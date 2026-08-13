package repository

import (
	"context"
	"errors"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrReferralNotFound = errors.New("referral link not found")
	ErrReferralExists   = errors.New("referral link already exists")
)

// IsUniqueViolation détecte l'erreur Postgres 23505 (contrainte unique).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type ReferralRepo struct {
	pool *pgxpool.Pool
}

func NewReferralRepo(pool *pgxpool.Pool) *ReferralRepo {
	return &ReferralRepo{pool: pool}
}

const referralColumns = `id, product_id, closer_id, slug, commission_pct, clicks, created_at`

func scanReferral(row pgx.Row) (*model.ReferralLink, error) {
	l := &model.ReferralLink{}
	err := row.Scan(&l.ID, &l.ProductID, &l.CloserID, &l.Slug, &l.CommissionPct, &l.Clicks, &l.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrReferralNotFound
		}
		return nil, err
	}
	return l, nil
}

// Create insère un lien d'affiliation. Un doublon (product_id, closer_id)
// est refusé : un closer ne génère qu'un lien par produit.
func (r *ReferralRepo) Create(ctx context.Context, closerID string, input model.CreateReferralLinkInput) (*model.ReferralLink, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO referral_links (product_id, closer_id, slug, commission_pct)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+referralColumns,
		input.ProductID, closerID, input.Slug, input.CommissionPct,
	)
	link, err := scanReferral(row)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrReferralExists
		}
		return nil, err
	}
	return link, nil
}

// SlugAvailable vérifie qu'un slug n'est pas déjà pris.
func (r *ReferralRepo) SlugAvailable(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM referral_links WHERE slug = $1)`, slug,
	).Scan(&exists)
	return !exists, err
}

func (r *ReferralRepo) FindByID(ctx context.Context, id string) (*model.ReferralLink, error) {
	return scanReferral(r.pool.QueryRow(ctx,
		`SELECT `+referralColumns+` FROM referral_links WHERE id = $1`, id))
}

func (r *ReferralRepo) FindBySlug(ctx context.Context, slug string) (*model.ReferralLink, error) {
	return scanReferral(r.pool.QueryRow(ctx,
		`SELECT `+referralColumns+` FROM referral_links WHERE slug = $1`, slug))
}

func (r *ReferralRepo) IncrementClicks(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE referral_links SET clicks = clicks + 1 WHERE id = $1`, id)
	return err
}

// ListByCloser retourne les liens d'un closer avec leurs stats de vente
// (nombre de ventes, chiffre d'affaires, commissions dues).
func (r *ReferralRepo) ListByCloser(ctx context.Context, closerID string) ([]*model.ReferralLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT rl.id, rl.product_id, p.title, rl.closer_id, rl.slug, rl.commission_pct, rl.clicks, rl.created_at,
			COUNT(s.id) FILTER (WHERE s.id IS NOT NULL AND s.status <> 'failed' AND s.status <> 'refunded') AS sales,
			COALESCE(SUM(s.amount_cfa) FILTER (WHERE s.status <> 'failed' AND s.status <> 'refunded'), 0) AS revenue,
			COALESCE(SUM(s.closer_commission_cfa) FILTER (WHERE s.status <> 'failed' AND s.status <> 'refunded'), 0) AS commissions
		 FROM referral_links rl
		 LEFT JOIN products p ON p.id = rl.product_id
		 LEFT JOIN sales s ON s.referral_link_id = rl.id
		 WHERE rl.closer_id = $1
		 GROUP BY rl.id, p.title
		 ORDER BY rl.created_at DESC`,
		closerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []*model.ReferralLink{}
	for rows.Next() {
		l := &model.ReferralLink{}
		err := rows.Scan(&l.ID, &l.ProductID, &l.ProductTitle, &l.CloserID, &l.Slug,
			&l.CommissionPct, &l.Clicks, &l.CreatedAt, &l.Sales, &l.RevenueCFA, &l.Commissions)
		if err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// TopClosersByConversion agrège tous les liens par closer : clics, ventes,
// revenu et taux de conversion (ventes/clics), classé par revenu.
func (r *ReferralRepo) TopClosersByConversion(ctx context.Context, limit int) ([]model.CloserConversionPoint, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.email,
			COALESCE(SUM(rl.clicks), 0) AS clicks,
			COUNT(s.id) FILTER (WHERE s.id IS NOT NULL AND s.status <> 'failed' AND s.status <> 'refunded') AS sales,
			COALESCE(SUM(s.amount_cfa) FILTER (WHERE s.status <> 'failed' AND s.status <> 'refunded'), 0) AS revenue_cfa
		 FROM referral_links rl
		 JOIN users u ON u.id = rl.closer_id
		 LEFT JOIN sales s ON s.referral_link_id = rl.id
		 GROUP BY u.id, u.email
		 ORDER BY revenue_cfa DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := []model.CloserConversionPoint{}
	for rows.Next() {
		var p model.CloserConversionPoint
		if err := rows.Scan(&p.CloserID, &p.Email, &p.Clicks, &p.Sales, &p.RevenueCFA); err != nil {
			return nil, err
		}
		if p.Clicks > 0 {
			p.ConversionPct = float64(p.Sales) / float64(p.Clicks) * 100
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
