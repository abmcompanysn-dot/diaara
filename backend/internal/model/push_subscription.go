package model

import "time"

// PushSubscription — abonnement Web Push d'un utilisateur (voir
// migrations/042_push_subscriptions.sql). Un utilisateur peut avoir
// plusieurs abonnements (un par appareil/navigateur).
type PushSubscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"-"` // jamais renvoyé au client, juste utilisé serveur -> navigateur push
	Auth      string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// CreatePushSubscriptionInput — corps envoyé par le frontend au moment de
// PushSubscription.toJSON() côté navigateur (voir lib/push.ts).
type CreatePushSubscriptionInput struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}
