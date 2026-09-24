package repository

import (
	"context"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnnouncementRepo struct {
	pool *pgxpool.Pool
}

func NewAnnouncementRepo(pool *pgxpool.Pool) *AnnouncementRepo {
	return &AnnouncementRepo{pool: pool}
}

const announcementColumns = `id, title, body, image_key, created_by, created_at`

// Create — posée par un admin (voir AdminHandler.CreateAnnouncement).
func (r *AnnouncementRepo) Create(ctx context.Context, createdBy string, input model.CreateAnnouncementInput) (*model.Announcement, error) {
	a := &model.Announcement{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO announcements (title, body, image_key, created_by) VALUES ($1, $2, $3, $4) RETURNING `+announcementColumns,
		input.Title, input.Body, input.ImageKey, createdBy,
	).Scan(&a.ID, &a.Title, &a.Body, &a.ImageKey, &a.CreatedBy, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// List — les plus récentes en premier, plafonné à 50 : cet espace est fait
// pour les dernières nouvelles, pas un historique complet à parcourir.
func (r *AnnouncementRepo) List(ctx context.Context) ([]*model.Announcement, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+announcementColumns+` FROM announcements ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*model.Announcement{}
	for rows.Next() {
		a := &model.Announcement{}
		if err := rows.Scan(&a.ID, &a.Title, &a.Body, &a.ImageKey, &a.CreatedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Delete — un admin retire une annonce publiée par erreur.
func (r *AnnouncementRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM announcements WHERE id = $1`, id)
	return err
}
