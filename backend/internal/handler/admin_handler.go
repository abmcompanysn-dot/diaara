package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"runtime"
	"sort"
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
	// webhook : réutilisé pour ConfirmPaidSale (statut + emails + notifs +
	// cagnotte + cache) depuis la vérification manuelle d'une vente chez le
	// prestataire — même effet exact qu'un webhook PawaPay reçu.
	webhook *WebhookHandler
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
	webhook *WebhookHandler,
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
		webhook:       webhook,
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

	// Passage vendeur décidé par un admin : même email de bienvenue vendeur
	// (avec le lien du groupe WhatsApp du pays) que le libre-service — voir
	// AuthService.AddRole.
	if input.Action == "grant" && input.Role == model.RoleVendeur && h.notifications != nil {
		userID := id
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			h.sendVendorWelcome(ctx, userID)
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "role_updated"})
}

// sendVendorWelcome envoie l'email « Bienvenue dans l'espace vendeur » avec le
// bon lien communauté WhatsApp (celui du pays du vendeur, déduit de son
// téléphone, sinon le lien général). Best-effort : le rôle est déjà accordé.
// Miroir de AuthService.sendVendorWelcome pour le passage vendeur décidé par
// un admin.
func (h *AdminHandler) sendVendorWelcome(ctx context.Context, userID string) {
	if h.notifications == nil {
		return
	}
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil || user.Email == "" {
		return
	}
	country := ""
	if user.Phone != nil {
		country = payment.CountryFromPhone(*user.Phone)
	}
	link := ""
	if country != "" {
		link = h.settingsRepo.Get(ctx, model.WhatsAppCommunitySettingKey(country), "")
	}
	if link == "" {
		link = h.settingsRepo.Get(ctx, model.SettingWhatsAppCommunityURL, "")
	}
	_ = h.notifications.SendVendorWelcome(ctx, user.Email, link, payment.CountryLabel(country), country != "")
}

