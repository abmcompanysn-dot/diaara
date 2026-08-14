package repository

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProductNotFound = errors.New("product not found")

type ProductRepo struct {
	pool *pgxpool.Pool
}

func NewProductRepo(pool *pgxpool.Pool) *ProductRepo {
	return &ProductRepo{pool: pool}
}

const productColumns = `id, vendor_id, title, description, price_cfa, price_mode, min_price_cfa, category, file_key,
	cover_image_key, moderation_status, moderation_note, affiliate_enabled, max_closer_commission_pct,
	preview_keys, preview_status, created_at, updated_at`

func scanProduct(row pgx.Row) (*model.Product, error) {
	p := &model.Product{}
	err := row.Scan(&p.ID, &p.VendorID, &p.Title, &p.Description, &p.PriceCFA, &p.PriceMode, &p.MinPriceCFA, &p.Category,
		&p.FileKey, &p.CoverImageKey, &p.ModerationStatus, &p.ModerationNote, &p.AffiliateEnabled,
		&p.MaxCloserCommissionPct, &p.PreviewKeys, &p.PreviewStatus, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *ProductRepo) Create(ctx context.Context, input model.CreateProductInput, vendorID string) (*model.Product, error) {
	priceMode := input.PriceMode
	if priceMode == "" {
		priceMode = "fixed"
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO products (vendor_id, title, description, price_cfa, price_mode, min_price_cfa, category, file_key,
			cover_image_key, affiliate_enabled, max_closer_commission_pct)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING `+productColumns,
		vendorID, input.Title, input.Description, input.PriceCFA, priceMode, input.MinPriceCFA, input.Category, input.FileKey,
		input.CoverImageKey, input.AffiliateEnabled, input.MaxCloserCommissionPct,
	)
	return scanProduct(row)
}

func (r *ProductRepo) FindByID(ctx context.Context, id string) (*model.Product, error) {
	return scanProduct(r.pool.QueryRow(ctx,
		`SELECT `+productColumns+` FROM products WHERE id = $1`, id))
}

func (r *ProductRepo) FindByVendor(ctx context.Context, vendorID string) ([]*model.Product, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+productColumns+` FROM products WHERE vendor_id = $1 ORDER BY created_at DESC`,
		vendorID)
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

func (r *ProductRepo) ListApproved(ctx context.Context, category, search string, limit, offset int) ([]*model.Product, error) {
	query := `SELECT ` + productColumns + ` FROM products WHERE moderation_status = 'approved'`
	args := []interface{}{}
	argIdx := 1

	if category != "" {
		query += ` AND category = $` + itoa(argIdx)
		args = append(args, category)
		argIdx++
	}
	if search != "" {
		query += ` AND title ILIKE $` + itoa(argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + itoa(argIdx)
	args = append(args, limit)
	argIdx++
	query += ` OFFSET $` + itoa(argIdx)
	args = append(args, offset)

	rows, err := r.pool.Query(ctx, query, args...)
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

func (r *ProductRepo) ListPending(ctx context.Context) ([]*model.Product, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+productColumns+` FROM products WHERE moderation_status = 'pending' ORDER BY created_at`)
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

// ListForAdmin — tous les produits (ou filtrés par statut) avec l'email du
// vendeur joint, pour l'écran de modération admin. status vide = tous statuts.
func (r *ProductRepo) ListForAdmin(ctx context.Context, status string) ([]*model.AdminProduct, error) {
	query := `SELECT p.id, p.vendor_id, p.title, p.description, p.price_cfa, p.price_mode, p.min_price_cfa,
		p.category, p.file_key, p.cover_image_key, p.moderation_status, p.moderation_note, p.affiliate_enabled,
		p.max_closer_commission_pct, p.preview_keys, p.preview_status, p.created_at, p.updated_at, u.email
		FROM products p JOIN users u ON u.id = p.vendor_id`
	args := []interface{}{}
	if status != "" {
		query += ` WHERE p.moderation_status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY p.created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []*model.AdminProduct{}
	for rows.Next() {
		p := &model.AdminProduct{}
		err := rows.Scan(&p.ID, &p.VendorID, &p.Title, &p.Description, &p.PriceCFA, &p.PriceMode, &p.MinPriceCFA, &p.Category,
			&p.FileKey, &p.CoverImageKey, &p.ModerationStatus, &p.ModerationNote, &p.AffiliateEnabled,
			&p.MaxCloserCommissionPct, &p.PreviewKeys, &p.PreviewStatus, &p.CreatedAt, &p.UpdatedAt, &p.VendorEmail)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *ProductRepo) Update(ctx context.Context, id string, input model.UpdateProductInput) (*model.Product, error) {
	query := `UPDATE products SET `
	args := []interface{}{}
	argIdx := 1
	sets := []string{}

	if input.Title != nil {
		sets = append(sets, `title = $`+itoa(argIdx))
		args = append(args, *input.Title)
		argIdx++
	}
	if input.Description != nil {
		sets = append(sets, `description = $`+itoa(argIdx))
		args = append(args, *input.Description)
		argIdx++
	}
	if input.PriceCFA != nil {
		sets = append(sets, `price_cfa = $`+itoa(argIdx))
		args = append(args, *input.PriceCFA)
		argIdx++
	}
	if input.PriceMode != nil {
		sets = append(sets, `price_mode = $`+itoa(argIdx))
		args = append(args, *input.PriceMode)
		argIdx++
	}
	if input.MinPriceCFA != nil {
		sets = append(sets, `min_price_cfa = $`+itoa(argIdx))
		args = append(args, *input.MinPriceCFA)
		argIdx++
	}
	if input.FileKey != nil {
		sets = append(sets, `file_key = $`+itoa(argIdx))
		args = append(args, *input.FileKey)
		argIdx++
		// Le fichier a changé : les aperçus filigranés existants ne
		// correspondent plus, on les régénère (même flux qu'à la création).
		sets = append(sets, `preview_status = 'pending'`, `preview_keys = '{}'`)
	}
	if input.Category != nil {
		sets = append(sets, `category = $`+itoa(argIdx))
		args = append(args, *input.Category)
		argIdx++
	}
	if input.CoverImageKey != nil {
		sets = append(sets, `cover_image_key = $`+itoa(argIdx))
		args = append(args, *input.CoverImageKey)
		argIdx++
	}
	if input.AffiliateEnabled != nil {
		sets = append(sets, `affiliate_enabled = $`+itoa(argIdx))
		args = append(args, *input.AffiliateEnabled)
		argIdx++
	}
	if input.MaxCloserCommissionPct != nil {
		sets = append(sets, `max_closer_commission_pct = $`+itoa(argIdx))
		args = append(args, *input.MaxCloserCommissionPct)
		argIdx++
	}

	sets = append(sets, `updated_at = NOW()`)
	query += join(sets, ", ")
	query += ` WHERE id = $` + itoa(argIdx)
	args = append(args, id)

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *ProductRepo) UpdateModerationStatus(ctx context.Context, id, status string, note *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE products SET moderation_status = $2, moderation_note = $3, updated_at = NOW() WHERE id = $1`,
		id, status, note)
	return err
}

// SetPreview — enregistre le résultat de la génération d'aperçu (tâche de
// fond après la création du produit). status : "ready", "unsupported" (type
// de fichier sans aperçu géré) ou "failed".
func (r *ProductRepo) SetPreview(ctx context.Context, id string, keys []string, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE products SET preview_keys = $2, preview_status = $3 WHERE id = $1`,
		id, keys, status)
	return err
}

func (r *ProductRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r *ProductRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&count)
	return count, err
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func join(parts []string, sep string) string {
	return strings.Join(parts, sep)
}
