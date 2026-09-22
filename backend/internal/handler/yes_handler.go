package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/diarra/backend/internal/storage"
)

// yes_handler.go — liaison DIARRA <-> YES Messaging (achat conversationnel
// "in-chat", voir migrations/037_yes_integration.sql et
// middleware/yes_client.go). YES gère l'état de la conversation ; DIARRA
// reste seul responsable de l'encaissement (PawaPay/PayPal), de la livraison
// (URL signée MinIO) et de la réputation vendeur.
//
// Isolé dans son propre fichier, comme gateway_relay.go, pour ne pas toucher
// à la logique déjà en place de sale_handler.go/webhook_handler.go — cette
// intégration réutilise leurs briques existantes (payment.PawaPayClient,
// calcul de commission, storage.S3Storage) sans les dupliquer.

type YesHandler struct {
	yesRepo       *repository.YesIntegrationRepo
	saleRepo      *repository.SaleRepo
	productRepo   *repository.ProductRepo
	referralRepo  *repository.ReferralRepo
	settingsRepo  *repository.SettingsRepo
	pawapay       *payment.PawaPayClient
	storage       *storage.S3Storage
	notifications *email.NotificationService
	// deliveryFulfillURL — endpoint YES qui reçoit la notification de
	// livraison (POST .../yes/delivery/fulfill côté YES, voir contrat
	// d'intégration). Un seul partenaire YES, contrairement à
	// gateway_clients où chaque client a sa propre callback_url — pas besoin
	// de la stocker en base pour un seul destinataire fixe.
	deliveryFulfillURL string
	// frontendURL — sert de base à ReturnUrl (PawaPay EXIGE une URL http(s)
	// valide, jamais un token brut — voir createPaymentPage). YES ne sert
	// aucune page web de retour ; l'acheteur atterrit ici après paiement
	// mobile money, DIARRA affiche juste une confirmation minimale (le vrai
	// statut de la conversation reste piloté par YES via ses propres appels).
	frontendURL string
}

func NewYesHandler(
	yesRepo *repository.YesIntegrationRepo,
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
	referralRepo *repository.ReferralRepo,
	settingsRepo *repository.SettingsRepo,
	pawapay *payment.PawaPayClient,
	storageSvc *storage.S3Storage,
	notifications *email.NotificationService,
	deliveryFulfillURL string,
	frontendURL string,
) *YesHandler {
	return &YesHandler{
		yesRepo: yesRepo, saleRepo: saleRepo, productRepo: productRepo,
		referralRepo: referralRepo, settingsRepo: settingsRepo,
		pawapay: pawapay, storage: storageSvc, notifications: notifications,
		deliveryFulfillURL: deliveryFulfillURL, frontendURL: frontendURL,
	}
}

// LookupClient — passé à middleware.RequireYesClient (voir main.go). Ne
// résout que des clients actifs (filtré dans le repo), comme LookupClient
// gateway.
func (h *YesHandler) LookupClient(ctx context.Context, apiKey string) (middleware.YesClientLookup, error) {
	c, err := h.yesRepo.FindClientByAPIKey(ctx, apiKey)
	if err != nil {
		return middleware.YesClientLookup{}, err
	}
	return middleware.YesClientLookup{ID: c.ID, APISecretHash: c.APISecretHash}, nil
}

// resolveReferralLink — même validation que SaleHandler.Create (produit
// correspondant, pas d'auto-référencement acheteur ni vendeur), mais
// silencieuse : un lien invalide ne doit pas faire échouer un micro-ticket,
// seulement priver la session de commission d'affiliation.
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

