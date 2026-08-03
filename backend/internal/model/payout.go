package model

import "time"

type Payout struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	AmountCFA   int        `json:"amount_cfa"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
}

type CreatePayoutInput struct {
	AmountCFA int `json:"amount"`
}
