package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
)

type WebhookHandler struct {
	saleRepo       *repository.SaleRepo
	userRepo       *repository.UserRepo
	productRepo    *repository.ProductRepo
	commissionSvc  *service.CommissionService
	notifications  *email.NotificationService
	webhookSecret  string
}

func NewWebhookHandler(
	saleRepo *repository.SaleRepo,
	userRepo *repository.UserRepo,
	productRepo *repository.ProductRepo,
	notifications *email.NotificationService,
	webhookSecret string,
) *WebhookHandler {
	return &WebhookHandler{
		saleRepo:      saleRepo,
		userRepo:      userRepo,
		productRepo:   productRepo,
		commissionSvc: service.NewCommissionService(),
		notifications: notifications,
		webhookSecret: webhookSecret,
	}
}

// PayDunyaWebhookPayload est le corps typique du callback PayDunya.
type PayDunyaWebhookPayload struct {
	Token        string `json:"token"`
	Status       string `json:"status"`
	ResponseCode string `json:"response_code"`
	ResponseText string `json:"response_text"`
	InvoiceToken string `json:"invoice_token"`
	CustomData   struct {
		SaleID string `json:"sale_id"`
	} `json:"custom_data"`
}

// PayDunyaWebhook reçoit la confirmation de paiement depuis PayDunya.
func (h *WebhookHandler) PayDunyaWebhook(w http.ResponseWriter, r *http.Request) {
	// Vérification signature (si PayDunya envoie un header de signature)
	sig := r.Header.Get("PAYDUNYA-SIGNATURE")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusBadRequest)
		return
	}
	if h.webhookSecret != "" && sig != "" {
		if !h.verifySignature(body, sig) {
			http.Error(w, `{"error":"invalid_signature"}`, http.StatusUnauthorized)
			return
		}
	}

	var payload PayDunyaWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	sale, err := h.saleRepo.FindByPaymentReference(r.Context(), payload.Token)
	if err != nil {
		sale, err = h.saleRepo.FindByPaymentReference(r.Context(), payload.InvoiceToken)
	}
	if err != nil && payload.CustomData.SaleID != "" {
		sale, err = h.saleRepo.FindByID(r.Context(), payload.CustomData.SaleID)
	}
	if err != nil {
		http.Error(w, `{"error":"sale_not_found"}`, http.StatusNotFound)
		return
	}

	if payload.ResponseCode == "00" || payload.Status == "completed" {
		// Confirmer le paiement
		if err := h.saleRepo.UpdateStatus(r.Context(), sale.ID, string(model.SalePaid)); err != nil {
			http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
			return
		}

		// Notifications email
		if h.notifications != nil {
			go h.notifyPaid(r.Context(), sale)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "paid"})
		return
	}

	if err := h.saleRepo.UpdateStatus(r.Context(), sale.ID, string(model.SaleFailed)); err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "failed"})
}

func (h *WebhookHandler) verifySignature(body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// notifyPaid envoie les emails de confirmation (acheteur + vendeur) en arrière-plan.
func (h *WebhookHandler) notifyPaid(ctx context.Context, sale *model.Sale) {
	buyer, err := h.userRepo.FindByID(ctx, sale.BuyerID)
	if err != nil {
		return
	}
	if buyer.Email != "" {
		h.notifications.SendOrderConfirmed(ctx, buyer.Email, sale.ID)
	}

	product, err := h.productRepo.FindByID(ctx, sale.ProductID)
	if err != nil {
		return
	}
	vendor, err := h.userRepo.FindByID(ctx, product.VendorID)
	if err != nil {
		return
	}
	if vendor.Email != "" {
		h.notifications.SendVendorSale(ctx, vendor.Email, product.Title, sale.VendorAmountCFA)
	}
}
