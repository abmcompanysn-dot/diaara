package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDonationRecipientNotFound = errors.New("donation recipient not found")
	ErrDonationPayoutNotFound    = errors.New("donation payout not found")
)

type DonationRepo struct {
	pool *pgxpool.Pool
}

func NewDonationRepo(pool *pgxpool.Pool) *DonationRepo {
	return &DonationRepo{pool: pool}
}

// Pool retourne l'état actuel de la cagnotte (ligne singleton id=1).
func (r *DonationRepo) Pool(ctx context.Context) (*model.DonationPool, error) {
	p := &model.DonationPool{}
	err := r.pool.QueryRow(ctx, `SELECT balance_cfa, updated_at FROM donation_pool WHERE id = 1`).
		Scan(&p.BalanceCFA, &p.UpdatedAt)
	return p, err
}

// Accumulate ajoute amountCFA à la cagnotte de façon atomique (UPDATE ...
// RETURNING) — safe même avec plusieurs replicas du backend traitant des
// webhooks de vente en parallèle.
func (r *DonationRepo) Accumulate(ctx context.Context, amountCFA int) (int, error) {
	var newBalance int
	err := r.pool.QueryRow(ctx,
		`UPDATE donation_pool SET balance_cfa = balance_cfa + $1, updated_at = now() WHERE id = 1 RETURNING balance_cfa`,
		amountCFA,
	).Scan(&newBalance)
	return newBalance, err
}

// DrainAndDistribute soustrait le montant distribué de la cagnotte et crée
// les lignes de versement en une seule transaction — le solde ne doit
// jamais être décrémenté sans que les DonationPayout correspondants
// existent (et inversement).
func (r *DonationRepo) DrainAndDistribute(ctx context.Context, totalAmountCFA int, shares map[string]int) ([]*model.DonationPayout, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE donation_pool SET balance_cfa = balance_cfa - $1, updated_at = now() WHERE id = 1`,
		totalAmountCFA,
	); err != nil {
		return nil, err
	}

	payouts := make([]*model.DonationPayout, 0, len(shares))
	for recipientID, amount := range shares {
		p := &model.DonationPayout{}
		err := tx.QueryRow(ctx,
			`INSERT INTO donation_payouts (recipient_id, amount_cfa) VALUES ($1, $2)
			 RETURNING id, recipient_id, amount_cfa, status, pawapay_payout_id, failure_reason, requested_at, paid_at`,
			recipientID, amount,
		).Scan(&p.ID, &p.RecipientID, &p.AmountCFA, &p.Status, &p.PawaPayPayoutID, &p.FailureReason, &p.RequestedAt, &p.PaidAt)
		if err != nil {
			return nil, err
		}
		payouts = append(payouts, p)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return payouts, nil
}

func (r *DonationRepo) SetPawaPayReference(ctx context.Context, id, pawapayPayoutID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE donation_payouts SET pawapay_payout_id = $2, status = 'processing' WHERE id = $1`,
		id, pawapayPayoutID)
	return err
}

