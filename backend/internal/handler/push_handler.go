package handler

import (
	"encoding/json"
	"net/http"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
)

// PushHandler — abonnement/désabonnement aux notifications push navigateur.
// L'envoi lui-même est géré par service.PushService (voir
// WebhookHandler.notify, branché sur le même point que les notifications
// in-app).
type PushHandler struct {
	repo *repository.PushRepo
}

func NewPushHandler(repo *repository.PushRepo) *PushHandler {
	return &PushHandler{repo: repo}
}

// Subscribe — POST /api/push/subscribe (utilisateur connecté). Corps :
// PushSubscription.toJSON() tel que renvoyé par le navigateur (voir
// frontend lib/push.ts).
func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var input model.CreatePushSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Endpoint == "" || input.Keys.P256dh == "" || input.Keys.Auth == "" {
		http.Error(w, `{"error":"invalid_subscription"}`, http.StatusBadRequest)
		return
	}
	if err := h.repo.Upsert(r.Context(), userID, input); err != nil {
		http.Error(w, `{"error":"subscribe_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "subscribed"})
}

// Unsubscribe — POST /api/push/unsubscribe. Corps : {"endpoint": "..."}.
func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Endpoint == "" {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), input.Endpoint); err != nil {
		http.Error(w, `{"error":"unsubscribe_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unsubscribed"})
}
