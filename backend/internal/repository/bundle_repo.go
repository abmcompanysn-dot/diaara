package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrBundleNotFound = errors.New("bundle not found")

type BundleRepo struct {
	pool *pgxpool.Pool
}

func NewBundleRepo(pool *pgxpool.Pool) *BundleRepo {
	return &BundleRepo{pool: pool}
}

const bundleColumns = `id, vendor_id, title, description, price_cfa, cover_image_key, moderation_status, created_at, updated_at`

func scanBundle(row pgx.Row) (*model.ProductBundle, error) {
	b := &model.ProductBundle{}
	err := row.Scan(&b.ID, &b.VendorID, &b.Title, &b.Description, &b.PriceCFA, &b.CoverImageKey,
		&b.ModerationStatus, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrBundleNotFound
		}
		return nil, err
	}
	return b, nil
}

// Create crée le pack puis ses éléments dans une transaction (soit tout,
// soit rien — un pack sans élément n'a pas de sens).
func (r *BundleRepo) Create(ctx context.Context, input model.CreateBundleInput, vendorID string) (*model.ProductBundle, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	bundle, err := scanBundle(tx.QueryRow(ctx,
		`INSERT INTO product_bundles (vendor_id, title, description, price_cfa, cover_image_key)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+bundleColumns,
		vendorID, input.Title, input.Description, input.PriceCFA, input.CoverImageKey,
	))
	if err != nil {
		return nil, err
	}

	for _, productID := range input.ProductIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO product_bundle_items (bundle_id, product_id) VALUES ($1, $2)`,
			bundle.ID, productID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return bundle, nil
}

func (r *BundleRepo) FindByID(ctx context.Context, id string) (*model.ProductBundle, error) {
	return scanBundle(r.pool.QueryRow(ctx, `SELECT `+bundleColumns+` FROM product_bundles WHERE id = $1`, id))
}

func (r *BundleRepo) ListByVendor(ctx context.Context, vendorID string) ([]*model.ProductBundle, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+bundleColumns+` FROM product_bundles WHERE vendor_id = $1 ORDER BY created_at DESC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bundles := []*model.ProductBundle{}
	for rows.Next() {
		b, err := scanBundle(rows)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, b)
	}
	return bundles, rows.Err()
}

// ListItems retourne les produits inclus dans un pack.
func (r *BundleRepo) ListItems(ctx context.Context, bundleID string) ([]*model.Product, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+productColumns+` FROM products p
		 JOIN product_bundle_items i ON i.product_id = p.id
		 WHERE i.bundle_id = $1`, bundleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []*model.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *BundleRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM product_bundles WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBundleNotFound
	}
	return nil
}
