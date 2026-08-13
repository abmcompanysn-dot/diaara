package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/storage"
	"github.com/go-chi/chi/v5"
)

const (
	deliveryMaxDownloads = 3
	deliveryExpiry       = 30 * time.Minute
	signedURLExpiry      = 5 * time.Minute
)

type DeliveryHandler struct {
	deliveryRepo *repository.DeliveryRepo
	saleRepo     *repository.SaleRepo
	productRepo  *repository.ProductRepo
	storage      *storage.S3Storage
}

func NewDeliveryHandler(
	deliveryRepo *repository.DeliveryRepo,
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
	storage *storage.S3Storage,
) *DeliveryHandler {
	return &DeliveryHandler{
		deliveryRepo: deliveryRepo,
		saleRepo:     saleRepo,
		productRepo:  productRepo,
		storage:      storage,
	}
}

// Generate — l'acheteur d'une commande payée peut générer/récupérer son lien de livraison.
// POST /api/orders/{id}/delivery
func (h *DeliveryHandler) Generate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	saleID := chi.URLParam(r, "id")
	sale, err := h.saleRepo.FindByID(r.Context(), saleID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	if sale.BuyerID != userID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	h.generate(w, r, sale)
}

// GenerateByToken — l'acheteur invité (sans session) récupère son lien de
// livraison via le checkout_token reçu à la commande. Mêmes règles que
// Generate, mais accessible sans authentification.
// POST /api/orders/delivery?token=...
func (h *DeliveryHandler) GenerateByToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"token_required"}`, http.StatusBadRequest)
		return
	}
	sale, err := h.saleRepo.FindByCheckoutToken(r.Context(), token)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	h.generate(w, r, sale)
}

func (h *DeliveryHandler) generate(w http.ResponseWriter, r *http.Request, sale *model.Sale) {
	if sale.Status != string(model.SalePaid) && sale.Status != string(model.SaleDelivered) {
		http.Error(w, `{"error":"order_not_paid"}`, http.StatusBadRequest)
		return
	}

	// Lien déjà existant ?
	if existing, err := h.deliveryRepo.FindBySale(r.Context(), sale.ID); err == nil {
		url, err := h.signedURLForSale(r.Context(), sale)
		if err != nil {
			http.Error(w, `{"error":"signing_failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"delivery":   existing,
			"signed_url": url,
		})
		return
	}

	// Création du lien de livraison
	token, err := generateDeliveryToken()
	if err != nil {
		http.Error(w, `{"error":"token_generation_failed"}`, http.StatusInternalServerError)
		return
	}

	delivery, err := h.deliveryRepo.Create(r.Context(), sale.ID, token, deliveryMaxDownloads, time.Now().Add(deliveryExpiry))
	if err != nil {
		http.Error(w, `{"error":"delivery_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	url, err := h.signedURLForSale(r.Context(), sale)
	if err != nil {
		http.Error(w, `{"error":"signing_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"delivery":   delivery,
		"signed_url": url,
	})
}

// Download — GET /api/delivery/{token} → redirige vers l'URL signée du stockage objet.
func (h *DeliveryHandler) Download(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	delivery, err := h.deliveryRepo.FindByToken(r.Context(), token)
	if err != nil {
		http.Error(w, `{"error":"link_not_found"}`, http.StatusNotFound)
		return
	}

	if time.Now().After(delivery.ExpiresAt) {
		http.Error(w, `{"error":"link_expired"}`, http.StatusGone)
		return
	}

	if delivery.DownloadCount >= delivery.MaxDownloads {
		http.Error(w, `{"error":"download_limit_reached"}`, http.StatusTooManyRequests)
		return
	}

	sale, err := h.saleRepo.FindByID(r.Context(), delivery.SaleID)
	if err != nil {
		http.Error(w, `{"error":"sale_not_found"}`, http.StatusNotFound)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), sale.ProductID)
	if err != nil {
		http.Error(w, `{"error":"product_not_found"}`, http.StatusNotFound)
		return
	}

	url, err := h.storage.GenerateSignedURL(r.Context(), product.FileKey, signedURLExpiry)
	if err != nil {
		http.Error(w, `{"error":"signing_failed"}`, http.StatusInternalServerError)
		return
	}

	if err := h.deliveryRepo.IncrementDownloadCount(r.Context(), delivery.ID); err != nil {
		http.Error(w, `{"error":"count_failed"}`, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

func (h *DeliveryHandler) signedURLForSale(ctx context.Context, sale *model.Sale) (string, error) {
	product, err := h.productRepo.FindByID(ctx, sale.ProductID)
	if err != nil {
		return "", err
	}
	return h.storage.GenerateSignedURL(ctx, product.FileKey, signedURLExpiry)
}

func generateDeliveryToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