// Users — GET /api/admin/users?country=SEN (filtre pays optionnel, ISO3).
// Le pays de chaque utilisateur est déduit de l'indicatif de son téléphone
// (payment.CountryFromPhone) — le modèle User n'a pas de champ pays.
func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListAllUsersWithStats(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	countryFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	filtered := users[:0]
	for _, u := range users {
		u.Country = ""
		if u.Phone != nil {
			u.Country = payment.CountryFromPhone(*u.Phone)
		}
		u.CountryLabel = payment.CountryLabel(u.Country)
		if countryFilter != "" {
			// "UNKNOWN" = pays non déterminé (pas de téléphone ou hors zone).
			if countryFilter == "UNKNOWN" {
				if u.Country != "" {
					continue
				}
			} else if u.Country != countryFilter {
				continue
			}
		}
		filtered = append(filtered, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": filtered})
}

// UsersCountrySummary — GET /api/admin/users/by-country : répartition des
// comptes par pays (déduit de l'indicatif téléphone), triée par effectif
// décroissant. Sert d'en-tête à la vue « Utilisateurs par pays » et alimente
// le sélecteur de pays du message groupé.
func (h *AdminHandler) UsersCountrySummary(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListAllUsersWithStats(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	type countryBucket struct {
		Country      string `json:"country"` // ISO3, "" pour inconnu
		CountryLabel string `json:"country_label"`
		Users        int    `json:"users"`
		Vendors      int    `json:"vendors"`
		WithPhone    int    `json:"with_phone"`
	}
	byCode := map[string]*countryBucket{}
	for _, u := range users {
		code := ""
		if u.Phone != nil {
			code = payment.CountryFromPhone(*u.Phone)
		}
		b := byCode[code]
		if b == nil {
			b = &countryBucket{Country: code, CountryLabel: payment.CountryLabel(code)}
			byCode[code] = b
		}
		b.Users++
		if u.Phone != nil && *u.Phone != "" {
			b.WithPhone++
		}
		for _, role := range u.Roles {
			if role == model.RoleVendeur {
				b.Vendors++
				break
			}
		}
	}

	buckets := make([]*countryBucket, 0, len(byCode))
	for _, b := range byCode {
		buckets = append(buckets, b)
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Users > buckets[j].Users })

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"countries": buckets})
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
		case key == model.SettingWhatsAppCommunityURL || strings.HasPrefix(key, "whatsapp_community_url_"):
			// Lien communauté WhatsApp : vide (= effacer) ou une URL http(s).
			if v := strings.TrimSpace(value); v != "" && !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
				http.Error(w, `{"error":"invalid_whatsapp_url"}`, http.StatusBadRequest)
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

// initiatePayoutWithProvider déclenche le versement mobile money chez le
// prestataire déjà résolu sur le versement (payout.Provider), passe le statut
// à "processing" et persiste la référence prestataire. Partagé par
// SettlePayoutAuto (demande en attente) et RetryPayout (échec relancé). En cas
// de rejet prestataire, marque le versement "failed" avec la raison et renvoie
// une erreur (destinée à l'admin uniquement — le vendeur ne voit jamais ça).
func (h *AdminHandler) initiatePayoutWithProvider(w http.ResponseWriter, r *http.Request, payout *model.Payout) bool {
	if payout.Provider == "kpay" {
		if h.kpay == nil {
			http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
			return false
		}
		resp, err := h.kpay.InitiatePayout(r.Context(), payment.PayoutInitRequest{
			Amount:      fmt.Sprintf("%d", payout.AmountCFA),
			Provider:    payout.Operator,
			PhoneNumber: payout.PhoneNumber,
			ExternalId:  payout.ID,
			Description: "Versement DIARRA",
		})
		if err != nil {
			reason := "payout_init_failed"
			h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "failed", &reason)
			h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))
			http.Error(w, `{"error":"payout_rejected","reason":"payout_init_failed"}`, http.StatusBadGateway)
			return false
		}
		if err := h.payoutRepo.SetProviderReference(r.Context(), payout.ID, resp.ID); err != nil {
			http.Error(w, `{"error":"payout_update_failed"}`, http.StatusInternalServerError)
			return false
		}
		return true
	}

	if h.pawapay == nil {
		http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
		return false
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
		reason := "payout_init_failed"
		if err == nil && resp.FailureReason != nil {
			reason = resp.FailureReason.FailureCode
		}
		h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "failed", &reason)
		h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))
		http.Error(w, fmt.Sprintf(`{"error":"payout_rejected","reason":"%s"}`, reason), http.StatusBadGateway)
		return false
	}
	if err := h.payoutRepo.SetProviderReference(r.Context(), payout.ID, pawapayID); err != nil {
		http.Error(w, `{"error":"payout_update_failed"}`, http.StatusInternalServerError)
		return false
	}
	return true
}

// SettlePayoutAuto — POST /api/admin/payouts/{id}/settle-auto (scope
// "finance") : l'admin choisit de régler AUTOMATIQUEMENT une demande de
// versement restée "en attente" (requested) — déclenche le versement chez le
// prestataire (PawaPay/KPay). Le versement passe "processing", puis
// "paid"/"failed" via le webhook prestataire (ou la réconciliation de fond).
// L'alternative est le règlement manuel (SettlePayoutManual).
func (h *AdminHandler) SettlePayoutAuto(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	payout, err := h.payoutRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if payout.Status != "requested" {
		http.Error(w, `{"error":"payout_not_pending"}`, http.StatusBadRequest)
		return
	}
	if payout.Provider == "" || payout.Provider == "off" || payout.Provider == "manual" {
		http.Error(w, `{"error":"no_auto_provider","detail":"Cette demande n'a pas de prestataire automatique — réglez-la à la main."}`, http.StatusBadRequest)
		return
	}
	if !h.initiatePayoutWithProvider(w, r, payout) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "processing"})
}

// RetryPayout — POST /api/admin/payouts/{id}/retry (scope "finance") :
// relance un versement resté en échec (nouvel essai chez le même prestataire).
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
	if payout.Provider == "" || payout.Provider == "off" || payout.Provider == "manual" {
		http.Error(w, `{"error":"no_auto_provider"}`, http.StatusBadRequest)
		return
	}
	if !h.initiatePayoutWithProvider(w, r, payout) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "processing"})
}

