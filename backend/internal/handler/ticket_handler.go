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
	ticketRepo *repository.TicketRepo
}

func NewTicketHandler(ticketRepo *repository.TicketRepo) *TicketHandler {
	return &TicketHandler{ticketRepo: ticketRepo}
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
