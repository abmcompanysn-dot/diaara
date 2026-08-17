package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/notify"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

// maxSupportFieldLen borne la taille des champs du widget de contact public
// (endpoint non authentifié, ouvert à n'importe quel visiteur) pour limiter
// l'abus et éviter des emails de notification disproportionnés.
const maxSupportFieldLen = 2000

// SupportContactHandler expose le widget de contact public (visiteur non
// authentifié, sans compte) et l'administration des agents support qui
// reçoivent les notifications. Accessible à tout admin, comme les tickets
// (pas de scope dédié).
type SupportContactHandler struct {
	repo          *repository.SupportContactRepo
	notifications *email.NotificationService
}

func NewSupportContactHandler(repo *repository.SupportContactRepo, notifications *email.NotificationService) *SupportContactHandler {
	return &SupportContactHandler{repo: repo, notifications: notifications}
}

// Contact — POST /api/support/contact (public). Un visiteur envoie un
// message ; chaque agent support actif reçoit un email de notification avec
// un lien de réponse directe (mailto ou WhatsApp selon le canal choisi).
func (h *SupportContactHandler) Contact(w http.ResponseWriter, r *http.Request) {
	var input model.CreateSupportContactInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.ContactValue = strings.TrimSpace(input.ContactValue)
	input.Message = strings.TrimSpace(input.Message)

	if input.Name == "" || input.ContactValue == "" || input.Message == "" {
		http.Error(w, `{"error":"fields_required"}`, http.StatusBadRequest)
		return
	}
	if input.ContactMethod != "email" && input.ContactMethod != "whatsapp" {
		http.Error(w, `{"error":"invalid_contact_method"}`, http.StatusBadRequest)
		return
	}
	if len(input.Name) > maxSupportFieldLen || len(input.ContactValue) > maxSupportFieldLen || len(input.Message) > maxSupportFieldLen {
		http.Error(w, `{"error":"field_too_long"}`, http.StatusBadRequest)
		return
	}
	if input.ContactMethod == "email" && !strings.Contains(input.ContactValue, "@") {
		http.Error(w, `{"error":"invalid_email"}`, http.StatusBadRequest)
		return
	}

	req, err := h.repo.CreateContactRequest(r.Context(), input)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}

	if h.notifications != nil {
		go h.notifyAgents(context.Background(), req)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *SupportContactHandler) notifyAgents(ctx context.Context, req *model.SupportContactRequest) {
	agents, err := h.repo.ListActiveAgents(ctx)
	if err != nil {
		return
	}
	waText := fmt.Sprintf("Nouveau message support de %s (%s : %s) :\n%s", req.Name, req.ContactMethod, req.ContactValue, req.Message)
	for _, agent := range agents {
		_ = h.notifications.SendSupportContact(ctx, agent.Email, req.Name, req.ContactMethod, req.ContactValue, req.Message)
		if agent.Phone != "" && agent.CallMeBotAPIKey != "" {
			_ = notify.SendCallMeBotWhatsApp(ctx, agent.Phone, agent.CallMeBotAPIKey, waText)
		}
	}
}

// --- Administration (tout admin, pas de scope dédié — comme les tickets) ---

// ListAgents — GET /api/admin/support-agents
func (h *SupportContactHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := h.repo.ListAgents(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"agents": agents})
}

// CreateAgent — POST /api/admin/support-agents
func (h *SupportContactHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var input model.CreateSupportAgentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	input.CallMeBotAPIKey = strings.TrimSpace(input.CallMeBotAPIKey)
	if input.Name == "" || input.Email == "" || !strings.Contains(input.Email, "@") {
		http.Error(w, `{"error":"name_and_email_required"}`, http.StatusBadRequest)
		return
	}

	agent, err := h.repo.CreateAgent(r.Context(), input.Name, input.Email, input.Phone, input.CallMeBotAPIKey)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"agent": agent})
}

// UpdateAgent — PUT /api/admin/support-agents/{id}
func (h *SupportContactHandler) UpdateAgent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input model.UpdateSupportAgentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	agent, err := h.repo.UpdateAgent(r.Context(), id, input)
	if err != nil {
		if err == repository.ErrSupportAgentNotFound {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"agent": agent})
}

// DeleteAgent — DELETE /api/admin/support-agents/{id}
func (h *SupportContactHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.DeleteAgent(r.Context(), id); err != nil {
		if err == repository.ErrSupportAgentNotFound {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListContacts — GET /api/admin/support-contacts (historique, 200 derniers)
func (h *SupportContactHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	requests, err := h.repo.ListContactRequests(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"requests": requests})
}
