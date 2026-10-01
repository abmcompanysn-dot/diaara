package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/diarra/backend/internal/cache"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

// AdHandler — sponsorisation de produits sur Meta (Facebook + Instagram)
// depuis l'espace vendeur (demande du 2026-10-01). Le vendeur paie avec son
// solde de gains ; DIARRA garde une commission et lance la campagne sur son
// propre compte publicitaire Meta (payment/meta_ads.go). Si Meta refuse la
// pub ou si la création échoue, la somme est rendue au solde du vendeur.
type AdHandler struct {
	repo          *repository.AdCampaignRepo
	productRepo   *repository.ProductRepo
	settingsRepo  *repository.SettingsRepo
	notifications *repository.NotificationRepo
	cache         *cache.Client
	meta          *payment.MetaAdsClient // nil tant que META_ADS_* n'est pas configuré
	frontendURL   string
	apiURL        string
}

func NewAdHandler(repo *repository.AdCampaignRepo, productRepo *repository.ProductRepo, settingsRepo *repository.SettingsRepo,
	notifications *repository.NotificationRepo, cacheClient *cache.Client, meta *payment.MetaAdsClient, frontendURL, apiURL string) *AdHandler {
	return &AdHandler{
		repo: repo, productRepo: productRepo, settingsRepo: settingsRepo, notifications: notifications,
		cache: cacheClient, meta: meta,
		frontendURL: strings.TrimSuffix(frontendURL, "/"), apiURL: strings.TrimSuffix(apiURL, "/"),
	}
}

func adJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *AdHandler) enabled(ctx context.Context) bool {
	return h.meta != nil && h.settingsRepo.GetBool(ctx, model.SettingAdsEnabled, true)
}

func (h *AdHandler) commissionPct(ctx context.Context) float64 {
	pct := h.settingsRepo.GetFloat(ctx, model.SettingAdsCommissionPct, model.DefaultAdsCommissionPct)
	if pct < 0 || pct >= 100 {
		return model.DefaultAdsCommissionPct
	}
	return pct
}

func (h *AdHandler) minDailyCFA(ctx context.Context) int {
	v := int(h.settingsRepo.GetFloat(ctx, model.SettingAdsMinDailyCFA, model.DefaultAdsMinDailyCFA))
	if v <= 0 {
		return model.DefaultAdsMinDailyCFA
	}
	return v
}

// splitAmount — part DIARRA (arrondie au FCFA supérieur) et budget pub.
func splitAmount(amount int, commissionPct float64) (commission, adBudget int) {
	commission = int(math.Ceil(float64(amount) * commissionPct / 100))
	return commission, amount - commission
}

// minAmountFor — montant minimum à payer pour durationDays jours, de sorte
// que le budget pub (après commission) atteigne minDaily par jour.
func minAmountFor(durationDays, minDaily int, commissionPct float64) int {
	return int(math.Ceil(float64(durationDays*minDaily) / (1 - commissionPct/100)))
}

// Config — GET /api/vendor/ads/config : de quoi construire le formulaire
// (disponibilité, commission, minimum, pays, solde disponible).
func (h *AdHandler) Config(w http.ResponseWriter, r *http.Request) {
	vendorID := middleware.GetUserID(r.Context())
	available, err := h.repo.AvailableBalance(r.Context(), vendorID)
	if err != nil {
		http.Error(w, `{"error":"balance_failed"}`, http.StatusInternalServerError)
		return
	}
	if available < 0 {
		available = 0
	}
	countries := make([]map[string]string, 0, len(model.AdCountries))
	for code, name := range model.AdCountries {
		countries = append(countries, map[string]string{"code": code, "name": name})
	}
	sort.Slice(countries, func(i, j int) bool { return countries[i]["name"] < countries[j]["name"] })
	adJSON(w, http.StatusOK, map[string]interface{}{
		"enabled":           h.enabled(r.Context()),
		"platforms":         []string{"meta"},
		"commission_pct":    h.commissionPct(r.Context()),
		"min_daily_cfa":     h.minDailyCFA(r.Context()),
		"max_duration_days": model.AdsMaxDurationDays,
		"max_amount_cfa":    model.AdsMaxAmountCFA,
		"available_cfa":     available,
		"countries":         countries,
	})
}

