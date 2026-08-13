package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// s3Pinger est l'interface minimale utilisée pour le health check infra
// (évite une dépendance directe sur le package storage).
type s3Pinger interface {
	Ping(ctx context.Context) error
}

type AdminHandler struct {
	productRepo   *repository.ProductRepo
	saleRepo      *repository.SaleRepo
	userRepo      *repository.UserRepo
	referralRepo  *repository.ReferralRepo
	adminPermRepo *repository.AdminPermissionRepo
	pool          *pgxpool.Pool
	storage       s3Pinger // nil si stockage objet non configuré
	startTime     time.Time
}

func NewAdminHandler(
	productRepo *repository.ProductRepo,
	saleRepo *repository.SaleRepo,
	userRepo *repository.UserRepo,
	referralRepo *repository.ReferralRepo,
	adminPermRepo *repository.AdminPermissionRepo,
	pool *pgxpool.Pool,
	storage s3Pinger,
	startTime time.Time,
) *AdminHandler {
	return &AdminHandler{
		productRepo:   productRepo,
		saleRepo:      saleRepo,
		userRepo:      userRepo,
		referralRepo:  referralRepo,
		adminPermRepo: adminPermRepo,
		pool:          pool,
		storage:       storage,
		startTime:     startTime,
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

// SetRole — PUT /api/admin/users/{id}/role
// Accord (grant) ou retire (revoke) un rôle cumulable (vendeur/closer).
// Les sessions existantes de l'utilisateur sont révoquées pour qu'il se
// reconnecte avec un token reflétant ses nouveaux rôles.
func (h *AdminHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input model.RoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if !model.ValidRole(input.Role) {
		http.Error(w, `{"error":"invalid_role"}`, http.StatusBadRequest)
		return
	}

	var err error
	switch input.Action {
	case "grant":
		err = h.userRepo.AddRole(r.Context(), id, input.Role)
	case "revoke":
		err = h.userRepo.RemoveRole(r.Context(), id, input.Role)
	default:
		http.Error(w, `{"error":"invalid_action"}`, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"role_update_failed"}`, http.StatusInternalServerError)
		return
	}

	_ = h.userRepo.RevokeAllUserRefreshTokens(r.Context(), id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "role_updated"})
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

// Admins — GET /api/admin/admins (RequireUnrestrictedAdmin)
func (h *AdminHandler) Admins(w http.ResponseWriter, r *http.Request) {
	admins, err := h.adminPermRepo.ListAdmins(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"admins": admins})
}

// SetAdminStatus — PUT /api/admin/users/{id}/admin (RequireUnrestrictedAdmin)
// Promeut ou rétrograde un utilisateur au statut administrateur. Interdit de
// se rétrograder soi-même (évite un verrouillage accidentel).
func (h *AdminHandler) SetAdminStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input model.AdminStatusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.Action == "revoke" && id == middleware.GetUserID(r.Context()) {
		http.Error(w, `{"error":"self_demote"}`, http.StatusForbidden)
		return
	}

	var isAdmin bool
	switch input.Action {
	case "grant":
		isAdmin = true
	case "revoke":
		isAdmin = false
	default:
		http.Error(w, `{"error":"invalid_action"}`, http.StatusBadRequest)
		return
	}

	if err := h.userRepo.SetAdminStatus(r.Context(), id, isAdmin); err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	_ = h.userRepo.RevokeAllUserRefreshTokens(r.Context(), id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "admin_status_updated"})
}

// SetAdminPermission — PUT /api/admin/admins/{id}/permission (RequireUnrestrictedAdmin)
func (h *AdminHandler) SetAdminPermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input model.AdminPermissionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if !model.ValidAdminPermission(input.Permission) {
		http.Error(w, `{"error":"invalid_permission"}`, http.StatusBadRequest)
		return
	}

	var err error
	switch input.Action {
	case "grant":
		err = h.adminPermRepo.Grant(r.Context(), id, input.Permission)
	case "revoke":
		err = h.adminPermRepo.Revoke(r.Context(), id, input.Permission)
	default:
		http.Error(w, `{"error":"invalid_action"}`, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"permission_update_failed"}`, http.StatusInternalServerError)
		return
	}
	_ = h.userRepo.RevokeAllUserRefreshTokens(r.Context(), id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "permission_updated"})
}

// SystemHealth — GET /api/admin/system/health (scope "infra")
func (h *AdminHandler) SystemHealth(w http.ResponseWriter, r *http.Request) {
	health := model.SystemHealth{
		Database:      "ok",
		UptimeSeconds: int64(time.Since(h.startTime).Seconds()),
	}

	if err := h.pool.Ping(r.Context()); err != nil {
		health.Database = "error"
	}

	switch {
	case h.storage == nil:
		health.Storage = "disabled"
	case h.storage.Ping(r.Context()) != nil:
		health.Storage = "error"
	default:
		health.Storage = "ok"
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	health.GoroutineCount = runtime.NumGoroutine()
	health.MemAllocMB = float64(mem.Alloc) / 1024 / 1024
	health.MemSysMB = float64(mem.Sys) / 1024 / 1024

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Analytics — GET /api/admin/analytics (scope "finance")
func (h *AdminHandler) Analytics(w http.ResponseWriter, r *http.Request) {
	salesByDay, err := h.saleRepo.SalesByDay(r.Context(), 30)
	if err != nil {
		http.Error(w, `{"error":"analytics_failed"}`, http.StatusInternalServerError)
		return
	}
	topProducts, err := h.saleRepo.TopProducts(r.Context(), 10)
	if err != nil {
		http.Error(w, `{"error":"analytics_failed"}`, http.StatusInternalServerError)
		return
	}
	topVendors, err := h.saleRepo.TopVendors(r.Context(), 10)
	if err != nil {
		http.Error(w, `{"error":"analytics_failed"}`, http.StatusInternalServerError)
		return
	}
	topClosers, err := h.referralRepo.TopClosersByConversion(r.Context(), 10)
	if err != nil {
		http.Error(w, `{"error":"analytics_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.AnalyticsOverview{
		SalesByDay:  salesByDay,
		TopProducts: topProducts,
		TopVendors:  topVendors,
		TopClosers:  topClosers,
	})
}
