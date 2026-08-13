package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPayoutNotFound = errors.New("payout not found")

type PayoutRepo struct {
	pool *pgxpool.Pool
}

func NewPayoutRepo(pool *pgxpool.Pool) *PayoutRepo {
	return &PayoutRepo{pool: pool}
}

const payoutColumns = `id, user_id, amount_cfa, status, phone_number, operator, pawapay_payout_id, failure_reason, requested_at, paid_at`

func scanPayout(row pgx.Row) (*model.Payout, error) {
	p := &model.Payout{}
	err := row.Scan(&p.ID, &p.UserID, &p.AmountCFA, &p.Status, &p.PhoneNumber, &p.Operator,
		&p.PawaPayPayoutID, &p.FailureReason, &p.RequestedAt, &p.PaidAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPayoutNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *PayoutRepo) Create(ctx context.Context, userID string, amount int, phone, operator string) (*model.Payout, error) {
	return scanPayout(r.pool.QueryRow(ctx,
		`INSERT INTO payouts (user_id, amount_cfa, phone_number, operator) VALUES ($1, $2, $3, $4) RETURNING `+payoutColumns,
		userID, amount, phone, operator))
}

func (r *PayoutRepo) ListByUser(ctx context.Context, userID string) ([]*model.Payout, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+payoutColumns+` FROM payouts WHERE user_id = $1 ORDER BY requested_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payouts := []*model.Payout{}
	for rows.Next() {
		p, err := scanPayout(rows)
		if err != nil {
			return nil, err
		}
		payouts = append(payouts, p)
	}
	return payouts, rows.Err()
}

func (r *PayoutRepo) FindByID(ctx context.Context, id string) (*model.Payout, error) {
	return scanPayout(r.pool.QueryRow(ctx,
		`SELECT `+payoutColumns+` FROM payouts WHERE id = $1`, id))
}

// PayoutWithUser — versement enrichi de l'email du demandeur, pour la vue admin globale.
type PayoutWithUser struct {
	model.Payout
	UserEmail string `json:"user_email"`
}

// ListAllAdmin — tous les versements, tous vendeurs/affiliés confondus (vue admin).
func (r *PayoutRepo) ListAllAdmin(ctx context.Context) ([]*PayoutWithUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.user_id, p.amount_cfa, p.status, p.phone_number, p.operator,
		        p.pawapay_payout_id, p.failure_reason, p.requested_at, p.paid_at, u.email
		 FROM payouts p JOIN users u ON u.id = p.user_id
		 ORDER BY p.requested_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*PayoutWithUser{}
	for rows.Next() {
		p := &PayoutWithUser{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.AmountCFA, &p.Status, &p.PhoneNumber, &p.Operator,
			&p.PawaPayPayoutID, &p.FailureReason, &p.RequestedAt, &p.PaidAt, &p.UserEmail); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PayoutRepo) FindByPawaPayID(ctx context.Context, pawapayPayoutID string) (*model.Payout, error) {
	return scanPayout(r.pool.QueryRow(ctx,
		`SELECT `+payoutColumns+` FROM payouts WHERE pawapay_payout_id = $1`, pawapayPayoutID))
}

// SetPawaPayReference — enregistre l'ID PawaPay juste après l'acceptation de la demande.
func (r *PayoutRepo) SetPawaPayReference(ctx context.Context, id, pawapayPayoutID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE payouts SET pawapay_payout_id = $2, status = 'processing' WHERE id = $1`,
		id, pawapayPayoutID)
	return err
}

// UpdateStatus — statut final ('paid' ou 'failed'), avec la raison d'échec le cas échéant.
func (r *PayoutRepo) UpdateStatus(ctx context.Context, id, status string, failureReason *string) error {
	if status == "paid" {
		_, err := r.pool.Exec(ctx,
			`UPDATE payouts SET status = $2, failure_reason = $3, paid_at = now() WHERE id = $1`,
			id, status, failureReason)
		return err
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE payouts SET status = $2, failure_reason = $3 WHERE id = $1`,
		id, status, failureReason)
	return err
}
