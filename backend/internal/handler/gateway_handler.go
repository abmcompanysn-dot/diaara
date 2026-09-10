package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

// GatewayHandler expose DIARRA comme passerelle de paiement pour des
// applications externes (ex. ABMCY Core) : elles créent des dépôts/versements/
// remboursements ici au lieu d'intégrer PawaPay/KPay/PayPal directement.
// Réutilise l'abstraction déjà unifiée entre les 3 agrégateurs (voir
// internal/payment/provider.go, PaymentProvider) — aucune nouvelle logique
// d'appel agrégateur, seulement une couche HTTP + persistance dédiée
// (gateway_transactions, migration 029) au-dessus.
//
// DIARRA garde la responsabilité des callbacks agrégateur (PawaPay/KPay/
// PayPal -> DIARRA, déjà géré par WebhookHandler) et les RELAIE vers le
// client externe (DIARRA -> callback_url du client, voir relayGatewayCallback
// dans webhook_handler.go) : le client n'a jamais besoin de recevoir un
// webhook agrégateur lui-même.
type GatewayHandler struct {
	gatewayRepo   *repository.GatewayRepo
	pawapay       *payment.PawaPayClient
	notifications *email.NotificationService
	frontendURL   string
}

func NewGatewayHandler(gatewayRepo *repository.GatewayRepo, pawapay *payment.PawaPayClient, notifications *email.NotificationService, frontendURL string) *GatewayHandler {
	return &GatewayHandler{
		gatewayRepo:   gatewayRepo,
		pawapay:       pawapay,
		notifications: notifications,
		frontendURL:   frontendURL,
	}
}

// LookupClient — adapte GatewayRepo au type attendu par
// middleware.RequireGatewayClient (évite un import de repository dans
// middleware, qui casserait la couche : middleware ne doit dépendre que
// d'interfaces/fonctions, jamais des types concrets du repository).
func (h *GatewayHandler) LookupClient(ctx context.Context, apiKeyHash string) (middleware.GatewayClientLookup, error) {
	c, err := h.gatewayRepo.FindClientByKeyHash(ctx, apiKeyHash)
	if err != nil {
		return middleware.GatewayClientLookup{}, err
	}
	return middleware.GatewayClientLookup{ID: c.ID, HMACSecretHash: c.HMACSecretHash}, nil
}

