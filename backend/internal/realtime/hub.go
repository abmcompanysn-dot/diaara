package realtime

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"
)

// SalesChannel et ModerationChannel sont les canaux NOTIFY de PostgreSQL.
const (
	SalesChannel     = "sales_channel"
	ModerationChannel = "moderation_channel"
)

// Hub maintient les connexions WebSocket par canal et relaie les notifications.
type Hub struct {
	mu       sync.RWMutex
	pool     *pgxpool.Pool
	clients  map[string]map[*Client]struct{} // channel -> set de clients
}

type Client struct {
	Channel string
	Send    chan []byte
}

func NewHub(pool *pgxpool.Pool) *Hub {
	h := &Hub{
		pool:    pool,
		clients: make(map[string]map[*Client]struct{}),
	}
	go h.listen()
	return h
}

func (h *Hub) Subscribe(channel string) (*Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	c := &Client{
		Channel: channel,
		Send:    make(chan []byte, 16),
	}
	if h.clients[channel] == nil {
		h.clients[channel] = make(map[*Client]struct{})
	}
	h.clients[channel][c] = struct{}{}
	return c, nil
}

func (h *Hub) Unsubscribe(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if set, ok := h.clients[c.Channel]; ok {
		delete(set, c)
	}
	close(c.Send)
}

func (h *Hub) broadcast(channel string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients[channel] {
		select {
		case c.Send <- payload:
		default:
			// client lent : on le déconnecte
			log.Printf("realtime: client %s lent sur %s", c.Channel, channel)
		}
	}
}

func (h *Hub) listen() {
	conn, err := h.pool.Acquire(context.Background())
	if err != nil {
		log.Printf("realtime: impossible d'acquérir la connexion LISTEN: %v", err)
		return
	}
	defer conn.Release()

	_, err = conn.Exec(context.Background(), "LISTEN "+SalesChannel)
	if err != nil {
		log.Printf("realtime: LISTEN sales_channel échoué: %v", err)
		return
	}
	_, err = conn.Exec(context.Background(), "LISTEN "+ModerationChannel)
	if err != nil {
		log.Printf("realtime: LISTEN moderation_channel échoué: %v", err)
		return
	}

	log.Println("realtime: écoute active sur sales_channel et moderation_channel")

	for {
		notification, err := conn.Conn().WaitForNotification(context.Background())
		if err != nil {
			log.Printf("realtime: erreur WaitForNotification: %v", err)
			return
		}
		h.dispatch(notification)
	}
}

func (h *Hub) dispatch(n *pgconn.Notification) {
	// On route chaque notification vers le bon canal WebSocket.
	channel := ""
	switch n.Channel {
	case SalesChannel:
		channel = "order"
	case ModerationChannel:
		channel = "moderation"
	default:
		return
	}

	var payload struct {
		Type    string `json:"type"`
		Channel string `json:"channel"`
		Data    any    `json:"data"`
	}
	payload.Type = "db_notify"
	payload.Channel = channel

	// On essaie de décoder le JSON du payload SQL (row_to_json) pour le renvoyer proprement.
	var data any
	if err := json.Unmarshal([]byte(n.Payload), &data); err == nil {
		payload.Data = data
	} else {
		payload.Data = n.Payload
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.broadcast(channel, raw)
}
