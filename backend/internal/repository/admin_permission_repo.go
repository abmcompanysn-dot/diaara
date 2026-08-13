package repository

import (
	"context"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminPermissionRepo struct {
	pool *pgxpool.Pool
}

func NewAdminPermissionRepo(pool *pgxpool.Pool) *AdminPermissionRepo {
	return &AdminPermissionRepo{pool: pool}
}

// ListForUser retourne les scopes d'un admin. Liste vide = accès complet.
func (r *AdminPermissionRepo) ListForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT permission FROM admin_permissions WHERE user_id = $1 ORDER BY permission`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perms := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *AdminPermissionRepo) Grant(ctx context.Context, userID, permission string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO admin_permissions (user_id, permission) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, permission)
	return err
}

func (r *AdminPermissionRepo) Revoke(ctx context.Context, userID, permission string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM admin_permissions WHERE user_id = $1 AND permission = $2`,
		userID, permission)
	return err
}

// ListAdmins retourne tous les administrateurs avec leurs scopes (pour la page permissions).
func (r *AdminPermissionRepo) ListAdmins(ctx context.Context) ([]*model.AdminSummary, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.email, u.created_at,
			COALESCE(ARRAY_AGG(ap.permission) FILTER (WHERE ap.permission IS NOT NULL), ARRAY[]::text[])
		 FROM users u
		 LEFT JOIN admin_permissions ap ON ap.user_id = u.id
		 WHERE u.is_admin = TRUE
		 GROUP BY u.id ORDER BY u.created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	admins := []*model.AdminSummary{}
	for rows.Next() {
		a := &model.AdminSummary{}
		if err := rows.Scan(&a.ID, &a.Email, &a.CreatedAt, &a.Permissions); err != nil {
			return nil, err
		}
		admins = append(admins, a)
	}
	return admins, rows.Err()
}
