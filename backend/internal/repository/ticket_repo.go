package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTicketNotFound = errors.New("ticket not found")

type TicketRepo struct {
	pool *pgxpool.Pool
}

func NewTicketRepo(pool *pgxpool.Pool) *TicketRepo {
	return &TicketRepo{pool: pool}
}

const ticketColumns = `id, user_id, sale_id, subject, status, created_at`

func scanTicket(row pgx.Row) (*model.SupportTicket, error) {
	t := &model.SupportTicket{}
	err := row.Scan(&t.ID, &t.UserID, &t.SaleID, &t.Subject, &t.Status, &t.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return t, nil
}

const messageColumns = `id, ticket_id, author_id, body, created_at`

func scanMessage(row pgx.Row) (*model.TicketMessage, error) {
	m := &model.TicketMessage{}
	err := row.Scan(&m.ID, &m.TicketID, &m.AuthorID, &m.Body, &m.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return m, nil
}

func (r *TicketRepo) Create(ctx context.Context, userID string, input model.CreateTicketInput) (*model.SupportTicket, error) {
	return scanTicket(r.pool.QueryRow(ctx,
		`INSERT INTO support_tickets (user_id, sale_id, subject)
		 VALUES ($1, $2, $3) RETURNING `+ticketColumns,
		userID, input.SaleID, input.Subject))
}

func (r *TicketRepo) FindByID(ctx context.Context, id string) (*model.SupportTicket, error) {
	return scanTicket(r.pool.QueryRow(ctx,
		`SELECT `+ticketColumns+` FROM support_tickets WHERE id = $1`, id))
}

func (r *TicketRepo) ListByUser(ctx context.Context, userID string) ([]*model.SupportTicket, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+ticketColumns+` FROM support_tickets WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := []*model.SupportTicket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (r *TicketRepo) ListAll(ctx context.Context) ([]*model.SupportTicket, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+ticketColumns+` FROM support_tickets ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := []*model.SupportTicket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (r *TicketRepo) AddMessage(ctx context.Context, ticketID, authorID, body string) (*model.TicketMessage, error) {
	return scanMessage(r.pool.QueryRow(ctx,
		`INSERT INTO ticket_messages (ticket_id, author_id, body)
		 VALUES ($1, $2, $3) RETURNING `+messageColumns,
		ticketID, authorID, body))
}

func (r *TicketRepo) ListMessages(ctx context.Context, ticketID string) ([]*model.TicketMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+messageColumns+` FROM ticket_messages WHERE ticket_id = $1 ORDER BY created_at`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*model.TicketMessage{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (r *TicketRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE support_tickets SET status = $2 WHERE id = $1`, id, status)
	return err
}