// gatewayUUID génère un UUID v4 correctement formaté (avec tirets) — PawaPay
// rejette un depositId qui n'a pas cette forme exacte (constaté : une chaîne
// hex de même longueur sans tirets échoue silencieusement côté PawaPay, sans
// message d'erreur exploitable, incident 2026-09-08). Même génération que
// SaleHandler.uuidString côté checkout DIARRA classique — dupliquée ici
// plutôt que partagée pour ne pas coupler gateway_handler.go à sale_handler.go
// pour une seule fonction utilitaire sans état.
func gatewayUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// CreateDeposit — POST /api/gateway/v1/deposits
// Ouvre une page de paiement PawaPay hébergée pour le client externe,
// exactement comme SaleHandler.initiatePaymentPage pour un achat DIARRA —
// seule la destination (gateway_transactions, pas sales) diffère.
func (h *GatewayHandler) CreateDeposit(w http.ResponseWriter, r *http.Request) {
	clientID := middleware.GetGatewayClientID(r.Context())

	var input model.GatewayCreateDepositInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.ClientRef == "" || input.AmountCFA <= 0 || input.Country == "" {
		http.Error(w, `{"error":"client_ref_amount_country_required"}`, http.StatusBadRequest)
		return
	}
	if h.pawapay == nil {
		http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	if payment.UnavailableCountries[input.Country] {
		http.Error(w, `{"error":"country_not_available"}`, http.StatusBadRequest)
		return
	}

	// Idempotence : un client_ref déjà utilisé pour un dépôt renvoie la
	// transaction existante plutôt que d'en créer une seconde — un client
	// externe qui retente un appel réseau échoué ne doit jamais créer deux
	// paiements pour la même commande.
	if existing, err := h.gatewayRepo.FindByClientRef(r.Context(), clientID, input.ClientRef, model.GatewayTxTypeDeposit); err == nil {
		writeGatewayTx(w, existing, http.StatusOK)
		return
	}

	var callbackURL *string
	if input.CallbackURL != "" {
		callbackURL = &input.CallbackURL
	}
	desc := input.Description
	var descPtr *string
	if desc != "" {
		descPtr = &desc
	}
	country := input.Country

	tx, err := h.gatewayRepo.CreateTransaction(r.Context(), repository.CreateGatewayTxParams{
		ClientID:    clientID,
		ClientRef:   input.ClientRef,
		Type:        model.GatewayTxTypeDeposit,
		Provider:    "pawapay",
		AmountCFA:   input.AmountCFA,
		Currency:    payment.CountryCurrency[input.Country],
		Country:     &country,
		Description: descPtr,
		CallbackURL: callbackURL,
	})
	if err != nil {
		http.Error(w, `{"error":"transaction_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	currency := tx.Currency
	if currency == "" {
		currency = "XOF"
	}
	amount, err := payment.ConvertFromXOF(input.AmountCFA, currency)
	if err != nil {
		http.Error(w, `{"error":"currency_conversion_failed"}`, http.StatusBadRequest)
		return
	}

	// depositId PawaPay = provider_ref, généré ici (pas l'UUID de la
	// transaction gateway) pour ne jamais exposer nos identifiants internes
	// à l'agrégateur — même principe que SaleHandler.Create.
	depositID := gatewayUUID()
	reason := payment.SanitizePaymentReason(input.Description, 50)
	if reason == "" {
		reason = "PAIEMENT"
	}
	// Pas de callbackUrl ici : POST /v2/paymentpage la rejette
	// (UNSUPPORTED_PARAMETER, voir le commentaire sur PaymentPageRequest) —
	// l'URL de callback PawaPay est configurée une fois dans leur dashboard,
	// commune à tous les dépôts (checkout DIARRA classique ET passerelle).
	// Où renvoyer le navigateur après paiement : l'URL fournie par le client
	// (ex. la page de suivi d'ABMCY Core), sinon la page générique
	// /gateway/return du frontend DIARRA. On garde depositId en query pour
	// que la page de retour puisse identifier la transaction.
	returnURL := h.frontendURL + "/gateway/return"
	if input.ReturnURL != "" {
		returnURL = input.ReturnURL
	}

	page, err := h.pawapay.CreatePaymentPage(r.Context(), payment.PaymentPageRequest{
		DepositId:       depositID,
		ReturnUrl:       returnURL,
		AmountDetails:   payment.AmountDetails{Amount: amount, Currency: currency},
		Country:         input.Country,
		Reason:          reason,
		CustomerMessage: "PAIEMENT",
		Language:        "FR",
		Metadata: []payment.MetadataItem{
			{"gatewayTxId": tx.ID},
		},
	})
	if err != nil || page == nil {
		reason := "payment_init_failed"
		_ = h.gatewayRepo.UpdateStatus(r.Context(), tx.ID, model.GatewayTxFailed, &reason)
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}
	if err := h.gatewayRepo.SetProviderRef(r.Context(), tx.ID, depositID); err != nil {
		http.Error(w, `{"error":"transaction_update_failed"}`, http.StatusInternalServerError)
		return
	}

	tx.ProviderRef = &depositID
	tx.RedirectURL = page.RedirectUrl
	writeGatewayTx(w, tx, http.StatusCreated)
}

// GetDeposit / GetPayout / GetRefund — GET /api/gateway/v1/{type}s/{client_ref}
// Toujours interrogé par client_ref (l'identifiant que le client externe
// connaît), jamais par l'UUID interne DIARRA.
func (h *GatewayHandler) GetTransaction(txType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID := middleware.GetGatewayClientID(r.Context())
		clientRef := chi.URLParam(r, "client_ref")
		tx, err := h.gatewayRepo.FindByClientRef(r.Context(), clientID, clientRef, txType)
		if err != nil {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		writeGatewayTx(w, tx, http.StatusOK)
	}
}

// CreatePayout — POST /api/gateway/v1/payouts
func (h *GatewayHandler) CreatePayout(w http.ResponseWriter, r *http.Request) {
	clientID := middleware.GetGatewayClientID(r.Context())

	var input model.GatewayCreatePayoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.ClientRef == "" || input.AmountCFA <= 0 || input.RecipientPhone == "" || input.RecipientOperator == "" || input.Country == "" {
		http.Error(w, `{"error":"missing_required_fields"}`, http.StatusBadRequest)
		return
	}
	if h.pawapay == nil {
		http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	if existing, err := h.gatewayRepo.FindByClientRef(r.Context(), clientID, input.ClientRef, model.GatewayTxTypePayout); err == nil {
		writeGatewayTx(w, existing, http.StatusOK)
		return
	}

	op, err := payment.ResolveOperator(input.Country, input.RecipientOperator)
	if err != nil {
		http.Error(w, `{"error":"unsupported_operator"}`, http.StatusBadRequest)
		return
	}
	msisdn, err := payment.NormalizePhone(op.DialCode, input.RecipientPhone)
	if err != nil {
		http.Error(w, `{"error":"invalid_phone_number"}`, http.StatusBadRequest)
		return
	}

	var callbackURL *string
	if input.CallbackURL != "" {
		callbackURL = &input.CallbackURL
	}
	country := input.Country
	recipientOp := op.Provider

	tx, err := h.gatewayRepo.CreateTransaction(r.Context(), repository.CreateGatewayTxParams{
		ClientID:          clientID,
		ClientRef:         input.ClientRef,
		Type:              model.GatewayTxTypePayout,
		Provider:          "pawapay",
		AmountCFA:         input.AmountCFA,
		Currency:          "XOF",
		RecipientPhone:    &msisdn,
		RecipientOperator: &recipientOp,
		Country:           &country,
		CallbackURL:       callbackURL,
	})
	if err != nil {
		http.Error(w, `{"error":"transaction_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	out, err := h.pawapay.AsProvider().InitiatePayout(r.Context(), payment.PayoutOp{
		Phone:     msisdn,
		Operator:  op.Provider,
		Amount:    fmt.Sprintf("%d", input.AmountCFA),
		Currency:  "XOF",
		ClientRef: tx.ID,
	})
	if err != nil || !out.Accepted {
		reason := "payout_init_failed"
		if out.FailureReason != "" {
			reason = out.FailureReason
		}
		_ = h.gatewayRepo.UpdateStatus(r.Context(), tx.ID, model.GatewayTxFailed, &reason)
		http.Error(w, fmt.Sprintf(`{"error":"payout_rejected","reason":%q}`, reason), http.StatusBadGateway)
		return
	}
	if err := h.gatewayRepo.SetProviderRef(r.Context(), tx.ID, out.ProviderRef); err != nil {
		http.Error(w, `{"error":"transaction_update_failed"}`, http.StatusInternalServerError)
		return
	}
	_ = h.gatewayRepo.UpdateStatus(r.Context(), tx.ID, model.GatewayTxProcessing, nil)

	tx.ProviderRef = &out.ProviderRef
	tx.Status = model.GatewayTxProcessing
	writeGatewayTx(w, tx, http.StatusCreated)
}

// CreateRefund — POST /api/gateway/v1/refunds
func (h *GatewayHandler) CreateRefund(w http.ResponseWriter, r *http.Request) {
	clientID := middleware.GetGatewayClientID(r.Context())

	var input model.GatewayCreateRefundInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.ClientRef == "" || input.DepositClientRef == "" {
		http.Error(w, `{"error":"client_ref_and_deposit_client_ref_required"}`, http.StatusBadRequest)
		return
	}
	if h.pawapay == nil {
		http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	if existing, err := h.gatewayRepo.FindByClientRef(r.Context(), clientID, input.ClientRef, model.GatewayTxTypeRefund); err == nil {
		writeGatewayTx(w, existing, http.StatusOK)
		return
	}

	deposit, err := h.gatewayRepo.FindByClientRef(r.Context(), clientID, input.DepositClientRef, model.GatewayTxTypeDeposit)
	if err != nil {
		http.Error(w, `{"error":"deposit_not_found"}`, http.StatusNotFound)
		return
	}
	if deposit.Status != model.GatewayTxCompleted {
		http.Error(w, `{"error":"deposit_not_completed"}`, http.StatusBadRequest)
		return
	}
	if deposit.ProviderRef == nil {
		http.Error(w, `{"error":"deposit_missing_provider_ref"}`, http.StatusInternalServerError)
		return
	}

	var callbackURL *string
	if input.CallbackURL != "" {
		callbackURL = &input.CallbackURL
	}

	tx, err := h.gatewayRepo.CreateTransaction(r.Context(), repository.CreateGatewayTxParams{
		ClientID:          clientID,
		ClientRef:         input.ClientRef,
		Type:              model.GatewayTxTypeRefund,
		Provider:          deposit.Provider,
		AmountCFA:         deposit.AmountCFA,
		Currency:          deposit.Currency,
		CallbackURL:       callbackURL,
		RelatedDepositRef: deposit.ProviderRef,
	})
	if err != nil {
		http.Error(w, `{"error":"transaction_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	out, err := h.pawapay.AsProvider().InitiateRefund(r.Context(), *deposit.ProviderRef, tx.ID)
	if err != nil || !out.Accepted {
		reason := "refund_init_failed"
		if out.FailureReason != "" {
			reason = out.FailureReason
		}
		_ = h.gatewayRepo.UpdateStatus(r.Context(), tx.ID, model.GatewayTxFailed, &reason)
		http.Error(w, fmt.Sprintf(`{"error":"refund_rejected","reason":%q}`, reason), http.StatusBadGateway)
		return
	}
	if err := h.gatewayRepo.SetProviderRef(r.Context(), tx.ID, out.ProviderRef); err != nil {
		http.Error(w, `{"error":"transaction_update_failed"}`, http.StatusInternalServerError)
		return
	}
	_ = h.gatewayRepo.UpdateStatus(r.Context(), tx.ID, model.GatewayTxProcessing, nil)

	tx.ProviderRef = &out.ProviderRef
	tx.Status = model.GatewayTxProcessing
	writeGatewayTx(w, tx, http.StatusCreated)
}

func writeGatewayTx(w http.ResponseWriter, tx *model.GatewayTransaction, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{"transaction": tx})
}
