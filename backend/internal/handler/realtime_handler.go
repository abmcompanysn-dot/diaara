package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/realtime"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Le Worker Cloudflare proxy la connexion ; on accepte en dev
	},
}

type RealtimeHandler struct {
	hub *realtime.Hub
}

func NewRealtimeHandler(hub *realtime.Hub) *RealtimeHandler {
	return &RealtimeHandler{hub: hub}
}

// OrderWS expose /ws/order/{id} — l'utilisateur reçoit les changements de statut de sa commande.
func (h *RealtimeHandler) OrderWS(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		http.Error(w, `{"error":"order_id_required"}`, http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// S'abonner au canal "order" et filtrer par id de commande côté client.
	client, err := h.hub.Subscribe(realtime.SalesChannel)
	if err != nil {
		return
	}
	defer h.hub.Unsubscribe(client)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Goroutine d'écriture : envoie les notifications du hub vers la socket.
	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		for {
			select {
			case msg, ok := <-client.Send:
				if !ok {
					return
				}
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Goroutine de lecture : maintient la connexion ouverte, ignore les messages entrants.
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	select {
	case <-writeDone:
	case <-readDone:
	case <-r.Context().Done():
	}
}

// ModerationWS expose /ws/admin — l'admin reçoit les changements de modération en direct.
func (h *RealtimeHandler) ModerationWS(w http.ResponseWriter, r *http.Request) {
	isAdmin := middleware.GetIsAdmin(r.Context())
	if !isAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	client, err := h.hub.Subscribe(realtime.ModerationChannel)
	if err != nil {
		return
	}
	defer h.hub.Unsubscribe(client)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		for {
			select {
			case msg, ok := <-client.Send:
				if !ok {
					return
				}
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	select {
	case <-writeDone:
	case <-readDone:
	case <-r.Context().Done():
	}
}
