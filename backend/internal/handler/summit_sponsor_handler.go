package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

// SummitSponsorHandler expose les paliers de sponsoring et les sponsors
// (lecture publique sur /summit/sponsors, gestion complète en admin).
type SummitSponsorHandler struct {
	repo    *repository.SummitSponsorRepo
	storage StorageService
}

func NewSummitSponsorHandler(repo *repository.SummitSponsorRepo, storage StorageService) *SummitSponsorHandler {
	return &SummitSponsorHandler{repo: repo, storage: storage}
}

// Logo — GET /api/summit/sponsors/{id}/logo (public). Sert le logo depuis le
// stockage objet plutôt que d'exposer la clé S3 brute côté client — même
// mécanique que ProductHandler.Cover.
func (h *SummitSponsorHandler) Logo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sponsors, err := h.repo.ListPublished(r.Context())
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	var logoKey string
	for _, s := range sponsors {
		if s.ID == id && s.LogoKey != nil {
			logoKey = *s.LogoKey
			break
		}
	}
	if logoKey == "" {
		http.Error(w, `{"error":"no_logo"}`, http.StatusNotFound)
		return
	}
	if h.storage == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	data, err := h.storage.Download(r.Context(), logoKey)
	if err != nil {
		http.Error(w, `{"error":"logo_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", imageContentType(logoKey, data))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}

// Public — GET /api/summit/sponsors. Paliers + sponsors publiés uniquement.
func (h *SummitSponsorHandler) Public(w http.ResponseWriter, r *http.Request) {
	tiers, err := h.repo.ListTiers(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	sponsors, err := h.repo.ListPublished(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tiers": tiers, "sponsors": sponsors})
}

// --- Admin : paliers ---

func (h *SummitSponsorHandler) ListTiers(w http.ResponseWriter, r *http.Request) {
	tiers, err := h.repo.ListTiers(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tiers": tiers})
}

func (h *SummitSponsorHandler) CreateTier(w http.ResponseWriter, r *http.Request) {
	var input model.CreateSummitSponsorTierInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || input.PriceCFA < 0 {
		http.Error(w, `{"error":"name_and_price_required"}`, http.StatusBadRequest)
		return
	}
	tier, err := h.repo.CreateTier(r.Context(), input)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tier)
}

func (h *SummitSponsorHandler) UpdateTier(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input model.UpdateSummitSponsorTierInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	tier, err := h.repo.UpdateTier(r.Context(), id, input)
	if err != nil {
		if err == repository.ErrSummitSponsorTierNotFound {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tier)
}

func (h *SummitSponsorHandler) DeleteTier(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.DeleteTier(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Admin : sponsors ---

func (h *SummitSponsorHandler) ListSponsors(w http.ResponseWriter, r *http.Request) {
	sponsors, err := h.repo.ListAll(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"sponsors": sponsors})
}

func (h *SummitSponsorHandler) CreateSponsor(w http.ResponseWriter, r *http.Request) {
	var input model.CreateSummitSponsorInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		http.Error(w, `{"error":"name_required"}`, http.StatusBadRequest)
		return
	}
	sponsor, err := h.repo.Create(r.Context(), input)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sponsor)
}

func (h *SummitSponsorHandler) UpdateSponsor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input model.UpdateSummitSponsorInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	sponsor, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if err == repository.ErrSummitSponsorNotFound {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sponsor)
}

func (h *SummitSponsorHandler) DeleteSponsor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
