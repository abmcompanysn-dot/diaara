package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
)

// maxSummitFieldLen borne la taille des champs du formulaire d'inscription
// public (endpoint non authentifié) pour limiter l'abus.
const maxSummitFieldLen = 200

// summitEventDateLabel / summitEventFrontendPath — date et page publique du
// DIARRA Summit, reprises dans l'email de confirmation. À ajuster ici si la
// date change (pas de table de config dédiée pour un événement ponctuel).
const summitEventDateLabel = "26 novembre 2026"
const summitEventFrontendPath = "/summit"

// SummitHandler expose l'inscription publique au DIARRA Summit et
// l'historique des inscrits côté admin.
type SummitHandler struct {
	repo          *repository.SummitRepo
	notifications *email.NotificationService
	frontendURL   string
}

func NewSummitHandler(repo *repository.SummitRepo, notifications *email.NotificationService, frontendURL string) *SummitHandler {
	return &SummitHandler{repo: repo, notifications: notifications, frontendURL: frontendURL}
}

// Register — POST /api/summit/register (public).
func (h *SummitHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input model.CreateSummitRegistrationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.FullName = strings.TrimSpace(input.FullName)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)
	input.Profile = strings.TrimSpace(strings.ToLower(input.Profile))

	if input.FullName == "" || input.Email == "" || input.Phone == "" || input.Profile == "" {
		http.Error(w, `{"error":"fields_required"}`, http.StatusBadRequest)
		return
	}
	if len(input.FullName) > maxSummitFieldLen || len(input.Email) > maxSummitFieldLen || len(input.Phone) > maxSummitFieldLen {
		http.Error(w, `{"error":"field_too_long"}`, http.StatusBadRequest)
		return
	}
	if !strings.Contains(input.Email, "@") {
		http.Error(w, `{"error":"invalid_email"}`, http.StatusBadRequest)
		return
	}
	if !model.SummitProfiles[input.Profile] {
		http.Error(w, `{"error":"invalid_profile"}`, http.StatusBadRequest)
		return
	}

	reg, err := h.repo.Create(r.Context(), input)
	if err != nil {
		if err == repository.ErrSummitAlreadyRegistered {
			http.Error(w, `{"error":"already_registered"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"registration_failed"}`, http.StatusInternalServerError)
		return
	}

	// Email de confirmation best-effort : une panne du fournisseur d'email
	// ne doit pas faire échouer l'inscription, déjà persistée en base.
	if h.notifications != nil {
		eventLink := strings.TrimSuffix(h.frontendURL, "/") + summitEventFrontendPath
		if err := h.notifications.SendSummitConfirmation(r.Context(), reg.Email, reg.FullName, summitEventDateLabel, eventLink); err != nil {
			log.Printf("WARNING: email confirmation DIARRA Summit échoué pour %s: %v", reg.Email, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reg)
}

// List — GET /api/admin/summit/registrations (admin).
func (h *SummitHandler) List(w http.ResponseWriter, r *http.Request) {
	regs, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	count, err := h.repo.Count(r.Context())
	if err != nil {
		http.Error(w, `{"error":"count_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"registrations": regs,
		"count":         count,
	})
}
