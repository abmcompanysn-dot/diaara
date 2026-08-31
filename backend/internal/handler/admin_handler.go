package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/cache"
	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
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
	payoutRepo    *repository.PayoutRepo
	settingsRepo  *repository.SettingsRepo
	ticketRepo    *repository.TicketRepo
	pool          *pgxpool.Pool
	storage       s3Pinger // nil si stockage objet non configuré
	// files : accès complet au stockage objet (URL signée) pour permettre à
	// un modérateur de télécharger le fichier livrable d'un produit avant de
	// l'approuver. nil si stockage non configuré.
	files         StorageService
	startTime     time.Time
	pawapay       *payment.PawaPayClient
	kpay          *payment.KPayClient
	notifications *email.NotificationService // nil si aucun fournisseur email configuré
	cache         *cache.Client
}

func NewAdminHandler(
	productRepo *repository.ProductRepo,
	saleRepo *repository.SaleRepo,
	userRepo *repository.UserRepo,
	referralRepo *repository.ReferralRepo,
	adminPermRepo *repository.AdminPermissionRepo,
	payoutRepo *repository.PayoutRepo,
	settingsRepo *repository.SettingsRepo,
	ticketRepo *repository.TicketRepo,
	pool *pgxpool.Pool,
	storage s3Pinger,
	files StorageService,
	startTime time.Time,
	pawapay *payment.PawaPayClient,
	kpay *payment.KPayClient,
	notifications *email.NotificationService,
	cacheClient *cache.Client,
) *AdminHandler {
	return &AdminHandler{
		productRepo:   productRepo,
		saleRepo:      saleRepo,
		userRepo:      userRepo,
		referralRepo:  referralRepo,
		adminPermRepo: adminPermRepo,
		payoutRepo:    payoutRepo,
		settingsRepo:  settingsRepo,
		ticketRepo:    ticketRepo,
		pool:          pool,
		storage:       storage,
		files:         files,
		startTime:     startTime,
		kpay:          kpay,
		pawapay:       pawapay,
		notifications: notifications,
		cache:         cacheClient,
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

// ListProducts — GET /api/admin/products?status=pending|approved|rejected
// (status omis = tous). Inclut l'email du vendeur, pour revoir/reverser une
// décision de modération passée (contrairement à PendingProducts qui ne
// montre que la file d'attente).
func (h *AdminHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	products, err := h.productRepo.ListForAdmin(r.Context(), status)
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

	// Notifie le vendeur de la décision par email (best-effort, non bloquant :
	// la modération est déjà enregistrée, un échec d'envoi ne doit pas la
	// faire échouer).
	if h.notifications != nil {
		note := ""
		if input.Note != nil {
			note = *input.Note
		}
		vendorID, title, status := product.VendorID, product.Title, input.Status
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			vendor, err := h.userRepo.FindByID(ctx, vendorID)
			if err != nil || vendor.Email == "" {
				return
			}
			_ = h.notifications.SendProductModerated(ctx, vendor.Email, title, status, note)
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"product": product})
}

// DownloadProductFile — GET /api/admin/products/{id}/download : renvoie une
// URL signée de courte durée vers le fichier livrable du produit, pour qu'un
// modérateur puisse en vérifier le contenu avant d'approuver. Même mécanique
// que le téléchargement acheteur (voir DeliveryHandler), mais réservé aux
// admins ayant le scope modération.
func (h *AdminHandler) DownloadProductFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if h.files == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if product.FileKey == "" {
		http.Error(w, `{"error":"no_file"}`, http.StatusNotFound)
		return
	}

	url, err := h.files.GenerateSignedURL(r.Context(), product.FileKey, 5*time.Minute)
	if err != nil {
		http.Error(w, `{"error":"signed_url_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"signed_url": url})
}

// ConfirmDeletion — DELETE /api/admin/products/{id} : supprime
// définitivement un produit dont le vendeur a demandé la suppression.
func (h *AdminHandler) ConfirmDeletion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.productRepo.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// CancelDeletion — PUT /api/admin/products/{id}/cancel-deletion : rejette la
// demande de suppression du vendeur, le produit reste actif.
func (h *AdminHandler) CancelDeletion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.productRepo.CancelDeletionRequest(r.Context(), id); err != nil {
		http.Error(w, `{"error":"cancel_failed"}`, http.StatusInternalServerError)
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
	users, err := h.userRepo.ListAllUsersWithStats(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": users})
}

// ReactivateUser — PUT /api/admin/users/{id}/reactivate
func (h *AdminHandler) ReactivateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.userRepo.UnlockAccount(r.Context(), id); err != nil {
		http.Error(w, `{"error":"reactivation_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "active"})
}

// Stats — GET /api/admin/stats
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	const cacheKey = "admin:stats"
	var cached map[string]interface{}
	if hit, _ := h.cache.GetJSON(r.Context(), cacheKey, &cached); hit {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

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

	roleCounts, err := h.userRepo.CountByRole(r.Context())
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

	gmv, err := h.saleRepo.GMV(r.Context())
	if err != nil {
		http.Error(w, `{"error":"stats_failed"}`, http.StatusInternalServerError)
		return
	}

	revenueThisMonth, revenueLastMonth, err := h.saleRepo.RevenueThisAndLastMonth(r.Context())
	if err != nil {
		http.Error(w, `{"error":"stats_failed"}`, http.StatusInternalServerError)
		return
	}
	growthPct := 0.0
	if revenueLastMonth > 0 {
		growthPct = (float64(revenueThisMonth) - float64(revenueLastMonth)) / float64(revenueLastMonth) * 100
	} else if revenueThisMonth > 0 {
		growthPct = 100
	}

	stats := map[string]interface{}{
		"total_sales":        totalSales,
		"total_products":     totalProducts,
		"pending_moderation": pendingCount,
		"active_products":    totalProducts - pendingCount,
		"total_users":        roleCounts.Total,
		"total_vendors":      roleCounts.Vendors,
		"total_closers":      roleCounts.Closers,
		"total_admins":       roleCounts.Admins,
		"total_revenue":      totalRevenue,
		"gmv":                gmv,
		"revenue_this_month": revenueThisMonth,
		"revenue_last_month": revenueLastMonth,
		"revenue_growth_pct": growthPct,
	}
	h.cache.SetJSON(r.Context(), cacheKey, stats, 30*time.Second)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetSettings — GET /api/admin/settings (scope "finance")
func (h *AdminHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsRepo.All(r.Context())
	if err != nil {
		http.Error(w, `{"error":"settings_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"settings": settings})
}

// UpdateSettings — PUT /api/admin/settings (scope "finance")
func (h *AdminHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var input model.UpdateSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if rate, ok := input[model.SettingCommissionRatePct]; ok {
		if f, err := parseRate(rate); err != nil || f < 0 || f > 100 {
			http.Error(w, `{"error":"invalid_commission_rate"}`, http.StatusBadRequest)
			return
		}
	}
	if pct, ok := input[model.SettingDonationSharePct]; ok {
		if f, err := parseRate(pct); err != nil || f < 0 || f > 100 {
			http.Error(w, `{"error":"invalid_donation_share_pct"}`, http.StatusBadRequest)
			return
		}
	}
	if threshold, ok := input[model.SettingDonationThresholdCFA]; ok {
		if f, err := parseRate(threshold); err != nil || f < 0 {
			http.Error(w, `{"error":"invalid_donation_threshold"}`, http.StatusBadRequest)
			return
		}
	}
	if enabled, ok := input[model.SettingDonationEnabled]; ok {
		if enabled != "true" && enabled != "false" {
			http.Error(w, `{"error":"invalid_donation_enabled"}`, http.StatusBadRequest)
			return
		}
	}
	// Réglages par opérateur (versements) et par pays (checkout) — voir
	// model.GatewayOperatorSettingKey / CheckoutProviderSettingKey. Empêche
	// notamment d'assigner KPay à un opérateur qu'il ne supporte pas (ex.
	// Wave, suspendu côté KPay).
	for key, value := range input {
		switch {
		case strings.HasPrefix(key, "gateway_op_"):
			if value != "off" && value != "pawapay" && value != "kpay" {
				http.Error(w, `{"error":"invalid_gateway_provider"}`, http.StatusBadRequest)
				return
			}
			if value == "kpay" {
				code := strings.ToUpper(strings.TrimPrefix(key, "gateway_op_"))
				if !payment.KPayProviderCodes[code] {
					http.Error(w, fmt.Sprintf(`{"error":"kpay_unsupported_operator","operator":"%s"}`, code), http.StatusBadRequest)
					return
				}
			}
		case strings.HasPrefix(key, "checkout_provider_"):
			if value != "pawapay" && value != "kpay" {
				http.Error(w, `{"error":"invalid_checkout_provider"}`, http.StatusBadRequest)
				return
			}
		}
	}
	if err := h.settingsRepo.SetMany(r.Context(), input); err != nil {
		http.Error(w, `{"error":"settings_update_failed"}`, http.StatusInternalServerError)
		return
	}
	settings, err := h.settingsRepo.All(r.Context())
	if err != nil {
		http.Error(w, `{"error":"settings_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"settings": settings})
}

// GetAutomationKey — GET /api/admin/automation/key (scope "finance").
// Renvoie la clé actuelle sans la régénérer ; vide si aucune n'a encore été
// générée.
func (h *AdminHandler) GetAutomationKey(w http.ResponseWriter, r *http.Request) {
	key := h.settingsRepo.Get(r.Context(), model.SettingAutomationAPIKey, "")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"key": key})
}

// RegenerateAutomationKey — POST /api/admin/automation/key/regenerate
// (scope "finance"). Génère une nouvelle clé et invalide l'ancienne
// immédiatement (un seul appel de script/IA en cours l'utilisant échouera
// après régénération — comportement voulu pour pouvoir révoquer un accès).
func (h *AdminHandler) RegenerateAutomationKey(w http.ResponseWriter, r *http.Request) {
	key, err := auth.GenerateToken()
	if err != nil {
		http.Error(w, `{"error":"key_generation_failed"}`, http.StatusInternalServerError)
		return
	}
	if err := h.settingsRepo.Set(r.Context(), model.SettingAutomationAPIKey, key); err != nil {
		http.Error(w, `{"error":"settings_update_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"key": key})
}

func parseRate(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// Payouts — GET /api/admin/payouts (scope "finance") : tous les versements,
// tous vendeurs/affiliés confondus.
func (h *AdminHandler) Payouts(w http.ResponseWriter, r *http.Request) {
	payouts, err := h.payoutRepo.ListAllAdmin(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"payouts": payouts})
}

// RetryPayout — POST /api/admin/payouts/{id}/retry (scope "finance") :
// relance un versement resté en échec (nouvel essai PawaPay avec un nouvel ID).
func (h *AdminHandler) RetryPayout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	payout, err := h.payoutRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if payout.Status != "failed" {
		http.Error(w, `{"error":"payout_not_retryable"}`, http.StatusBadRequest)
		return
	}

	// Le prestataire est celui déjà résolu à la création du versement (voir
	// PayoutHandler.Create) — une relance ne doit jamais basculer un
	// versement vers un autre prestataire que celui d'origine.
	if payout.Provider == "kpay" {
		if h.kpay == nil {
			http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
			return
		}
		resp, err := h.kpay.InitiatePayout(r.Context(), payment.PayoutInitRequest{
			Amount:      fmt.Sprintf("%d", payout.AmountCFA),
			Provider:    payout.Operator,
			PhoneNumber: payout.PhoneNumber,
			ExternalId:  payout.ID,
			Description: "Versement DIARRA",
		})
		if err != nil {
			http.Error(w, `{"error":"payout_rejected","reason":"payout_retry_failed"}`, http.StatusBadGateway)
			return
		}
		if err := h.payoutRepo.SetProviderReference(r.Context(), payout.ID, resp.ID); err != nil {
			http.Error(w, `{"error":"payout_update_failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "processing"})
		return
	}

	if h.pawapay == nil {
		http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	pawapayID := uuidString()
	resp, err := h.pawapay.InitiatePayout(r.Context(), payment.PayoutRequest{
		PayoutId: pawapayID,
		Recipient: payment.Payer{
			Type: "MMO",
			AccountDetails: payment.AccountDetails{
				PhoneNumber: payout.PhoneNumber,
				Provider:    payout.Operator,
			},
		},
		Amount:            fmt.Sprintf("%d", payout.AmountCFA),
		Currency:          "XOF",
		ClientReferenceId: payout.ID,
		CustomerMessage:   "VERSEMENT DIARRA",
	})
	if err != nil || resp.Status != "ACCEPTED" {
		reason := "payout_retry_failed"
		if err == nil && resp.FailureReason != nil {
			reason = resp.FailureReason.FailureCode
		}
		http.Error(w, fmt.Sprintf(`{"error":"payout_rejected","reason":"%s"}`, reason), http.StatusBadGateway)
		return
	}
	if err := h.payoutRepo.SetProviderReference(r.Context(), payout.ID, pawapayID); err != nil {
		http.Error(w, `{"error":"payout_update_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "processing"})
}

// ActivityFeed — GET /api/admin/activity (scope "finance") : dernières
// actions sur la plateforme (ventes, inscriptions, demandes de versement).
func (h *AdminHandler) ActivityFeed(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		(SELECT 'sale' AS kind, s.id, s.created_at AS at,
		        json_build_object('product_title', p.title, 'amount_cfa', s.amount_cfa, 'status', s.status) AS data
		 FROM sales s JOIN products p ON p.id = s.product_id
		 ORDER BY s.created_at DESC LIMIT 10)
		UNION ALL
		(SELECT 'user' AS kind, u.id, u.created_at AS at,
		        json_build_object('email', u.email) AS data
		 FROM users u ORDER BY u.created_at DESC LIMIT 10)
		UNION ALL
		(SELECT 'payout' AS kind, po.id, po.requested_at AS at,
		        json_build_object('amount_cfa', po.amount_cfa, 'status', po.status, 'email', u.email) AS data
		 FROM payouts po JOIN users u ON u.id = po.user_id
		 ORDER BY po.requested_at DESC LIMIT 10)
		ORDER BY at DESC LIMIT 20`)
	if err != nil {
		http.Error(w, `{"error":"activity_failed"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type activityItem struct {
		Kind string                 `json:"kind"`
		ID   string                 `json:"id"`
		At   time.Time              `json:"at"`
		Data map[string]interface{} `json:"data"`
	}
	items := []activityItem{}
	for rows.Next() {
		var it activityItem
		if err := rows.Scan(&it.Kind, &it.ID, &it.At, &it.Data); err != nil {
			http.Error(w, `{"error":"activity_failed"}`, http.StatusInternalServerError)
			return
		}
		items = append(items, it)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"activity": items})
}

// Notifications — GET /api/admin/notifications (tout admin, pas de scope
// dédié : chacun doit voir ce qui touche à son périmètre au minimum).
func (h *AdminHandler) Notifications(w http.ResponseWriter, r *http.Request) {
	pendingModeration, _ := h.pendingCount(r.Context())
	openTickets, _ := h.ticketRepo.CountOpen(r.Context())
	failedSales24h, _ := h.saleRepo.FailedCount24h(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pending_moderation": pendingModeration,
		"open_tickets":       openTickets,
		"failed_sales_24h":   failedSales24h,
		"total":              pendingModeration + openTickets + failedSales24h,
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

// RefundSale — POST /api/admin/sales/{id}/refund (remboursement total, mobile money)
func (h *AdminHandler) RefundSale(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sale, err := h.saleRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if sale.Status != string(model.SalePaid) {
		http.Error(w, `{"error":"sale_not_refundable"}`, http.StatusBadRequest)
		return
	}

	if sale.PaymentProvider == "kpay" {
		if h.kpay == nil {
			http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
			return
		}
		if sale.ProviderTransactionID == nil {
			http.Error(w, `{"error":"missing_provider_transaction_id"}`, http.StatusInternalServerError)
			return
		}
		resp, err := h.kpay.InitiateRefund(r.Context(), *sale.ProviderTransactionID, payment.RefundInitRequest{
			ExternalId: sale.ID,
		})
		if err != nil {
			http.Error(w, `{"error":"refund_rejected","reason":"refund_init_failed"}`, http.StatusBadGateway)
			return
		}
		if err := h.saleRepo.SetRefundReference(r.Context(), sale.ID, resp.ID); err != nil {
			http.Error(w, `{"error":"refund_update_failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "refund_pending"})
		return
	}

	if h.pawapay == nil {
		http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	refundId := uuidString()
	resp, err := h.pawapay.InitiateRefund(r.Context(), payment.RefundRequest{
		RefundId:          refundId,
		DepositId:         sale.PaymentReference,
		Currency:          "XOF",
		ClientReferenceId: sale.ID,
	})
	if err != nil || resp.Status != "ACCEPTED" {
		reason := "refund_init_failed"
		if err == nil && resp.FailureReason != nil {
			reason = resp.FailureReason.FailureCode
		}
		http.Error(w, fmt.Sprintf(`{"error":"refund_rejected","reason":"%s"}`, reason), http.StatusBadGateway)
		return
	}

	if err := h.saleRepo.SetRefundReference(r.Context(), sale.ID, refundId); err != nil {
		http.Error(w, `{"error":"refund_update_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "refund_pending", "refund_id": refundId})
}

func (h *AdminHandler) totalRevenue(ctx context.Context) (int, error) {
	sales, err := h.saleRepo.ListAll(ctx)
	if err != nil {
		return 0, err
	}

	total := 0
	for _, s := range sales {
		// Même règle que PayoutHandler.totalEarned : seules les ventes payées
		// comptent dans le revenu, pas les commandes en attente.
		if s.Status == "paid" || s.Status == "delivered" {
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

	if h.notifications == nil {
		health.Email = "disabled"
	} else {
		health.Email = "ok"
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	health.GoroutineCount = runtime.NumGoroutine()
	health.MemAllocMB = float64(mem.Alloc) / 1024 / 1024
	health.MemSysMB = float64(mem.Sys) / 1024 / 1024

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Analytics — GET /api/admin/analytics?days=7|30|365 (scope "finance")
func (h *AdminHandler) Analytics(w http.ResponseWriter, r *http.Request) {
	days := 30
	switch r.URL.Query().Get("days") {
	case "7":
		days = 7
	case "365":
		days = 365
	}
	salesByDay, err := h.saleRepo.SalesByDay(r.Context(), days)
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

// SendBroadcast — POST /api/admin/broadcast (scope "users"). Diffuse un
// email (HTML composé par l'admin) à tous les comptes DIARRA. TestOnly
// n'envoie qu'à l'admin connecté, pour prévisualiser avant l'envoi réel.
// L'envoi en masse part en tâche de fond, espacé pour ne pas dépasser les
// limites de débit du fournisseur email — la réponse HTTP ne reflète que
// le démarrage, pas la livraison individuelle de chaque email.
func (h *AdminHandler) SendBroadcast(w http.ResponseWriter, r *http.Request) {
	if h.notifications == nil {
		http.Error(w, `{"error":"email_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	var input struct {
		Subject  string `json:"subject"`
		HTML     string `json:"html"`
		TestOnly bool   `json:"test_only"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Subject == "" || input.HTML == "" {
		http.Error(w, `{"error":"subject_and_html_required"}`, http.StatusBadRequest)
		return
	}

	if input.TestOnly {
		userID := middleware.GetUserID(r.Context())
		admin, err := h.userRepo.FindByID(r.Context(), userID)
		if err != nil {
			http.Error(w, `{"error":"admin_not_found"}`, http.StatusInternalServerError)
			return
		}
		if err := h.notifications.SendBroadcast(r.Context(), admin.Email, input.Subject, input.HTML); err != nil {
			http.Error(w, `{"error":"send_failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "test_sent", "to": admin.Email})
		return
	}

	emails, err := h.userRepo.ListAllEmails(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	subject := input.Subject
	bodyHTML := input.HTML
	go func() {
		ctx := context.Background()
		for _, to := range emails {
			_ = h.notifications.SendBroadcast(ctx, to, subject, bodyHTML)
			time.Sleep(150 * time.Millisecond)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "started", "recipients": len(emails)})
}

// SendUserMessage — POST /api/admin/users/{id}/message (scope "users").
// Envoie un email direct à un utilisateur précis (ex: expliquer un incident
// de paiement à un vendeur), indépendamment de tout ticket support existant.
func (h *AdminHandler) SendUserMessage(w http.ResponseWriter, r *http.Request) {
	if h.notifications == nil {
		http.Error(w, `{"error":"email_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	userID := chi.URLParam(r, "id")

	var input struct {
		Subject string `json:"subject"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Subject == "" || input.Message == "" {
		http.Error(w, `{"error":"subject_and_message_required"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"user_not_found"}`, http.StatusNotFound)
		return
	}

	if err := h.notifications.SendAdminMessage(r.Context(), user.Email, input.Subject, input.Message); err != nil {
		http.Error(w, `{"error":"send_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
