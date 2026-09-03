package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/diarra/backend/internal/cache"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
)

type PayoutHandler struct {
	payoutRepo   *repository.PayoutRepo
	saleRepo     *repository.SaleRepo
	productRepo  *repository.ProductRepo
	userRepo     *repository.UserRepo
	settingsRepo *repository.SettingsRepo
	pawapay      *payment.PawaPayClient
	kpay         *payment.KPayClient
	cache        *cache.Client

	// Cache en mémoire des limites de versement par opérateur (PawaPay
	// Active Configuration) : ces valeurs changent rarement, on évite un
	// appel PawaPay à chaque chargement de la page vendeur.
	limitsMu      sync.Mutex
	limitsCache   map[string]struct{ Min, Max int }
	limitsFetched time.Time
}

func NewPayoutHandler(
	payoutRepo *repository.PayoutRepo,
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
	userRepo *repository.UserRepo,
	settingsRepo *repository.SettingsRepo,
	pawapay *payment.PawaPayClient,
	kpay *payment.KPayClient,
	cacheClient *cache.Client,
) *PayoutHandler {
	return &PayoutHandler{
		payoutRepo:   payoutRepo,
		saleRepo:     saleRepo,
		productRepo:  productRepo,
		userRepo:     userRepo,
		settingsRepo: settingsRepo,
		pawapay:      pawapay,
		kpay:         kpay,
		cache:        cacheClient,
	}
}

// resolvePayoutProvider lit le réglage admin par opérateur exact (voir
// model.GatewayOperatorSettingKey) et retourne le nom du prestataire à
// utiliser ("off" bloque explicitement l'opérateur). Défaut "pawapay" si le
// réglage est absent. KPay est suspendu (2026-09-03) : une valeur "kpay"
// éventuellement restée en base est traitée comme "pawapay" — voir aussi le
// refus à l'écriture dans AdminHandler.UpdateSettings.
func (h *PayoutHandler) resolvePayoutProvider(ctx context.Context, providerCode string) string {
	v := h.settingsRepo.Get(ctx, model.GatewayOperatorSettingKey(providerCode), "pawapay")
	if v == "kpay" {
		return "pawapay"
	}
	return v
}

// vendorBalanceCacheKey — même package que webhook_handler.go, qui invalide
// cette clé quand une vente passe "paid" (voir Earnings, qui l'alimente, et
// Create, qui l'invalide aussi après un versement).
func vendorBalanceCacheKey(vendorID string) string {
	return "vendor:balance:" + vendorID
}

// GetPayoutMethod — GET /api/vendor/payout-method
// Retourne le moyen de versement enregistré du vendeur (nuls si jamais renseigné).
func (h *PayoutHandler) GetPayoutMethod(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	phone, operator, country, err := h.userRepo.GetPayoutMethod(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"payout_method_lookup_failed"}`, http.StatusInternalServerError)
		return
	}
	paypalEmail, err := h.userRepo.GetPayoutPayPalEmail(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"payout_method_lookup_failed"}`, http.StatusInternalServerError)
		return
	}

	label := ""
	if operator != nil {
		for _, op := range payment.XOFOperators {
			if op.Provider == *operator {
				label = op.Label
				break
			}
		}
	}

	// Canal actif par défaut : PayPal s'il est renseigné (prioritaire), sinon
	// mobile money. Le frontend s'en sert pour préselectionner l'onglet.
	activeChannel := "mobile_money"
	if paypalEmail != nil && *paypalEmail != "" {
		activeChannel = "paypal"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payout_method": map[string]interface{}{
			"active_channel": activeChannel,
			"phone":          phone,
			"operator":       operator,
			"operator_label": label,
			"country":        country,
			"paypal_email":   paypalEmail,
		},
	})
}