// ListVendor — GET /api/vendor/ads
func (h *AdHandler) ListVendor(w http.ResponseWriter, r *http.Request) {
	campaigns, err := h.repo.ListByVendor(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	adJSON(w, http.StatusOK, map[string]interface{}{"campaigns": campaigns})
}

// Create — POST /api/vendor/ads : valide, débite le solde (transaction
// verrouillée), puis lance la campagne chez Meta. Échec Meta = remboursement
// immédiat du solde et message d'erreur au vendeur.
func (h *AdHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vendorID := middleware.GetUserID(ctx)
	if !h.enabled(ctx) {
		http.Error(w, `{"error":"ads_unavailable"}`, http.StatusServiceUnavailable)
		return
	}

	var input model.CreateAdCampaignInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	product, err := h.productRepo.FindByID(ctx, input.ProductID)
	if err != nil || product.VendorID != vendorID {
		http.Error(w, `{"error":"product_not_found"}`, http.StatusNotFound)
		return
	}
	if product.ModerationStatus != "approved" {
		http.Error(w, `{"error":"product_not_approved"}`, http.StatusBadRequest)
		return
	}

	if input.DurationDays < 1 || input.DurationDays > model.AdsMaxDurationDays {
		http.Error(w, `{"error":"invalid_duration"}`, http.StatusBadRequest)
		return
	}
	countries := []string{}
	seen := map[string]bool{}
	for _, c := range input.Countries {
		c = strings.ToUpper(strings.TrimSpace(c))
		if _, ok := model.AdCountries[c]; ok && !seen[c] {
			seen[c] = true
			countries = append(countries, c)
		}
	}
	if len(countries) == 0 {
		http.Error(w, `{"error":"countries_required"}`, http.StatusBadRequest)
		return
	}
	message := strings.TrimSpace(input.Message)
	if message == "" {
		message = product.Title
	}
	if utf8.RuneCountInString(message) > model.AdsMessageMaxLen {
		http.Error(w, `{"error":"message_too_long"}`, http.StatusBadRequest)
		return
	}

	pct := h.commissionPct(ctx)
	minAmount := minAmountFor(input.DurationDays, h.minDailyCFA(ctx), pct)
	if input.AmountCFA < minAmount {
		adJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "ad_amount_below_minimum", "min_amount_cfa": minAmount})
		return
	}
	if input.AmountCFA > model.AdsMaxAmountCFA {
		http.Error(w, `{"error":"ad_amount_above_maximum"}`, http.StatusBadRequest)
		return
	}
	commission, adBudget := splitAmount(input.AmountCFA, pct)

	campaign, err := h.repo.CreateFromBalance(ctx, &model.AdCampaign{
		VendorID: vendorID, ProductID: product.ID, Platform: "meta",
		AmountCFA: input.AmountCFA, CommissionCFA: commission, AdBudgetCFA: adBudget,
		DurationDays: input.DurationDays, Countries: countries, Message: message,
	})
	if errors.Is(err, repository.ErrInsufficientBalance) {
		http.Error(w, `{"error":"insufficient_balance"}`, http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("ads: création campagne vendeur=%s: %v", vendorID, err)
		http.Error(w, `{"error":"ad_creation_failed"}`, http.StatusInternalServerError)
		return
	}
	h.cache.Del(ctx, vendorBalanceCacheKey(vendorID))

	// Lancement chez Meta : contexte détaché de la requête (un vendeur qui
	// ferme la page ne doit pas interrompre la création à mi-chemin).
	launchCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := h.launch(launchCtx, campaign, product); err != nil {
		reason := metaErrorReason(err)
		log.Printf("ads: lancement Meta campagne=%s: %v", campaign.ID, err)
		h.refund(launchCtx, campaign, model.AdStatusFailed, reason)
		adJSON(w, http.StatusBadGateway, map[string]interface{}{"error": "ad_launch_failed", "details": reason})
		return
	}

	updated, err := h.repo.FindByID(ctx, campaign.ID)
	if err != nil {
		updated = campaign
	}
	adJSON(w, http.StatusCreated, map[string]interface{}{"campaign": updated})
}