func (r *DonationRepo) UpdatePayoutStatus(ctx context.Context, id, status string, failureReason *string) error {
	if status == "paid" {
		_, err := r.pool.Exec(ctx,
			`UPDATE donation_payouts SET status = $2, failure_reason = $3, paid_at = now() WHERE id = $1`,
			id, status, failureReason)
		return err
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE donation_payouts SET status = $2, failure_reason = $3 WHERE id = $1`,
		id, status, failureReason)
	return err
}

func (r *DonationRepo) FindPayoutByID(ctx context.Context, id string) (*model.DonationPayout, error) {
	p := &model.DonationPayout{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, recipient_id, amount_cfa, status, pawapay_payout_id, failure_reason, requested_at, paid_at
		 FROM donation_payouts WHERE id = $1`, id,
	).Scan(&p.ID, &p.RecipientID, &p.AmountCFA, &p.Status, &p.PawaPayPayoutID, &p.FailureReason, &p.RequestedAt, &p.PaidAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrDonationPayoutNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *DonationRepo) FindPayoutByPawaPayID(ctx context.Context, pawapayPayoutID string) (*model.DonationPayout, error) {
	p := &model.DonationPayout{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, recipient_id, amount_cfa, status, pawapay_payout_id, failure_reason, requested_at, paid_at
		 FROM donation_payouts WHERE pawapay_payout_id = $1`, pawapayPayoutID,
	).Scan(&p.ID, &p.RecipientID, &p.AmountCFA, &p.Status, &p.PawaPayPayoutID, &p.FailureReason, &p.RequestedAt, &p.PaidAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrDonationPayoutNotFound
		}
		return nil, err
	}
	return p, nil
}

// ListPayouts — historique complet (tous destinataires confondus), pour l'admin.
func (r *DonationRepo) ListPayouts(ctx context.Context) ([]*model.DonationPayoutWithRecipient, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.recipient_id, p.amount_cfa, p.status, p.pawapay_payout_id, p.failure_reason,
		        p.requested_at, p.paid_at, r.name, r.phone_number
		 FROM donation_payouts p JOIN donation_recipients r ON r.id = p.recipient_id
		 ORDER BY p.requested_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*model.DonationPayoutWithRecipient{}
	for rows.Next() {
		p := &model.DonationPayoutWithRecipient{}
		if err := rows.Scan(&p.ID, &p.RecipientID, &p.AmountCFA, &p.Status, &p.PawaPayPayoutID, &p.FailureReason,
			&p.RequestedAt, &p.PaidAt, &p.RecipientName, &p.RecipientPhone); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// --- Destinataires ---

func scanDonationRecipient(row pgx.Row) (*model.DonationRecipient, error) {
	rec := &model.DonationRecipient{}
	err := row.Scan(&rec.ID, &rec.Name, &rec.PhoneNumber, &rec.Operator, &rec.Country, &rec.Active, &rec.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrDonationRecipientNotFound
		}
		return nil, err
	}
	return rec, nil
}

const donationRecipientColumns = `id, name, phone_number, operator, country, active, created_at`

func (r *DonationRepo) CreateRecipient(ctx context.Context, name, phone, operator, country string) (*model.DonationRecipient, error) {
	return scanDonationRecipient(r.pool.QueryRow(ctx,
		`INSERT INTO donation_recipients (name, phone_number, operator, country) VALUES ($1, $2, $3, $4)
		 RETURNING `+donationRecipientColumns,
		name, phone, operator, country))
}

func (r *DonationRepo) ListRecipients(ctx context.Context) ([]*model.DonationRecipient, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+donationRecipientColumns+` FROM donation_recipients ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*model.DonationRecipient{}
	for rows.Next() {
		rec, err := scanDonationRecipient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *DonationRepo) ListActiveRecipients(ctx context.Context) ([]*model.DonationRecipient, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+donationRecipientColumns+` FROM donation_recipients WHERE active = true ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*model.DonationRecipient{}
	for rows.Next() {
		rec, err := scanDonationRecipient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *DonationRepo) UpdateRecipient(ctx context.Context, id string, input model.UpdateDonationRecipientInput) (*model.DonationRecipient, error) {
	query := `UPDATE donation_recipients SET `
	args := []interface{}{}
	sets := []string{}
	argIdx := 1

	if input.Name != nil {
		sets = append(sets, `name = $`+itoa(argIdx))
		args = append(args, *input.Name)
		argIdx++
	}
	if input.Active != nil {
		sets = append(sets, `active = $`+itoa(argIdx))
		args = append(args, *input.Active)
		argIdx++
	}
	if len(sets) == 0 {
		return r.FindRecipientByID(ctx, id)
	}

	query += join(sets, ", ") + ` WHERE id = $` + itoa(argIdx)
	args = append(args, id)

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return nil, err
	}
	return r.FindRecipientByID(ctx, id)
}

func (r *DonationRepo) FindRecipientByID(ctx context.Context, id string) (*model.DonationRecipient, error) {
	return scanDonationRecipient(r.pool.QueryRow(ctx,
		`SELECT `+donationRecipientColumns+` FROM donation_recipients WHERE id = $1`, id))
}

func (r *DonationRepo) DeleteRecipient(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM donation_recipients WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDonationRecipientNotFound
	}
	return nil
}
