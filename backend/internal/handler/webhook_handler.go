package handler

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/diarra/backend/internal/cache"
	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/diarra/backend/internal/storage"
)

type WebhookHandler struct {
	saleRepo          *repository.SaleRepo
	userRepo          *repository.UserRepo
	productRepo       *repository.ProductRepo
	payoutRepo        *repository.PayoutRepo
	pawapay           *payment.PawaPayClient
	kpay              *payment.KPayClient
	kpayWebhookSecret string
	paypal            *payment.PayPalClient
	commissionSvc     *service.CommissionService
	donationSvc       *service.DonationService
	notifications     *email.NotificationService
	notificationRepo  *repository.NotificationRepo
	storage           *storage.S3Storage
	allowedIPs        map[string]bool
	cache             *cache.Client
}

func NewWebhookHandler(
	saleRepo *repository.SaleRepo,
	userRepo *repository.UserRepo,
	productRepo *repository.ProductRepo,
	payoutRepo *repository.PayoutRepo,
	pawapay *payment.PawaPayClient,
	kpay *payment.KPayClient,
	kpayWebhookSecret string,
	paypal *payment.PayPalClient,
	donationSvc *service.DonationService,
	notifications *email.NotificationService,
	notificationRepo *repository.NotificationRepo,
	storage *storage.S3Storage,
	allowedIPs []string,
	cacheClient *cache.Client,
) *WebhookHandler {
	ips := make(map[string]bool, len(allowedIPs))
	for _, ip := range allowedIPs {
		ips[strings.TrimSpace(ip)] = true
	}
	return &WebhookHandler{
		saleRepo:          saleRepo,
		userRepo:          userRepo,
		productRepo:       productRepo,
		payoutRepo:        payoutRepo,
		pawapay:           pawapay,
		kpay:              kpay,
		kpayWebhookSecret: kpayWebhookSecret,
		paypal:            paypal,
		commissionSvc:     service.NewCommissionService(),
		donationSvc:       donationSvc,
		notifications:     notifications,
		notificationRepo:  notificationRepo,
		storage:           storage,
		allowedIPs:        ips,
		cache:             cacheClient,
	}
}

// verifyKPayRequest — HMAC-SHA256 (X-KPAY-Signature) sur le corps brut, avec
// le secret webhook DÉDIÉ (distinct de la clé secrète API) — schéma différent
// de verifyRequest (Content-Digest + liste blanche IP, PawaPay).
func (h *WebhookHandler) verifyKPayRequest(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusBadRequest)
		return nil, false
	}
	sig := r.Header.Get("X-KPAY-Signature")
	if !payment.VerifyKPaySignature(h.kpayWebhookSecret, body, sig) {
		http.Error(w, `{"error":"invalid_signature"}`, http.StatusUnauthorized)
		return nil, false
	}
	return body, true
}

// notify insère une notification in-app, en tâche de fond, sans jamais faire
// échouer l'appelant (les notifications sont secondaires au flux principal).
func (h *WebhookHandler) notify(ctx context.Context, userID, notifType, title, body, link string) {
	if h.notificationRepo == nil || userID == "" {
		return
	}
	h.notificationRepo.Create(ctx, userID, notifType, title, body, link)
}

// verifyRequest applique les mêmes vérifications (digest + IP) que le webhook
// de dépôt, et retourne le corps lu. Renvoie false si la requête a déjà été
// rejetée (réponse déjà écrite).
func (h *WebhookHandler) verifyRequest(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusBadRequest)
		return nil, false
	}
	if digest := r.Header.Get("Content-Digest"); digest != "" {
		if !h.verifyContentDigest(body, digest) {
			http.Error(w, `{"error":"invalid_digest"}`, http.StatusUnauthorized)
			return nil, false
		}
	}
	if len(h.allowedIPs) > 0 {
		ip := clientIP(r)
		if !h.allowedIPs[ip] {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return nil, false
		}
	}
	return body, true
}

