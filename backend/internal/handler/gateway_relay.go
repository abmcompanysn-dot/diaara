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
	"time"

	"github.com/diarra/backend/internal/model"
)

// gateway_relay.go : DIARRA garde la responsabilité des callbacks agrégateur
// (PawaPay -> DIARRA, vérifiés par verifyRequest/verifyContentDigest comme
// tout le reste de webhook_handler.go) et les RELAIE vers le client externe
// qui a créé la transaction (DIARRA -> callback_url du client). Le client
// (ex. ABMCY Core) n'a donc jamais besoin de recevoir un webhook agrégateur
// lui-même — il reçoit uniquement le vocabulaire déjà normalisé par
// payment.PaymentProvider (pending|processing|completed|failed|cancelled).
//
// Isolé dans son propre fichier plutôt qu'ajouté à webhook_handler.go : ce
// dernier documente plusieurs incidents de production sur sa logique
// existante (voir ses commentaires) — limiter la surface touchée réduit le
// risque de régression sur le flux de paiement principal DIARRA pendant
// qu'on ajoute la passerelle.

// tryRelayGatewayDeposit — appelé quand un depositId de callback PawaPay ne
// correspond à AUCUNE vente DIARRA : peut-être un dépôt initié via la
// passerelle (GatewayHandler.CreateDeposit). Revérifie le statut auprès de
// PawaPay (même principe défensif que le reste du fichier — jamais confiance
// dans le corps du webhook) puis met à jour + relaie. Renvoie false si ce
// depositId n'est pas non plus une transaction gateway connue (l'appelant
// retombe alors sur son 404 sale_not_found habituel).
func (h *WebhookHandler) tryRelayGatewayDeposit(w http.ResponseWriter, r *http.Request, depositID string) bool {
	tx, err := h.gatewayRepo.FindByProviderRef(r.Context(), "pawapay", depositID)
	if err != nil {
		return false
	}
	if tx.Status != model.GatewayTxPending && tx.Status != model.GatewayTxProcessing {
		writeJSON(w, map[string]string{"status": tx.Status})
		return true
	}

	status, err := h.pawapay.GetDepositStatus(r.Context(), depositID)
	if err != nil || status.Data == nil {
		http.Error(w, `{"error":"status_check_failed"}`, http.StatusBadGateway)
		return true
	}

	newStatus := normalizeGatewayPawaPayStatus(status.Data.Status)
	var failureReason *string
	if status.Data.FailureReason != nil {
		failureReason = &status.Data.FailureReason.FailureMessage
	}
	if err := h.gatewayRepo.UpdateStatus(r.Context(), tx.ID, newStatus, failureReason); err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return true
	}
	tx.Status = newStatus
	tx.FailureReason = failureReason
	go h.relayGatewayCallback(tx)

	writeJSON(w, map[string]string{"status": newStatus})
	return true
}

func (h *WebhookHandler) tryRelayGatewayPayout(w http.ResponseWriter, r *http.Request, payoutID string) bool {
	tx, err := h.gatewayRepo.FindByProviderRef(r.Context(), "pawapay", payoutID)
	if err != nil {
		return false
	}
	if tx.Status == model.GatewayTxCompleted || tx.Status == model.GatewayTxFailed {
		writeJSON(w, map[string]string{"status": tx.Status})
		return true
	}

	status, err := h.pawapay.GetPayoutStatus(r.Context(), payoutID)
	if err != nil || status.Data == nil {
		http.Error(w, `{"error":"status_check_failed"}`, http.StatusBadGateway)
		return true
	}

	newStatus := normalizeGatewayPawaPayStatus(status.Data.Status)
	var failureReason *string
	if status.Data.FailureReason != nil {
		failureReason = &status.Data.FailureReason.FailureMessage
	}
	if err := h.gatewayRepo.UpdateStatus(r.Context(), tx.ID, newStatus, failureReason); err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return true
	}
	tx.Status = newStatus
	tx.FailureReason = failureReason
	go h.relayGatewayCallback(tx)

	writeJSON(w, map[string]string{"status": newStatus})
	return true
}

