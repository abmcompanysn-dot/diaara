package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
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

// Create insère une inscription (historique : utilisé avant que
// l'inscription devienne payante). Renvoie ErrSummitAlreadyRegistered si
// l'email (insensible à la casse) est déjà inscrit.
func (r *SummitRepo) Create(ctx context.Context, in model.CreateSummitRegistrationInput) (*model.SummitRegistration, error) {
	reg := &model.SummitRegistration{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO summit_registrations (full_name, email, phone_number, profile)
		VALUES ($1, $2, $3, $4)
		RETURNING id, full_name, email, phone_number, profile, status, created_at`,
		in.FullName, in.Email, in.Phone, in.Profile,
	).Scan(&reg.ID, &reg.FullName, &reg.Email, &reg.PhoneNumber, &reg.Profile, &reg.Status, &reg.CreatedAt)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrSummitAlreadyRegistered
		}
		return nil, err
	}
	return reg, nil
}

// CreatePending insère une inscription "pending", liée au checkout_token
// de la vente du Tier Essentiel qui vient d'être créée (voir
// SaleHandler.Create). Pas d'email de confirmation à ce stade — elle part
// seulement une fois le paiement confirmé (voir ConfirmByCheckoutToken).
// Renvoie ErrSummitAlreadyRegistered si l'email est déjà inscrit — l'appelant
// doit alors laisser la vente du billet se poursuivre normalement (l'achat
// reste valide même si cette personne a déjà une inscription Summit).
func (r *SummitRepo) CreatePending(ctx context.Context, fullName, email, phone, profile, checkoutToken string) (*model.SummitRegistration, error) {
	reg := &model.SummitRegistration{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO summit_registrations (full_name, email, phone_number, profile, status, sale_checkout_token)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id, full_name, email, phone_number, profile, status, created_at`,
		fullName, email, phone, profile, checkoutToken,
	).Scan(&reg.ID, &reg.FullName, &reg.Email, &reg.PhoneNumber, &reg.Profile, &reg.Status, &reg.CreatedAt)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrSummitAlreadyRegistered
		}
		return nil, err
	}
	return reg, nil
}

// ConfirmByCheckoutToken passe une inscription "pending" à "confirmed" une
// fois le paiement du billet validé. Renvoie (nil, nil) si aucune
// inscription Summit n'est liée à ce jeton (vente normale, sans rapport
// avec le Summit) — pas une erreur.
func (r *SummitRepo) ConfirmByCheckoutToken(ctx context.Context, checkoutToken string) (*model.SummitRegistration, error) {
	reg := &model.SummitRegistration{}
	err := r.pool.QueryRow(ctx, `
		UPDATE summit_registrations SET status = 'confirmed'
		WHERE sale_checkout_token = $1 AND status = 'pending'
		RETURNING id, full_name, email, phone_number, profile, status, created_at`,
		checkoutToken,
	).Scan(&reg.ID, &reg.FullName, &reg.Email, &reg.PhoneNumber, &reg.Profile, &reg.Status, &reg.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return reg, nil
}

// Count — nombre d'inscrits confirmés (payés), affiché côté admin / page
// publique — une inscription "pending" jamais payée ne doit pas gonfler ce
// chiffre.
func (r *SummitRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM summit_registrations WHERE status = 'confirmed'`).Scan(&n)
	return n, err
}

// List — historique des inscriptions, plus récentes d'abord (admin).
func (r *SummitRepo) List(ctx context.Context) ([]*model.SummitRegistration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, full_name, email, phone_number, profile, status, created_at
		FROM summit_registrations
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.SummitRegistration
	for rows.Next() {
		reg := &model.SummitRegistration{}
		if err := rows.Scan(&reg.ID, &reg.FullName, &reg.Email, &reg.PhoneNumber, &reg.Profile, &reg.Status, &reg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, reg)
	}
	return out, rows.Err()
}
