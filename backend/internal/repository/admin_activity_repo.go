package repository

import (
	"context"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminActivityRepo struct {
	pool *pgxpool.Pool
}

func NewAdminActivityRepo(pool *pgxpool.Pool) *AdminActivityRepo {
	return &AdminActivityRepo{pool: pool}
}

// Log enregistre une action admin. Best-effort par construction (appelé en
// tâche de fond par les handlers, jamais dans le chemin critique) : ce n'est
// qu'un journal de traçabilité, une écriture ratée ne doit jamais faire
// échouer l'action réelle qu'elle décrit — voir les appels dans
// admin_handler.go/product_handler.go (toujours `go repo.Log(...)`, erreur
// ignorée).
func (r *AdminActivityRepo) Log(ctx context.Context, adminID *string, action, targetType string, targetID *string, description string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO admin_activity_log (admin_id, action, target_type, target_id, description)
		 VALUES ($1, $2, $3, $4, $5)`,
		adminID, action, targetType, targetID, description)
	return err
}

func scanAdminActivity(row pgx.Row) (*model.AdminActivityLog, error) {
	a := &model.AdminActivityLog{}
	var adminEmail *string
	err := row.Scan(&a.ID, &a.AdminID, &adminEmail, &a.Action, &a.TargetType, &a.TargetID, &a.Description, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if adminEmail != nil {
		a.AdminEmail = *adminEmail
	}
	return a, nil
}

const adminActivityColumns = `l.id, l.admin_id, u.email, l.action, l.target_type, l.target_id, l.description, l.created_at`

// List — les entrées les plus récentes, paginées. limit/offset gérés par
// l'appelant (pas de valeur par défaut ici, voir AdminHandler.ActivityLog).
func (r *AdminActivityRepo) List(ctx context.Context, limit, offset int) ([]*model.AdminActivityLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+adminActivityColumns+`
		 FROM admin_activity_log l
		 LEFT JOIN users u ON u.id = l.admin_id
		 ORDER BY l.created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []*model.AdminActivityLog{}
	for rows.Next() {
		a, err := scanAdminActivity(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, a)
	}
	return logs, rows.Err()
}

// Count — total des entrées, pour la pagination côté frontend.
func (r *AdminActivityRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_activity_log`).Scan(&count)
	return count, err
}