// InitiateSession — POST /api/yes/session/initiate (YES -> DIARRA) :
// encaisse le micro-ticket (600 FCFA), crée la session de liaison. N'ouvre
// PAS elle-même la conversation côté YES (YES l'a déjà créée avant cet
// appel, c'est lui qui pilote son propre état) — DIARRA se contente de
// confirmer que le paiement est en cours et de renvoyer l'URL de paiement
// hébergée (mobile money, seul canal pour un montant aussi faible — voir
// model.MicroTicketAmountCFA).
func (h *YesHandler) InitiateSession(w http.ResponseWriter, r *http.Request) {
	var input model.InitiateSessionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.YesConversationID == "" || input.ProductID == "" || input.BuyerID == "" || input.Country == "" {
		http.Error(w, `{"error":"missing_required_fields"}`, http.StatusBadRequest)
		return
	}
	if h.pawapay == nil {
		http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
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
	if product.PriceCFA <= model.MicroTicketAmountCFA {
		http.Error(w, `{"error":"price_too_low_for_conversational_flow"}`, http.StatusBadRequest)
		return
	}

	var referralLinkID *string
	if input.ReferralLinkID != nil {
		referralLinkID = h.resolveReferralLink(r.Context(), *input.ReferralLinkID, input.BuyerID, product.ID)
	}

	// Vente "micro-ticket" : une Sale normale, montant fixe, commission
	// calculée EXACTEMENT comme une vente classique (même taux
	// SettingCommissionRatePct) — voir SaleHandler.Create pour la même
	// logique, dupliquée ici volontairement car il n'existe pas encore de
	// fonction partagée extraite (à faire si un 3e appelant apparaît).
	rate := h.settingsRepo.GetFloat(r.Context(), model.SettingCommissionRatePct, service.DefaultPlatformFeePct)
	rate = service.EffectivePlatformFeePct(model.MicroTicketAmountCFA, rate)
	platformFee := int(float64(model.MicroTicketAmountCFA) * rate / 100.0)
	checkoutToken := newUUID()
	microSale := &model.Sale{
		ProductID:        product.ID,
		BuyerID:          input.BuyerID,
		BuyerName:        "Acheteur YES",
		Country:          &input.Country,
		AmountCFA:        model.MicroTicketAmountCFA,
		PlatformFeeCFA:   platformFee,
		VendorAmountCFA:  model.MicroTicketAmountCFA - platformFee,
		PaymentProvider:  "pawapay",
		PaymentReference: newUUID(),
		CheckoutToken:    &checkoutToken,
		Status:           string(model.SalePending),
	}

	created, err := h.saleRepo.Create(r.Context(), microSale)
	if err != nil {
		http.Error(w, `{"error":"sale_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	page, err := h.createPaymentPage(r.Context(), created, product, input.Country)
	if err != nil {
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}

	session, err := h.yesRepo.CreateSession(r.Context(), &model.ConversationalSession{
		YesConversationID: input.YesConversationID,
		ProductID:         product.ID,
		BuyerID:           input.BuyerID,
		SellerID:          product.VendorID,
		ReferralLinkID:    referralLinkID,
		MicroTicketSaleID: &created.ID,
		Status:            model.SessionMicroTicketPaid,
	})
	if err != nil {
		if repository.IsUniqueViolation(err) {
			http.Error(w, `{"error":"session_already_exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"session_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"session_id":            session.ID,
		"micro_ticket_sale_id":  created.ID,
		"payment_redirect_url":  page.RedirectUrl,
	})
}

// createPaymentPage — même construction que SaleHandler.initiatePaymentPage
// (PawaPay Payment Page hébergée), reprise ici car ce handler n'a pas accès
// à SaleHandler directement. ReturnUrl réutilise /checkout/return (page
// DIARRA existante, affiche juste une confirmation minimale) — PawaPay
// EXIGE une URL http(s) valide, jamais un token brut. YES ne sert aucune
// page web ; c'est lui qui pilote le retour dans la conversation via ses
// propres appels (in-chat checkout), cette URL n'est qu'un filet de
// sécurité pour l'acheteur qui atterrit dessus après paiement mobile money.
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

// InChatCheckout — POST /api/yes/checkout/in-chat (YES -> DIARRA) : le
// client a cliqué "Payer et débloquer" dans le chat. Encaisse le SOLDE
// (product.PriceCFA - MicroTicketAmountCFA) via une nouvelle Sale.
func (h *YesHandler) InChatCheckout(w http.ResponseWriter, r *http.Request) {
	var input model.InChatCheckoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.SessionID == "" {
		http.Error(w, `{"error":"missing_session_id"}`, http.StatusBadRequest)
		return
	}
	if h.pawapay == nil {
		http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	session, err := h.yesRepo.FindSessionByID(r.Context(), input.SessionID)
	if err != nil {
		http.Error(w, `{"error":"session_not_found"}`, http.StatusNotFound)
		return
	}
	if session.Status != model.SessionMicroTicketPaid {
		http.Error(w, `{"error":"session_not_payable","detail":"session déjà complétée ou annulée"}`, http.StatusConflict)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), session.ProductID)
	if err != nil {
		http.Error(w, `{"error":"product_not_found"}`, http.StatusNotFound)
		return
	}

	balance := product.PriceCFA - model.MicroTicketAmountCFA
	if balance <= 0 {
		http.Error(w, `{"error":"invalid_balance","detail":"le prix du produit doit dépasser le montant du micro-ticket"}`, http.StatusConflict)
		return
	}

	rate := h.settingsRepo.GetFloat(r.Context(), model.SettingCommissionRatePct, service.DefaultPlatformFeePct)
	rate = service.EffectivePlatformFeePct(balance, rate)
	platformFee := int(float64(balance) * rate / 100.0)
	checkoutToken := newUUID()
	sale := &model.Sale{
		ProductID:        product.ID,
		BuyerID:          session.BuyerID,
		BuyerName:        "Acheteur YES",
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

	// Le pays n'est pas reçu du client ici (contrairement à InitiateSession) —
	// XOF par défaut, cohérent avec createPaymentPage quand country est vide.
	page, err := h.createPaymentPage(r.Context(), created, product, "")
	if err != nil {
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}

	if err := h.yesRepo.CompleteSession(r.Context(), session.ID, created.ID); err != nil {
		log.Printf("yes in-chat checkout: session %s non mise à jour après vente %s: %v", session.ID, created.ID, err)
	}

	writeJSON(w, map[string]interface{}{
		"sale_id":              created.ID,
		"payment_redirect_url": page.RedirectUrl,
		"status":               "processing",
	})
}

// NotifyDelivery — appelé quand une vente issue d'une session conversationnelle
// est confirmée payée (voir webhook_handler.go, à brancher dans
// ConfirmPaidSale : si sale.ID correspond à une conversational_sessions.sale_id,
// appeler ceci après la confirmation normale). Génère l'URL signée et notifie
// YES — jamais le paiement lui-même, qui suit le flux PawaPay standard.
func (h *YesHandler) NotifyDelivery(ctx context.Context, saleID string) {
	session, err := h.yesRepo.FindSessionBySaleID(ctx, saleID)
	if err != nil {
		return // pas une vente issue du flux conversationnel, rien à faire
	}
	product, err := h.productRepo.FindByID(ctx, session.ProductID)
	if err != nil {
		log.Printf("yes notify delivery: produit introuvable pour session=%s: %v", session.ID, err)
		return
	}
	url, err := h.storage.GenerateSignedURL(ctx, product.FileKey, 5*time.Minute)
	if err != nil {
		log.Printf("yes notify delivery: URL signée échouée pour session=%s: %v", session.ID, err)
		return
	}
	h.notifyDelivery(ctx, session, url)
}

func (h *YesHandler) notifyDelivery(ctx context.Context, session *model.ConversationalSession, downloadURL string) {
	if h.deliveryFulfillURL == "" {
		return
	}
	payload := map[string]interface{}{
		"yes_conversation_id": session.YesConversationID,
		"session_id":          session.ID,
		"download_url":        downloadURL,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, h.deliveryFulfillURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set("X-Timestamp", ts)

	// Signé avec le secret DIARRA -> YES (côté sortant) — distinct du secret
	// YES -> DIARRA vérifié par middleware.RequireYesClient (entrant), voir
	// le commentaire de yes_client.go pour le principe de séparation.
	outboundSecret := h.settingsRepo.Get(ctx, "yes_outbound_hmac_secret", "")
	if outboundSecret == "" {
		log.Printf("yes notify delivery: secret sortant introuvable, notification non envoyée pour session=%s", session.ID)
		return
	}
	mac := hmac.New(sha256.New, []byte(outboundSecret))
	mac.Write([]byte(ts + "." + string(body)))
	req.Header.Set("X-Signature-SHA256", hex.EncodeToString(mac.Sum(nil)))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("yes notify delivery: %s injoignable pour session=%s: %v", h.deliveryFulfillURL, session.ID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("yes notify delivery: %s a répondu %d pour session=%s", h.deliveryFulfillURL, resp.StatusCode, session.ID)
	}
}

// SubmitVendorReview — POST /api/yes/vendors/review (YES -> DIARRA).
func (h *YesHandler) SubmitVendorReview(w http.ResponseWriter, r *http.Request) {
	var input model.VendorReviewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.SessionID == "" || input.Rating < 1 || input.Rating > 5 {
		http.Error(w, `{"error":"invalid_review"}`, http.StatusBadRequest)
		return
	}

	session, err := h.yesRepo.FindSessionByID(r.Context(), input.SessionID)
	if err != nil {
		http.Error(w, `{"error":"session_not_found"}`, http.StatusNotFound)
		return
	}
	if session.Status != model.SessionCompleted {
		http.Error(w, `{"error":"session_not_completed","detail":"un avis ne peut être laissé qu'après un achat abouti"}`, http.StatusConflict)
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

	writeJSON(w, map[string]interface{}{"review": review})
}
