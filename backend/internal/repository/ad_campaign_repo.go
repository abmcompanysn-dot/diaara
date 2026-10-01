package repository

import (
	"context"
	"errors"
	"time"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAdCampaignNotFound  = errors.New("ad campaign not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

type AdCampaignRepo struct {
	pool *pgxpool.Pool
}

func NewAdCampaignRepo(pool *pgxpool.Pool) *AdCampaignRepo {
	return &AdCampaignRepo{pool: pool}
}

// availableBalanceSQL — solde disponible d'un vendeur ($1), MÊME définition
// que PayoutHandler.Earnings/Create (ventes paid/delivered de ses produits,
// moins les versements non échoués/non remboursés), moins les sponsorisations
// payées par le solde et non remboursées. Calculé en SQL pour pouvoir être
// lu dans la même transaction que l'insertion (voir CreateFromBalance).
const availableBalanceSQL = `
	SELECT
	  COALESCE((SELECT SUM(s.vendor_amount_cfa) FROM sales s JOIN products p ON p.id = s.product_id
	            WHERE p.vendor_id = $1 AND s.status IN ('paid', 'delivered')), 0)
	- COALESCE((SELECT SUM(amount_cfa) FROM payouts
	            WHERE user_id = $1 AND status NOT IN ('failed', 'refunded')), 0)
	- COALESCE((SELECT SUM(amount_cfa) FROM ad_campaigns
	            WHERE vendor_id = $1 AND payment_method = 'balance' AND NOT refunded), 0)`

const adColumns = `c.id, c.vendor_id, c.product_id, c.platform, c.status, c.amount_cfa, c.commission_cfa,
	c.ad_budget_cfa, c.duration_days, c.countries, c.message, c.payment_method, c.refunded, c.starts_at, c.ends_at,
	c.external_campaign_id, c.external_adset_id, c.external_creative_id, c.external_ad_id, c.failure_reason,
	c.impressions, c.reach, c.clicks, c.spend_cfa, c.stats_updated_at, c.created_at, c.updated_at,
	COALESCE(p.title, ''), COALESCE(u.email, '')`

const adFrom = ` FROM ad_campaigns c
	LEFT JOIN products p ON p.id = c.product_id
	LEFT JOIN users u ON u.id = c.vendor_id`

func scanAd(row pgx.Row) (*model.AdCampaign, error) {
	c := &model.AdCampaign{}
	err := row.Scan(&c.ID, &c.VendorID, &c.ProductID, &c.Platform, &c.Status, &c.AmountCFA, &c.CommissionCFA,
		&c.AdBudgetCFA, &c.DurationDays, &c.Countries, &c.Message, &c.PaymentMethod, &c.Refunded, &c.StartsAt, &c.EndsAt,
		&c.ExternalCampaignID, &c.ExternalAdSetID, &c.ExternalCreativeID, &c.ExternalAdID, &c.FailureReason,
		&c.Impressions, &c.Reach, &c.Clicks, &c.SpendCFA, &c.StatsUpdatedAt, &c.CreatedAt, &c.UpdatedAt,
		&c.ProductTitle, &c.VendorEmail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAdCampaignNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *AdCampaignRepo) list(ctx context.Context, where string, args ...interface{}) ([]*model.AdCampaign, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+adColumns+adFrom+` `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.AdCampaign{}
	for rows.Next() {
		c, err := scanAd(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AvailableBalance — solde disponible actuel (affichage ; la création relit
// le solde sous verrou, voir CreateFromBalance).
func (r *AdCampaignRepo) AvailableBalance(ctx context.Context, vendorID string) (int, error) {
	var v int
	err := r.pool.QueryRow(ctx, availableBalanceSQL, vendorID).Scan(&v)
	return v, err
}

// BalanceDebits — total des sponsorisations payées par le solde et non
// remboursées, à retrancher du solde dans PayoutHandler (Earnings/Create).
func (r *AdCampaignRepo) BalanceDebits(ctx context.Context, vendorID string) (int, error) {
	var v int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount_cfa), 0) FROM ad_campaigns
		 WHERE vendor_id = $1 AND payment_method = 'balance' AND NOT refunded`, vendorID).Scan(&v)
	return v, err
}

// CreateFromBalance insère une campagne payée par le solde, dans une
// transaction verrouillée par vendeur (pg_advisory_xact_lock) : deux
// demandes simultanées ne peuvent jamais dépenser deux fois le même solde.
// ErrInsufficientBalance si amount_cfa dépasse le solde disponible.
func (r *AdCampaignRepo) CreateFromBalance(ctx context.Context, c *model.AdCampaign) (*model.AdCampaign, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('vendor_balance:' || $1::text))`, c.VendorID); err != nil {
		return nil, err
	}
	var available int
	if err := tx.QueryRow(ctx, availableBalanceSQL, c.VendorID).Scan(&available); err != nil {
		return nil, err
	}
	if c.AmountCFA > available {
		return nil, ErrInsufficientBalance
	}

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO ad_campaigns (vendor_id, product_id, platform, status, amount_cfa, commission_cfa,
			ad_budget_cfa, duration_days, countries, message, payment_method)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'balance')
		 RETURNING id`,
		c.VendorID, c.ProductID, c.Platform, model.AdStatusLaunching, c.AmountCFA, c.CommissionCFA,
		c.AdBudgetCFA, c.DurationDays, c.Countries, c.Message).Scan(&id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *AdCampaignRepo) FindByID(ctx context.Context, id string) (*model.AdCampaign, error) {
	return scanAd(r.pool.QueryRow(ctx, `SELECT `+adColumns+adFrom+` WHERE c.id = $1`, id))
}

func (r *AdCampaignRepo) ListByVendor(ctx context.Context, vendorID string) ([]*model.AdCampaign, error) {
	return r.list(ctx, `WHERE c.vendor_id = $1 ORDER BY c.created_at DESC`, vendorID)
}

func (r *AdCampaignRepo) ListAll(ctx context.Context) ([]*model.AdCampaign, error) {
	return r.list(ctx, `ORDER BY c.created_at DESC LIMIT 500`)
}

// ListToSync — campagnes dont le statut ou les statistiques peuvent encore
// bouger chez la régie : en vérification, actives, ou terminées depuis
// moins de 2 jours (les statistiques Meta se consolident après la fin).
func (r *AdCampaignRepo) ListToSync(ctx context.Context) ([]*model.AdCampaign, error) {
	return r.list(ctx, `WHERE c.status IN ('in_review', 'active')
		OR (c.status IN ('completed', 'stopped') AND c.ends_at > now() - interval '2 days')
		ORDER BY c.created_at LIMIT 300`)
}

// ListStuckLaunching — campagnes restées "launching" plus de olderThan
// (serveur redémarré pendant la création chez la régie) : à rembourser.
func (r *AdCampaignRepo) ListStuckLaunching(ctx context.Context, olderThan time.Duration) ([]*model.AdCampaign, error) {
	return r.list(ctx, `WHERE c.status = 'launching' AND c.created_at < $1 ORDER BY c.created_at`,
		time.Now().Add(-olderThan))
}

// MarkLaunched — la campagne existe chez la régie (envoyée en vérification).
func (r *AdCampaignRepo) MarkLaunched(ctx context.Context, id, campaignID, adSetID, creativeID, adID string, startsAt, endsAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ad_campaigns SET status = 'in_review', external_campaign_id = $2, external_adset_id = $3,
			external_creative_id = $4, external_ad_id = $5, starts_at = $6, ends_at = $7, updated_at = now()
		 WHERE id = $1 AND status = 'launching'`,
		id, campaignID, adSetID, creativeID, adID, startsAt, endsAt)
	return err
}

// MarkRefunded — échec définitif (failed/rejected) : statut + raison, et la
// somme revient au solde du vendeur. Idempotent : false si déjà remboursée
// (jamais deux remboursements pour la même campagne).
func (r *AdCampaignRepo) MarkRefunded(ctx context.Context, id, status, reason string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE ad_campaigns SET status = $2, failure_reason = NULLIF($3, ''), refunded = true, updated_at = now()
		 WHERE id = $1 AND NOT refunded`,
		id, status, reason)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// SetStatus — changement de statut sans mouvement d'argent (active,
// completed, stopped). reason vide = inchangée.
func (r *AdCampaignRepo) SetStatus(ctx context.Context, id, status, reason string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ad_campaigns SET status = $2, failure_reason = COALESCE(NULLIF($3, ''), failure_reason), updated_at = now()
		 WHERE id = $1`,
		id, status, reason)
	return err
}

func (r *AdCampaignRepo) UpdateStats(ctx context.Context, id string, impressions, reach, clicks int64, spendCFA int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ad_campaigns SET impressions = $2, reach = $3, clicks = $4, spend_cfa = $5,
			stats_updated_at = now(), updated_at = now()
		 WHERE id = $1`,
		id, impressions, reach, clicks, spendCFA)
	return err
}
