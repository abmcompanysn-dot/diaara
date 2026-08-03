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

const saleColumns = `id, product_id, buyer_id, referral_link_id, amount_cfa, platform_fee_cfa,
	closer_commission_cfa, vendor_amount_cfa, payment_provider, payment_reference, status, delivered_at, created_at`

func scanSale(row pgx.Row) (*model.Sale, error) {
	s := &model.Sale{}
	err := row.Scan(&s.ID, &s.ProductID, &s.BuyerID, &s.ReferralLinkID, &s.AmountCFA,
		&s.PlatformFeeCFA, &s.CloserCommissionCFA, &s.VendorAmountCFA, &s.PaymentProvider,
		&s.PaymentReference, &s.Status, &s.DeliveredAt, &s.CreatedAt)
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
		`INSERT INTO sales (product_id, buyer_id, referral_link_id, amount_cfa, platform_fee_cfa,
			closer_commission_cfa, vendor_amount_cfa, payment_provider, payment_reference, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING `+saleColumns,
		s.ProductID, s.BuyerID, s.ReferralLinkID, s.AmountCFA, s.PlatformFeeCFA,
		s.CloserCommissionCFA, s.VendorAmountCFA, s.PaymentProvider, s.PaymentReference, s.Status)
	return scanSale(row)
}

func (r *SaleRepo) FindByID(ctx context.Context, id string) (*model.Sale, error) {
	return scanSale(r.pool.QueryRow(ctx,
		`SELECT `+saleColumns+` FROM sales WHERE id = $1`, id))
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
