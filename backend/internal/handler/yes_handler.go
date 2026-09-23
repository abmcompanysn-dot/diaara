package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/diarra/backend/internal/storage"
	"github.com/go-chi/chi/v5"
)

// yes_handler.go — achat conversationnel "in-chat" via YES.abmcy Business.
// DIARRA est le SEUL appelant des 5 endpoints YES Business (session/initiate,
// session/{id}/status, session/{id}/send-offer, session/{id}/review,
// delivery/fulfill) — YES n'appelle jamais DIARRA en retour (voir doc
// d'intégration fournie par YES le 2026-09-22, JOURNAL-MODIFICATIONS.md).
// DIARRA reste seul responsable de l'encaissement (PawaPay, micro-ticket +
// solde), de la livraison (URL signée MinIO) et du recueil des avis
// (l'interface YES n'a pas de composant de notation propre).
//
// Toutes les routes de ce handler sont des routes DIARRA classiques
// (authentification JWT normale, acheteur ou vendeur connecté).

type YesHandler struct {
	yesRepo       *repository.YesIntegrationRepo
	saleRepo      *repository.SaleRepo
	productRepo   *repository.ProductRepo
	userRepo      *repository.UserRepo
	referralRepo  *repository.ReferralRepo
	settingsRepo  *repository.SettingsRepo
	pawapay       *payment.PawaPayClient
	yesBusiness   *payment.YesBusinessClient
	storage       *storage.S3Storage
	notifications *email.NotificationService
	frontendURL   string
}

func NewYesHandler(
	yesRepo *repository.YesIntegrationRepo,
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
	userRepo *repository.UserRepo,
	referralRepo *repository.ReferralRepo,
	settingsRepo *repository.SettingsRepo,
	pawapay *payment.PawaPayClient,
	yesBusiness *payment.YesBusinessClient,
	storageSvc *storage.S3Storage,
	notifications *email.NotificationService,
	frontendURL string,
) *YesHandler {
	return &YesHandler{
		yesRepo: yesRepo, saleRepo: saleRepo, productRepo: productRepo, userRepo: userRepo,
		referralRepo: referralRepo, settingsRepo: settingsRepo,
		pawapay: pawapay, yesBusiness: yesBusiness, storage: storageSvc, notifications: notifications,
		frontendURL: frontendURL,
	}
}

// resolveReferralLink — même validation que SaleHandler.Create (produit
// correspondant, pas d'auto-référencement acheteur), mais silencieuse : un
// lien invalide ne doit pas faire échouer l'ouverture d'une conversation.
func (h *YesHandler) resolveReferralLink(ctx context.Context, referralLinkID, buyerID, productID string) *string {
	if referralLinkID == "" {
		return nil
	}
	link, err := h.referralRepo.FindByID(ctx, referralLinkID)
	if err != nil || link.ProductID != productID || link.CloserID == buyerID {
		return nil
	}
	id := link.ID
	return &id
}

