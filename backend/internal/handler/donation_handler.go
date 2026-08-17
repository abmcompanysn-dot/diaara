package handler

import (
	"encoding/json"
	"net/http"

	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/go-chi/chi/v5"
)

// DonationHandler expose l'administration du programme de reversement
// automatique ("Fidélisation") — cagnotte, destinataires, historique.
// Scope admin "finance", même groupe que /api/admin/settings.
type DonationHandler struct {
	repo         *repository.DonationRepo
	settingsRepo *repository.SettingsRepo
	svc          *service.DonationService
}

func NewDonationHandler(repo *repository.DonationRepo, settingsRepo *repository.SettingsRepo, svc *service.DonationService) *DonationHandler {
	return &DonationHandler{repo: repo, settingsRepo: settingsRepo, svc: svc}
}

// Get — GET /api/admin/donations. Solde de la cagnotte, réglages actuels,
// destinataires et historique des versements.
func (h *DonationHandler) Get(w http.ResponseWriter, r *http.Request) {
	pool, err := h.repo.Pool(r.Context())
	if err != nil {
		http.Error(w, `{"error":"donation_pool_failed"}`, http.StatusInternalServerError)
		return
	}
	recipients, err := h.repo.ListRecipients(r.Context())
	if err != nil {
		http.Error(w, `{"error":"donation_recipients_failed"}`, http.StatusInternalServerError)
		return
	}
	payouts, err := h.repo.ListPayouts(r.Context())
	if err != nil {
		http.Error(w, `{"error":"donation_payouts_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pool":       pool,
		"recipients": recipients,
		"payouts":    payouts,
		"settings": map[string]interface{}{
			"share_pct":     h.settingsRepo.GetFloat(r.Context(), model.SettingDonationSharePct, service.DefaultDonationSharePct),
			"threshold_cfa": h.settingsRepo.GetFloat(r.Context(), model.SettingDonationThresholdCFA, service.DefaultDonationThresholdCFA),
			"enabled":       h.settingsRepo.GetBool(r.Context(), model.SettingDonationEnabled, true),
		},
	})
}

// CreateRecipient — POST /api/admin/donations/recipients
func (h *DonationHandler) CreateRecipient(w http.ResponseWriter, r *http.Request) {
	var input model.CreateDonationRecipientInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Name == "" || input.Phone == "" || input.Operator == "" || input.Country == "" {
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

	rec, err := h.repo.CreateRecipient(r.Context(), input.Name, msisdn, op.Provider, input.Country)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"recipient": rec})
}

// UpdateRecipient — PUT /api/admin/donations/recipients/{id}
func (h *DonationHandler) UpdateRecipient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input model.UpdateDonationRecipientInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	rec, err := h.repo.UpdateRecipient(r.Context(), id, input)
	if err != nil {
		if err == repository.ErrDonationRecipientNotFound {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"recipient": rec})
}

// DeleteRecipient — DELETE /api/admin/donations/recipients/{id}
func (h *DonationHandler) DeleteRecipient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.DeleteRecipient(r.Context(), id); err != nil {
		if err == repository.ErrDonationRecipientNotFound {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RetryPayout — POST /api/admin/donations/payouts/{id}/retry
func (h *DonationHandler) RetryPayout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.RetryPayout(r.Context(), id); err != nil {
		http.Error(w, `{"error":"retry_failed","reason":"`+err.Error()+`"}`, http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
