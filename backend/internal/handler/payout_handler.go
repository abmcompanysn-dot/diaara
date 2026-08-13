package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
)

type PayoutHandler struct {
	payoutRepo  *repository.PayoutRepo
	saleRepo    *repository.SaleRepo
	productRepo *repository.ProductRepo
	userRepo    *repository.UserRepo
	pawapay     *payment.PawaPayClient
}

func NewPayoutHandler(
	payoutRepo *repository.PayoutRepo,
	saleRepo *repository.SaleRepo,
	productRepo *repository.ProductRepo,
	userRepo *repository.UserRepo,
	pawapay *payment.PawaPayClient,
) *PayoutHandler {
	return &PayoutHandler{
		payoutRepo:  payoutRepo,
		saleRepo:    saleRepo,
		productRepo: productRepo,
		userRepo:    userRepo,
		pawapay:     pawapay,
	}
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
			"phone":         phone,
			"operator":      operator,
			"operator_label": label,
			"country":       country,
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

	payout, err := h.payoutRepo.Create(r.Context(), userID, input.AmountCFA, *msisdn, *provider)
	if err != nil {
		http.Error(w, `{"error":"payout_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	// Déclenche le versement mobile money PawaPay (asynchrone, comme le checkout).
	if h.pawapay != nil {
		pawapayID := uuidString()
		resp, err := h.pawapay.InitiatePayout(r.Context(), payment.PayoutRequest{
			PayoutId: pawapayID,
			Recipient: payment.Payer{
				Type: "MMO",
				AccountDetails: payment.AccountDetails{
					PhoneNumber: *msisdn,
					Provider:    *provider,
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
			http.Error(w, fmt.Sprintf(`{"error":"payout_rejected","reason":"%s"}`, reason), http.StatusBadGateway)
			return
		}
		if err := h.payoutRepo.SetPawaPayReference(r.Context(), payout.ID, pawapayID); err != nil {
			http.Error(w, `{"error":"payout_update_failed"}`, http.StatusInternalServerError)
			return
		}
		payout.PawaPayPayoutID = &pawapayID
		payout.Status = "processing"
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
