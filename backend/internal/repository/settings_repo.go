package repository

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsRepo struct {
	pool *pgxpool.Pool
}

func NewSettingsRepo(pool *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{pool: pool}
}

// All retourne tous les réglages sous forme de map.
func (r *SettingsRepo) All(ctx context.Context) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// Get retourne un réglage, ou fallback si la clé n'existe pas.
func (r *SettingsRepo) Get(ctx context.Context, key, fallback string) string {
	var v string
	err := r.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&v)
	if err != nil {
		return fallback
	}
	return v
}

// GetBool lit un réglage booléen ("true"/"false"), fallback si absent ou invalide.
func (r *SettingsRepo) GetBool(ctx context.Context, key string, fallback bool) bool {
	v := r.Get(ctx, key, "")
	if v == "" {
		return fallback
	}
	return v == "true"
}

// GetFloat lit un réglage numérique, fallback si absent ou invalide.
func (r *SettingsRepo) GetFloat(ctx context.Context, key string, fallback float64) float64 {
	v := r.Get(ctx, key, "")
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

// Set upsert une valeur (crée ou met à jour).
func (r *SettingsRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, now())
		 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = now()`,
		key, value)
	return err
}

// SetMany upsert plusieurs clés en une fois.
func (r *SettingsRepo) SetMany(ctx context.Context, values map[string]string) error {
	for k, v := range values {
		if err := r.Set(ctx, k, v); err != nil {
			return err
		}
	}
	return nil
}
