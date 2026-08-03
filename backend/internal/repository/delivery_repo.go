package repository

import (
	"context"
	"errors"
	"time"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDeliveryLinkNotFound = errors.New("delivery link not found")
var ErrDeliveryExpired = errors.New("delivery link expired")
var ErrDeliveryLimitReached = errors.New("download limit reached")

type DeliveryRepo struct {
	pool *pgxpool.Pool
}

func NewDeliveryRepo(pool *pgxpool.Pool) *DeliveryRepo {
	return &DeliveryRepo{pool: pool}
}

const deliveryColumns = `id, sale_id, signed_url_token, max_downloads, download_count, expires_at, created_at`

func scanDelivery(row pgx.Row) (*model.DeliveryLink, error) {
	d := &model.DeliveryLink{}
	err := row.Scan(&d.ID, &d.SaleID, &d.SignedURLToken, &d.MaxDownloads, &d.DownloadCount, &d.ExpiresAt, &d.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrDeliveryLinkNotFound
		}
		return nil, err
	}
	return d, nil
}

func (r *DeliveryRepo) Create(ctx context.Context, saleID, token string, maxDownloads int, expiresAt time.Time) (*model.DeliveryLink, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO delivery_links (sale_id, signed_url_token, max_downloads, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+deliveryColumns,
		saleID, token, maxDownloads, expiresAt)
	return scanDelivery(row)
}

func (r *DeliveryRepo) FindByToken(ctx context.Context, token string) (*model.DeliveryLink, error) {
	return scanDelivery(r.pool.QueryRow(ctx,
		`SELECT `+deliveryColumns+` FROM delivery_links WHERE signed_url_token = $1`, token))
}

func (r *DeliveryRepo) FindBySale(ctx context.Context, saleID string) (*model.DeliveryLink, error) {
	return scanDelivery(r.pool.QueryRow(ctx,
		`SELECT `+deliveryColumns+` FROM delivery_links WHERE sale_id = $1`, saleID))
}

func (r *DeliveryRepo) IncrementDownloadCount(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE delivery_links SET download_count = download_count + 1 WHERE id = $1`, id)
	return err
}
