// Package cache fournit un accès Redis optionnel : si REDIS_URL n'est pas
// configuré, Client reste nil et toutes les méthodes deviennent des no-op
// (cache toujours "miss", jamais d'erreur) — même convention que le
// stockage S3/l'email/PawaPay dans cmd/server/main.go. Les appelants n'ont
// donc jamais besoin de vérifier "le cache est-il configuré ?" eux-mêmes.
package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

// New retourne un Client no-op si rawURL est vide — jamais d'erreur ici,
// pour que l'appelant (main.go) puisse toujours logger un WARNING et
// continuer plutôt que planter au démarrage pour un cache manquant.
func New(rawURL string) (*Client, error) {
	if rawURL == "" {
		return &Client{}, nil
	}
	opt, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	return &Client{rdb: redis.NewClient(opt)}, nil
}

func (c *Client) enabled() bool {
	return c != nil && c.rdb != nil
}

func (c *Client) Ping(ctx context.Context) error {
	if !c.enabled() {
		return nil
	}
	return c.rdb.Ping(ctx).Err()
}

// GetJSON désérialise dans dest ; retourne (false, nil) sur cache miss (clé
// absente ou cache désactivé) — dest n'est alors pas modifié, l'appelant
// continue vers la source de vérité normale (base de données).
func (c *Client) GetJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	if !c.enabled() {
		return false, nil
	}
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Client) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !c.enabled() {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, raw, ttl).Err()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if !c.enabled() || len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

// IncrWithExpire incrémente un compteur et fixe son expiration au premier
// incrément seulement (NX : n'écrase pas un TTL déjà en cours) — pattern
// standard de fenêtre glissante fixe pour le rate limiting. Retourne la
// valeur du compteur après incrément ; sur cache désactivé, retourne 0
// (l'appelant doit alors laisser passer la requête, voir middleware).
func (c *Client) IncrWithExpire(ctx context.Context, key string, window time.Duration) (int64, error) {
	if !c.enabled() {
		return 0, nil
	}
	count, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		c.rdb.Expire(ctx, key, window)
	}
	return count, nil
}
