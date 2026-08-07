package repository

import (
	"context"
	"time"

	"github.com/diarra/backend/internal/otp"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OTPRepo struct {
	pool *pgxpool.Pool
}

func NewOTPRepo(pool *pgxpool.Pool) *OTPRepo {
	return &OTPRepo{pool: pool}
}

// CreateCode insère un nouveau code OTP (hash du code stocké, jamais le code en clair).
// On expire préventivement l'ancien code non utilisé du même (user, channel,
// purpose) pour garantir qu'un seul code est valide à la fois.
func (r *OTPRepo) CreateCode(ctx context.Context, userID, channel, purpose, codeHash string, expiresAt time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Invalide l'ancien code actif du même usage (s'il existe).
	_, err = tx.Exec(ctx, `
		UPDATE otp_codes SET used_at = now()
		WHERE user_id = $1 AND channel = $2 AND purpose = $3
		  AND used_at IS NULL AND expires_at > now()`,
		userID, channel, purpose)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO otp_codes (user_id, code_hash, channel, purpose, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		userID, codeHash, channel, purpose, expiresAt)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// LatestCode retourne le code actif le plus récent pour (user, channel, purpose).
// Un code actif = non utilisé et non expiré.
func (r *OTPRepo) LatestCode(ctx context.Context, userID, channel, purpose string) (*otp.OTPCode, error) {
	var c otp.OTPCode
	err := r.pool.QueryRow(ctx, `
		SELECT id, code_hash, attempts
		FROM otp_codes
		WHERE user_id = $1 AND channel = $2 AND purpose = $3
		  AND used_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC
		LIMIT 1`,
		userID, channel, purpose).Scan(&c.ID, &c.CodeHash, &c.Attempts)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// IncAttempts incrémente le compteur de tentatives et renvoie sa nouvelle valeur.
func (r *OTPRepo) IncAttempts(ctx context.Context, id string) (int, error) {
	var attempts int
	err := r.pool.QueryRow(ctx, `
		UPDATE otp_codes SET attempts = attempts + 1
		WHERE id = $1 RETURNING attempts`,
		id).Scan(&attempts)
	return attempts, err
}

// MarkUsed marque un code comme consommé.
func (r *OTPRepo) MarkUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE otp_codes SET used_at = now() WHERE id = $1`, id)
	return err
}

// LastIssuedAt renvoie la date du dernier code émis pour (user, channel, purpose),
// afin de calculer le cooldown de resend (anti-spam). Renvoie le zero time si aucun.
func (r *OTPRepo) LastIssuedAt(ctx context.Context, userID, channel, purpose string) (time.Time, error) {
	var t time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT created_at FROM otp_codes
		WHERE user_id = $1 AND channel = $2 AND purpose = $3
		ORDER BY created_at DESC LIMIT 1`,
		userID, channel, purpose).Scan(&t)
	if err != nil {
		return time.Time{}, nil // aucune ligne = pas encore de code (tolérant)
	}
	return t, nil
}
