package model

import "time"

// DonationPool — cagnotte alimentée par une part de la commission DIARRA
// sur chaque vente payée (voir DonationService.Accumulate). Une seule ligne
// en base (id=1) ; le solde n'est jamais tenu en mémoire, le backend
// tournant en plusieurs replicas.
type DonationPool struct {
	BalanceCFA int       `json:"balance_cfa"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// DonationRecipient — bénéficiaire externe (association, personne) qui
// reçoit une part égale de la cagnotte à chaque distribution automatique.
// Distinct de `users`/`payouts` : ce ne sont pas des vendeurs de la plateforme.
type DonationRecipient struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	PhoneNumber string    `json:"phone_number"`
	Operator    string    `json:"operator"`
	Country     string    `json:"country"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateDonationRecipientInput struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Operator string `json:"operator"`
	Country  string `json:"country"`
}

type UpdateDonationRecipientInput struct {
	Name   *string `json:"name,omitempty"`
	Active *bool   `json:"active,omitempty"`
}

// DonationPayout — un versement mobile money vers un DonationRecipient,
// émis lors d'une distribution automatique (voir DonationService.Distribute).
// Même mécanique que model.Payout (statuts, référence PawaPay) mais sans le
// lien vers `users` : les destinataires sont externes à la plateforme.
type DonationPayout struct {
	ID              string     `json:"id"`
	RecipientID     string     `json:"recipient_id"`
	AmountCFA       int        `json:"amount_cfa"`
	Status          string     `json:"status"`
	PawaPayPayoutID *string    `json:"pawapay_payout_id,omitempty"`
	FailureReason   *string    `json:"failure_reason,omitempty"`
	RequestedAt     time.Time  `json:"requested_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
}

// DonationPayoutWithRecipient — versement enrichi du nom/téléphone du
// destinataire, pour l'historique affiché dans l'admin.
type DonationPayoutWithRecipient struct {
	DonationPayout
	RecipientName  string `json:"recipient_name"`
	RecipientPhone string `json:"recipient_phone"`
}