// PawaPayPayoutWebhook reçoit la confirmation d'un versement vendeur. Par
// prudence (le schéma exact du corps du callback n'est pas garanti), on ne
// se fie qu'à l'ID transmis puis on revérifie le statut via l'API PawaPay.
func (h *WebhookHandler) PawaPayPayoutWebhook(w http.ResponseWriter, r *http.Request) {
	body, ok := h.verifyRequest(w, r)
	if !ok {
		return
	}

	var payload struct {
		PayoutId string `json:"payoutId"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.PayoutId == "" {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	payout, err := h.payoutRepo.FindByProviderReference(r.Context(), "pawapay", payload.PayoutId)
	if err != nil {
		http.Error(w, `{"error":"payout_not_found"}`, http.StatusNotFound)
		return
	}
	if payout.Status == "paid" || payout.Status == "failed" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": payout.Status})
		return
	}

	status, err := h.pawapay.GetPayoutStatus(r.Context(), payload.PayoutId)
	if err != nil || status.Data == nil {
		http.Error(w, `{"error":"status_check_failed"}`, http.StatusBadGateway)
		return
	}

	switch status.Data.Status {
	case "COMPLETED":
		h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "paid", nil)
		h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))
		h.notify(r.Context(), payout.UserID, "payout_paid", "Versement effectué",
			fmt.Sprintf("Votre versement de %d FCFA a été envoyé.", payout.AmountCFA), "/vendor/earnings")
	case "FAILED":
		reason := ""
		if status.Data.FailureReason != nil {
			reason = status.Data.FailureReason.FailureCode
		}
		h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "failed", &reason)
		h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))
		// Pas de notification au vendeur sur un échec de versement : il ne doit
		// jamais voir d'erreur sur une demande de retrait. L'admin voit le
		// statut "failed" et décide (relancer / régler à la main).
		h.notifyAdminsPayoutFailed(r.Context(), payout, reason)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status.Data.Status})
}

// PawaPayRefundWebhook reçoit la confirmation d'un remboursement. Même
// principe défensif que pour les payouts : revérification via l'API.
func (h *WebhookHandler) PawaPayRefundWebhook(w http.ResponseWriter, r *http.Request) {
	body, ok := h.verifyRequest(w, r)
	if !ok {
		return
	}

	var payload struct {
		RefundId string `json:"refundId"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.RefundId == "" {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	sale, err := h.saleRepo.FindByRefundReference(r.Context(), payload.RefundId)
	if err != nil {
		http.Error(w, `{"error":"sale_not_found"}`, http.StatusNotFound)
		return
	}
	if sale.Status == string(model.SaleRefunded) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": sale.Status})
		return
	}

	status, err := h.pawapay.GetRefundStatus(r.Context(), payload.RefundId)
	if err != nil || status.Data == nil {
		http.Error(w, `{"error":"status_check_failed"}`, http.StatusBadGateway)
		return
	}

	if status.Data.Status == "COMPLETED" {
		h.saleRepo.UpdateStatus(r.Context(), sale.ID, string(model.SaleRefunded))
		go h.notifyRefunded(context.Background(), sale)
		h.notify(r.Context(), sale.BuyerID, "refund", "Remboursement effectué",
			fmt.Sprintf("Votre remboursement de %d FCFA a été traité.", sale.AmountCFA), "/orders")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status.Data.Status})
}

// notifyRefunded envoie l'email de confirmation de remboursement à l'acheteur.
func (h *WebhookHandler) notifyRefunded(ctx context.Context, sale *model.Sale) {
	buyer, err := h.userRepo.FindByID(ctx, sale.BuyerID)
	if err != nil || buyer.Email == "" {
		return
	}
	product, err := h.productRepo.FindByID(ctx, sale.ProductID)
	if err != nil {
		return
	}
	h.notifications.SendRefundConfirmed(ctx, buyer.Email, sale.BuyerName, product.Title, sale.AmountCFA)
}

// PawaPayDepositCallback est le corps du callback PawaPay (statut final d'un
// dépôt). Seul DepositId est utilisé : le statut n'est jamais lu depuis ce
// payload (voir PawaPayWebhook — même principe défensif que pour les
// payouts/remboursements, la forme exacte de ce corps n'est pas garantie).
type PawaPayDepositCallback struct {
	DepositId string `json:"depositId"`
}

