package handler

import (
	"context"
	"encoding/json"
	"net/http"
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
// réglage est absent (compatible avec le comportement d'avant KPay).
func (h *PayoutHandler) resolvePayoutProvider(ctx context.Context, providerCode string) string {
	return h.settingsRepo.Get(ctx, model.GatewayOperatorSettingKey(providerCode), "pawapay")
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

	label := ""
	if operator != nil {
		for _, op := range payment.XOFOperators {
			if op.Provider == *operator {
				label = op.Label
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payout_method": map[string]interface{}{
			"phone":          phone,
			"operator":       operator,
			"operator_label": label,
			"country":        country,
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
			"phone":          msisdn,
			"operator":       op.Provider,
			"operator_label": op.Label,
			"country":        input.Country,
		},
	})
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

	// Le compte destinataire est celui enregistré au préalable (voir SetPayoutMethod),
	// pas resaisi à chaque demande de versement.
	msisdn, provider, _, err := h.userRepo.GetPayoutMethod(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"payout_method_lookup_failed"}`, http.StatusInternalServerError)
		return
	}
	if msisdn == nil || provider == nil || *msisdn == "" || *provider == "" {
		http.Error(w, `{"error":"payout_method_required"}`, http.StatusBadRequest)
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

	// Prestataire pressenti, résolu et persisté ici (voir PayoutRepo.Create) —
	// un changement de réglage admin après coup ne doit jamais rediriger ce
	// versement. NB : "off" n'est plus bloquant côté vendeur (il ne doit jamais
	// voir d'erreur sur une demande de retrait) ; l'admin verra le cas au
	// moment de régler la demande et pourra la traiter manuellement.
	providerName := h.resolvePayoutProvider(r.Context(), *provider)

	// La demande est SEULEMENT enregistrée "en attente" (requested). Aucun
	// versement n'est déclenché automatiquement : c'est l'admin qui décide,
	// demande par demande, de régler automatiquement (PawaPay/KPay) ou
	// manuellement (voir AdminHandler.SettlePayoutAuto / SettlePayoutManual).
	// Le solde disponible est déduit dès maintenant (Earnings compte tout
	// sauf 'failed' dans "requested").
	payout, err := h.payoutRepo.Create(r.Context(), userID, input.AmountCFA, *msisdn, *provider, providerName)
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