// SetPayoutMethod — PUT /api/vendor/payout-method
// Le vendeur enregistre ou modifie le compte mobile money qui recevra ses versements.
func (h *PayoutHandler) SetPayoutMethod(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.SetPayoutMethodInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	// Canal PayPal : on n'enregistre qu'un email (validation de forme
	// minimale). N'écrase jamais le mobile money — les deux coexistent.
	if input.Channel == "paypal" {
		email := strings.TrimSpace(input.PayPalEmail)
		if !looksLikeEmail(email) {
			http.Error(w, `{"error":"invalid_paypal_email"}`, http.StatusBadRequest)
			return
		}
		if err := h.userRepo.SetPayoutPayPalEmail(r.Context(), userID, email); err != nil {
			http.Error(w, `{"error":"payout_method_save_failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"payout_method": map[string]interface{}{
				"active_channel": "paypal",
				"paypal_email":   email,
			},
		})
		return
	}

	// Canal mobile money (défaut).
	if input.Phone == "" || input.Operator == "" || input.Country == "" {
		http.Error(w, `{"error":"payment_details_required"}`, http.StatusBadRequest)
		return
	}
	op, err := payment.ResolveOperator(input.Country, input.Operator)
	if err != nil {
		http.Error(w, `{"error":"unsupported_operator"}`, http.StatusBadRequest)
		return
	}
	if h.resolvePayoutProvider(r.Context(), op.Provider) == "off" {
		http.Error(w, `{"error":"gateway_disabled"}`, http.StatusBadRequest)
		return
	}
	msisdn, err := payment.NormalizePhone(op.DialCode, input.Phone)
	if err != nil {
		http.Error(w, `{"error":"invalid_phone_number"}`, http.StatusBadRequest)
		return
	}

	if err := h.userRepo.SetPayoutMethod(r.Context(), userID, msisdn, op.Provider, input.Country); err != nil {
		http.Error(w, `{"error":"payout_method_save_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payout_method": map[string]interface{}{
			"active_channel": "mobile_money",
			"phone":          msisdn,
			"operator":       op.Provider,
			"operator_label": op.Label,
			"country":        input.Country,
		},
	})
}

// looksLikeEmail — validation de forme minimale (un @ au milieu, un point
// après). Suffisant : PayPal rejettera de toute façon un email invalide au
// moment du versement, on veut juste éviter les saisies vides/absurdes.
func looksLikeEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	return strings.IndexByte(s[at+1:], '.') > 0
}

// Earnings — GET /api/vendor/earnings
// Retourne le total gagné, le disponible et l'historique des versements.
func (h *PayoutHandler) Earnings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Endpoint d'affichage seulement — la demande de versement elle-même
	// (Create, plus bas) recalcule toujours totalEarned() en direct sur la
	// base, donc un solde caché ici ne peut jamais permettre un retrait
	// au-delà du solde réel.
	cacheKey := vendorBalanceCacheKey(userID)
	var cached map[string]interface{}
	if hit, _ := h.cache.GetJSON(r.Context(), cacheKey, &cached); hit {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Total gagné par le vendeur (somme des vendor_amount_cfa sur les ventes payées)
	totalEarned, err := h.totalEarned(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"stats_failed"}`, http.StatusInternalServerError)
		return
	}

	// Montant déjà demandé en versement
	payouts, err := h.payoutRepo.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"payouts_failed"}`, http.StatusInternalServerError)
		return
	}

	requested := 0
	for _, p := range payouts {
		if p.Status != "failed" {
			requested += p.AmountCFA
		}
	}

	available := totalEarned - requested
	if available < 0 {
		available = 0
	}

	earnings := map[string]interface{}{
		"total_earned": totalEarned,
		"available":    available,
		"pending":      requested,
		"history":      payouts,
		"tier":         model.VendorTier(totalEarned),
	}
	h.cache.SetJSON(r.Context(), cacheKey, earnings, 30*time.Second)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(earnings)
}

// Limits — GET /api/vendor/payout-limits
// Renvoie, pour chaque opérateur PawaPay, le montant minimum/maximum réel
// accepté pour un versement — pour que le vendeur voie ces limites avant de
// tenter un retrait qui serait sinon refusé par PawaPay (payout_rejected).
// Mise en cache 6h en mémoire (les limites changent rarement).
func (h *PayoutHandler) Limits(w http.ResponseWriter, r *http.Request) {
	if h.pawapay == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"limits": map[string]interface{}{}})
		return
	}

	h.limitsMu.Lock()
	defer h.limitsMu.Unlock()

	if h.limitsCache == nil || time.Since(h.limitsFetched) > 6*time.Hour {
		config, err := h.pawapay.GetActiveConfig(r.Context(), "PAYOUT")
		if err != nil {
			// En cas d'échec, on sert le cache existant s'il y en a un (même
			// périmé) plutôt que de bloquer l'affichage de la page vendeur.
			if h.limitsCache == nil {
				http.Error(w, `{"error":"limits_unavailable"}`, http.StatusBadGateway)
				return
			}
		} else {
			h.limitsCache = config.PayoutLimits()
			h.limitsFetched = time.Now()
		}
	}

	limits := make(map[string]map[string]int, len(h.limitsCache))
	for provider, l := range h.limitsCache {
		limits[provider] = map[string]int{"min": l.Min, "max": l.Max}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"limits": limits})
}

