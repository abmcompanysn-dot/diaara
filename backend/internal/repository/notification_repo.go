package repository

import (
	"context"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepo struct {
	pool *pgxpool.Pool
}

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{pool: pool}
}

const notificationColumns = `id, user_id, type, title, body, link, read_at, created_at`

func scanNotification(row pgx.Row) (*model.Notification, error) {
	n := &model.Notification{}
	err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Link, &n.ReadAt, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// Create — insère une notification pour un utilisateur. Appelée en tâche de
// fond depuis les webhooks (paiement confirmé, versement, remboursement...),
// jamais bloquante pour le flux principal : l'appelant ignore les erreurs.
func (r *NotificationRepo) Create(ctx context.Context, userID, notifType, title, body, link string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notifications (user_id, type, title, body, link) VALUES ($1, $2, $3, $4, $5)`,
		userID, notifType, title, body, link)
	return err
}

// ListByUser — les 30 notifications les plus récentes d'un utilisateur.
func (r *NotificationRepo) ListByUser(ctx context.Context, userID string) ([]*model.Notification, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+notificationColumns+` FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 30`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := []*model.Notification{}
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&count)
	return count, err
}

// MarkRead — marque une notification comme lue. Scopé au propriétaire pour
// qu'un utilisateur ne puisse pas marquer les notifications d'un autre.
func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2 AND read_at IS NULL`,
		id, userID)
	return err
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
