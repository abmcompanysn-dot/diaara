package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
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
	paydunya      *payment.PayDunyaClient
	yesBusiness   *payment.YesBusinessClient
	storage       *storage.S3Storage
	notifications *email.NotificationService
	frontendURL   string
	// apiURL — domaine public du backend (ex. https://api.diarra.app),
	// utilisé pour construire product_image_url envoyé à YES
	// session/initiate : doit être une URL publique chargeable dans un
	// <img> sans authentification (voir doc YES Business 2026-09-23,
	// GET /api/products/{id}/cover est déjà public — OptionalAuth).
	apiURL string
}

func NewYesHandler(
	yesRepo *repository.YesIntegrationRepo,
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
	userRepo *repository.UserRepo,
	referralRepo *repository.ReferralRepo,
	settingsRepo *repository.SettingsRepo,
	pawapay *payment.PawaPayClient,
	paydunya *payment.PayDunyaClient,
	yesBusiness *payment.YesBusinessClient,
	storageSvc *storage.S3Storage,
	notifications *email.NotificationService,
	frontendURL string,
	apiURL string,
) *YesHandler {
	return &YesHandler{
		yesRepo: yesRepo, saleRepo: saleRepo, productRepo: productRepo, userRepo: userRepo,
		referralRepo: referralRepo, settingsRepo: settingsRepo,
		pawapay: pawapay, paydunya: paydunya, yesBusiness: yesBusiness, storage: storageSvc, notifications: notifications,
		frontendURL: frontendURL, apiURL: strings.TrimSuffix(apiURL, "/"),
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

// microTicketAmount — montant du ticket d'entrée, réglage admin modifiable
// (voir model.SettingYesMicroTicketAmountCFA), pas une constante figée.
func (h *YesHandler) microTicketAmount(ctx context.Context) int {
	return int(h.settingsRepo.GetFloat(ctx, model.SettingYesMicroTicketAmountCFA, model.DefaultYesMicroTicketAmountCFA))
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
	if input.ProductID == "" || input.Country == "" || input.Phone == "" || input.Operator == "" {
		http.Error(w, `{"error":"missing_required_fields"}`, http.StatusBadRequest)
		return
	}
	if h.yesBusiness == nil {
		http.Error(w, `{"error":"yes_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	if isOperatorBlocked(r.Context(), h.settingsRepo, input.Operator) {
		http.Error(w, `{"error":"operator_unavailable"}`, http.StatusBadRequest)
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
	// Bêta réservée aux vendeurs choisis par un admin (voir migration 039,
	// AdminHandler.SetYesChatEnabled) — pas ouvert à tout le catalogue.
	seller, err := h.userRepo.FindByID(r.Context(), product.VendorID)
	if err != nil || !seller.YesChatEnabled {
		http.Error(w, `{"error":"vendor_chat_not_available"}`, http.StatusForbidden)
		return
	}

	microTicketAmount := h.microTicketAmount(r.Context())
	if product.PriceCFA <= microTicketAmount {
		http.Error(w, `{"error":"price_too_low_for_conversational_flow"}`, http.StatusBadRequest)
		return
	}

	referralLinkID := h.resolveReferralLink(r.Context(), safeStr(input.ReferralLinkID), userID, product.ID)

	rate := h.settingsRepo.GetFloat(r.Context(), model.SettingCommissionRatePct, service.DefaultPlatformFeePct)
	rate = service.EffectivePlatformFeePct(microTicketAmount, rate)
	platformFee := int(float64(microTicketAmount) * rate / 100.0)
	providerName := resolveMobileMoneyProvider(r.Context(), h.settingsRepo, input.Country, input.Operator)
	checkoutToken := newUUID()
	microSale := &model.Sale{
		ProductID:        product.ID,
		BuyerID:          userID,
		BuyerName:        "Acheteur",
		Country:          &input.Country,
		AmountCFA:        microTicketAmount,
		PlatformFeeCFA:   platformFee,
		VendorAmountCFA:  microTicketAmount - platformFee,
		PaymentProvider:  providerName,
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

	buyerEmail := ""
	if buyer, err := h.userRepo.FindByID(r.Context(), userID); err == nil {
		buyerEmail = buyer.Email
	}
	returnURL := h.frontendURL + "/checkout/return?token=" + *created.CheckoutToken
	redirectURL, err := initiateMobileMoneyDeposit(r.Context(), h.pawapay, h.paydunya, h.saleRepo,
		created.ID, created.PaymentReference, product.Title, created.BuyerName, buyerEmail, created.AmountCFA,
		providerName, input.Operator, input.Phone, returnURL, input.OTP)
	if err != nil {
		log.Printf("yes micro-ticket payment_init_failed sale=%s: %v", created.ID, err)
		h.saleRepo.UpdateStatus(r.Context(), created.ID, string(model.SaleFailed))
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}

	writeJSON(w, map[string]interface{}{
		"micro_ticket_sale_id": created.ID,
		"payment_redirect_url": redirectURL,
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

// IsMicroTicketSale — vrai si sale.ID correspond au micro-ticket d'une
// session conversationnelle YES (voir WebhookHandler.notifyPaid, qui doit
// alors ne PAS joindre le fichier complet du produit à l'email de
// confirmation : le micro-ticket ne paie que l'accès à la conversation,
// jamais le produit — sa livraison n'arrive qu'après paiement du solde,
// voir fulfillDelivery). Incident 2026-09-24 : l'email de confirmation du
// micro-ticket joignait le fichier complet, contournant le paiement du solde.
func (h *YesHandler) IsMicroTicketSale(ctx context.Context, saleID string) bool {
	_, err := h.yesRepo.FindSessionByMicroTicketSaleID(ctx, saleID)
	return err == nil
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

// productImageURL — URL publique de la couverture produit (GET /api/products/{id}/cover
// est déjà public, OptionalAuth), envoyée à YES pour le widget produit
// épinglé dans le fil de conversation. Vide si pas de couverture ou si
// h.apiURL n'est pas configuré — YES traite ce cas normalement (pas
// d'erreur, juste pas d'image, voir doc 2026-09-23).
func (h *YesHandler) productImageURL(product *model.Product) string {
	if product.CoverImageKey == nil || *product.CoverImageKey == "" || h.apiURL == "" {
		return ""
	}
	return h.apiURL + "/api/products/" + product.ID + "/cover"
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

	microTicketAmount := h.microTicketAmount(ctx)
	resp, err := h.yesBusiness.InitiateSession(ctx, payment.InitiateSessionRequest{
		ProductID:         product.ID,
		ProductName:       product.Title,
		ProductImageURL:   h.productImageURL(product), // widget produit dans le fil de conversation — voir doc 2026-09-23
		SellerHandle:      seller.Email,               // handle YES = email DIARRA du vendeur (voir doc 2026-09-22)
		BuyerExternalID:   buyer.ID,
		BuyerDisplayName:  buyerDisplayName(buyer),
		BuyerContactEmail: buyer.Email, // pour les notifications YES (conversation démarrée, offre) — voir doc 2026-09-23
		MicroTicketAmount: microTicketAmount,
		FinalAmount:       product.PriceCFA - microTicketAmount,
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
		Country  string `json:"country"`
		Phone    string `json:"phone"`
		Operator string `json:"operator"`
		OTP      string `json:"otp,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	if input.Country == "" {
		input.Country = "SEN"
	}
	if input.Phone == "" || input.Operator == "" {
		http.Error(w, `{"error":"missing_required_fields"}`, http.StatusBadRequest)
		return
	}
	if isOperatorBlocked(r.Context(), h.settingsRepo, input.Operator) {
		http.Error(w, `{"error":"operator_unavailable"}`, http.StatusBadRequest)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), session.ProductID)
	if err != nil {
		http.Error(w, `{"error":"product_not_found"}`, http.StatusNotFound)
		return
	}
	balance := product.PriceCFA - h.microTicketAmount(r.Context())
	if balance <= 0 {
		http.Error(w, `{"error":"invalid_balance"}`, http.StatusConflict)
		return
	}

	rate := h.settingsRepo.GetFloat(r.Context(), model.SettingCommissionRatePct, service.DefaultPlatformFeePct)
	rate = service.EffectivePlatformFeePct(balance, rate)
	platformFee := int(float64(balance) * rate / 100.0)
	providerName := resolveMobileMoneyProvider(r.Context(), h.settingsRepo, input.Country, input.Operator)
	checkoutToken := newUUID()
	sale := &model.Sale{
		ProductID:        product.ID,
		BuyerID:          session.BuyerID,
		BuyerName:        "Acheteur",
		Country:          &input.Country,
		AmountCFA:        balance,
		PlatformFeeCFA:   platformFee,
		VendorAmountCFA:  balance - platformFee,
		PaymentProvider:  providerName,
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
	buyerEmail := ""
	if buyer, err := h.userRepo.FindByID(r.Context(), session.BuyerID); err == nil {
		buyerEmail = buyer.Email
	}
	returnURL := h.frontendURL + "/checkout/return?token=" + *created.CheckoutToken
	redirectURL, err := initiateMobileMoneyDeposit(r.Context(), h.pawapay, h.paydunya, h.saleRepo,
		created.ID, created.PaymentReference, product.Title, created.BuyerName, buyerEmail, created.AmountCFA,
		providerName, input.Operator, input.Phone, returnURL, input.OTP)
	if err != nil {
		log.Printf("yes balance payment_init_failed sale=%s: %v", created.ID, err)
		h.saleRepo.UpdateStatus(r.Context(), created.ID, string(model.SaleFailed))
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
		"payment_redirect_url": redirectURL,
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