// launch crée la campagne chez Meta et enregistre ses identifiants.
func (h *AdHandler) launch(ctx context.Context, c *model.AdCampaign, product *model.Product) error {
	// Début dans 10 minutes (Meta refuse un start_time dans le passé), fin
	// après la durée choisie.
	start := time.Now().Add(10 * time.Minute)
	end := start.Add(time.Duration(c.DurationDays) * 24 * time.Hour)

	image := ""
	if product.CoverImageKey != nil && *product.CoverImageKey != "" && h.apiURL != "" {
		image = h.apiURL + "/api/products/" + product.ID + "/cover"
	}
	res, err := h.meta.CreateSponsoredAd(ctx, payment.SponsoredAdRequest{
		Name:      fmt.Sprintf("DIARRA · %s · %s", truncateRunes(product.Title, 60), c.ID[:8]),
		LinkURL:   fmt.Sprintf("%s/product?id=%s&utm_source=meta&utm_medium=paid&utm_campaign=%s", h.frontendURL, product.ID, c.ID),
		Headline:  truncateRunes(product.Title, 40),
		Message:   c.Message,
		ImageURL:  image,
		Countries: c.Countries,
		BudgetXOF: c.AdBudgetCFA,
		StartTime: start,
		EndTime:   end,
	})
	if err != nil {
		return err
	}
	if err := h.repo.MarkLaunched(ctx, c.ID, res.CampaignID, res.AdSetID, res.CreativeID, res.AdID, start, end); err != nil {
		// La campagne tourne chez Meta mais DIARRA n'a pas pu l'enregistrer :
		// l'appelant va rembourser le vendeur, donc on coupe la diffusion
		// d'abord — jamais une pub qui dépense sur un montant remboursé.
		if pauseErr := h.meta.PauseCampaign(ctx, res.CampaignID); pauseErr != nil {
			log.Printf("ads: ALERTE campagne Meta %s active mais non enregistrée (campagne DIARRA %s) et pause impossible: %v",
				res.CampaignID, c.ID, pauseErr)
		}
		return err
	}
	return nil
}

// refund — échec définitif : la somme revient au solde (une seule fois,
// MarkRefunded est idempotent) et le vendeur est prévenu.
func (h *AdHandler) refund(ctx context.Context, c *model.AdCampaign, status, reason string) {
	done, err := h.repo.MarkRefunded(ctx, c.ID, status, reason)
	if err != nil {
		log.Printf("ads: remboursement campagne=%s: %v", c.ID, err)
		return
	}
	if !done {
		return
	}
	h.cache.Del(ctx, vendorBalanceCacheKey(c.VendorID))
	title := "Sponsorisation refusée, montant remboursé"
	if status == model.AdStatusFailed {
		title = "Sponsorisation non lancée, montant remboursé"
	}
	body := fmt.Sprintf("%d FCFA ont été rendus à votre solde.", c.AmountCFA)
	if reason != "" {
		body += " Raison : " + truncateRunes(reason, 200)
	}
	h.notify(ctx, c.VendorID, "ad_refunded", title, body)
}

func (h *AdHandler) notify(ctx context.Context, userID, notifType, title, body string) {
	if h.notifications != nil {
		_ = h.notifications.Create(ctx, userID, notifType, title, body, "/vendor/ads")
	}
}