// CheckPayoutProvider — POST /api/admin/payouts/{id}/check-provider (scope
// "finance") : interroge le prestataire (PawaPay/KPay) sur le VRAI statut d'un
// versement "processing", et applique COMPLETED→paid (+ email vendeur) /
// FAILED→failed. Équivalent de CheckSaleProvider pour les versements — sert de
// bouton « Vérifier chez PawaPay » si le webnook s'est perdu.
//
// Réponse : { provider, provider_status, payout_status }.
func (h *AdminHandler) CheckPayoutProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	payout, err := h.payoutRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if payout.ProviderReference == nil || *payout.ProviderReference == "" {
		http.Error(w, `{"error":"no_provider_reference","detail":"Ce versement n'a pas encore été envoyé à un prestataire."}`, http.StatusBadRequest)
		return
	}

	resp := map[string]string{"provider": payout.Provider, "payout_status": payout.Status}
	providerStatus := ""

	switch payout.Provider {
	case "kpay":
		if h.kpay == nil {
			http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
			return
		}
		st, err := h.kpay.GetPayoutStatus(r.Context(), *payout.ProviderReference)
		if err != nil {
			http.Error(w, `{"error":"provider_unreachable"}`, http.StatusBadGateway)
			return
		}
		providerStatus = st.Status
	default: // pawapay
		if h.pawapay == nil {
			http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
			return
		}
		st, err := h.pawapay.GetPayoutStatus(r.Context(), *payout.ProviderReference)
		if err != nil || st.Data == nil {
			http.Error(w, `{"error":"provider_unreachable"}`, http.StatusBadGateway)
			return
		}
		providerStatus = st.Data.Status
	}
	resp["provider_status"] = providerStatus

	if payout.Status != "paid" && payout.Status != "failed" {
		switch providerStatus {
		case "COMPLETED":
			if err := h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "paid", nil); err == nil {
				resp["payout_status"] = "paid"
				h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))
				if h.notifications != nil {
					userID, amount := payout.UserID, payout.AmountCFA
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
						defer cancel()
						if u, uerr := h.userRepo.FindByID(ctx, userID); uerr == nil && u.Email != "" {
							_ = h.notifications.SendPayoutConfirmed(ctx, u.Email, amount)
						}
					}()
				}
			}
		case "FAILED":
			reason := "provider_failed"
			if err := h.payoutRepo.UpdateStatus(r.Context(), payout.ID, "failed", &reason); err == nil {
				resp["payout_status"] = "failed"
				h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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

// PendingSales — GET /api/admin/sales/pending : toutes les commandes restées
// "pending" ou "failed", avec le contact acheteur (nom, email, téléphone,
// pays), le produit et le vendeur. Sert à voir qui a essayé d'acheter sans
// aboutir, et à les relancer.
func (h *AdminHandler) PendingSales(w http.ResponseWriter, r *http.Request) {
	sales, err := h.saleRepo.ListPendingAndFailed(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"sales": sales})
}

