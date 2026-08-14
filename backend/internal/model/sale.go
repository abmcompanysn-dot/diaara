package model

import "time"

type Sale struct {
	ID                  string     `json:"id"`
	ProductID           string     `json:"product_id"`
	BuyerID             string     `json:"buyer_id"`
	BuyerName           string     `json:"buyer_name"`
	Country             *string    `json:"country,omitempty"`
	ReferralLinkID      *string    `json:"referral_link_id,omitempty"`
	AmountCFA           int        `json:"amount_cfa"`
	PlatformFeeCFA      int        `json:"platform_fee_cfa"`
	CloserCommissionCFA int        `json:"closer_commission_cfa"`
	VendorAmountCFA     int        `json:"vendor_amount_cfa"`
	PaymentProvider     string     `json:"payment_provider"`
	PaymentReference    string     `json:"payment_reference"`
	CheckoutToken       *string    `json:"checkout_token,omitempty"`
	Status              string     `json:"status"`
	RefundReference     *string    `json:"refund_reference,omitempty"`
	DeliveredAt         *time.Time `json:"delivered_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type CreateOrderInput struct {
	ProductID      string  `json:"product_id"`
	BuyerName      string  `json:"buyer_name"`
	BuyerEmail     *string `json:"buyer_email,omitempty"`
	Country        string  `json:"country"` // ISO 3166-1 alpha-3, ex "SEN" — requis par la Payment Page PawaPay
	ReferralLinkID *string `json:"referral_link_id,omitempty"`
	// AmountCFA : montant choisi par l'acheteur, pris en compte UNIQUEMENT si
	// le produit est en price_mode "flexible" (et doit alors être >=
	// MinPriceCFA). Pour un produit à prix fixe, ce champ est ignoré et le
	// serveur utilise toujours product.PriceCFA — ne jamais faire confiance
	// à un montant client pour un produit à prix fixe.
	AmountCFA *int `json:"amount_cfa,omitempty"`
}

type SaleStatus string

const (
	SalePending       SaleStatus = "pending"
	SalePaid          SaleStatus = "paid"
	SaleFailed        SaleStatus = "failed"
	SaleRefundPending SaleStatus = "refund_pending"
	SaleRefunded      SaleStatus = "refunded"
	SaleDelivered     SaleStatus = "delivered"
)
