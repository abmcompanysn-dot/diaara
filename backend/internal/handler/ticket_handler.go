package handler

import (
	"encoding/json"
	"net/http"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type TicketHandler struct {
	ticketRepo    *repository.TicketRepo
	adminPermRepo *repository.AdminPermissionRepo
}

func NewTicketHandler(ticketRepo *repository.TicketRepo, adminPermRepo *repository.AdminPermissionRepo) *TicketHandler {
	return &TicketHandler{ticketRepo: ticketRepo, adminPermRepo: adminPermRepo}
}

// Create — POST /api/tickets
func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.CreateTicketInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.Subject == "" || input.Message == "" {
		http.Error(w, `{"error":"subject_and_message_required"}`, http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketRepo.Create(r.Context(), userID, input)
	if err != nil {
		http.Error(w, `{"error":"ticket_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	// Premier message du ticket
	if _, err := h.ticketRepo.AddMessage(r.Context(), ticket.ID, userID, input.Message); err != nil {
		http.Error(w, `{"error":"message_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"ticket": ticket})
}

// List — GET /api/tickets (utilisateur = ses tickets, admin = tous)
func (h *TicketHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var tickets []*model.SupportTicket
	var err error
	if isAdmin {
		tickets, err = h.ticketRepo.ListAll(r.Context())
	} else {
		tickets, err = h.ticketRepo.ListByUser(r.Context(), userID)
	}
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tickets": tickets})
}

// GetMessages — GET /api/tickets/{id}/messages
func (h *TicketHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())
	id := chi.URLParam(r, "id")

	ticket, err := h.ticketRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	if ticket.UserID != userID && !isAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	messages, err := h.ticketRepo.ListMessages(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ticket":   ticket,
		"messages": messages,
	})
}

// AddMessage — POST /api/tickets/{id}/messages
func (h *TicketHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())
	id := chi.URLParam(r, "id")

	ticket, err := h.ticketRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	if ticket.UserID != userID && !isAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	var input model.CreateTicketMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.Body == "" {
		http.Error(w, `{"error":"message_required"}`, http.StatusBadRequest)
		return
	}

	message, err := h.ticketRepo.AddMessage(r.Context(), id, userID, input.Body)
	if err != nil {
		http.Error(w, `{"error":"message_failed"}`, http.StatusInternalServerError)
		return
	}

	// Passer le ticket à "answered" ou "open"
	if isAdmin {
		h.ticketRepo.UpdateStatus(r.Context(), id, "answered")
	} else {
		h.ticketRepo.UpdateStatus(r.Context(), id, "open")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": message})
}

// Claim — PUT /api/admin/tickets/{id}/claim (admin uniquement).
// Le premier agent à cliquer "Prendre en charge" gagne le ticket ; les
// autres agents reçoivent une erreur explicite (voir ClaimTicket, UPDATE
// atomique) au lieu de traiter la même conversation en double.
func (h *TicketHandler) Claim(w http.ResponseWriter, r *http.Request) {
	isAdmin := middleware.GetIsAdmin(r.Context())
	if !isAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	adminID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	ticket, err := h.ticketRepo.ClaimTicket(r.Context(), id, adminID)
	if err != nil {
		switch err {
		case repository.ErrTicketAlreadyClaimed:
			http.Error(w, `{"error":"ticket_already_claimed"}`, http.StatusConflict)
		case repository.ErrTicketNotFound:
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		default:
			http.Error(w, `{"error":"claim_failed"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ticket": ticket})
}

// Assign — PUT /api/admin/tickets/{id}/assign (admin uniquement).
// Redirige un ticket vers un agent précis, qu'il soit déjà pris en charge ou
// non — utile quand le mauvais agent a répondu, ou pour équilibrer la charge.
func (h *TicketHandler) Assign(w http.ResponseWriter, r *http.Request) {
	isAdmin := middleware.GetIsAdmin(r.Context())
	if !isAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")

	var input model.AssignTicketInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.AdminID == "" {
		http.Error(w, `{"error":"admin_id_required"}`, http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketRepo.AssignTicket(r.Context(), id, input.AdminID)
	if err != nil {
		if err == repository.ErrTicketNotFound {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"assign_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ticket": ticket})
}

// Assignees — GET /api/admin/tickets/assignees (admin uniquement).
// Liste des admins vers qui un ticket peut être redirigé.
func (h *TicketHandler) Assignees(w http.ResponseWriter, r *http.Request) {
	isAdmin := middleware.GetIsAdmin(r.Context())
	if !isAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	admins, err := h.adminPermRepo.ListAdmins(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"admins": admins})
}
