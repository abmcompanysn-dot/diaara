package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSummitSponsorTierNotFound = errors.New("summit sponsor tier not found")
	ErrSummitSponsorNotFound     = errors.New("summit sponsor not found")
)

type SummitSponsorRepo struct {
	pool *pgxpool.Pool
}

func NewSummitSponsorRepo(pool *pgxpool.Pool) *SummitSponsorRepo {
	return &SummitSponsorRepo{pool: pool}
}

// --- Paliers ---

const summitSponsorTierColumns = `id, name, price_cfa, perks, highlight, sort_order, created_at, updated_at`

func scanSummitSponsorTier(row pgx.Row) (*model.SummitSponsorTier, error) {
	t := &model.SummitSponsorTier{}
	err := row.Scan(&t.ID, &t.Name, &t.PriceCFA, &t.Perks, &t.Highlight, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSummitSponsorTierNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *SummitSponsorRepo) ListTiers(ctx context.Context) ([]*model.SummitSponsorTier, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+summitSponsorTierColumns+` FROM summit_sponsor_tiers ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.SummitSponsorTier
	for rows.Next() {
		t := &model.SummitSponsorTier{}
		if err := rows.Scan(&t.ID, &t.Name, &t.PriceCFA, &t.Perks, &t.Highlight, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *SummitSponsorRepo) CreateTier(ctx context.Context, in model.CreateSummitSponsorTierInput) (*model.SummitSponsorTier, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO summit_sponsor_tiers (name, price_cfa, perks, highlight, sort_order)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+summitSponsorTierColumns,
		in.Name, in.PriceCFA, in.Perks, in.Highlight, in.SortOrder,
	)
	return scanSummitSponsorTier(row)
}

func (r *SummitSponsorRepo) UpdateTier(ctx context.Context, id string, in model.UpdateSummitSponsorTierInput) (*model.SummitSponsorTier, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE summit_sponsor_tiers SET
			name = COALESCE($2, name),
			price_cfa = COALESCE($3, price_cfa),
			perks = COALESCE($4, perks),
			highlight = COALESCE($5, highlight),
			sort_order = COALESCE($6, sort_order),
			updated_at = now()
		 WHERE id = $1
		 RETURNING `+summitSponsorTierColumns,
		id, in.Name, in.PriceCFA, in.Perks, in.Highlight, in.SortOrder,
	)
	return scanSummitSponsorTier(row)
}

func (r *SummitSponsorRepo) DeleteTier(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM summit_sponsor_tiers WHERE id = $1`, id)
	return err
}

// --- Sponsors ---

const summitSponsorColumns = `id, tier_id, name, logo_key, website_url, published, sort_order, created_at, updated_at`

func scanSummitSponsor(row pgx.Row) (*model.SummitSponsor, error) {
	s := &model.SummitSponsor{}
	err := row.Scan(&s.ID, &s.TierID, &s.Name, &s.LogoKey, &s.WebsiteURL, &s.Published, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSummitSponsorNotFound
		}
		return nil, err
	}
	return s, nil
}

// ListPublished — sponsors visibles sur /summit/sponsors (published=true),
// plus récents en dernier au sein d'un même sort_order.
func (r *SummitSponsorRepo) ListPublished(ctx context.Context) ([]*model.SummitSponsor, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+summitSponsorColumns+` FROM summit_sponsors WHERE published = true ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSummitSponsors(rows)
}

// ListAll — vue admin (publiés et non publiés).
func (r *SummitSponsorRepo) ListAll(ctx context.Context) ([]*model.SummitSponsor, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+summitSponsorColumns+` FROM summit_sponsors ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSummitSponsors(rows)
}

func scanSummitSponsors(rows pgx.Rows) ([]*model.SummitSponsor, error) {
	var out []*model.SummitSponsor
	for rows.Next() {
		s := &model.SummitSponsor{}
		if err := rows.Scan(&s.ID, &s.TierID, &s.Name, &s.LogoKey, &s.WebsiteURL, &s.Published, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *SummitSponsorRepo) Create(ctx context.Context, in model.CreateSummitSponsorInput) (*model.SummitSponsor, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO summit_sponsors (tier_id, name, logo_key, website_url, published, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+summitSponsorColumns,
		in.TierID, in.Name, in.LogoKey, in.WebsiteURL, in.Published, in.SortOrder,
	)
	return scanSummitSponsor(row)
}

func (r *SummitSponsorRepo) Update(ctx context.Context, id string, in model.UpdateSummitSponsorInput) (*model.SummitSponsor, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE summit_sponsors SET
			tier_id = COALESCE($2, tier_id),
			name = COALESCE($3, name),
			logo_key = COALESCE($4, logo_key),
			website_url = COALESCE($5, website_url),
			published = COALESCE($6, published),
			sort_order = COALESCE($7, sort_order),
			updated_at = now()
		 WHERE id = $1
		 RETURNING `+summitSponsorColumns,
		id, in.TierID, in.Name, in.LogoKey, in.WebsiteURL, in.Published, in.SortOrder,
	)
	return scanSummitSponsor(row)
}

func (r *SummitSponsorRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM summit_sponsors WHERE id = $1`, id)
	return err
}
