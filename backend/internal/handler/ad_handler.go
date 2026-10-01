package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/secretbox"
	"github.com/go-chi/chi/v5"
)

// AdHandler — sponsorisation de produits sur Meta (Facebook + Instagram)
// depuis l'espace vendeur (demande du 2026-10-01, modèle revu le même jour).
//
// Le vendeur connecte SON compte Facebook (Facebook Login, voir
// ad_meta_connect.go), choisit SA page et SON compte publicitaire, puis
// DIARRA crée la pub avec son jeton (payment/meta_ads.go). C'est le vendeur
// qui paie Meta, sur le moyen de paiement de son compte publicitaire :
// aucun argent ne transite par DIARRA (pas de solde débité, pas de
// commission, pas de remboursement).
type AdHandler struct {
	repo          *repository.AdCampaignRepo
	conns         *repository.MetaConnectionRepo
	productRepo   *repository.ProductRepo
	settingsRepo  *repository.SettingsRepo
	notifications *repository.NotificationRepo
	meta          *payment.MetaApp // nil tant que META_APP_* n'est pas configuré
	box           *secretbox.Box   // nil tant que META_TOKEN_ENCRYPTION_KEY n'est pas valide
	frontendURL   string
	apiURL        string
	secureCookie  bool
}

func NewAdHandler(repo *repository.AdCampaignRepo, conns *repository.MetaConnectionRepo, productRepo *repository.ProductRepo,
	settingsRepo *repository.SettingsRepo, notifications *repository.NotificationRepo, meta *payment.MetaApp, box *secretbox.Box,
	frontendURL, apiURL string, secureCookie bool) *AdHandler {
	return &AdHandler{
		repo: repo, conns: conns, productRepo: productRepo, settingsRepo: settingsRepo, notifications: notifications,
		meta: meta, box: box, secureCookie: secureCookie,
		frontendURL: strings.TrimSuffix(frontendURL, "/"), apiURL: strings.TrimSuffix(apiURL, "/"),
	}
}

func adJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// configured — app Meta + clé de chiffrement des jetons présentes.
func (h *AdHandler) configured() bool {
	return h.meta != nil && h.box != nil
}

// enabled — configuré ET interrupteur admin ads_enabled actif.
func (h *AdHandler) enabled(ctx context.Context) bool {
	return h.configured() && h.settingsRepo.GetBool(ctx, model.SettingAdsEnabled, true)
}

// clientFor — client Meta du vendeur pour un compte pub / une page / une
// devise donnés (ceux de la connexion, ou ceux d'une campagne existante).
// Jeton indéchiffrable (clé META_TOKEN_ENCRYPTION_KEY changée) : la
// connexion passe "à reconnecter".
func (h *AdHandler) clientFor(ctx context.Context, conn *model.VendorMetaConnection, adAccountID, pageID, currency string) (*payment.MetaAdsClient, error) {
	token, err := h.box.Decrypt(conn.AccessTokenEnc)
	if err != nil {
		log.Printf("ads: jeton Meta indéchiffrable vendeur=%s: %v", conn.VendorID, err)
		h.markReconnect(ctx, conn.VendorID)
		return nil, errDecrypt
	}
	return h.meta.Client(payment.MetaAccount{AccessToken: token, AdAccountID: adAccountID, PageID: pageID, Currency: currency}), nil
}

// errDecrypt — jeton stocké indéchiffrable : le vendeur doit reconnecter.
var errDecrypt = errors.New("jeton indéchiffrable")

// handleTokenError — jeton expiré/révoqué (erreur Meta 190) : la connexion
// passe "à reconnecter" et le vendeur est prévenu UNE fois. true si err est
// bien une erreur de jeton.
func (h *AdHandler) handleTokenError(ctx context.Context, vendorID string, err error) bool {
	if errors.Is(err, errDecrypt) {
		return true // déjà marqué par clientFor
	}
	if !payment.IsMetaTokenError(err) {
		return false
	}
	h.markReconnect(ctx, vendorID)
	return true
}