// OpenConversation — POST /api/vendor-chat/open (acheteur connecté) : le
// clic "Discuter avec le vendeur" sur une fiche produit. Encaisse le
// micro-ticket (PawaPay) et crée une session "pending". L'appel à YES
// session/initiate n'intervient qu'APRÈS confirmation du paiement (voir
// OnSaleConfirmed, branché sur ConfirmPaidSale) — tant que le micro-ticket
// n'est pas payé, DIARRA n'a encore rien demandé à YES.
func (h *YesHandler) OpenConversation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var input model.OpenConversationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.ProductID == "" || input.Country == "" {
		http.Error(w, `{"error":"missing_required_fields"}`, http.StatusBadRequest)
		return
	}
	if h.pawapay == nil {
		http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	if h.yesBusiness == nil {
		http.Error(w, `{"error":"yes_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), input.ProductID)
	if err != nil {
		http.Error(w, `{"error":"product_not_found"}`, http.StatusNotFound)
		return
	}
	if product.ModerationStatus != "approved" {
		http.Error(w, `{"error":"product_not_available"}`, http.StatusBadRequest)
		return
	}
	if product.VendorID == userID {
		http.Error(w, `{"error":"cannot_chat_with_self"}`, http.StatusBadRequest)
		return
	}
	if product.PriceCFA <= model.MicroTicketAmountCFA {
		http.Error(w, `{"error":"price_too_low_for_conversational_flow"}`, http.StatusBadRequest)
		return
	}

	referralLinkID := h.resolveReferralLink(r.Context(), safeStr(input.ReferralLinkID), userID, product.ID)

	rate := h.settingsRepo.GetFloat(r.Context(), model.SettingCommissionRatePct, service.DefaultPlatformFeePct)
	rate = service.EffectivePlatformFeePct(model.MicroTicketAmountCFA, rate)
	platformFee := int(float64(model.MicroTicketAmountCFA) * rate / 100.0)
	checkoutToken := newUUID()
	microSale := &model.Sale{
		ProductID:        product.ID,
		BuyerID:          userID,
		BuyerName:        "Acheteur",
		Country:          &input.Country,
		AmountCFA:        model.MicroTicketAmountCFA,
		PlatformFeeCFA:   platformFee,
		VendorAmountCFA:  model.MicroTicketAmountCFA - platformFee,
		PaymentProvider:  "pawapay",
		PaymentReference: newUUID(),
		CheckoutToken:    &checkoutToken,
		ReferralLinkID:   referralLinkID,
		Status:           string(model.SalePending),
	}
	created, err := h.saleRepo.Create(r.Context(), microSale)
	if err != nil {
		http.Error(w, `{"error":"sale_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	if _, err := h.yesRepo.CreatePendingSession(r.Context(), &model.ConversationalSession{
		ProductID:         product.ID,
		BuyerID:           userID,
		SellerID:          product.VendorID,
		ReferralLinkID:    referralLinkID,
		MicroTicketSaleID: created.ID,
	}); err != nil {
		http.Error(w, `{"error":"session_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	page, err := h.createPaymentPage(r.Context(), created, product, input.Country)
	if err != nil {
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}

	writeJSON(w, map[string]interface{}{
		"micro_ticket_sale_id": created.ID,
		"payment_redirect_url": page.RedirectUrl,
	})
}

// createPaymentPage — même construction que SaleHandler.initiatePaymentPage
// (PawaPay Payment Page hébergée). ReturnUrl pointe vers /checkout/return
// (page DIARRA existante) — PawaPay exige une URL http(s) valide.
func (h *YesHandler) createPaymentPage(ctx context.Context, sale *model.Sale, product *model.Product, country string) (*payment.PaymentPageResponse, error) {
	reason := payment.SanitizePaymentReason(product.Title, 50)
	currency := payment.CountryCurrency[country]
	if currency == "" {
		currency = "XOF"
	}
	amount, err := payment.ConvertFromXOF(sale.AmountCFA, currency)
	if err != nil {
		return nil, err
	}
	returnURL := h.frontendURL + "/checkout/return?token=" + *sale.CheckoutToken
	return h.pawapay.CreatePaymentPage(ctx, payment.PaymentPageRequest{
		DepositId: sale.PaymentReference,
		ReturnUrl: returnURL,
		AmountDetails: payment.AmountDetails{
			Amount:   amount,
			Currency: currency,
		},
		Country:         country,
		Reason:          reason,
		CustomerMessage: "PAIEMENT DIARRA",
		Language:        "FR",
		Metadata: []payment.MetadataItem{
			{"saleId": sale.ID},
			{"product": product.Title},
		},
	})
}

func buyerDisplayName(u *model.User) string {
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	return "Acheteur DIARRA"
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// OnSaleConfirmed — appelé par ConfirmPaidSale (webhook_handler.go) pour
// TOUTE vente confirmée payée, pas seulement celles du flux conversationnel
// — no-op silencieux si sale.ID ne correspond à aucune conversational_sessions
// (micro_ticket_sale_id ni sale_id). Distingue les deux étapes possibles :
//   - micro-ticket confirmé (session "pending") -> appelle YES session/initiate
//   - solde confirmé (session "offer_sent", sale_id déjà posé) -> génère
//     l'URL signée et appelle YES delivery/fulfill
func (h *YesHandler) OnSaleConfirmed(ctx context.Context, sale *model.Sale) {
	if h.yesBusiness == nil {
		return
	}
	if session, err := h.yesRepo.FindSessionByMicroTicketSaleID(ctx, sale.ID); err == nil {
		h.openSessionOnYes(ctx, session)
		return
	}
	if session, err := h.yesRepo.FindSessionBySaleID(ctx, sale.ID); err == nil {
		h.fulfillDelivery(ctx, session)
		return
	}
	// Ni l'un ni l'autre : vente classique, hors flux conversationnel.
}

// openSessionOnYes — micro-ticket confirmé payé : DIARRA appelle enfin YES
// session/initiate (jamais avant, voir doc : "un paiement échoué en amont ne
// produit aucun webhook" — DIARRA ne veut pas ouvrir de session YES pour un
// paiement qui pourrait encore échouer).
func (h *YesHandler) openSessionOnYes(ctx context.Context, session *model.ConversationalSession) {
	product, err := h.productRepo.FindByID(ctx, session.ProductID)
	if err != nil {
		log.Printf("yes open session: produit introuvable pour session=%s: %v", session.ID, err)
		return
	}
	seller, err := h.userRepo.FindByID(ctx, session.SellerID)
	if err != nil || seller.Email == "" {
		log.Printf("yes open session: vendeur/email introuvable pour session=%s: %v", session.ID, err)
		return
	}
	buyer, err := h.userRepo.FindByID(ctx, session.BuyerID)
	if err != nil {
		log.Printf("yes open session: acheteur introuvable pour session=%s: %v", session.ID, err)
		return
	}

	resp, err := h.yesBusiness.InitiateSession(ctx, payment.InitiateSessionRequest{
		ProductID:         product.ID,
		ProductName:       product.Title,
		SellerHandle:      seller.Email, // handle YES = email DIARRA du vendeur (voir doc 2026-09-22)
		BuyerExternalID:   buyer.ID,
		BuyerDisplayName:  buyerDisplayName(buyer),
		MicroTicketAmount: model.MicroTicketAmountCFA,
		FinalAmount:       product.PriceCFA - model.MicroTicketAmountCFA,
		Currency:          "XOF",
	})
	if err != nil {
		log.Printf("yes open session: session/initiate échoué pour session=%s: %v", session.ID, err)
		return
	}
	if err := h.yesRepo.OpenSessionAfterPayment(ctx, session.ID, resp.SessionID, resp.ChatURL); err != nil {
		log.Printf("yes open session: statut DIARRA non mis à jour pour session=%s (yes_session_id=%s): %v", session.ID, resp.SessionID, err)
	}
}

// fulfillDelivery — solde confirmé payé : génère l'URL signée et appelle YES
// delivery/fulfill.
func (h *YesHandler) fulfillDelivery(ctx context.Context, session *model.ConversationalSession) {
	if session.YesSessionID == nil {
		log.Printf("yes fulfill delivery: session=%s sans yes_session_id, incohérence (jamais ouverte côté YES)", session.ID)
		return
	}
	product, err := h.productRepo.FindByID(ctx, session.ProductID)
	if err != nil {
		log.Printf("yes fulfill delivery: produit introuvable pour session=%s: %v", session.ID, err)
		return
	}
	expiry := 5 * time.Minute
	url, err := h.storage.GenerateSignedURL(ctx, product.FileKey, expiry)
	if err != nil {
		log.Printf("yes fulfill delivery: URL signée échouée pour session=%s: %v", session.ID, err)
		return
	}
	err = h.yesBusiness.FulfillDelivery(ctx, payment.FulfillDeliveryRequest{
		SessionID:         *session.YesSessionID,
		Status:            "COMPLETED",
		DeliveryURL:       url,
		DeliveryExpiresAt: time.Now().Add(expiry).UTC().Format(time.RFC3339),
	})
	if err != nil {
		log.Printf("yes fulfill delivery: delivery/fulfill échoué pour session=%s: %v", session.ID, err)
		return
	}
	if err := h.yesRepo.CompleteSession(ctx, session.ID); err != nil {
		log.Printf("yes fulfill delivery: statut DIARRA non mis à jour pour session=%s: %v", session.ID, err)
	}
}

// SendOffer — POST /api/vendor-chat/{id}/send-offer (vendeur connecté) : le
// vendeur décide que la conversation est prête à passer au paiement du
// solde. Appelle YES send-offer.
func (h *YesHandler) SendOffer(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if h.yesBusiness == nil {
		http.Error(w, `{"error":"yes_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	sessionID := chi.URLParam(r, "id")
	session, err := h.yesRepo.FindSessionByID(r.Context(), sessionID)
	if err != nil {
		http.Error(w, `{"error":"session_not_found"}`, http.StatusNotFound)
		return
	}
	if session.SellerID != userID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if session.Status != model.SessionOpened || session.YesSessionID == nil {
		http.Error(w, `{"error":"session_not_open"}`, http.StatusConflict)
		return
	}

	resp, err := h.yesBusiness.SendOffer(r.Context(), *session.YesSessionID)
	if err != nil {
		http.Error(w, `{"error":"send_offer_failed"}`, http.StatusBadGateway)
		return
	}
	if err := h.yesRepo.SetOfferSent(r.Context(), session.ID); err != nil {
		log.Printf("yes send-offer: statut DIARRA non mis à jour pour session=%s: %v", session.ID, err)
	}

	writeJSON(w, map[string]interface{}{"status": resp.Status, "checkout_url": resp.CheckoutURL})
}

// InitiateBalanceCheckout — POST /api/vendor-chat/{id}/checkout (acheteur
// connecté) : l'acheteur clique "Payer" pour le solde. Encaisse le SOLDE via
// PawaPay, comme OpenConversation encaisse le micro-ticket.
func (h *YesHandler) InitiateBalanceCheckout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	sessionID := chi.URLParam(r, "id")
	session, err := h.yesRepo.FindSessionByID(r.Context(), sessionID)
	if err != nil {
		http.Error(w, `{"error":"session_not_found"}`, http.StatusNotFound)
		return
	}
	if session.BuyerID != userID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if session.Status != model.SessionOfferSent {
		http.Error(w, `{"error":"session_not_payable"}`, http.StatusConflict)
		return
	}

	var input struct {
		Country string `json:"country"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	if input.Country == "" {
		input.Country = "SEN"
	}

	product, err := h.productRepo.FindByID(r.Context(), session.ProductID)
	if err != nil {
		http.Error(w, `{"error":"product_not_found"}`, http.StatusNotFound)
		return
	}
	balance := product.PriceCFA - model.MicroTicketAmountCFA
	if balance <= 0 {
		http.Error(w, `{"error":"invalid_balance"}`, http.StatusConflict)
		return
	}

	rate := h.settingsRepo.GetFloat(r.Context(), model.SettingCommissionRatePct, service.DefaultPlatformFeePct)
	rate = service.EffectivePlatformFeePct(balance, rate)
	platformFee := int(float64(balance) * rate / 100.0)
	checkoutToken := newUUID()
	sale := &model.Sale{
		ProductID:        product.ID,
		BuyerID:          session.BuyerID,
		BuyerName:        "Acheteur",
		Country:          &input.Country,
		AmountCFA:        balance,
		PlatformFeeCFA:   platformFee,
		VendorAmountCFA:  balance - platformFee,
		PaymentProvider:  "pawapay",
		PaymentReference: newUUID(),
		CheckoutToken:    &checkoutToken,
		Status:           string(model.SalePending),
	}
	if session.ReferralLinkID != nil {
		link, err := h.referralRepo.FindByID(r.Context(), *session.ReferralLinkID)
		if err == nil {
			closerFee := int(float64(balance) * link.CommissionPct / 100.0)
			rest := balance - platformFee
			if closerFee > rest {
				closerFee = rest
			}
			sale.ReferralLinkID = session.ReferralLinkID
			sale.CloserCommissionCFA = closerFee
			sale.VendorAmountCFA = rest - closerFee
		}
	}

	created, err := h.saleRepo.Create(r.Context(), sale)
	if err != nil {
		http.Error(w, `{"error":"sale_creation_failed"}`, http.StatusInternalServerError)
		return
	}
	page, err := h.createPaymentPage(r.Context(), created, product, input.Country)
	if err != nil {
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}
	// La session ne passe "completed" qu'après confirmation réelle du
	// paiement (voir OnSaleConfirmed, branché sur ConfirmPaidSale) — sale_id
	// est posé maintenant pour que FindSessionBySaleID le retrouve ensuite.
	if err := h.yesRepo.SetBalanceSale(r.Context(), session.ID, created.ID); err != nil {
		log.Printf("yes balance checkout: sale_id non posé pour session=%s: %v", session.ID, err)
	}

	writeJSON(w, map[string]interface{}{
		"sale_id":              created.ID,
		"payment_redirect_url": page.RedirectUrl,
	})
}

// SubmitVendorReview — POST /api/vendor-chat/{id}/review (acheteur
// connecté) : DIARRA recueille l'avis puis le relaie à YES.
func (h *YesHandler) SubmitVendorReview(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var input model.VendorReviewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Rating < 1 || input.Rating > 5 {
		http.Error(w, `{"error":"invalid_review"}`, http.StatusBadRequest)
		return
	}
	sessionID := chi.URLParam(r, "id")
	session, err := h.yesRepo.FindSessionByID(r.Context(), sessionID)
	if err != nil {
		http.Error(w, `{"error":"session_not_found"}`, http.StatusNotFound)
		return
	}
	if session.BuyerID != userID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if session.Status != model.SessionCompleted || session.YesSessionID == nil {
		http.Error(w, `{"error":"session_not_completed"}`, http.StatusConflict)
		return
	}

	review, err := h.yesRepo.CreateReview(r.Context(), &model.VendorReview{
		SessionID: session.ID,
		VendorID:  session.SellerID,
		BuyerID:   session.BuyerID,
		Rating:    input.Rating,
		Comment:   input.Comment,
	})
	if err != nil {
		if repository.IsUniqueViolation(err) {
			http.Error(w, `{"error":"review_already_submitted"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"review_save_failed"}`, http.StatusInternalServerError)
		return
	}

	if h.yesBusiness != nil {
		yesSessionID := *session.YesSessionID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := h.yesBusiness.SubmitReview(ctx, yesSessionID, payment.SubmitReviewRequest{
				Rating: input.Rating, Comment: input.Comment,
			}); err != nil {
				log.Printf("yes review: relais vers YES échoué pour session=%s: %v", session.ID, err)
			}
		}()
	}

	writeJSON(w, map[string]interface{}{"review": review})
}
