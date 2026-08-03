package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	productRepo *repository.ProductRepo
	saleRepo    *repository.SaleRepo
	userRepo    *repository.UserRepo
}

func NewAdminHandler(
	productRepo *repository.ProductRepo,
	saleRepo *repository.SaleRepo,
	userRepo *repository.UserRepo,
) *AdminHandler {
	return &AdminHandler{
		productRepo: productRepo,
		saleRepo:    saleRepo,
		userRepo:    userRepo,
	}
}

// PendingProducts — GET /api/admin/products/pending
func (h *AdminHandler) PendingProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.productRepo.ListPending(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"products": products})
}

// Moderate — PUT /api/admin/products/{id}/moderate
func (h *AdminHandler) Moderate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input model.ModerateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.Status != "approved" && input.Status != "rejected" {
		http.Error(w, `{"error":"invalid_status"}`, http.StatusBadRequest)
		return
	}

	if err := h.productRepo.UpdateModerationStatus(r.Context(), id, input.Status, input.Note); err != nil {
		http.Error(w, `{"error":"moderation_failed"}`, http.StatusInternalServerError)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"product": product})
}

// SuspendUser — PUT /api/admin/users/{id}/suspend
func (h *AdminHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	until := time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)

	if err := h.userRepo.LockAccount(r.Context(), id, until); err != nil {
		http.Error(w, `{"error":"suspension_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "suspended"})
}

// Users — GET /api/admin/users
func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListAllUsers(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": users})
}

// Stats — GET /api/admin/stats
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	totalSales, err := h.saleRepo.Count(r.Context())
	if err != nil {
		http.Error(w, `{"error":"stats_failed"}`, http.StatusInternalServerError)
		return
	}

	totalProducts, err := h.productRepo.Count(r.Context())
	if err != nil {
		http.Error(w, `{"error":"stats_failed"}`, http.StatusInternalServerError)
		return
	}

	totalUsers, err := h.userRepo.CountAllUsers(r.Context())
	if err != nil {
		http.Error(w, `{"error":"stats_failed"}`, http.StatusInternalServerError)
		return
	}

	totalRevenue, err := h.totalRevenue(r.Context())
	if err != nil {
		http.Error(w, `{"error":"stats_failed"}`, http.StatusInternalServerError)
		return
	}

	pendingCount, err := h.pendingCount(r.Context())
	if err != nil {
		http.Error(w, `{"error":"stats_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_sales":        totalSales,
		"total_products":     totalProducts,
		"total_users":        totalUsers,
		"total_revenue":      totalRevenue,
		"pending_moderation": pendingCount,
	})
}

// Sales — GET /api/admin/sales
func (h *AdminHandler) Sales(w http.ResponseWriter, r *http.Request) {
	sales, err := h.saleRepo.ListAll(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"sales": sales})
}

func (h *AdminHandler) totalRevenue(ctx context.Context) (int, error) {
	sales, err := h.saleRepo.ListAll(ctx)
	if err != nil {
		return 0, err
	}

	total := 0
	for _, s := range sales {
		if s.Status != "failed" && s.Status != "refunded" {
			total += s.PlatformFeeCFA
		}
	}
	return total, nil
}

func (h *AdminHandler) pendingCount(ctx context.Context) (int, error) {
	products, err := h.productRepo.ListPending(ctx)
	if err != nil {
		return 0, err
	}
	return len(products), nil
}
