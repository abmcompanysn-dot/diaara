package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/diarra/backend/internal/storage"
	"github.com/go-chi/chi/v5"
)

type ProductHandler struct {
	productRepo *repository.ProductRepo
	storage     StorageService
}

// StorageService est l'interface minimale dont le handler a besoin.
type StorageService interface {
	Upload(ctx context.Context, key string, data []byte) error
}

func NewProductHandler(productRepo *repository.ProductRepo, storage StorageService) *ProductHandler {
	return &ProductHandler{productRepo: productRepo, storage: storage}
}

// List — public, produits approuvés
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")

	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	products, err := h.productRepo.ListApproved(r.Context(), category, search, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"products": products})
}

// ListVendor — vendeur authentifié, ses propres produits
func (h *ProductHandler) ListVendor(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	products, err := h.productRepo.FindByVendor(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"products": products})
}

// Get — public, fiche détaillée
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrProductNotFound {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"get_failed"}`, http.StatusInternalServerError)
		return
	}

	// Un produit non approuvé n'est visible que par son vendeur ou un admin
	if product.ModerationStatus != "approved" {
		userID := middleware.GetUserID(r.Context())
		isAdmin := middleware.GetIsAdmin(r.Context())
		if userID != product.VendorID && !isAdmin {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"product": product})
}

// Upload — vendeur authentifié, fichier multipart → R2
func (h *ProductHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if h.storage == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		http.Error(w, `{"error":"file_too_large"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file_required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusInternalServerError)
		return
	}

	key := storage.NewFileKey(userID, header.Filename)
	if err := h.storage.Upload(r.Context(), key, data); err != nil {
		http.Error(w, `{"error":"upload_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"file_key": key,
		"size":     strconv.Itoa(len(data)),
	})
}

// Create — vendeur authentifié
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.Title == "" || input.PriceCFA < 0 || input.Category == "" || input.FileKey == "" {
		http.Error(w, `{"error":"title_price_category_file_required"}`, http.StatusBadRequest)
		return
	}

	if !validAffiliateConfig(input.AffiliateEnabled, input.MaxCloserCommissionPct) {
		http.Error(w, `{"error":"invalid_affiliate_config"}`, http.StatusBadRequest)
		return
	}

	product, err := h.productRepo.Create(r.Context(), input, userID)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"product": product})
}

// Update — vendeur propriétaire
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if product.VendorID != userID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	var input model.UpdateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	// Si le statut d'affiliation est modifié, vérifier la cohérence du plafond.
	if input.AffiliateEnabled != nil {
		capVal := product.MaxCloserCommissionPct
		if input.MaxCloserCommissionPct != nil {
			capVal = *input.MaxCloserCommissionPct
		}
		if !validAffiliateConfig(*input.AffiliateEnabled, capVal) {
			http.Error(w, `{"error":"invalid_affiliate_config"}`, http.StatusBadRequest)
			return
		}
	}

	updated, err := h.productRepo.Update(r.Context(), id, input)
	if err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"product": updated})
}

// Delete — vendeur propriétaire ou admin
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())
	id := chi.URLParam(r, "id")

	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if product.VendorID != userID && !isAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	if err := h.productRepo.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// validAffiliateConfig vérifie la cohérence des paramètres d'affiliation :
// si le produit est ouvert à l'affiliation, le plafond doit être strictement
// positif et ne pas dépasser la part vendeur (100% - 15% plateforme).
func validAffiliateConfig(enabled bool, maxPct float64) bool {
	if !enabled {
		return true
	}
	return maxPct > 0 && maxPct <= (100-float64(service.PlatformFeePct))
}
