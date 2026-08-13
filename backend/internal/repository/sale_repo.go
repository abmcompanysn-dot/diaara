package repository

import (
	"context"
	"errors"

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

const saleColumns = `id, product_id, buyer_id, buyer_name, referral_link_id, amount_cfa, platform_fee_cfa,
	closer_commission_cfa, vendor_amount_cfa, payment_provider, payment_reference, checkout_token, status, refund_reference, delivered_at, created_at`

func scanSale(row pgx.Row) (*model.Sale, error) {
	s := &model.Sale{}
	err := row.Scan(&s.ID, &s.ProductID, &s.BuyerID, &s.BuyerName, &s.ReferralLinkID, &s.AmountCFA,
		&s.PlatformFeeCFA, &s.CloserCommissionCFA, &s.VendorAmountCFA, &s.PaymentProvider,
		&s.PaymentReference, &s.CheckoutToken, &s.Status, &s.RefundReference, &s.DeliveredAt, &s.CreatedAt)
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
		`INSERT INTO sales (product_id, buyer_id, buyer_name, referral_link_id, amount_cfa, platform_fee_cfa,
			closer_commission_cfa, vendor_amount_cfa, payment_provider, payment_reference, checkout_token, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING `+saleColumns,
		s.ProductID, s.BuyerID, s.BuyerName, s.ReferralLinkID, s.AmountCFA, s.PlatformFeeCFA,
		s.CloserCommissionCFA, s.VendorAmountCFA, s.PaymentProvider, s.PaymentReference, s.CheckoutToken, s.Status)
	return scanSale(row)
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

func (r *SaleRepo) UpdatePaymentReference(ctx context.Context, id, ref string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sales SET payment_reference = $2 WHERE id = $1`, id, ref)
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
			AND created_at >= now() - ($1 || ' days')::interval
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
