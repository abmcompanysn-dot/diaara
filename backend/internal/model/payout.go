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
	// Règlement manuel (argent envoyé au vendeur hors PawaPay/KPay) :
	// IsManual + note libre + frais/taxe retenus. Voir PayoutRepo.SettleManually.
	IsManual   bool    `json:"is_manual"`
	ManualNote *string `json:"manual_note,omitempty"`
	FeeCFA     int     `json:"fee_cfa"`
}

// SettlePayoutInput — règlement manuel d'un versement existant.
type SettlePayoutInput struct {
	Note   string `json:"note"`
	FeeCFA int    `json:"fee_cfa"`
}

// ManualPayoutInput — création d'un versement manuel de toutes pièces.
type ManualPayoutInput struct {
	UserID    string `json:"user_id"`
	AmountCFA int    `json:"amount"`
	FeeCFA    int    `json:"fee_cfa"`
	Phone     string `json:"phone"`
	Note      string `json:"note"`
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