func (h *WebhookHandler) tryRelayGatewayRefund(w http.ResponseWriter, r *http.Request, refundID string) bool {
	tx, err := h.gatewayRepo.FindByProviderRef(r.Context(), "pawapay", refundID)
	if err != nil {
		return false
	}
	if tx.Status == model.GatewayTxCompleted {
		writeJSON(w, map[string]string{"status": tx.Status})
		return true
	}

	status, err := h.pawapay.GetRefundStatus(r.Context(), refundID)
	if err != nil || status.Data == nil {
		http.Error(w, `{"error":"status_check_failed"}`, http.StatusBadGateway)
		return true
	}

	newStatus := normalizeGatewayPawaPayStatus(status.Data.Status)
	var failureReason *string
	if status.Data.FailureReason != nil {
		failureReason = &status.Data.FailureReason.FailureMessage
	}
	if err := h.gatewayRepo.UpdateStatus(r.Context(), tx.ID, newStatus, failureReason); err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return true
	}
	tx.Status = newStatus
	tx.FailureReason = failureReason
	go h.relayGatewayCallback(tx)

	writeJSON(w, map[string]string{"status": newStatus})
	return true
}

// normalizeGatewayPawaPayStatus — même vocabulaire que
// payment.normalizePawaPayStatus (non exporté, donc reconstruit ici plutôt
// que couplé) : le client externe ne doit jamais voir le vocabulaire brut
// PawaPay (ACCEPTED/ENQUEUED/PROCESSING/IN_RECONCILIATION/COMPLETED/FAILED).
func normalizeGatewayPawaPayStatus(s string) string {
	switch s {
	case "ACCEPTED", "ENQUEUED":
		return model.GatewayTxPending
	case "PROCESSING", "IN_RECONCILIATION":
		return model.GatewayTxProcessing
	case "COMPLETED":
		return model.GatewayTxCompleted
	case "FAILED":
		return model.GatewayTxFailed
	default:
		return model.GatewayTxPending
	}
}

// gatewayCallbackPayload — corps envoyé au client externe. Volontairement
// minimal et déjà normalisé (voir normalizeGatewayPawaPayStatus) : le client
// n'a jamais besoin de connaître PawaPay/KPay/PayPal en tant que tels.
type gatewayCallbackPayload struct {
	ClientRef     string `json:"client_ref"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	FailureReason string `json:"failure_reason,omitempty"`
	AmountCFA     int    `json:"amount_cfa"`
}

// relayGatewayCallback notifie le client externe (POST callback_url) qu'une
// transaction a changé de statut. Best-effort, en tâche de fond : un client
// externe injoignable ne doit jamais faire échouer le traitement du webhook
// agrégateur (dont la réponse 200 est ce qui empêche PawaPay de re-livrer le
// même callback en boucle). Le client est censé re-synchroniser via
// GET /api/gateway/v1/{type}s/{client_ref} s'il rate une notification.
func (h *WebhookHandler) relayGatewayCallback(tx *model.GatewayTransaction) {
	if tx.CallbackURL == nil || *tx.CallbackURL == "" {
		return
	}
	payload := gatewayCallbackPayload{
		ClientRef: tx.ClientRef,
		Type:      tx.Type,
		Status:    tx.Status,
		AmountCFA: tx.AmountCFA,
	}
	if tx.FailureReason != nil {
		payload.FailureReason = *tx.FailureReason
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *tx.CallbackURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	// Signature HMAC-SHA256 du corps avec le secret HMAC DÉDIÉ du client
	// (jamais sa clé API ni son ID — voir model.GatewayClient.HMACSecretHash
	// et middleware.RequireGatewayClient pour le même principe en sens
	// inverse) : le client externe doit pouvoir vérifier que ce callback
	// vient bien de DIARRA. On ne connaît que le HASH du secret (jamais le
	// secret brut, remis une seule fois à la création du client) — c'est ce
	// hash qui sert de clé HMAC des deux côtés, voir doSigned côté ABMCY Core.
	gwClient, err := h.gatewayRepo.FindClientByID(context.Background(), tx.ClientID)
	if err != nil || gwClient.HMACSecretHash == "" {
		log.Printf("relais gateway: secret HMAC introuvable pour client=%s (tx=%s), callback non signé, abandon", tx.ClientID, tx.ID)
		return
	}
	mac := hmac.New(sha256.New, []byte(gwClient.HMACSecretHash))
	mac.Write(body)
	req.Header.Set("X-Diarra-Gateway-Signature", hex.EncodeToString(mac.Sum(nil)))

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("relais gateway: callback %s injoignable pour tx=%s: %v", *tx.CallbackURL, tx.ID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("relais gateway: callback %s a répondu %d pour tx=%s", *tx.CallbackURL, resp.StatusCode, tx.ID)
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
