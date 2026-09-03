package repository

import (
	"context"
	"errors"
	"time"

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

const payoutColumns = `id, user_id, amount_cfa, status, phone_number, operator, provider, provider_reference, failure_reason, requested_at, paid_at, is_manual, manual_note, fee_cfa, paypal_email, paypal_batch_id`

func scanPayout(row pgx.Row) (*model.Payout, error) {
	p := &model.Payout{}
	err := row.Scan(&p.ID, &p.UserID, &p.AmountCFA, &p.Status, &p.PhoneNumber, &p.Operator,
		&p.Provider, &p.ProviderReference, &p.FailureReason, &p.RequestedAt, &p.PaidAt,
		&p.IsManual, &p.ManualNote, &p.FeeCFA, &p.PayPalEmail, &p.PayPalBatchID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPayoutNotFound
		}
		return nil, err
	}
	return p, nil
}

// Create — versement MOBILE MONEY. provider est résolu par l'appelant depuis
// GatewayOperatorSettingKey AU MOMENT de la création (pas relu à chaque
// tentative/webhook ensuite), pour qu'un changement de réglage admin ne
// redirige jamais un versement déjà en cours vers un autre prestataire.
func (r *PayoutRepo) Create(ctx context.Context, userID string, amount int, phone, operator, provider string) (*model.Payout, error) {
	return scanPayout(r.pool.QueryRow(ctx,
		`INSERT INTO payouts (user_id, amount_cfa, phone_number, operator, provider) VALUES ($1, $2, $3, $4, $5) RETURNING `+payoutColumns,
		userID, amount, phone, operator, provider))
}

// CreatePayPal — versement vers un email PayPal (provider = "paypal", canal
// prioritaire quand le vendeur a renseigné un email PayPal). L'email est figé
// ici, jamais relu après (même principe que phone_number pour le mobile money).
func (r *PayoutRepo) CreatePayPal(ctx context.Context, userID string, amount int, paypalEmail string) (*model.Payout, error) {
	return scanPayout(r.pool.QueryRow(ctx,
		`INSERT INTO payouts (user_id, amount_cfa, provider, paypal_email) VALUES ($1, $2, 'paypal', $3) RETURNING `+payoutColumns,
		userID, amount, paypalEmail))
}

