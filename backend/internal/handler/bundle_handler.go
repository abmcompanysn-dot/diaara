package handler

import (
	"encoding/json"
	"net/http"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type BundleHandler struct {
	bundleRepo  *repository.BundleRepo
	productRepo *repository.ProductRepo
}

func NewBundleHandler(bundleRepo *repository.BundleRepo, productRepo *repository.ProductRepo) *BundleHandler {
	return &BundleHandler{bundleRepo: bundleRepo, productRepo: productRepo}
}

// Create — POST /api/vendor/bundles
func (h *BundleHandler) Create(w http.ResponseWriter, r *http.Request) {
	vendorID := middleware.GetUserID(r.Context())
	if vendorID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.CreateBundleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.Title == "" || input.PriceCFA <= 0 || len(input.ProductIDs) < 2 {
		http.Error(w, `{"error":"title_price_and_two_products_required"}`, http.StatusBadRequest)
		return
	}

	// Chaque produit inclus doit appartenir au vendeur qui crée le pack.
	for _, productID := range input.ProductIDs {
		product, err := h.productRepo.FindByID(r.Context(), productID)
		if err != nil {
			http.Error(w, `{"error":"product_not_found"}`, http.StatusNotFound)
			return
		}
		if product.VendorID != vendorID {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
	}

	bundle, err := h.bundleRepo.Create(r.Context(), input, vendorID)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"bundle": bundle})
}

// ListVendor — GET /api/vendor/bundles
func (h *BundleHandler) ListVendor(w http.ResponseWriter, r *http.Request) {
	vendorID := middleware.GetUserID(r.Context())
	if vendorID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	bundles, err := h.bundleRepo.ListByVendor(r.Context(), vendorID)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"bundles": bundles})
}

// Get — GET /api/bundles/{id} (public) — le pack et ses produits inclus.
func (h *BundleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	bundle, err := h.bundleRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	items, err := h.bundleRepo.ListItems(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"items_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"bundle": bundle, "products": items})
}

// Delete — DELETE /api/vendor/bundles/{id}
func (h *BundleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vendorID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	bundle, err := h.bundleRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if bundle.VendorID != vendorID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	if err := h.bundleRepo.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