// RemindSale — POST /api/admin/sales/{id}/remind : relance par email l'acheteur
// d'une commande restée "pending" (tout produit, tout vendeur).
func (h *AdminHandler) RemindSale(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	view, err := h.saleRepo.FindPendingView(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	// Depuis l'admin, on n'impose pas le cooldown : c'est une action explicite,
	// pas une boucle automatique.
	if err := sendPendingReminder(r.Context(), h.saleRepo, h.notifications, view, false); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// MarkSalePaid — POST /api/admin/sales/{id}/mark-paid : confirme À LA MAIN un
// paiement (l'acheteur a payé mais le webhook n'est jamais arrivé). Passe la
// vente pending/failed → paid, trace l'admin, puis déclenche la livraison :
// email de confirmation à l'acheteur (avec le fichier) + email au vendeur,
// exactement comme le webhook PawaPay le ferait.
func (h *AdminHandler) MarkSalePaid(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := middleware.GetUserID(r.Context())

	sale, err := h.saleRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if sale.Status != string(model.SalePending) && sale.Status != string(model.SaleFailed) {
		http.Error(w, `{"error":"sale_not_confirmable"}`, http.StatusBadRequest)
		return
	}

	ok, err := h.saleRepo.MarkManuallyConfirmed(r.Context(), id, adminID)
	if err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	if !ok {
		// Course : la vente a changé d'état entre-temps.
		http.Error(w, `{"error":"sale_not_confirmable"}`, http.StatusConflict)
		return
	}

	// Livraison + emails, en tâche de fond (best-effort, ne bloque pas la
	// réponse — même principe que WebhookHandler.notifyPaid).
	sale.Status = string(model.SalePaid)
	go h.deliverManuallyConfirmed(context.Background(), sale)

	if product, err := h.productRepo.FindByID(r.Context(), sale.ProductID); err == nil {
		h.cache.Del(r.Context(), vendorBalanceCacheKey(product.VendorID))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "paid"})
}

// CheckSaleProvider — POST /api/admin/sales/{id}/check-provider : interroge en
// direct le prestataire (PawaPay pour l'instant) sur le VRAI statut du dépôt
// d'une vente restée "pending", et — si le prestataire répond COMPLETED —
// confirme la vente exactement comme le webhook l'aurait fait (email acheteur
// avec fichier, email + notif vendeur, crédit du solde). Sert de bouton
// « Vérifier chez PawaPay » : contrairement à MarkSalePaid, ne force rien à
// l'aveugle, on ne confirme que sur réponse positive du prestataire.
//
// Réponse : { provider, provider_status, provider_transaction_id, sale_status }.
func (h *AdminHandler) CheckSaleProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	sale, err := h.saleRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if sale.PaymentProvider == "kpay" {
		http.Error(w, `{"error":"provider_check_unsupported","provider":"kpay"}`, http.StatusBadRequest)
		return
	}
	if h.pawapay == nil {
		http.Error(w, `{"error":"payment_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	status, err := h.pawapay.GetDepositStatus(r.Context(), sale.PaymentReference)
	if err != nil {
		http.Error(w, `{"error":"provider_unreachable"}`, http.StatusBadGateway)
		return
	}

	resp := map[string]string{
		"provider":    "pawapay",
		"sale_status": sale.Status,
	}
	if status.Data == nil {
		// NOT_FOUND côté PawaPay : l'acheteur n'a jamais confirmé le paiement
		// sur la page hébergée (panier abandonné).
		resp["provider_status"] = "NOT_FOUND"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	resp["provider_status"] = status.Data.Status
	resp["provider_transaction_id"] = status.Data.ProviderTransactionId

	// On persiste l'ID de transaction du prestataire dès qu'on le connaît
	// (traçabilité), même si le paiement n'est pas encore COMPLETED.
	if status.Data.ProviderTransactionId != "" && (sale.ProviderTransactionID == nil || *sale.ProviderTransactionID == "") {
		_ = h.saleRepo.SetProviderTransactionID(r.Context(), sale.ID, status.Data.ProviderTransactionId)
	}

	switch status.Data.Status {
	case "COMPLETED":
		if sale.Status == string(model.SalePending) && h.webhook != nil {
			if err := h.webhook.ConfirmPaidSale(r.Context(), sale); err != nil {
				http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
				return
			}
			resp["sale_status"] = string(model.SalePaid)
		}
	case "FAILED":
		if sale.Status == string(model.SalePending) {
			if err := h.saleRepo.UpdateStatus(r.Context(), sale.ID, string(model.SaleFailed)); err == nil {
				resp["sale_status"] = string(model.SaleFailed)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// deliverManuallyConfirmed reproduit les effets post-paiement du webhook pour
// une vente confirmée manuellement : email acheteur (fichier joint si possible)
// + email vendeur.
func (h *AdminHandler) deliverManuallyConfirmed(ctx context.Context, sale *model.Sale) {
	if h.notifications == nil {
		return
	}
	product, err := h.productRepo.FindByID(ctx, sale.ProductID)
	if err != nil {
		return
	}

	buyer, err := h.userRepo.FindByID(ctx, sale.BuyerID)
	if err == nil && buyer.Email != "" && sale.CheckoutToken != nil {
		var attachment *email.Attachment
		if h.files != nil && product.FileKey != "" {
			if content, derr := h.files.Download(ctx, product.FileKey); derr == nil {
				ctype := mime.TypeByExtension(filepath.Ext(product.FileKey))
				if ctype == "" {
					ctype = "application/octet-stream"
				}
				attachment = &email.Attachment{
					Filename:    filepath.Base(product.FileKey),
					Content:     content,
					ContentType: ctype,
				}
			}
		}
		_ = h.notifications.SendOrderConfirmed(ctx, buyer.Email, sale.BuyerName, product.Title, sale.AmountCFA, *sale.CheckoutToken, attachment)
	}

	if vendor, verr := h.userRepo.FindByID(ctx, product.VendorID); verr == nil && vendor.Email != "" {
		_ = h.notifications.SendVendorSale(ctx, vendor.Email, product.Title, sale.VendorAmountCFA)
	}
}

// SettlePayoutManual — POST /api/admin/payouts/{id}/settle-manual : marque un
// versement "paid" SANS appel PawaPay/KPay — l'argent a été envoyé au vendeur
// autrement (Wave perso, espèces, virement). Corps : { note, fee_cfa }.
func (h *AdminHandler) SettlePayoutManual(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	adminID := middleware.GetUserID(r.Context())

	var input model.SettlePayoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.FeeCFA < 0 {
		http.Error(w, `{"error":"invalid_fee"}`, http.StatusBadRequest)
		return
	}

	payout, err := h.payoutRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if payout.Status == "paid" {
		http.Error(w, `{"error":"payout_already_paid"}`, http.StatusBadRequest)
		return
	}

	ok, err := h.payoutRepo.SettleManually(r.Context(), id, input.Note, input.FeeCFA, adminID)
	if err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, `{"error":"payout_not_settleable"}`, http.StatusConflict)
		return
	}
	h.cache.Del(r.Context(), vendorBalanceCacheKey(payout.UserID))

	if h.notifications != nil {
		userID, amount := payout.UserID, payout.AmountCFA
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if u, uerr := h.userRepo.FindByID(ctx, userID); uerr == nil && u.Email != "" {
				_ = h.notifications.SendPayoutConfirmed(ctx, u.Email, amount)
			}
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "paid"})
}

// CreateManualPayout — POST /api/admin/payouts/manual : enregistre un versement
// déjà effectué à la main pour un vendeur (aucune demande côté vendeur requise).
// Corps : { user_id, amount, fee_cfa, phone, note }.
func (h *AdminHandler) CreateManualPayout(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetUserID(r.Context())

	var input model.ManualPayoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.UserID == "" {
		http.Error(w, `{"error":"user_id_required"}`, http.StatusBadRequest)
		return
	}
	if input.AmountCFA <= 0 || input.FeeCFA < 0 {
		http.Error(w, `{"error":"invalid_amount"}`, http.StatusBadRequest)
		return
	}
	if _, err := h.userRepo.FindByID(r.Context(), input.UserID); err != nil {
		http.Error(w, `{"error":"user_not_found"}`, http.StatusNotFound)
		return
	}

	// À défaut de numéro fourni, on reprend celui enregistré par le vendeur
	// (moyen de versement) — pour que la traçabilité comptable montre bien où
	// l'argent a été envoyé.
	phone := input.Phone
	if phone == "" {
		if regPhone, _, _, perr := h.userRepo.GetPayoutMethod(r.Context(), input.UserID); perr == nil && regPhone != nil {
			phone = *regPhone
		}
	}

	payout, err := h.payoutRepo.CreateManual(r.Context(), input.UserID, input.AmountCFA, input.FeeCFA, phone, input.Note, adminID)
	if err != nil {
		http.Error(w, `{"error":"payout_creation_failed"}`, http.StatusInternalServerError)
		return
	}
	h.cache.Del(r.Context(), vendorBalanceCacheKey(input.UserID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"payout": payout})
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
// email (HTML composé par l'admin) aux comptes DIARRA. TestOnly n'envoie qu'à
// l'admin connecté, pour prévisualiser avant l'envoi réel. Country (ISO3
// optionnel, ou "UNKNOWN") restreint la diffusion aux comptes dont le pays,
// déduit de l'indicatif téléphone, correspond. Un bloc « Rejoindre la
// communauté WhatsApp » est ajouté en pied si un lien est configuré (général
// ou celui du pays ciblé). L'envoi en masse part en tâche de fond, espacé
// pour ne pas dépasser les limites du fournisseur email — la réponse HTTP ne
// reflète que le démarrage.
func (h *AdminHandler) SendBroadcast(w http.ResponseWriter, r *http.Request) {
	if h.notifications == nil {
		http.Error(w, `{"error":"email_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	var input struct {
		Subject  string `json:"subject"`
		HTML     string `json:"html"`
		TestOnly bool   `json:"test_only"`
		Country  string `json:"country,omitempty"` // ISO3, "UNKNOWN", ou vide = tous
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Subject == "" || input.HTML == "" {
		http.Error(w, `{"error":"subject_and_html_required"}`, http.StatusBadRequest)
		return
	}
	countryFilter := strings.ToUpper(strings.TrimSpace(input.Country))

	if input.TestOnly {
		userID := middleware.GetUserID(r.Context())
		admin, err := h.userRepo.FindByID(r.Context(), userID)
		if err != nil {
			http.Error(w, `{"error":"admin_not_found"}`, http.StatusInternalServerError)
			return
		}
		waCountry := countryFilter
		if waCountry == "" || waCountry == "UNKNOWN" {
			waCountry = ""
			if admin.Phone != nil {
				waCountry = payment.CountryFromPhone(*admin.Phone)
			}
		}
		html := input.HTML + h.whatsappCommunityBlock(r.Context(), waCountry)
		if err := h.notifications.SendBroadcast(r.Context(), admin.Email, input.Subject, html); err != nil {
			http.Error(w, `{"error":"send_failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "test_sent", "to": admin.Email})
		return
	}

	recipients, err := h.userRepo.ListEmailsWithPhone(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	// Pré-calcul du bloc WhatsApp : si un pays précis est ciblé, le bloc est
	// identique pour tous → une seule lecture des réglages. Sinon on le
	// calcule par destinataire (pays déduit de son téléphone).
	subject, bodyHTML := input.Subject, input.HTML
	sharedBlock := ""
	perRecipientBlock := true
	if countryFilter != "" && countryFilter != "UNKNOWN" {
		sharedBlock = h.whatsappCommunityBlock(r.Context(), countryFilter)
		perRecipientBlock = false
	}
	generalBlock := h.whatsappCommunityBlock(r.Context(), "")

	type target struct{ email, html string }
	targets := make([]target, 0, len(recipients))
	for _, rcp := range recipients {
		code := ""
		if rcp.Phone != nil {
			code = payment.CountryFromPhone(*rcp.Phone)
		}
		if countryFilter == "UNKNOWN" {
			if code != "" {
				continue
			}
		} else if countryFilter != "" && code != countryFilter {
			continue
		}
		block := sharedBlock
		if perRecipientBlock {
			if code != "" {
				block = h.whatsappCommunityBlock(r.Context(), code)
			} else {
				block = generalBlock
			}
		}
		targets = append(targets, target{email: rcp.Email, html: bodyHTML + block})
	}

	go func() {
		ctx := context.Background()
		for _, t := range targets {
			_ = h.notifications.SendBroadcast(ctx, t.email, subject, t.html)
			time.Sleep(150 * time.Millisecond)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "started", "recipients": len(targets)})
}

// whatsappCommunityBlock renvoie un fragment HTML « Rejoindre la communauté
// WhatsApp » à concaténer au corps d'un email, ou "" si aucun lien n'est
// configuré. countryISO3 vide = lien général ; sinon on prend le lien du pays
// s'il existe, à défaut le lien général.
func (h *AdminHandler) whatsappCommunityBlock(ctx context.Context, countryISO3 string) string {
	link := ""
	if countryISO3 != "" {
		link = h.settingsRepo.Get(ctx, model.WhatsAppCommunitySettingKey(countryISO3), "")
	}
	if link == "" {
		link = h.settingsRepo.Get(ctx, model.SettingWhatsAppCommunityURL, "")
	}
	if link == "" {
		return ""
	}
	return email.WhatsAppCommunityHTML(link, payment.CountryLabel(countryISO3), countryISO3 != "")
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

	// Bloc communauté WhatsApp en pied : lien du pays du destinataire (déduit
	// de son téléphone) si configuré, sinon lien général.
	country := ""
	if user.Phone != nil {
		country = payment.CountryFromPhone(*user.Phone)
	}
	link := ""
	if country != "" {
		link = h.settingsRepo.Get(r.Context(), model.WhatsAppCommunitySettingKey(country), "")
	}
	if link == "" {
		link = h.settingsRepo.Get(r.Context(), model.SettingWhatsAppCommunityURL, "")
	}

	if err := h.notifications.SendAdminMessageWithCommunity(r.Context(), user.Email, input.Subject, input.Message, link, payment.CountryLabel(country), country != ""); err != nil {
		http.Error(w, `{"error":"send_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