// PawaPayWebhook reçoit la confirmation de paiement depuis PawaPay.
func (h *WebhookHandler) PawaPayWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusBadRequest)
		return
	}

	// Vérification d'intégrité du corps (Content-Digest: sha-256=:base64:)
	if digest := r.Header.Get("Content-Digest"); digest != "" {
		if !h.verifyContentDigest(body, digest) {
			http.Error(w, `{"error":"invalid_digest"}`, http.StatusUnauthorized)
			return
		}
	}

	// Restriction d'adresse IP des serveurs PawaPay (si configurée).
	if len(h.allowedIPs) > 0 {
		ip := clientIP(r)
		if !h.allowedIPs[ip] {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
	}

	var payload PawaPayDepositCallback
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if payload.DepositId == "" {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	sale, err := h.saleRepo.FindByPaymentReference(r.Context(), payload.DepositId)
	if err != nil {
		http.Error(w, `{"error":"sale_not_found"}`, http.StatusNotFound)
		return
	}

	// Idempotence : ne pas re-traiter un paiement déjà terminé.
	if sale.Status != string(model.SalePending) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": sale.Status})
		return
	}

	// Le corps du callback sert uniquement de signal de réveil : le statut
	// réel est revérifié via l'API (même principe défensif que pour les
	// payouts/remboursements) plutôt que de faire confiance au champ "status"
	// du webhook, dont la forme exacte (top-level ou nested sous "data")
	// n'est pas fiable — un bug ici avait pour effet de marquer "failed"
	// des paiements réellement complétés.
	status, err := h.pawapay.GetDepositStatus(r.Context(), payload.DepositId)
	if err != nil || status.Data == nil {
		http.Error(w, `{"error":"status_check_failed"}`, http.StatusBadGateway)
		return
	}

	if status.Data.Status == "COMPLETED" {
		if err := h.ConfirmPaidSale(r.Context(), sale); err != nil {
			http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "paid"})
		return
	}

	if status.Data.Status == "FAILED" {
		if err := h.saleRepo.UpdateStatus(r.Context(), sale.ID, string(model.SaleFailed)); err != nil {
			http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
			return
		}
		if h.notifications != nil {
			go h.notifyFailed(context.Background(), sale)
		}
		h.notify(r.Context(), sale.BuyerID, "order_failed", "Paiement échoué",
			"Votre paiement n'a pas pu être traité, réessayez.", "/orders")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status.Data.Status})
}

// --- Webhooks KPay -----------------------------------------------------------
//
// Même principe défensif que PawaPay ci-dessus (ne jamais faire confiance au
// statut du corps du webhook, toujours revérifier via l'API), mais
// vérification par HMAC réel (verifyKPayRequest) au lieu de Content-Digest+IP.

// KPayPaymentWebhook reçoit la confirmation de paiement depuis KPay.
func (h *WebhookHandler) KPayPaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if h.kpay == nil {
		http.Error(w, `{"error":"kpay_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	body, ok := h.verifyKPayRequest(w, r)
	if !ok {
		return
	}

	var payload struct {
		ExternalId string `json:"externalId"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.ExternalId == "" {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	sale, err := h.saleRepo.FindByPaymentReference(r.Context(), payload.ExternalId)
	if err != nil {
		http.Error(w, `{"error":"sale_not_found"}`, http.StatusNotFound)
		return
	}
	if sale.Status != string(model.SalePending) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": sale.Status})
		return
	}
	if sale.ProviderTransactionID == nil {
		http.Error(w, `{"error":"missing_provider_transaction_id"}`, http.StatusInternalServerError)
		return
	}

	status, err := h.kpay.GetPaymentStatus(r.Context(), *sale.ProviderTransactionID)
	if err != nil {
		http.Error(w, `{"error":"status_check_failed"}`, http.StatusBadGateway)
		return
	}

	if status.Status == "COMPLETED" {
		if err := h.saleRepo.UpdateStatus(r.Context(), sale.ID, string(model.SalePaid)); err != nil {
			http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
			return
		}
		if h.notifications != nil {
			go h.notifyPaid(context.Background(), sale)
		}
		if h.donationSvc != nil {
			go h.donationSvc.Accumulate(context.Background(), sale.PlatformFeeCFA)
		}
		h.notify(r.Context(), sale.BuyerID, "order_paid", "Commande confirmée",
			fmt.Sprintf("Votre paiement de %d FCFA a été confirmé.", sale.AmountCFA), "/orders")
		if product, err := h.productRepo.FindByID(r.Context(), sale.ProductID); err == nil {
			h.notify(r.Context(), product.VendorID, "sale", "Nouvelle vente",
				fmt.Sprintf("%s vient d'acheter « %s » pour %d FCFA.", sale.BuyerName, product.Title, sale.VendorAmountCFA), "/vendor/sales")
			h.cache.Del(r.Context(), vendorBalanceCacheKey(product.VendorID))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "paid"})
		return
	}

	if status.Status == "FAILED" || status.Status == "CANCELLED" {
		if err := h.saleRepo.UpdateStatus(r.Context(), sale.ID, string(model.SaleFailed)); err != nil {
			http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
			return
		}
		if h.notifications != nil {
			go h.notifyFailed(context.Background(), sale)
		}
		h.notify(r.Context(), sale.BuyerID, "order_failed", "Paiement échoué",
			"Votre paiement n'a pas pu être traité, réessayez.", "/orders")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status.Status})
}

