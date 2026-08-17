package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSupportAgentNotFound = errors.New("support agent not found")

type SupportContactRepo struct {
	pool *pgxpool.Pool
}

func NewSupportContactRepo(pool *pgxpool.Pool) *SupportContactRepo {
	return &SupportContactRepo{pool: pool}
}

// --- Agents support ---

// COALESCE : phone/callmebot_apikey sont NULL pour un agent qui n'a pas
// configuré CallMeBot — convertis en chaîne vide côté SQL pour garder le
// scan Go simple (pas de sql.NullString à propager partout).
const supportAgentColumns = `id, name, email, COALESCE(phone, ''), COALESCE(callmebot_apikey, ''), active, created_at`

func scanSupportAgent(row pgx.Row) (*model.SupportAgent, error) {
	a := &model.SupportAgent{}
	err := row.Scan(&a.ID, &a.Name, &a.Email, &a.Phone, &a.CallMeBotAPIKey, &a.Active, &a.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSupportAgentNotFound
		}
		return nil, err
	}
	return a, nil
}

func (r *SupportContactRepo) CreateAgent(ctx context.Context, name, email, phone, callmebotAPIKey string) (*model.SupportAgent, error) {
	return scanSupportAgent(r.pool.QueryRow(ctx,
		`INSERT INTO support_agents (name, email, phone, callmebot_apikey) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''))
		 RETURNING `+supportAgentColumns,
		name, email, phone, callmebotAPIKey))
}

func (r *SupportContactRepo) ListAgents(ctx context.Context) ([]*model.SupportAgent, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+supportAgentColumns+` FROM support_agents ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*model.SupportAgent{}
	for rows.Next() {
		a, err := scanSupportAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SupportContactRepo) ListActiveAgents(ctx context.Context) ([]*model.SupportAgent, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+supportAgentColumns+` FROM support_agents WHERE active = true ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*model.SupportAgent{}
	for rows.Next() {
		a, err := scanSupportAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SupportContactRepo) FindAgentByID(ctx context.Context, id string) (*model.SupportAgent, error) {
	return scanSupportAgent(r.pool.QueryRow(ctx,
		`SELECT `+supportAgentColumns+` FROM support_agents WHERE id = $1`, id))
}

func (r *SupportContactRepo) UpdateAgent(ctx context.Context, id string, input model.UpdateSupportAgentInput) (*model.SupportAgent, error) {
	query := `UPDATE support_agents SET `
	args := []interface{}{}
	sets := []string{}
	argIdx := 1

	if input.Name != nil {
		sets = append(sets, `name = $`+itoa(argIdx))
		args = append(args, *input.Name)
		argIdx++
	}
	if input.Email != nil {
		sets = append(sets, `email = $`+itoa(argIdx))
		args = append(args, *input.Email)
		argIdx++
	}
	if input.Phone != nil {
		sets = append(sets, `phone = NULLIF($`+itoa(argIdx)+`, '')`)
		args = append(args, *input.Phone)
		argIdx++
	}
	if input.CallMeBotAPIKey != nil {
		sets = append(sets, `callmebot_apikey = NULLIF($`+itoa(argIdx)+`, '')`)
		args = append(args, *input.CallMeBotAPIKey)
		argIdx++
	}
	if input.Active != nil {
		sets = append(sets, `active = $`+itoa(argIdx))
		args = append(args, *input.Active)
		argIdx++
	}
	if len(sets) == 0 {
		return r.FindAgentByID(ctx, id)
	}

	query += join(sets, ", ") + ` WHERE id = $` + itoa(argIdx)
	args = append(args, id)

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return nil, err
	}
	return r.FindAgentByID(ctx, id)
}

func (r *SupportContactRepo) DeleteAgent(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM support_agents WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSupportAgentNotFound
	}
	return nil
}

// --- Demandes de contact ---

func (r *SupportContactRepo) CreateContactRequest(ctx context.Context, input model.CreateSupportContactInput) (*model.SupportContactRequest, error) {
	req := &model.SupportContactRequest{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO support_contact_requests (name, contact_method, contact_value, message)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, contact_method, contact_value, message, created_at`,
		input.Name, input.ContactMethod, input.ContactValue, input.Message,
	).Scan(&req.ID, &req.Name, &req.ContactMethod, &req.ContactValue, &req.Message, &req.CreatedAt)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (r *SupportContactRepo) ListContactRequests(ctx context.Context) ([]*model.SupportContactRequest, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, contact_method, contact_value, message, created_at
		 FROM support_contact_requests ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*model.SupportContactRequest{}
	for rows.Next() {
		req := &model.SupportContactRequest{}
		if err := rows.Scan(&req.ID, &req.Name, &req.ContactMethod, &req.ContactValue, &req.Message, &req.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}
