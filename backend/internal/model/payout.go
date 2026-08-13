package model

import "time"

type Payout struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	AmountCFA       int        `json:"amount_cfa"`
	Status          string     `json:"status"`
	PhoneNumber     string     `json:"phone_number"`
	Operator        string     `json:"operator"`
	PawaPayPayoutID *string    `json:"pawapay_payout_id,omitempty"`
	FailureReason   *string    `json:"failure_reason,omitempty"`
	RequestedAt     time.Time  `json:"requested_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
}

// CreatePayoutInput — le vendeur choisit le compte mobile money qui recevra
// le versement (même format que le checkout : pays + libellé opérateur).
type CreatePayoutInput struct {
	AmountCFA int    `json:"amount"`
	Phone     string `json:"phone"`
	Operator  string `json:"operator"`
	Country   string `json:"country"`
}