// metaErrorReason — message affichable : celui que Meta destine à
// l'utilisateur s'il existe, sinon un message générique (jamais un détail
// technique interne).
func metaErrorReason(err error) string {
	var metaErr *payment.MetaAPIError
	if errors.As(err, &metaErr) {
		if metaErr.UserMessage != "" {
			return metaErr.UserMessage
		}
		if metaErr.StatusCode >= 500 || metaErr.StatusCode == http.StatusBadGateway {
			return "Meta est momentanément indisponible."
		}
	}
	return "La publicité n'a pas pu être créée chez Meta."
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

// --- Synchronisation périodique ---------------------------------------------

// RunSyncLoop — toutes les 15 minutes : statut Meta (vérification, refus,
// diffusion, fin), statistiques, et remboursement des campagnes restées
// bloquées en création (serveur redémarré au mauvais moment).
func (h *AdHandler) RunSyncLoop(ctx context.Context) {
	if h.meta == nil {
		log.Println("ads: synchronisation désactivée (Meta non configuré)")
		return
	}
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		h.syncPass(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (h *AdHandler) syncPass(ctx context.Context) {
	stuck, err := h.repo.ListStuckLaunching(ctx, 10*time.Minute)
	if err == nil {
		for _, c := range stuck {
			h.refund(ctx, c, model.AdStatusFailed, "La création chez Meta a été interrompue.")
		}
	}

	campaigns, err := h.repo.ListToSync(ctx)
	if err != nil {
		log.Printf("ads: lecture des campagnes à synchroniser: %v", err)
		return
	}
	for _, c := range campaigns {
		callCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		h.syncOne(callCtx, c)
		cancel()
	}
}

func (h *AdHandler) syncOne(ctx context.Context, c *model.AdCampaign) {
	if c.ExternalAdID == nil || c.ExternalCampaignID == nil {
		return
	}
	if c.Status == model.AdStatusInReview || c.Status == model.AdStatusActive {
		st, err := h.meta.GetAdStatus(ctx, *c.ExternalAdID)
		if err != nil {
			log.Printf("ads: statut Meta campagne=%s: %v", c.ID, err)
			return
		}
		switch st.EffectiveStatus {
		case "DISAPPROVED":
			// Une pub refusée n'a rien dépensé : remboursement intégral.
			h.refund(ctx, c, model.AdStatusRejected, st.ReviewFeedback)
			return
		case "ACTIVE":
			if c.Status == model.AdStatusInReview {
				_ = h.repo.SetStatus(ctx, c.ID, model.AdStatusActive, "")
				h.notify(ctx, c.VendorID, "ad_active", "Votre pub est en ligne",
					fmt.Sprintf("« %s » est diffusé sur Facebook et Instagram.", c.ProductTitle))
				c.Status = model.AdStatusActive
			}
		case "WITH_ISSUES":
			_ = h.repo.SetStatus(ctx, c.ID, c.Status, st.ReviewFeedback)
		}
		if c.Status == model.AdStatusActive && c.EndsAt != nil && time.Now().After(*c.EndsAt) {
			_ = h.repo.SetStatus(ctx, c.ID, model.AdStatusCompleted, "")
		}
	}
	if c.Status == model.AdStatusInReview {
		return // pas encore de statistiques
	}
	ins, err := h.meta.GetCampaignInsights(ctx, *c.ExternalCampaignID)
	if err != nil {
		log.Printf("ads: statistiques Meta campagne=%s: %v", c.ID, err)
		return
	}
	_ = h.repo.UpdateStats(ctx, c.ID, ins.Impressions, ins.Reach, ins.Clicks, ins.SpendXOF)
}

// --- Admin ---------------------------------------------------------------------

// ListAdmin — GET /api/admin/ads : toutes les campagnes + état de la config.
func (h *AdHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	campaigns, err := h.repo.ListAll(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	adJSON(w, http.StatusOK, map[string]interface{}{
		"campaigns":       campaigns,
		"meta_configured": h.meta != nil,
		"enabled":         h.enabled(r.Context()),
		"commission_pct":  h.commissionPct(r.Context()),
		"min_daily_cfa":   h.minDailyCFA(r.Context()),
	})
}

// Stop — POST /api/admin/ads/{id}/stop : met la campagne en pause chez Meta
// (diffusion arrêtée). Pas de remboursement automatique : le budget déjà
// dépensé ne revient pas, l'admin décide au cas par cas.
func (h *AdHandler) Stop(w http.ResponseWriter, r *http.Request) {
	c, err := h.repo.FindByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if c.Status != model.AdStatusActive && c.Status != model.AdStatusInReview {
		http.Error(w, `{"error":"ad_not_running"}`, http.StatusConflict)
		return
	}
	if h.meta == nil || c.ExternalCampaignID == nil {
		http.Error(w, `{"error":"ads_unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	if err := h.meta.PauseCampaign(r.Context(), *c.ExternalCampaignID); err != nil {
		log.Printf("ads: arrêt campagne=%s: %v", c.ID, err)
		adJSON(w, http.StatusBadGateway, map[string]interface{}{"error": "ad_stop_failed", "details": metaErrorReason(err)})
		return
	}
	_ = h.repo.SetStatus(r.Context(), c.ID, model.AdStatusStopped, "Arrêtée par un administrateur DIARRA.")
	h.notify(r.Context(), c.VendorID, "ad_stopped", "Votre pub a été arrêtée",
		fmt.Sprintf("La sponsorisation de « %s » a été arrêtée par DIARRA.", c.ProductTitle))
	updated, _ := h.repo.FindByID(r.Context(), c.ID)
	adJSON(w, http.StatusOK, map[string]interface{}{"campaign": updated})
}
