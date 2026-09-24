-- Migration 042 : notifications push navigateur (Web Push + VAPID).
-- Une ligne par abonnement (endpoint) — un même utilisateur peut avoir
-- plusieurs appareils/navigateurs abonnés simultanément.
CREATE TABLE push_subscriptions (
	id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	endpoint   TEXT NOT NULL UNIQUE,
	-- p256dh/auth : clés publiques du navigateur pour chiffrer le payload
	-- (spec Web Push, RFC 8291) — fournies telles quelles par
	-- PushSubscription.toJSON() côté client, jamais recalculées côté serveur.
	p256dh     TEXT NOT NULL,
	auth       TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_push_subscriptions_user ON push_subscriptions (user_id);
