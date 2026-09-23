package handler

import (
	"encoding/json"
	"net/http"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

// AnnouncementHandler — "Dernières mises à jour" côté admin -> dashboard
// vendeur (voir migrations/040_announcements.sql). Pas de notification
// email : lecture passive au chargement du dashboard vendeur.
type AnnouncementHandler struct {
	repo *repository.AnnouncementRepo
}

func NewAnnouncementHandler(repo *repository.AnnouncementRepo) *AnnouncementHandler {
	return &AnnouncementHandler{repo: repo}
}

// Create — POST /api/admin/announcements (scope "users" ou équivalent large,
// voir routing dans main.go).
func (h *AnnouncementHandler) Create(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetUserID(r.Context())
	var input model.CreateAnnouncementInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Title == "" || input.Body == "" {
		http.Error(w, `{"error":"title_and_body_required"}`, http.StatusBadRequest)
		return
	}
	a, err := h.repo.Create(r.Context(), adminID, input)
	if err != nil {
		http.Error(w, `{"error":"create_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"announcement": a})
}

// List — GET /api/admin/announcements (vue admin, pour gérer/supprimer) ET
// GET /api/vendor/announcements (vue vendeur, lecture seule) — même contenu,
// deux routes distinctes pour ne pas exposer les routes admin (/api/admin/*)
// à un compte vendeur non-admin.
func (h *AnnouncementHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"announcements": items})
}

// Delete — DELETE /api/admin/announcements/{id}.
func (h *AnnouncementHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
