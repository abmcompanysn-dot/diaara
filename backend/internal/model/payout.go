package model

import "time"

type Payout struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	AmountCFA         int        `json:"amount_cfa"`
	Status            string     `json:"status"`
	PhoneNumber       string     `json:"phone_number"`
	Operator          string     `json:"operator"`
	Provider          string     `json:"provider"` // "pawapay" | "kpay", résolu à la création (voir GatewayOperatorSettingKey)
	ProviderReference *string    `json:"provider_reference,omitempty"`
	FailureReason     *string    `json:"failure_reason,omitempty"`
	RequestedAt       time.Time  `json:"requested_at"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
}

// CreatePayoutInput — le montant à verser. Le compte mobile money destinataire
// est celui enregistré au préalable via SetPayoutMethodInput (voir plus bas),
// pas resaisi à chaque demande.
type CreatePayoutInput struct {
	AmountCFA int `json:"amount"`
}

// SetPayoutMethodInput — le vendeur choisit/modifie le compte mobile money qui
// recevra ses versements (même format que l'ancien checkout : pays + libellé
// opérateur), enregistré une fois pour toutes.
type SetPayoutMethodInput struct {
	Phone    string `json:"phone"`
	Operator string `json:"operator"`
	Country  string `json:"country"`
}
