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

const payoutColumns = `id, user_id, amount_cfa, status, requested_at, paid_at`

func scanPayout(row pgx.Row) (*model.Payout, error) {
	p := &model.Payout{}
	err := row.Scan(&p.ID, &p.UserID, &p.AmountCFA, &p.Status, &p.RequestedAt, &p.PaidAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPayoutNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *PayoutRepo) Create(ctx context.Context, userID string, amount int) (*model.Payout, error) {
	return scanPayout(r.pool.QueryRow(ctx,
		`INSERT INTO payouts (user_id, amount_cfa) VALUES ($1, $2) RETURNING `+payoutColumns,
		userID, amount))
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
