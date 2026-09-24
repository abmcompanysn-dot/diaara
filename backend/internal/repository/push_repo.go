package repository

import (
	"context"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PushRepo struct {
	pool *pgxpool.Pool
}

func NewPushRepo(pool *pgxpool.Pool) *PushRepo {
	return &PushRepo{pool: pool}
}

// Upsert — un même endpoint (navigateur+appareil précis) peut être renvoyé
// plusieurs fois (ex. l'utilisateur revisite la page après avoir déjà
// autorisé) : ON CONFLICT évite les doublons plutôt qu'une contrainte
// UNIQUE qui ferait échouer l'appel silencieusement.
func (r *PushRepo) Upsert(ctx context.Context, userID string, input model.CreatePushSubscriptionInput) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (endpoint) DO UPDATE SET user_id = $1, p256dh = $3, auth = $4`,
		userID, input.Endpoint, input.Keys.P256dh, input.Keys.Auth)
	return err
}

func (r *PushRepo) Delete(ctx context.Context, endpoint string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1`, endpoint)
	return err
}

// ListByUser — tous les abonnements actifs d'un utilisateur (potentiellement
// plusieurs appareils).
func (r *PushRepo) ListByUser(ctx context.Context, userID string) ([]*model.PushSubscription, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, endpoint, p256dh, auth, created_at FROM push_subscriptions WHERE user_id = $1`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*model.PushSubscription{}
	for rows.Next() {
		s := &model.PushSubscription{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