// SetPayPalBatchID — enregistre le payout_batch_id PayPal juste après
// l'acceptation du lot, et passe le versement à "processing" (miroir de
// SetProviderReference pour le mobile money).
func (r *PayoutRepo) SetPayPalBatchID(ctx context.Context, id, batchID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE payouts SET paypal_batch_id = $2, provider_reference = $2, status = 'processing' WHERE id = $1`,
		id, batchID)
	return err
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

// PayoutWithUser — versement enrichi de l'email du demandeur et du moyen de
// versement enregistré par le vendeur (users.payout_*), pour la vue admin
// globale. Les champs VendorPayout* servent de repli quand le versement
// lui-même n'a pas d'opérateur/numéro (versement manuel créé de toutes pièces)
// et à afficher « où envoyer l'argent » avant un règlement manuel.
type PayoutWithUser struct {
	model.Payout
	UserEmail           string  `json:"user_email"`
	VendorPayoutPhone   *string `json:"vendor_payout_phone,omitempty"`
	VendorPayoutOperator *string `json:"vendor_payout_operator,omitempty"`
	VendorPayoutCountry *string `json:"vendor_payout_country,omitempty"`
	VendorPayoutPayPalEmail *string `json:"vendor_payout_paypal_email,omitempty"`
}

// ListAllAdmin — tous les versements, tous vendeurs/affiliés confondus (vue admin).
func (r *PayoutRepo) ListAllAdmin(ctx context.Context) ([]*PayoutWithUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.user_id, p.amount_cfa, p.status, p.phone_number, p.operator,
		        p.provider, p.provider_reference, p.failure_reason, p.requested_at, p.paid_at,
		        p.is_manual, p.manual_note, p.fee_cfa, p.paypal_email, p.paypal_batch_id,
		        u.email, u.payout_phone, u.payout_operator, u.payout_country, u.payout_paypal_email
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
			&p.Provider, &p.ProviderReference, &p.FailureReason, &p.RequestedAt, &p.PaidAt,
			&p.IsManual, &p.ManualNote, &p.FeeCFA, &p.PayPalEmail, &p.PayPalBatchID,
			&p.UserEmail, &p.VendorPayoutPhone, &p.VendorPayoutOperator, &p.VendorPayoutCountry, &p.VendorPayoutPayPalEmail); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// FindByProviderReference — clé composite (provider, reference) : les deux
// prestataires génèrent leurs propres identifiants dans des espaces
// indépendants, donc la référence brute seule ne suffit pas à désambiguïser.
func (r *PayoutRepo) FindByProviderReference(ctx context.Context, provider, reference string) (*model.Payout, error) {
	return scanPayout(r.pool.QueryRow(ctx,
		`SELECT `+payoutColumns+` FROM payouts WHERE provider = $1 AND provider_reference = $2`, provider, reference))
}

// ListProcessing — versements encore "processing" (envoyés à un prestataire
// mais pas confirmés), plus anciens d'abord, dans la fenêtre maxAge. Utilisé
// par le job de réconciliation de fond (filet de sécurité si un webhook
// prestataire s'est perdu).
func (r *PayoutRepo) ListProcessing(ctx context.Context, maxAge time.Duration) ([]*model.Payout, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+payoutColumns+`
		 FROM payouts
		 WHERE status = 'processing' AND provider_reference IS NOT NULL
		   AND requested_at >= now() - $1::interval
		 ORDER BY requested_at ASC
		 LIMIT 200`, intervalStr(maxAge))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.Payout{}
	for rows.Next() {
		p, err := scanPayout(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetProviderReference — enregistre l'ID du prestataire juste après l'acceptation de la demande.
func (r *PayoutRepo) SetProviderReference(ctx context.Context, id, reference string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE payouts SET provider_reference = $2, status = 'processing' WHERE id = $1`,
		id, reference)
	return err
}

// SettleManually — marque un versement "paid" SANS appel prestataire : l'argent
// a été envoyé au vendeur hors PawaPay/KPay (Wave perso, espèces, virement).
// note = référence/commentaire libre, feeCFA = frais/taxe éventuellement retenus
// (0 si aucun), adminID = l'admin qui a effectué le règlement. N'agit que sur un
// versement non encore réglé (requested/processing/failed).
func (r *PayoutRepo) SettleManually(ctx context.Context, id, note string, feeCFA int, adminID string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE payouts
		 SET status = 'paid', paid_at = now(), is_manual = TRUE,
		     manual_note = $2, fee_cfa = $3, settled_by = $4, failure_reason = NULL
		 WHERE id = $1 AND status IN ('requested', 'processing', 'failed')`,
		id, note, feeCFA, adminID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// CreateManual — crée un versement déjà réglé à la main, de toutes pièces, pour
// un vendeur (cas où aucune demande n'existe côté vendeur mais l'admin veut
// tracer un paiement fait hors plateforme). provider = "manual".
func (r *PayoutRepo) CreateManual(ctx context.Context, userID string, amount, feeCFA int, phone, note, adminID string) (*model.Payout, error) {
	return scanPayout(r.pool.QueryRow(ctx,
		`INSERT INTO payouts (user_id, amount_cfa, phone_number, operator, provider, status,
		     paid_at, is_manual, manual_note, fee_cfa, settled_by)
		 VALUES ($1, $2, $3, '', 'manual', 'paid', now(), TRUE, $4, $5, $6)
		 RETURNING `+payoutColumns,
		userID, amount, phone, note, feeCFA, adminID))
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
