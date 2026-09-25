package model

import "time"

type Sale struct {
	ID                  string  `json:"id"`
	ProductID           string  `json:"product_id"`
	BuyerID             string  `json:"buyer_id"`
	BuyerName           string  `json:"buyer_name"`
	Country             *string `json:"country,omitempty"`
	ReferralLinkID      *string `json:"referral_link_id,omitempty"`
	AmountCFA           int     `json:"amount_cfa"`
	PlatformFeeCFA      int     `json:"platform_fee_cfa"`
	CloserCommissionCFA int     `json:"closer_commission_cfa"`
	VendorAmountCFA     int     `json:"vendor_amount_cfa"`
	PaymentProvider     string  `json:"payment_provider"`
	PaymentReference    string  `json:"payment_reference"`
	// ProviderTransactionID : ID propre à PayPal (retourné à l'initiation,
	// remplacé par l'ID de CAPTURE une fois le paiement confirmé), nécessaire
	// pour ses appels GET statut/remboursement — reste nil pour une vente
	// PawaPay/PayDunya (payment_reference est déjà l'identifiant utilisé pour
	// ces appels).
	ProviderTransactionID *string    `json:"provider_transaction_id,omitempty"`
	CheckoutToken         *string    `json:"checkout_token,omitempty"`
	Status                string     `json:"status"`
	RefundReference       *string    `json:"refund_reference,omitempty"`
	DeliveredAt           *time.Time `json:"delivered_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	// Relance des commandes restées "pending" : date de la dernière relance
	// envoyée à l'acheteur et nombre total de relances (voir SaleRepo.MarkReminded).
	RemindedAt    *time.Time `json:"reminded_at,omitempty"`
	ReminderCount int        `json:"reminder_count"`
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
	// PaymentMethod : "mobile_money" (défaut, vide accepté) | "card" | "paypal".
	// Carte/PayPal forcent PayPal (ni PawaPay ni PayDunya ne la supportent) —
	// voir SaleHandler.initiateCheckout.
	PaymentMethod string `json:"payment_method,omitempty"`
	// Phone/Operator : requis pour mobile_money — dépôt PawaPay direct
	// (POST /v2/deposits) au lieu de la Payment Page hébergée, l'acheteur
	// choisit son opérateur et saisit son numéro directement sur DIARRA.
	// Phone : chiffres locaux uniquement, sans l'indicatif (voir
	// payment.NormalizePhone qui l'ajoute à partir du pays).
	// Operator : code PawaPay, ex "ORANGE_SEN" (voir payment.XOFOperators).
	Phone    string `json:"phone,omitempty"`
	Operator string `json:"operator,omitempty"`
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