// KPayPayoutWebhook reçoit la confirmation d'un versement vendeur via KPay.
func (h *WebhookHandler) KPayPayoutWebhook(w http.ResponseWriter, r *http.Request) {
	if h.kpay == nil {
		http.Error(w, `{"error":"kpay_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	body, ok := h.verifyKPayRequest(w, r)
	if !ok {
		return
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.ID == "" {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	payout, err := h.payoutRepo.FindByProviderReference(r.Context(), "kpay", payload.ID)
	if err != nil {
		http.Error(w, `{"error":"payout_not_found"}`, http.StatusNotFound)
		return
	}
	if payout.Status == "paid" || payout.Status == "failed" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": payout.Status})
		return
	}

	status, err := h.kpay.GetPayoutStatus(r.Context(), payload.ID)
	if err != nil {
		http.Error(w, `{"error":"status_check_failed"}`, http.StatusBadGateway)
		return
	}

	switch status.Status {
	case "COMPLETED":
		h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "paid", nil)
		h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))
		h.notify(r.Context(), payout.UserID, "payout_paid", "Versement effectué",
			fmt.Sprintf("Votre versement de %d FCFA a été envoyé.", payout.AmountCFA), "/vendor/earnings")
	case "FAILED", "CANCELLED":
		reason := status.FailureReason
		h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "failed", &reason)
		h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))
		// Pas de notification vendeur sur un échec (voir PawaPayPayoutWebhook).
		h.notifyAdminsPayoutFailed(r.Context(), payout, reason)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status.Status})
}

// KPayRefundWebhook reçoit la confirmation d'un remboursement via KPay.
//
// La doc fournie par KPay ne liste pas d'endpoint GET dédié au statut d'un
// remboursement (contrairement à PawaPay GetRefundStatus) — à confirmer
// contre leur API réelle avant mise en production. En attendant, on
// revérifie via le statut du PAIEMENT parent (GetPaymentStatus sur
// sale.ProviderTransactionID) plutôt que de faire confiance au webhook seul.
func (h *WebhookHandler) KPayRefundWebhook(w http.ResponseWriter, r *http.Request) {
	if h.kpay == nil {
		http.Error(w, `{"error":"kpay_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	body, ok := h.verifyKPayRequest(w, r)
	if !ok {
		return
	}

	var payload struct {
		ID                string `json:"id"`
		OriginalPaymentId string `json:"originalPaymentId"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.ID == "" {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	sale, err := h.saleRepo.FindByRefundReference(r.Context(), payload.ID)
	if err != nil {
		http.Error(w, `{"error":"sale_not_found"}`, http.StatusNotFound)
		return
	}
	if sale.Status == string(model.SaleRefunded) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": sale.Status})
		return
	}
	if sale.ProviderTransactionID == nil {
		http.Error(w, `{"error":"missing_provider_transaction_id"}`, http.StatusInternalServerError)
		return
	}

	// Pas d'endpoint de statut dédié au remboursement (voir commentaire de
	// fonction) : on relit le paiement parent, dont le statut redevient
	// "COMPLETED" (jamais un statut "refunded" dédié côté KPay d'après leur
	// doc) une fois le remboursement traité — on se fie donc ici au statut
	// annoncé par le WEBHOOK lui-même pour décider, contrairement au reste de
	// ce fichier (à corriger dès confirmation d'un vrai endpoint de statut).
	_, err = h.kpay.GetPaymentStatus(r.Context(), *sale.ProviderTransactionID)
	if err != nil {
		http.Error(w, `{"error":"status_check_failed"}`, http.StatusBadGateway)
		return
	}

	var payloadStatus struct {
		Status string `json:"status"`
	}
	json.Unmarshal(body, &payloadStatus)

	if payloadStatus.Status == "COMPLETED" {
		h.saleRepo.UpdateStatus(r.Context(), sale.ID, string(model.SaleRefunded))
		go h.notifyRefunded(context.Background(), sale)
		h.notify(r.Context(), sale.BuyerID, "refund", "Remboursement effectué",
			fmt.Sprintf("Votre remboursement de %d FCFA a été traité.", sale.AmountCFA), "/orders")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": payloadStatus.Status})
}

// PayPalWebhook reçoit les événements PayPal (paiement approuvé/capturé,
// remboursement). Sert de filet de sécurité si l'acheteur ferme l'onglet
// avant que le polling depuis checkout/return (SaleHandler.CheckoutStatus)
// n'ait capturé la commande — la vérification de signature se fait auprès de
// PayPal lui-même (pas de HMAC local possible, voir VerifyWebhookSignature).
func (h *WebhookHandler) PayPalWebhook(w http.ResponseWriter, r *http.Request) {
	if h.paypal == nil {
		http.Error(w, `{"error":"paypal_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusBadRequest)
		return
	}
	valid, err := h.paypal.VerifyWebhookSignature(r.Context(), r.Header, body)
	if err != nil || !valid {
		http.Error(w, `{"error":"invalid_signature"}`, http.StatusUnauthorized)
		return
	}

	var event struct {
		EventType string `json:"event_type"`
		Resource  struct {
			ID            string `json:"id"`
			CustomID      string `json:"custom_id"`
			PurchaseUnits []struct {
				ReferenceID string `json:"reference_id"`
				CustomID    string `json:"custom_id"`
			} `json:"purchase_units"`
			// Événements PAYMENT.PAYOUTS-ITEM.* : sender_item_id = ID du
			// versement DIARRA ; payout_item.sender_item_id sur certains formats.
			SenderItemID string `json:"sender_item_id"`
			PayoutItem   struct {
				SenderItemID string `json:"sender_item_id"`
			} `json:"payout_item"`
			TransactionStatus string `json:"transaction_status"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	switch event.EventType {
	case "PAYMENT.PAYOUTS-ITEM.SUCCEEDED", "PAYMENT.PAYOUTS-ITEM.FAILED",
		"PAYMENT.PAYOUTS-ITEM.BLOCKED", "PAYMENT.PAYOUTS-ITEM.RETURNED",
		"PAYMENT.PAYOUTS-ITEM.DENIED":
		payoutID := event.Resource.SenderItemID
		if payoutID == "" {
			payoutID = event.Resource.PayoutItem.SenderItemID
		}
		if payoutID == "" {
			w.WriteHeader(http.StatusOK) // rien à rapprocher
			return
		}
		payout, err := h.payoutRepo.FindByID(r.Context(), payoutID)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		if payout.Status == "paid" || payout.Status == "failed" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": payout.Status})
			return
		}
		if event.EventType == "PAYMENT.PAYOUTS-ITEM.SUCCEEDED" {
			if err := h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "paid", nil); err == nil {
				h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))
				if h.notifications != nil {
					uid, amt := payout.UserID, payout.AmountCFA
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
						defer cancel()
						if u, uerr := h.userRepo.FindByID(ctx, uid); uerr == nil && u.Email != "" {
							_ = h.notifications.SendPayoutConfirmed(ctx, u.Email, amt)
						}
					}()
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "paid"})
			return
		}
		reason := event.Resource.TransactionStatus
		if reason == "" {
			reason = "paypal_payout_failed"
		}
		if err := h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "failed", &reason); err == nil {
			h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))
			h.notifyAdminsPayoutFailed(r.Context(), payout, reason)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "failed"})
		return
	}

	switch event.EventType {
	case "CHECKOUT.ORDER.APPROVED", "PAYMENT.CAPTURE.COMPLETED":
		saleID := event.Resource.CustomID
		if saleID == "" && len(event.Resource.PurchaseUnits) > 0 {
			saleID = event.Resource.PurchaseUnits[0].CustomID
		}
		if saleID == "" {
			http.Error(w, `{"error":"missing_sale_id"}`, http.StatusBadRequest)
			return
		}
		sale, err := h.saleRepo.FindByID(r.Context(), saleID)
		if err != nil {
			http.Error(w, `{"error":"sale_not_found"}`, http.StatusNotFound)
			return
		}
		if sale.Status != string(model.SalePending) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": sale.Status})
			return
		}
		if sale.ProviderTransactionID == nil {
			http.Error(w, `{"error":"missing_provider_transaction_id"}`, http.StatusInternalServerError)
			return
		}

		outcome, err := h.paypal.AsProvider().GetDepositStatus(r.Context(), *sale.ProviderTransactionID)
		if err != nil {
			http.Error(w, `{"error":"status_check_failed"}`, http.StatusBadGateway)
			return
		}
		if outcome.UpdatedProviderRef != "" {
			h.saleRepo.SetProviderTransactionID(r.Context(), sale.ID, outcome.UpdatedProviderRef)
		}
		if outcome.Status == "completed" {
			if err := h.ConfirmPaidSale(r.Context(), sale); err != nil {
				http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": outcome.Status})
		return

	case "PAYMENT.CAPTURE.DECLINED":
		// Paiement carte refusé par la banque / PayPal après redirection : la
		// vente reste sinon "pending" indéfiniment (le polling checkout/return
		// ne voit qu'un ordre non capturé). On la passe "failed" et on prévient
		// l'acheteur, comme pour un échec PawaPay.
		saleID := event.Resource.CustomID
		if saleID == "" && len(event.Resource.PurchaseUnits) > 0 {
			saleID = event.Resource.PurchaseUnits[0].CustomID
		}
		if saleID == "" {
			http.Error(w, `{"error":"missing_sale_id"}`, http.StatusBadRequest)
			return
		}
		sale, err := h.saleRepo.FindByID(r.Context(), saleID)
		if err != nil {
			http.Error(w, `{"error":"sale_not_found"}`, http.StatusNotFound)
			return
		}
		if sale.Status == string(model.SalePending) {
			h.saleRepo.UpdateStatus(r.Context(), sale.ID, string(model.SaleFailed))
			if h.notifications != nil {
				go h.notifyFailed(context.Background(), sale)
			}
			h.notify(r.Context(), sale.BuyerID, "order_failed", "Paiement échoué",
				"Votre paiement par carte n'a pas abouti, réessayez.", "/orders")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "failed"})
		return

	case "PAYMENT.CAPTURE.REFUNDED":
		// event.Resource.ID est l'ID du REMBOURSEMENT PayPal — celui-là même
		// stocké par AdminHandler.RefundSale via SetRefundReference.
		sale, err := h.saleRepo.FindByRefundReference(r.Context(), event.Resource.ID)
		if err != nil {
			// Remboursement initié ailleurs que par RefundSale (depuis le
			// dashboard PayPal directement) : rien à rapprocher côté DIARRA.
			w.WriteHeader(http.StatusOK)
			return
		}
		if sale.Status != string(model.SaleRefunded) {
			h.saleRepo.UpdateStatus(r.Context(), sale.ID, string(model.SaleRefunded))
			go h.notifyRefunded(context.Background(), sale)
			h.notify(r.Context(), sale.BuyerID, "refund", "Remboursement effectué",
				fmt.Sprintf("Votre remboursement de %d FCFA a été traité.", sale.AmountCFA), "/orders")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "refunded"})
		return
	}

	w.WriteHeader(http.StatusOK)
}

// notifyFailed envoie l'email d'échec de paiement à l'acheteur.
func (h *WebhookHandler) notifyFailed(ctx context.Context, sale *model.Sale) {
	buyer, err := h.userRepo.FindByID(ctx, sale.BuyerID)
	if err != nil || buyer.Email == "" {
		return
	}
	product, err := h.productRepo.FindByID(ctx, sale.ProductID)
	if err != nil {
		return
	}
	h.notifications.SendPaymentFailed(ctx, buyer.Email, sale.BuyerName, product.Title, product.ID)
}

// verifyContentDigest vérifie le header Content-Digest (RFC 9530) : "sha-256=:base64:".
func (h *WebhookHandler) verifyContentDigest(body []byte, digest string) bool {
	parts := strings.SplitN(digest, "=", 2)
	if len(parts) != 2 {
		return false
	}
	algo := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	value = strings.TrimPrefix(value, ":")
	value = strings.TrimSuffix(value, ":")

	var sum []byte
	switch algo {
	case "sha-256":
		s := sha256.Sum256(body)
		sum = s[:]
	case "sha-512":
		s := sha512.Sum512(body)
		sum = s[:]
	default:
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return false
	}
	return secureEqual(decoded, sum)
}

// secureEqual compare deux octets en temps constant.
func secureEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

// Cadence du job de réconciliation des dépôts PawaPay restés "pending" :
// balaie toutes les depositReconcileEvery les ventes de moins de
// depositReconcileMaxAge, revérifie leur statut via l'API PawaPay et applique
// COMPLETED → paid (ConfirmPaidSale) / FAILED → failed. Filet de sécurité si
// un webhook s'est perdu (incident du 2026-09-02 : aucun webhook reçu, toutes
// les ventes bloquées "pending" malgré des paiements réussis).
const (
	depositReconcileEvery  = 10 * time.Minute
	depositReconcileMaxAge = 3 * 24 * time.Hour
)

// RunDepositReconcileLoop réconcilie en tâche de fond les ventes PawaPay
// restées "pending" avec le statut réel côté PawaPay. À lancer une fois au
// démarrage : go webhookHandler.RunDepositReconcileLoop(ctx).
func (h *WebhookHandler) RunDepositReconcileLoop(ctx context.Context) {
	if h.pawapay == nil {
		log.Printf("réconciliation dépôts: désactivée (PawaPay non configuré)")
		return
	}
	ticker := time.NewTicker(depositReconcileEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.reconcileDepositsPass(ctx)
		}
	}
}

func (h *WebhookHandler) reconcileDepositsPass(ctx context.Context) {
	sales, err := h.saleRepo.ListPendingForProvider(ctx, "pawapay", depositReconcileMaxAge)
	if err != nil {
		log.Printf("réconciliation dépôts: lecture des ventes échouée: %v", err)
		return
	}
	confirmed, failed := 0, 0
	for _, sale := range sales {
		status, err := h.pawapay.GetDepositStatus(ctx, sale.PaymentReference)
		if err != nil || status.Data == nil {
			continue // NOT_FOUND (panier abandonné) ou erreur transitoire : on retentera
		}
		switch status.Data.Status {
		case "COMPLETED":
			if err := h.ConfirmPaidSale(ctx, sale); err != nil {
				log.Printf("réconciliation dépôts: sale=%s confirmation échouée: %v", sale.ID, err)
				continue
			}
			confirmed++
		case "FAILED":
			if err := h.saleRepo.UpdateStatus(ctx, sale.ID, string(model.SaleFailed)); err != nil {
				continue
			}
			if h.notifications != nil {
				go h.notifyFailed(context.Background(), sale)
			}
			h.notify(ctx, sale.BuyerID, "order_failed", "Paiement échoué",
				"Votre paiement n'a pas pu être traité, réessayez.", "/orders")
			failed++
		}
	}
	if confirmed > 0 || failed > 0 {
		log.Printf("réconciliation dépôts: %d confirmée(s), %d échouée(s) sur %d en attente", confirmed, failed, len(sales))
	}
}

// notifyAdminsPayoutFailed prévient les admins (notification in-app) qu'un
// versement a échoué côté prestataire — le vendeur, lui, n'est jamais notifié
// d'un échec de retrait. Best-effort.
func (h *WebhookHandler) notifyAdminsPayoutFailed(ctx context.Context, payout *model.Payout, reason string) {
	if h.notificationRepo == nil || h.userRepo == nil {
		return
	}
	adminIDs, err := h.userRepo.ListAdminIDs(ctx)
	if err != nil {
		return
	}
	msg := fmt.Sprintf("Versement de %d FCFA en échec (%s) — à relancer ou régler à la main.", payout.AmountCFA, reason)
	for _, adminID := range adminIDs {
		h.notificationRepo.Create(ctx, adminID, "payout_failed_admin", "Versement à traiter", msg, "/admin/payouts")
	}
}

// Cadence du job de réconciliation des versements PawaPay/KPay restés
// "processing" : même principe que RunDepositReconcileLoop pour les dépôts.
const (
	payoutReconcileEvery  = 10 * time.Minute
	payoutReconcileMaxAge = 3 * 24 * time.Hour
)

// RunPayoutReconcileLoop réconcilie en tâche de fond les versements restés
// "processing" avec leur statut réel côté prestataire (filet de sécurité si un
// webhook s'est perdu). À lancer une fois au démarrage.
func (h *WebhookHandler) RunPayoutReconcileLoop(ctx context.Context) {
	if h.pawapay == nil && h.kpay == nil {
		log.Printf("réconciliation versements: désactivée (aucun prestataire configuré)")
		return
	}
	ticker := time.NewTicker(payoutReconcileEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.reconcilePayoutsPass(ctx)
		}
	}
}

func (h *WebhookHandler) reconcilePayoutsPass(ctx context.Context) {
	payouts, err := h.payoutRepo.ListProcessing(ctx, payoutReconcileMaxAge)
	if err != nil {
		log.Printf("réconciliation versements: lecture échouée: %v", err)
		return
	}
	paid, failed := 0, 0
	for _, p := range payouts {
		if p.ProviderReference == nil || *p.ProviderReference == "" {
			continue
		}
		var providerStatus, failReason string
		switch p.Provider {
		case "paypal":
			if h.paypal == nil {
				continue
			}
			st, err := h.paypal.AsProvider().GetPayoutStatus(ctx, *p.ProviderReference)
			if err != nil {
				continue
			}
			// Vocabulaire commun → brut attendu par le switch ci-dessous.
			switch st.Status {
			case "completed":
				providerStatus = "COMPLETED"
			case "failed":
				providerStatus = "FAILED"
			}
			failReason = st.FailureReason
		case "kpay":
			if h.kpay == nil {
				continue
			}
			st, err := h.kpay.GetPayoutStatus(ctx, *p.ProviderReference)
			if err != nil {
				continue
			}
			providerStatus, failReason = st.Status, st.FailureReason
		default:
			if h.pawapay == nil {
				continue
			}
			st, err := h.pawapay.GetPayoutStatus(ctx, *p.ProviderReference)
			if err != nil || st.Data == nil {
				continue
			}
			providerStatus = st.Data.Status
			if st.Data.FailureReason != nil {
				failReason = st.Data.FailureReason.FailureCode
			}
		}
		switch providerStatus {
		case "COMPLETED":
			if err := h.payoutRepo.UpdateStatus(ctx, p.ID, "paid", nil); err == nil {
				h.cache.Del(ctx, vendorBalanceCacheKey(p.UserID))
				h.notify(ctx, p.UserID, "payout_paid", "Versement effectué",
					fmt.Sprintf("Votre versement de %d FCFA a été envoyé.", p.AmountCFA), "/vendor/earnings")
				paid++
			}
		case "FAILED", "CANCELLED":
			if err := h.payoutRepo.UpdateStatus(ctx, p.ID, "failed", &failReason); err == nil {
				h.cache.Del(ctx, vendorBalanceCacheKey(p.UserID))
				h.notifyAdminsPayoutFailed(ctx, p, failReason)
				failed++
			}
		}
	}
	if paid > 0 || failed > 0 {
		log.Printf("réconciliation versements: %d payé(s), %d échoué(s) sur %d en traitement", paid, failed, len(payouts))
	}
}

// ConfirmPaidSale applique tous les effets d'un paiement confirmé pour une
// vente restée "pending" : passage en "paid", emails acheteur (fichier joint) +
// vendeur, notifications in-app, alimentation de la cagnotte de fidélisation,
// invalidation du solde vendeur en cache. Idempotent au niveau appelant (ne
// rien faire si la vente n'est plus "pending"). Partagé par les trois chemins
// de confirmation : webhook PawaPay, job de réconciliation de fond
// (SaleHandler.RunDepositReconcileLoop) et vérification manuelle admin
// (AdminHandler.CheckSaleProvider).
func (h *WebhookHandler) ConfirmPaidSale(ctx context.Context, sale *model.Sale) error {
	if sale.Status != string(model.SalePending) {
		return nil
	}
	if err := h.saleRepo.UpdateStatus(ctx, sale.ID, string(model.SalePaid)); err != nil {
		return err
	}
	sale.Status = string(model.SalePaid)

	if h.notifications != nil {
		go h.notifyPaid(context.Background(), sale)
	}
	if h.donationSvc != nil {
		go h.donationSvc.Accumulate(context.Background(), sale.PlatformFeeCFA)
	}
	h.notify(ctx, sale.BuyerID, "order_paid", "Commande confirmée",
		fmt.Sprintf("Votre paiement de %d FCFA a été confirmé.", sale.AmountCFA), "/orders")
	if product, err := h.productRepo.FindByID(ctx, sale.ProductID); err == nil {
		h.notify(ctx, product.VendorID, "sale", "Nouvelle vente",
			fmt.Sprintf("%s vient d'acheter « %s » pour %d FCFA.", sale.BuyerName, product.Title, sale.VendorAmountCFA), "/vendor/sales")
		// La vente vient de créditer ce vendeur : le solde caché
		// (PayoutHandler.Earnings) serait sinon obsolète jusqu'à
		// expiration de son TTL (30s).
		h.cache.Del(ctx, vendorBalanceCacheKey(product.VendorID))
	}
	return nil
}

// notifyPaid envoie les emails de confirmation (acheteur + vendeur) en arrière-plan.
func (h *WebhookHandler) notifyPaid(ctx context.Context, sale *model.Sale) {
	product, err := h.productRepo.FindByID(ctx, sale.ProductID)
	if err != nil {
		return
	}

	buyer, err := h.userRepo.FindByID(ctx, sale.BuyerID)
	if err == nil && buyer.Email != "" && sale.CheckoutToken != nil {
		h.notifications.SendOrderConfirmed(ctx, buyer.Email, sale.BuyerName, product.Title, sale.AmountCFA, *sale.CheckoutToken, h.fileAttachment(ctx, product.FileKey))
	}

	vendor, err := h.userRepo.FindByID(ctx, product.VendorID)
	if err != nil {
		return
	}
	if vendor.Email != "" {
		h.notifications.SendVendorSale(ctx, vendor.Email, product.Title, sale.VendorAmountCFA)
	}
}

// fileAttachment télécharge le fichier acheté pour le joindre à l'email de
// confirmation. Retourne nil (pas d'échec) si le stockage n'est pas
// configuré ou si le téléchargement échoue — l'email part quand même, avec
// juste le lien de téléchargement (voir NotificationService.SendOrderConfirmed).
func (h *WebhookHandler) fileAttachment(ctx context.Context, fileKey string) *email.Attachment {
	if h.storage == nil || fileKey == "" {
		return nil
	}
	content, err := h.storage.Download(ctx, fileKey)
	if err != nil {
		return nil
	}
	contentType := mime.TypeByExtension(filepath.Ext(fileKey))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &email.Attachment{
		Filename:    filepath.Base(fileKey),
		Content:     content,
		ContentType: contentType,
	}
}