// Create — POST /api/vendor/payouts
func (h *PayoutHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.CreatePayoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.AmountCFA <= 0 {
		http.Error(w, `{"error":"invalid_amount"}`, http.StatusBadRequest)
		return
	}

	// Le compte destinataire est celui enregistré au préalable (voir
	// SetPayoutMethod), pas resaisi à chaque demande. Deux canaux possibles ;
	// PayPal est prioritaire quand le vendeur a renseigné un email PayPal.
	msisdn, provider, _, err := h.userRepo.GetPayoutMethod(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"payout_method_lookup_failed"}`, http.StatusInternalServerError)
		return
	}
	paypalEmail, err := h.userRepo.GetPayoutPayPalEmail(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"payout_method_lookup_failed"}`, http.StatusInternalServerError)
		return
	}

	hasMobileMoney := msisdn != nil && provider != nil && *msisdn != "" && *provider != ""
	hasPayPal := paypalEmail != nil && *paypalEmail != ""
	if !hasMobileMoney && !hasPayPal {
		http.Error(w, `{"error":"payout_method_required"}`, http.StatusBadRequest)
		return
	}
	usePayPal := hasPayPal
	if usePayPal && input.AmountCFA < model.PayPalPayoutMinCFA && hasMobileMoney {
		// Sous le plancher PayPal mais un mobile money existe : on route vers
		// le mobile money plutôt que de refuser.
		usePayPal = false
	}
	if usePayPal && input.AmountCFA < model.PayPalPayoutMinCFA {
		http.Error(w, `{"error":"amount_below_paypal_minimum"}`, http.StatusBadRequest)
		return
	}

	// Vérifier que le montant est disponible
	totalEarned, err := h.totalEarned(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"stats_failed"}`, http.StatusInternalServerError)
		return
	}

	payouts, err := h.payoutRepo.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"payouts_failed"}`, http.StatusInternalServerError)
		return
	}

	requested := 0
	for _, p := range payouts {
		if p.Status != "failed" {
			requested += p.AmountCFA
		}
	}

	if input.AmountCFA > totalEarned-requested {
		http.Error(w, `{"error":"insufficient_balance"}`, http.StatusBadRequest)
		return
	}

	// La demande est SEULEMENT enregistrée "en attente" (requested). Aucun
	// versement n'est déclenché automatiquement : c'est l'admin qui décide,
	// demande par demande, de régler automatiquement (PawaPay/PayPal) ou
	// manuellement (voir AdminHandler.SettlePayoutAuto / SettlePayoutManual).
	// Le solde disponible est déduit dès maintenant (Earnings compte tout
	// sauf 'failed' dans "requested").
	var payout *model.Payout
	if usePayPal {
		payout, err = h.payoutRepo.CreatePayPal(r.Context(), userID, input.AmountCFA, *paypalEmail)
	} else {
		// Prestataire pressenti, résolu et persisté ici (voir PayoutRepo.Create)
		// — un changement de réglage admin après coup ne doit jamais rediriger
		// ce versement. "off" n'est pas bloquant côté vendeur ; l'admin le verra
		// au moment de régler.
		providerName := h.resolvePayoutProvider(r.Context(), *provider)
		payout, err = h.payoutRepo.Create(r.Context(), userID, input.AmountCFA, *msisdn, *provider, providerName)
	}
	if err != nil {
		http.Error(w, `{"error":"payout_creation_failed"}`, http.StatusInternalServerError)
		return
	}
	h.cache.Del(r.Context(), vendorBalanceCacheKey(userID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"payout": payout})
}

func (h *PayoutHandler) totalEarned(ctx context.Context, vendorID string) (int, error) {
	sales, err := h.saleRepo.ListAll(ctx)
	if err != nil {
		return 0, err
	}

	total := 0
	for _, s := range sales {
		// Seules les ventes réellement payées comptent dans le chiffre d'affaires
		// du vendeur — une commande "pending"/"refund_pending" n'est pas de
		// l'argent reçu et ne doit jamais être disponible pour un versement.
		if s.Status != "paid" && s.Status != "delivered" {
			continue
		}
		product, err := h.productRepo.FindByID(ctx, s.ProductID)
		if err != nil {
			continue
		}
		if product.VendorID == vendorID {
			total += s.VendorAmountCFA
		}
	}
	return total, nil
}
