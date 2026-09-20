package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEventTicketCheckinNotFound = errors.New("event ticket checkin not found")

type EventTicketRepo struct {
	pool *pgxpool.Pool
}

func NewEventTicketRepo(pool *pgxpool.Pool) *EventTicketRepo {
	return &EventTicketRepo{pool: pool}
}

const eventTicketCheckinColumns = `id, sale_id, check_in_token, checked_in_at, checked_in_by, created_at`

func scanEventTicketCheckin(row pgx.Row) (*model.EventTicketCheckin, error) {
	c := &model.EventTicketCheckin{}
	err := row.Scan(&c.ID, &c.SaleID, &c.CheckInToken, &c.CheckedInAt, &c.CheckedInBy, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrEventTicketCheckinNotFound
		}
		return nil, err
	}
	return c, nil
}

// FindOrCreateBySale renvoie le check-in existant pour cette vente, ou en
// crée un nouveau (avec un token frais) s'il n'existe pas encore. Idempotent
// par construction (contrainte UNIQUE sur sale_id) : appelé à chaque
// génération du PDF, jamais au moment du paiement.
func (r *EventTicketRepo) FindOrCreateBySale(ctx context.Context, saleID, token string) (*model.EventTicketCheckin, error) {
	existing, err := scanEventTicketCheckin(r.pool.QueryRow(ctx,
		`SELECT `+eventTicketCheckinColumns+` FROM event_ticket_checkins WHERE sale_id = $1`, saleID))
	if err == nil {
		return existing, nil
	}
	if err != ErrEventTicketCheckinNotFound {
		return nil, err
	}

	row := r.pool.QueryRow(ctx,
		`INSERT INTO event_ticket_checkins (sale_id, check_in_token) VALUES ($1, $2)
		 ON CONFLICT (sale_id) DO UPDATE SET sale_id = EXCLUDED.sale_id
		 RETURNING `+eventTicketCheckinColumns,
		saleID, token,
	)
	// ON CONFLICT DO UPDATE (no-op) plutôt que DO NOTHING : garantit une ligne
	// en retour même si un appel concurrent a créé la ligne entre le SELECT
	// et l'INSERT ci-dessus (rare mais possible, deux onglets ouvrant le
	// PDF en même temps).
	return scanEventTicketCheckin(row)
}

func (r *EventTicketRepo) FindByToken(ctx context.Context, token string) (*model.EventTicketCheckin, error) {
	return scanEventTicketCheckin(r.pool.QueryRow(ctx,
		`SELECT `+eventTicketCheckinColumns+` FROM event_ticket_checkins WHERE check_in_token = $1`, token))
}

// CheckIn marque le billet comme utilisé. Renvoie déjàUtilisé=true sans
// erreur si un check-in existait déjà (première utilisation gagne — la
// page de scan affiche alors "déjà scanné" plutôt qu'une erreur opaque).
func (r *EventTicketRepo) CheckIn(ctx context.Context, token, staffUserID string) (checkin *model.EventTicketCheckin, alreadyUsed bool, err error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE event_ticket_checkins
		SET checked_in_at = now(), checked_in_by = $2
		WHERE check_in_token = $1 AND checked_in_at IS NULL
		RETURNING `+eventTicketCheckinColumns,
		token, staffUserID,
	)
	checkin, err = scanEventTicketCheckin(row)
	if err == nil {
		return checkin, false, nil
	}
	if err != ErrEventTicketCheckinNotFound {
		return nil, false, err
	}
	// Soit le token n'existe pas, soit il est déjà utilisé — on distingue
	// les deux pour ne pas dire "déjà scanné" à un token invalide.
	existing, findErr := r.FindByToken(ctx, token)
	if findErr != nil {
		return nil, false, ErrEventTicketCheckinNotFound
	}
	return existing, true, nil
}
