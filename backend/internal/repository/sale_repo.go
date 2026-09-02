package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diarra/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSaleNotFound = errors.New("sale not found")

type SaleRepo struct {
	pool *pgxpool.Pool
}

func NewSaleRepo(pool *pgxpool.Pool) *SaleRepo {
	return &SaleRepo{pool: pool}
}

const saleColumns = `id, product_id, buyer_id, buyer_name, country, referral_link_id, amount_cfa, platform_fee_cfa,
	closer_commission_cfa, vendor_amount_cfa, payment_provider, payment_reference, provider_transaction_id, checkout_token, status, refund_reference, delivered_at, created_at`

func scanSale(row pgx.Row) (*model.Sale, error) {
	s := &model.Sale{}
	err := row.Scan(&s.ID, &s.ProductID, &s.BuyerID, &s.BuyerName, &s.Country, &s.ReferralLinkID, &s.AmountCFA,
		&s.PlatformFeeCFA, &s.CloserCommissionCFA, &s.VendorAmountCFA, &s.PaymentProvider,
		&s.PaymentReference, &s.ProviderTransactionID, &s.CheckoutToken, &s.Status, &s.RefundReference, &s.DeliveredAt, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *SaleRepo) Create(ctx context.Context, s *model.Sale) (*model.Sale, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO sales (product_id, buyer_id, buyer_name, country, referral_link_id, amount_cfa, platform_fee_cfa,
			closer_commission_cfa, vendor_amount_cfa, payment_provider, payment_reference, checkout_token, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING `+saleColumns,
		s.ProductID, s.BuyerID, s.BuyerName, s.Country, s.ReferralLinkID, s.AmountCFA, s.PlatformFeeCFA,
		s.CloserCommissionCFA, s.VendorAmountCFA, s.PaymentProvider, s.PaymentReference, s.CheckoutToken, s.Status)
	return scanSale(row)
}

// VendorSaleView — vente enrichie pour l'espace vendeur : le vendeur voit
// qui a acheté (nom, email, pays) et quel produit, sans exposer les autres
// champs internes (frais plateforme, etc. déjà visibles ailleurs).
type VendorSaleView struct {
	*model.Sale
	BuyerEmail   string `json:"buyer_email"`
	ProductTitle string `json:"product_title"`
}

// ListByVendor retourne les ventes des produits d'un vendeur, avec les
// coordonnées de l'acheteur (jointure users) et le titre du produit.
func (r *SaleRepo) ListByVendor(ctx context.Context, vendorID string) ([]*VendorSaleView, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.id, s.product_id, s.buyer_id, s.buyer_name, s.country, s.referral_link_id, s.amount_cfa,
			s.platform_fee_cfa, s.closer_commission_cfa, s.vendor_amount_cfa, s.payment_provider,
			s.payment_reference, s.provider_transaction_id, s.checkout_token, s.status, s.refund_reference, s.delivered_at, s.created_at,
			u.email, p.title
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 JOIN users u ON u.id = s.buyer_id
		 WHERE p.vendor_id = $1
		 ORDER BY s.created_at DESC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	views := []*VendorSaleView{}
	for rows.Next() {
		s := &model.Sale{}
		v := &VendorSaleView{Sale: s}
		if err := rows.Scan(&s.ID, &s.ProductID, &s.BuyerID, &s.BuyerName, &s.Country, &s.ReferralLinkID, &s.AmountCFA,
			&s.PlatformFeeCFA, &s.CloserCommissionCFA, &s.VendorAmountCFA, &s.PaymentProvider,
			&s.PaymentReference, &s.ProviderTransactionID, &s.CheckoutToken, &s.Status, &s.RefundReference, &s.DeliveredAt, &s.CreatedAt,
			&v.BuyerEmail, &v.ProductTitle); err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, rows.Err()
}

// TotalEarnedByVendor — somme du vendor_amount_cfa de toutes les ventes
// payées/livrées d'un vendeur (chiffre d'affaires cumulé depuis toujours),
// pour le niveau/trophée affiché sur sa boutique.
func (r *SaleRepo) TotalEarnedByVendor(ctx context.Context, vendorID string) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(s.vendor_amount_cfa), 0)
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 WHERE p.vendor_id = $1 AND s.status IN ('paid', 'delivered')`,
		vendorID).Scan(&total)
	return total, err
}

func (r *SaleRepo) FindByID(ctx context.Context, id string) (*model.Sale, error) {
	return scanSale(r.pool.QueryRow(ctx,
		`SELECT `+saleColumns+` FROM sales WHERE id = $1`, id))
}

func (r *SaleRepo) FindByCheckoutToken(ctx context.Context, token string) (*model.Sale, error) {
	return scanSale(r.pool.QueryRow(ctx,
		`SELECT `+saleColumns+` FROM sales WHERE checkout_token = $1`, token))
}

func (r *SaleRepo) FindByPaymentReference(ctx context.Context, ref string) (*model.Sale, error) {
	return scanSale(r.pool.QueryRow(ctx,
		`SELECT `+saleColumns+` FROM sales WHERE payment_reference = $1`, ref))
}

func (r *SaleRepo) ListByBuyer(ctx context.Context, buyerID string) ([]*model.Sale, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+saleColumns+` FROM sales WHERE buyer_id = $1 ORDER BY created_at DESC`, buyerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sales := []*model.Sale{}
	for rows.Next() {
		s, err := scanSale(rows)
		if err != nil {
			return nil, err
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

func (r *SaleRepo) ListAll(ctx context.Context) ([]*model.Sale, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+saleColumns+` FROM sales ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sales := []*model.Sale{}
	for rows.Next() {
		s, err := scanSale(rows)
		if err != nil {
			return nil, err
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

func (r *SaleRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sales SET status = $2 WHERE id = $1`, id, status)
	return err
}

// ListPendingForProvider — ventes encore "pending" pour un prestataire donné
// ("pawapay"/"kpay"), créées il y a moins de maxAge, plus anciennes d'abord.
// Utilisé par le job de réconciliation de fond qui va revérifier chaque
// dépôt via l'API du prestataire (filet de sécurité si un webhook s'est
// perdu — incident du 2026-09-02).
func (r *SaleRepo) ListPendingForProvider(ctx context.Context, provider string, maxAge time.Duration) ([]*model.Sale, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+saleColumns+`
		 FROM sales
		 WHERE status = 'pending' AND payment_provider = $1
		   AND created_at >= now() - $2::interval
		 ORDER BY created_at ASC
		 LIMIT 200`, provider, intervalStr(maxAge))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sales := []*model.Sale{}
	for rows.Next() {
		s, err := scanSale(rows)
		if err != nil {
			return nil, err
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

// PendingSaleView — commande non aboutie (pending/failed) enrichie du contact
// acheteur (nom, email, téléphone, pays) et du titre produit + vendeur, pour
// la vue admin « paiements en attente & échoués » et la relance vendeur.
type PendingSaleView struct {
	*model.Sale
	BuyerEmail    string  `json:"buyer_email"`
	BuyerPhone    *string `json:"buyer_phone,omitempty"`
	ProductTitle  string  `json:"product_title"`
	VendorID      string  `json:"vendor_id"`
	VendorEmail   string  `json:"vendor_email"`
}

const pendingSaleSelect = `
	SELECT s.id, s.product_id, s.buyer_id, s.buyer_name, s.country, s.referral_link_id, s.amount_cfa,
		s.platform_fee_cfa, s.closer_commission_cfa, s.vendor_amount_cfa, s.payment_provider,
		s.payment_reference, s.provider_transaction_id, s.checkout_token, s.status, s.refund_reference,
		s.delivered_at, s.created_at,
		s.reminded_at, s.reminder_count,
		u.email, u.phone, p.title, p.vendor_id, vu.email
	 FROM sales s
	 JOIN products p ON p.id = s.product_id
	 JOIN users u   ON u.id = s.buyer_id
	 JOIN users vu  ON vu.id = p.vendor_id`

func scanPendingSaleRows(rows pgx.Rows) ([]*PendingSaleView, error) {
	defer rows.Close()
	views := []*PendingSaleView{}
	for rows.Next() {
		s := &model.Sale{}
		v := &PendingSaleView{Sale: s}
		if err := rows.Scan(&s.ID, &s.ProductID, &s.BuyerID, &s.BuyerName, &s.Country, &s.ReferralLinkID, &s.AmountCFA,
			&s.PlatformFeeCFA, &s.CloserCommissionCFA, &s.VendorAmountCFA, &s.PaymentProvider,
			&s.PaymentReference, &s.ProviderTransactionID, &s.CheckoutToken, &s.Status, &s.RefundReference,
			&s.DeliveredAt, &s.CreatedAt,
			&s.RemindedAt, &s.ReminderCount,
			&v.BuyerEmail, &v.BuyerPhone, &v.ProductTitle, &v.VendorID, &v.VendorEmail); err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, rows.Err()
}

// ListPendingAndFailed — toutes les commandes pending/failed, plus récentes
// d'abord (vue admin : voir qui a essayé d'acheter sans aboutir).
func (r *SaleRepo) ListPendingAndFailed(ctx context.Context) ([]*PendingSaleView, error) {
	rows, err := r.pool.Query(ctx, pendingSaleSelect+`
	 WHERE s.status IN ('pending', 'failed')
	 ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, err
	}
	return scanPendingSaleRows(rows)
}

// FindPendingViewForVendor — une commande pending précise appartenant à un
// produit du vendeur donné (contrôle d'accès pour la relance vendeur).
func (r *SaleRepo) FindPendingViewForVendor(ctx context.Context, saleID, vendorID string) (*PendingSaleView, error) {
	rows, err := r.pool.Query(ctx, pendingSaleSelect+`
	 WHERE s.id = $1 AND p.vendor_id = $2`, saleID, vendorID)
	if err != nil {
		return nil, err
	}
	views, err := scanPendingSaleRows(rows)
	if err != nil {
		return nil, err
	}
	if len(views) == 0 {
		return nil, ErrSaleNotFound
	}
	return views[0], nil
}

// FindPendingView — une commande précise (admin, sans restriction vendeur).
func (r *SaleRepo) FindPendingView(ctx context.Context, saleID string) (*PendingSaleView, error) {
	rows, err := r.pool.Query(ctx, pendingSaleSelect+` WHERE s.id = $1`, saleID)
	if err != nil {
		return nil, err
	}
	views, err := scanPendingSaleRows(rows)
	if err != nil {
		return nil, err
	}
	if len(views) == 0 {
		return nil, ErrSaleNotFound
	}
	return views[0], nil
}

// MarkReminded — enregistre l'envoi d'une relance (incrémente le compteur).
func (r *SaleRepo) MarkReminded(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sales SET reminded_at = now(), reminder_count = reminder_count + 1 WHERE id = $1`, id)
	return err
}

// ListRemindable — ventes "pending" candidates à une relance automatique, selon
// la cadence « 1ʳᵉ relance à firstAfter, 2ᵉ à secondAfter, puis stop » :
//   - reminder_count = 0 et âge ≥ firstAfter        → 1ʳᵉ relance
//   - reminder_count = 1 et âge ≥ secondAfter       → 2ᵉ relance
//   - reminder_count ≥ 2                             → plus jamais
// Les ventes plus vieilles que maxAge sont abandonnées (pas de relance).
// Jointe au contact acheteur.
func (r *SaleRepo) ListRemindable(ctx context.Context, firstAfter, secondAfter, maxAge time.Duration) ([]*PendingSaleView, error) {
	rows, err := r.pool.Query(ctx, pendingSaleSelect+`
	 WHERE s.status = 'pending'
	   AND s.created_at >= now() - $3::interval
	   AND (
	         (s.reminder_count = 0 AND s.created_at <= now() - $1::interval)
	      OR (s.reminder_count = 1 AND s.created_at <= now() - $2::interval)
	       )
	 ORDER BY s.created_at ASC
	 LIMIT 200`,
		intervalStr(firstAfter), intervalStr(secondAfter), intervalStr(maxAge))
	if err != nil {
		return nil, err
	}
	return scanPendingSaleRows(rows)
}

// MarkManuallyConfirmed — bascule une vente pending/failed en 'paid' et trace
// l'admin qui l'a fait. Ne touche pas une vente déjà payée/livrée/remboursée.
func (r *SaleRepo) MarkManuallyConfirmed(ctx context.Context, id, adminID string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE sales
		 SET status = 'paid', manually_confirmed_by = $2, manually_confirmed_at = now()
		 WHERE id = $1 AND status IN ('pending', 'failed')`, id, adminID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// intervalStr formate une durée pour un cast ::interval PostgreSQL ("3600 seconds").
func intervalStr(d time.Duration) string {
	return fmt.Sprintf("%d seconds", int64(d.Seconds()))
}

func (r *SaleRepo) UpdatePaymentReference(ctx context.Context, id, ref string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sales SET payment_reference = $2 WHERE id = $1`, id, ref)
	return err
}

// SetProviderTransactionID — enregistre l'ID KPay du paiement juste après
// l'initiation (voir model.Sale.ProviderTransactionID) : nécessaire pour les
// appels GET statut/remboursement ultérieurs, KPay n'utilisant pas notre
// payment_reference comme identifiant côté serveur.
func (r *SaleRepo) SetProviderTransactionID(ctx context.Context, id, providerTxID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sales SET provider_transaction_id = $2 WHERE id = $1`, id, providerTxID)
	return err
}

// SetRefundReference — enregistre l'ID de remboursement PawaPay et bascule
// la vente en 'refund_pending' (le webhook confirmera avec 'refunded').
func (r *SaleRepo) SetRefundReference(ctx context.Context, id, refundReference string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sales SET refund_reference = $2, status = 'refund_pending' WHERE id = $1`,
		id, refundReference)
	return err
}

func (r *SaleRepo) FindByRefundReference(ctx context.Context, ref string) (*model.Sale, error) {
	return scanSale(r.pool.QueryRow(ctx,
		`SELECT `+saleColumns+` FROM sales WHERE refund_reference = $1`, ref))
}

func (r *SaleRepo) MarkDelivered(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sales SET status = 'delivered', delivered_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *SaleRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sales`).Scan(&count)
	return count, err
}

// GMV — volume total des transactions abouties (Gross Merchandise Value).
func (r *SaleRepo) GMV(ctx context.Context) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount_cfa), 0) FROM sales WHERE status NOT IN ('failed', 'pending')`,
	).Scan(&total)
	return total, err
}

// RevenueThisAndLastMonth — revenu plateforme (frais) du mois en cours et du
// mois précédent, pour calculer un indicateur de croissance (%).
func (r *SaleRepo) RevenueThisAndLastMonth(ctx context.Context) (thisMonth, lastMonth int, err error) {
	err = r.pool.QueryRow(ctx, `SELECT
		COALESCE(SUM(platform_fee_cfa) FILTER (
			WHERE created_at >= date_trunc('month', now())
		), 0),
		COALESCE(SUM(platform_fee_cfa) FILTER (
			WHERE created_at >= date_trunc('month', now() - interval '1 month')
			  AND created_at < date_trunc('month', now())
		), 0)
		FROM sales WHERE status NOT IN ('failed', 'pending')`,
	).Scan(&thisMonth, &lastMonth)
	return thisMonth, lastMonth, err
}

// FailedCount24h — nombre de ventes échouées dans les dernières 24h
// (pour les alertes/notifications admin).
func (r *SaleRepo) FailedCount24h(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM sales WHERE status = 'failed' AND created_at >= now() - interval '24 hours'`,
	).Scan(&count)
	return count, err
}

func (r *SaleRepo) CountByVendor(ctx context.Context, vendorID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM sales s JOIN products p ON s.product_id = p.id WHERE p.vendor_id = $1`,
		vendorID).Scan(&count)
	return count, err
}

// SalesByDay agrège le nombre de ventes et le revenu plateforme par jour sur
// les `days` derniers jours (ventes ni échouées ni remboursées).
func (r *SaleRepo) SalesByDay(ctx context.Context, days int) ([]model.SalesByDayPoint, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT date_trunc('day', created_at)::date AS day,
			COUNT(*) AS sales,
			COALESCE(SUM(amount_cfa), 0) AS revenue_cfa
		 FROM sales
		 WHERE status <> 'failed' AND status <> 'refunded'
			AND created_at >= now() - make_interval(days => $1)
		 GROUP BY day
		 ORDER BY day`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := []model.SalesByDayPoint{}
	for rows.Next() {
		var p model.SalesByDayPoint
		if err := rows.Scan(&p.Day, &p.Sales, &p.RevenueCFA); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// TopProducts classe les produits par revenu (ventes ni échouées ni remboursées).
func (r *SaleRepo) TopProducts(ctx context.Context, limit int) ([]model.TopProductPoint, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.title, COUNT(s.id) AS sales, COALESCE(SUM(s.amount_cfa), 0) AS revenue_cfa
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 WHERE s.status <> 'failed' AND s.status <> 'refunded'
		 GROUP BY p.id, p.title
		 ORDER BY revenue_cfa DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := []model.TopProductPoint{}
	for rows.Next() {
		var p model.TopProductPoint
		if err := rows.Scan(&p.ProductID, &p.Title, &p.Sales, &p.RevenueCFA); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// TopVendors classe les vendeurs par revenu (ventes ni échouées ni remboursées).
func (r *SaleRepo) TopVendors(ctx context.Context, limit int) ([]model.TopVendorPoint, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.email, COUNT(s.id) AS sales, COALESCE(SUM(s.vendor_amount_cfa), 0) AS revenue_cfa
		 FROM sales s
		 JOIN products p ON p.id = s.product_id
		 JOIN users u ON u.id = p.vendor_id
		 WHERE s.status <> 'failed' AND s.status <> 'refunded'
		 GROUP BY u.id, u.email
		 ORDER BY revenue_cfa DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := []model.TopVendorPoint{}
	for rows.Next() {
		var p model.TopVendorPoint
		if err := rows.Scan(&p.VendorID, &p.Email, &p.Sales, &p.RevenueCFA); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
