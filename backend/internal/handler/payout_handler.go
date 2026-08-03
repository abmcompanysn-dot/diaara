package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
)

type PayoutHandler struct {
	payoutRepo  *repository.PayoutRepo
	saleRepo    *repository.SaleRepo
	productRepo *repository.ProductRepo
}

func NewPayoutHandler(
	payoutRepo *repository.PayoutRepo,
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
) *PayoutHandler {
	return &PayoutHandler{
		payoutRepo:  payoutRepo,
		saleRepo:    saleRepo,
		productRepo: productRepo,
	}
}

// Earnings — GET /api/vendor/earnings
// Retourne le total gagné, le disponible et l'historique des versements.
func (h *PayoutHandler) Earnings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_earned": totalEarned,
		"available":    available,
		"pending":      requested,
		"history":      payouts,
	})
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

	payout, err := h.payoutRepo.Create(r.Context(), userID, input.AmountCFA)
	if err != nil {
		http.Error(w, `{"error":"payout_creation_failed"}`, http.StatusInternalServerError)
		return
	}

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
		if s.Status != "failed" && s.Status != "refunded" {
			product, err := h.productRepo.FindByID(ctx, s.ProductID)
			if err != nil {
				continue
			}
			if product.VendorID == vendorID {
				total += s.VendorAmountCFA
			}
		}
	}
	return total, nil
}
