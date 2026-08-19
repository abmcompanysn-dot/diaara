package handler

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/diarra/backend/internal/cache"
	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/diarra/backend/internal/storage"
)

type WebhookHandler struct {
	saleRepo         *repository.SaleRepo
	userRepo         *repository.UserRepo
	productRepo      *repository.ProductRepo
	payoutRepo       *repository.PayoutRepo
	pawapay          *payment.PawaPayClient
	commissionSvc    *service.CommissionService
	donationSvc      *service.DonationService
	notifications    *email.NotificationService
	notificationRepo *repository.NotificationRepo
	storage          *storage.S3Storage
	allowedIPs       map[string]bool
	cache            *cache.Client
}

func NewWebhookHandler(
	saleRepo *repository.SaleRepo,
	userRepo *repository.UserRepo,
	productRepo *repository.ProductRepo,
	payoutRepo *repository.PayoutRepo,
	pawapay *payment.PawaPayClient,
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
		saleRepo:         saleRepo,
		userRepo:         userRepo,
		productRepo:      productRepo,
		payoutRepo:       payoutRepo,
		pawapay:          pawapay,
		commissionSvc:    service.NewCommissionService(),
		donationSvc:      donationSvc,
		notifications:    notifications,
		notificationRepo: notificationRepo,
		storage:          storage,
		allowedIPs:       ips,
		cache:            cacheClient,
	}
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

	payout, err := h.payoutRepo.FindByPawaPayID(r.Context(), payload.PayoutId)
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
		h.notify(r.Context(), payout.UserID, "payout_paid", "Versement effectué",
			fmt.Sprintf("Votre versement de %d FCFA a été envoyé.", payout.AmountCFA), "/vendor/earnings")
	case "FAILED":
		reason := ""
		if status.Data.FailureReason != nil {
			reason = status.Data.FailureReason.FailureCode
		}
		h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "failed", &reason)
		h.notify(r.Context(), payout.UserID, "payout_failed", "Versement échoué",
			"Votre demande de versement a été refusée, vérifiez votre moyen de versement.", "/vendor/earnings")
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
			// La vente vient de créditer ce vendeur : le solde caché
			// (PayoutHandler.Earnings) serait sinon obsolète jusqu'à
			// expiration de son TTL (30s).
			h.cache.Del(r.Context(), vendorBalanceCacheKey(product.VendorID))
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
