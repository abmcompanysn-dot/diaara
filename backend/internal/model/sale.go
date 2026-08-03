package model

import "time"

type Sale struct {
	ID                  string     `json:"id"`
	ProductID           string     `json:"product_id"`
	BuyerID             string     `json:"buyer_id"`
	ReferralLinkID      *string    `json:"referral_link_id,omitempty"`
	AmountCFA           int        `json:"amount_cfa"`
	PlatformFeeCFA      int        `json:"platform_fee_cfa"`
	CloserCommissionCFA int        `json:"closer_commission_cfa"`
	VendorAmountCFA     int        `json:"vendor_amount_cfa"`
	PaymentProvider     string     `json:"payment_provider"`
	PaymentReference    string     `json:"payment_reference"`
	Status              string     `json:"status"`
	DeliveredAt         *time.Time `json:"delivered_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type CreateOrderInput struct {
	ProductID string `json:"product_id"`
}

type SaleStatus string

const (
	SalePending   SaleStatus = "pending"
	SalePaid      SaleStatus = "paid"
	SaleFailed    SaleStatus = "failed"
	SaleRefunded  SaleStatus = "refunded"
	SaleDelivered SaleStatus = "delivered"
)