func (h *AdHandler) markReconnect(ctx context.Context, vendorID string) {
	first, err := h.conns.MarkNeedsReconnect(ctx, vendorID)
	if err != nil {
		log.Printf("ads: marquage reconnexion vendeur=%s: %v", vendorID, err)
		return
	}
	if first {
		h.notify(ctx, vendorID, "meta_reconnect", "Reconnectez votre compte Facebook",
			"La connexion à votre compte Facebook a expiré ou a été retirée. Reconnectez-le pour suivre et lancer vos pubs.")
	}
}

// Config — GET /api/vendor/ads/config : de quoi construire le formulaire.
func (h *AdHandler) Config(w http.ResponseWriter, r *http.Request) {
	countries := make([]map[string]string, 0, len(model.AdCountries))
	for code, name := range model.AdCountries {
		countries = append(countries, map[string]string{"code": code, "name": name})
	}
	sort.Slice(countries, func(i, j int) bool { return countries[i]["name"] < countries[j]["name"] })
	adJSON(w, http.StatusOK, map[string]interface{}{
		"enabled":              h.enabled(r.Context()),
		"platforms":            []string{"meta"},
		"min_daily_cfa":        model.AdsMinDailyCFA,
		"max_duration_days":    model.AdsMaxDurationDays,
		"max_budget_cfa":       model.AdsMaxBudgetCFA,
		"supported_currencies": payment.MetaSupportedCurrencies,
		"countries":            countries,
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

// Create — POST /api/vendor/ads : valide, puis lance la campagne sur le
// compte publicitaire du vendeur. Échec Meta = campagne "failed" + message
// d'erreur au vendeur (rien n'est diffusé, donc rien n'est facturé par Meta).
func (h *AdHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vendorID := middleware.GetUserID(ctx)
	if !h.enabled(ctx) {
		http.Error(w, `{"error":"ads_unavailable"}`, http.StatusServiceUnavailable)
		return
	}

	conn, err := h.conns.Get(ctx, vendorID)
	if errors.Is(err, repository.ErrMetaConnectionNotFound) {
		http.Error(w, `{"error":"meta_not_connected"}`, http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"ad_creation_failed"}`, http.StatusInternalServerError)
		return
	}
	if conn.NeedsReconnect {
		http.Error(w, `{"error":"meta_reconnect_required"}`, http.StatusConflict)
		return
	}
	if !conn.Ready() {
		http.Error(w, `{"error":"meta_selection_required"}`, http.StatusConflict)
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
	countries := normalizeAdCountries(input.Countries)
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

	if minBudget := minAdBudget(input.DurationDays); input.BudgetCFA < minBudget {
		adJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "ad_budget_below_minimum", "min_budget_cfa": minBudget})
		return
	}
	if input.BudgetCFA > model.AdsMaxBudgetCFA {
		http.Error(w, `{"error":"ad_budget_above_maximum"}`, http.StatusBadRequest)
		return
	}
	// Devise du compte pub : vérifiée AVANT tout appel Meta et toute écriture.
	if _, err := payment.XOFToMetaMinorUnits(input.BudgetCFA, *conn.Currency); err != nil {
		adJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "meta_currency_unsupported", "currency": *conn.Currency})
		return
	}

	client, err := h.clientFor(ctx, conn, *conn.AdAccountID, *conn.PageID, *conn.Currency)
	if err != nil {
		http.Error(w, `{"error":"meta_reconnect_required"}`, http.StatusConflict)
		return
	}

	campaign, err := h.repo.Create(ctx, &model.AdCampaign{
		VendorID: vendorID, ProductID: product.ID, Platform: "meta",
		BudgetCFA: input.BudgetCFA, AdAccountID: *conn.AdAccountID, PageID: *conn.PageID, Currency: *conn.Currency,
		DurationDays: input.DurationDays, Countries: countries, Message: message,
	})
	if err != nil {
		log.Printf("ads: création campagne vendeur=%s: %v", vendorID, err)
		http.Error(w, `{"error":"ad_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	// Lancement chez Meta : contexte détaché de la requête (un vendeur qui
	// ferme la page ne doit pas interrompre la création à mi-chemin).
	launchCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := h.launch(launchCtx, client, campaign, product); err != nil {
		log.Printf("ads: lancement Meta campagne=%s: %v", campaign.ID, err)
		if h.handleTokenError(launchCtx, vendorID, err) {
			_, _ = h.repo.MarkEnded(launchCtx, campaign.ID, model.AdStatusFailed, "Connexion Facebook expirée : reconnectez votre compte puis relancez la pub.")
			http.Error(w, `{"error":"meta_reconnect_required"}`, http.StatusConflict)
			return
		}
		reason := metaErrorReason(err)
		_, _ = h.repo.MarkEnded(launchCtx, campaign.ID, model.AdStatusFailed, reason)
		adJSON(w, http.StatusBadGateway, map[string]interface{}{"error": "ad_launch_failed", "details": reason})
		return
	}

	updated, err := h.repo.FindByID(ctx, campaign.ID)
	if err != nil {
		updated = campaign
	}
	adJSON(w, http.StatusCreated, map[string]interface{}{"campaign": updated})
}

// normalizeAdCountries — codes ISO2 connus (model.AdCountries), majuscules,
// sans doublon, dans l'ordre reçu.
func normalizeAdCountries(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, c := range in {
		c = strings.ToUpper(strings.TrimSpace(c))
		if _, ok := model.AdCountries[c]; ok && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

// minAdBudget — budget total minimum pour durationDays jours.
func minAdBudget(durationDays int) int {
	return durationDays * model.AdsMinDailyCFA
}

// launch crée la campagne chez Meta et enregistre ses identifiants.
func (h *AdHandler) launch(ctx context.Context, client *payment.MetaAdsClient, c *model.AdCampaign, product *model.Product) error {
	// Début dans 10 minutes (Meta refuse un start_time dans le passé), fin
	// après la durée choisie.
	start := time.Now().Add(10 * time.Minute)
	end := start.Add(time.Duration(c.DurationDays) * 24 * time.Hour)

	image := ""
	if product.CoverImageKey != nil && *product.CoverImageKey != "" && h.apiURL != "" {
		image = h.apiURL + "/api/products/" + product.ID + "/cover"
	}
	res, err := client.CreateSponsoredAd(ctx, payment.SponsoredAdRequest{
		Name:      fmt.Sprintf("DIARRA · %s · %s", truncateRunes(product.Title, 60), c.ID[:8]),
		LinkURL:   fmt.Sprintf("%s/product?id=%s&utm_source=meta&utm_medium=paid&utm_campaign=%s", h.frontendURL, product.ID, c.ID),
		Headline:  truncateRunes(product.Title, 40),
		Message:   c.Message,
		ImageURL:  image,
		Countries: c.Countries,
		BudgetXOF: c.BudgetCFA,
		StartTime: start,
		EndTime:   end,
	})
	if err != nil {
		return err
	}
	if err := h.repo.MarkLaunched(ctx, c.ID, res.CampaignID, res.AdSetID, res.CreativeID, res.AdID, start, end); err != nil {
		// La campagne tourne chez Meta mais DIARRA n'a pas pu l'enregistrer :
		// elle serait invisible pour le vendeur tout en dépensant son budget.
		// On coupe la diffusion d'abord.
		if pauseErr := client.PauseCampaign(ctx, res.CampaignID); pauseErr != nil {
			log.Printf("ads: ALERTE campagne Meta %s active mais non enregistrée (campagne DIARRA %s) et pause impossible: %v",
				res.CampaignID, c.ID, pauseErr)
		}
		return err
	}
	return nil
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
	if errors.Is(err, payment.ErrMetaCurrencyUnsupported) {
		return "La devise de votre compte publicitaire n'est pas prise en charge (FCFA, EUR ou USD uniquement)."
	}
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

// --- Arrêt -------------------------------------------------------------------

// StopVendor — POST /api/vendor/ads/{id}/stop : le vendeur arrête sa pub
// (mise en pause chez Meta, plus aucune dépense).
func (h *AdHandler) StopVendor(w http.ResponseWriter, r *http.Request) {
	c, err := h.repo.FindByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil || c.VendorID != middleware.GetUserID(r.Context()) {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	h.stop(w, r, c, "Arrêtée par vous.", false)
}

// Stop — POST /api/admin/ads/{id}/stop : un admin arrête une pub (contenu
// problématique...), avec le jeton du vendeur.
func (h *AdHandler) Stop(w http.ResponseWriter, r *http.Request) {
	c, err := h.repo.FindByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	h.stop(w, r, c, "Arrêtée par un administrateur DIARRA.", true)
}

func (h *AdHandler) stop(w http.ResponseWriter, r *http.Request, c *model.AdCampaign, reason string, byAdmin bool) {
	ctx := r.Context()
	if c.Status != model.AdStatusActive && c.Status != model.AdStatusInReview {
		http.Error(w, `{"error":"ad_not_running"}`, http.StatusConflict)
		return
	}
	if !h.configured() || c.ExternalCampaignID == nil {
		http.Error(w, `{"error":"ads_unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	conn, err := h.conns.Get(ctx, c.VendorID)
	if err != nil {
		http.Error(w, `{"error":"meta_not_connected"}`, http.StatusConflict)
		return
	}
	if conn.NeedsReconnect {
		http.Error(w, `{"error":"meta_reconnect_required"}`, http.StatusConflict)
		return
	}
	client, err := h.clientFor(ctx, conn, c.AdAccountID, c.PageID, c.Currency)
	if err != nil {
		http.Error(w, `{"error":"meta_reconnect_required"}`, http.StatusConflict)
		return
	}
	if err := client.PauseCampaign(ctx, *c.ExternalCampaignID); err != nil {
		log.Printf("ads: arrêt campagne=%s: %v", c.ID, err)
		if h.handleTokenError(ctx, c.VendorID, err) {
			http.Error(w, `{"error":"meta_reconnect_required"}`, http.StatusConflict)
			return
		}
		adJSON(w, http.StatusBadGateway, map[string]interface{}{"error": "ad_stop_failed", "details": metaErrorReason(err)})
		return
	}
	_ = h.repo.SetStatus(ctx, c.ID, model.AdStatusStopped, reason)
	if byAdmin {
		h.notify(ctx, c.VendorID, "ad_stopped", "Votre pub a été arrêtée",
			fmt.Sprintf("La sponsorisation de « %s » a été arrêtée par DIARRA.", c.ProductTitle))
	}
	updated, _ := h.repo.FindByID(ctx, c.ID)
	adJSON(w, http.StatusOK, map[string]interface{}{"campaign": updated})
}

// --- Synchronisation périodique ---------------------------------------------

// RunSyncLoop — toutes les 15 minutes : statut Meta (vérification, refus,
// diffusion, pause, fin) et statistiques, avec le jeton de chaque vendeur.
func (h *AdHandler) RunSyncLoop(ctx context.Context) {
	if !h.configured() {
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
			done, _ := h.repo.MarkEnded(ctx, c.ID, model.AdStatusFailed,
				"La création chez Meta a été interrompue. Vérifiez votre gestionnaire de publicités Meta et supprimez-y la campagne si elle existe.")
			if done {
				h.notify(ctx, c.VendorID, "ad_failed", "Pub non lancée",
					fmt.Sprintf("La création de la pub pour « %s » a été interrompue. Vous pouvez la relancer.", c.ProductTitle))
			}
		}
	}

	campaigns, err := h.repo.ListToSync(ctx)
	if err != nil {
		log.Printf("ads: lecture des campagnes à synchroniser: %v", err)
		return
	}
	// Une connexion par vendeur et par passage ; nil = vendeur à ignorer
	// (déconnecté, jeton à renouveler).
	conns := map[string]*model.VendorMetaConnection{}
	for _, c := range campaigns {
		conn, seen := conns[c.VendorID]
		if !seen {
			conn, err = h.conns.Get(ctx, c.VendorID)
			if err != nil || conn.NeedsReconnect {
				conn = nil
			}
			conns[c.VendorID] = conn
		}
		if conn == nil {
			continue
		}
		client, err := h.clientFor(ctx, conn, c.AdAccountID, c.PageID, c.Currency)
		if err != nil {
			conns[c.VendorID] = nil
			continue
		}
		callCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		err = h.syncOne(callCtx, client, c)
		cancel()
		if err != nil && h.handleTokenError(ctx, c.VendorID, err) {
			conns[c.VendorID] = nil
		}
	}
}

// syncOne met à jour une campagne. Renvoie l'erreur Meta éventuelle (pour
// détecter un jeton expiré) ; les autres erreurs sont seulement loguées.
func (h *AdHandler) syncOne(ctx context.Context, client *payment.MetaAdsClient, c *model.AdCampaign) error {
	if c.ExternalAdID == nil || c.ExternalCampaignID == nil {
		return nil
	}
	if c.Status == model.AdStatusInReview || c.Status == model.AdStatusActive {
		st, err := client.GetAdStatus(ctx, *c.ExternalAdID)
		if err != nil {
			log.Printf("ads: statut Meta campagne=%s: %v", c.ID, err)
			return err
		}
		ended := c.EndsAt != nil && time.Now().After(*c.EndsAt)
		switch st.EffectiveStatus {
		case "DISAPPROVED":
			reason := st.ReviewFeedback
			if reason == "" {
				reason = "Refusée par Meta (règles publicitaires)."
			}
			if done, _ := h.repo.MarkEnded(ctx, c.ID, model.AdStatusRejected, reason); done {
				h.notify(ctx, c.VendorID, "ad_rejected", "Pub refusée par Meta",
					fmt.Sprintf("Meta a refusé la pub pour « %s ». Raison : %s", c.ProductTitle, truncateRunes(reason, 200)))
			}
			return nil
		case "ACTIVE":
			if c.Status == model.AdStatusInReview {
				_ = h.repo.SetStatus(ctx, c.ID, model.AdStatusActive, "")
				h.notify(ctx, c.VendorID, "ad_active", "Votre pub est en ligne",
					fmt.Sprintf("« %s » est diffusé sur Facebook et Instagram.", c.ProductTitle))
				c.Status = model.AdStatusActive
			}
		case "WITH_ISSUES":
			_ = h.repo.SetStatus(ctx, c.ID, c.Status, st.ReviewFeedback)
		case "PAUSED", "CAMPAIGN_PAUSED", "ADSET_PAUSED", "ARCHIVED", "DELETED":
			// Le vendeur a arrêté la pub directement dans son gestionnaire de
			// publicités Meta (c'est son compte).
			if !ended {
				_ = h.repo.SetStatus(ctx, c.ID, model.AdStatusStopped, "Arrêtée depuis le gestionnaire de publicités Meta.")
				c.Status = model.AdStatusStopped
			}
		}
		if c.Status == model.AdStatusActive && ended {
			_ = h.repo.SetStatus(ctx, c.ID, model.AdStatusCompleted, "")
		}
	}
	if c.Status == model.AdStatusInReview {
		return nil // pas encore de statistiques
	}
	ins, err := client.GetCampaignInsights(ctx, *c.ExternalCampaignID)
	if err != nil {
		log.Printf("ads: statistiques Meta campagne=%s: %v", c.ID, err)
		return err
	}
	_ = h.repo.UpdateStats(ctx, c.ID, ins.Impressions, ins.Reach, ins.Clicks, ins.SpendXOF)
	return nil
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
		"meta_configured": h.configured(),
		"enabled":         h.enabled(r.Context()),
	})
}
