package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrSummitAlreadyRegistered — l'email est déjà inscrit (index unique sur
// lower(email), voir migration 033).
var ErrSummitAlreadyRegistered = errors.New("summit: email already registered")

type SummitRepo struct {
	pool *pgxpool.Pool
}

func NewSummitRepo(pool *pgxpool.Pool) *SummitRepo {
	return &SummitRepo{pool: pool}
}

// Create insère une inscription. Renvoie ErrSummitAlreadyRegistered si
// l'email (insensible à la casse) est déjà inscrit.
func (r *SummitRepo) Create(ctx context.Context, in model.CreateSummitRegistrationInput) (*model.SummitRegistration, error) {
	reg := &model.SummitRegistration{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO summit_registrations (full_name, email, phone_number, profile)
		VALUES ($1, $2, $3, $4)
		RETURNING id, full_name, email, phone_number, profile, created_at`,
		in.FullName, in.Email, in.Phone, in.Profile,
	).Scan(&reg.ID, &reg.FullName, &reg.Email, &reg.PhoneNumber, &reg.Profile, &reg.CreatedAt)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrSummitAlreadyRegistered
		}
		return nil, err
	}
	return reg, nil
}

// Count — nombre total d'inscrits (affiché côté admin / page publique).
func (r *SummitRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM summit_registrations`).Scan(&n)
	return n, err
}

// List — historique des inscriptions, plus récentes d'abord (admin).
func (r *SummitRepo) List(ctx context.Context) ([]*model.SummitRegistration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, full_name, email, phone_number, profile, created_at
		FROM summit_registrations
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.SummitRegistration
	for rows.Next() {
		reg := &model.SummitRegistration{}
		if err := rows.Scan(&reg.ID, &reg.FullName, &reg.Email, &reg.PhoneNumber, &reg.Profile, &reg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, reg)
	}
	return out, rows.Err()
}
