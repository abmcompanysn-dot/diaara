package model

import "time"

type Payout struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	AmountCFA         int        `json:"amount_cfa"`
	Status            string     `json:"status"`
	PhoneNumber       string     `json:"phone_number"`
	Operator          string     `json:"operator"`
	Provider          string     `json:"provider"` // "pawapay" | "paypal" | "manual" | "kpay"(suspendu), résolu à la création
	ProviderReference *string    `json:"provider_reference,omitempty"`
	FailureReason     *string    `json:"failure_reason,omitempty"`
	RequestedAt       time.Time  `json:"requested_at"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	// Règlement manuel (argent envoyé au vendeur hors PawaPay/KPay) :
	// IsManual + note libre + frais/taxe retenus. Voir PayoutRepo.SettleManually.
	IsManual   bool    `json:"is_manual"`
	ManualNote *string `json:"manual_note,omitempty"`
	FeeCFA     int     `json:"fee_cfa"`
	// Versement PayPal (provider == "paypal") : email destinataire figé à la
	// création + payout_batch_id renvoyé par l'API. Nil pour un versement
	// mobile money. Voir PayoutRepo.CreatePayPal / SetPayPalBatchID.
	PayPalEmail   *string `json:"paypal_email,omitempty"`
	PayPalBatchID *string `json:"paypal_batch_id,omitempty"`
	// Remboursement reconnu a posteriori (payout déjà "paid"/"failed" mais
	// l'argent a été rendu au vendeur/à la plateforme hors plateforme) — voir
	// PayoutRepo.MarkRefunded. Statut "refunded", exclu du solde "requested"
	// au même titre que "failed" (PayoutHandler.totalEarned/Earnings/Create).
	RefundedAt *time.Time `json:"refunded_at,omitempty"`
	RefundedBy *string    `json:"refunded_by,omitempty"`
}

// RefundPayoutInput — reconnaissance admin d'un remboursement de versement déjà
// payé (ou échoué). Note libre optionnelle (raison du remboursement).
type RefundPayoutInput struct {
	Note string `json:"note"`
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

// DirectPayoutInput — versement RÉEL (déclenché via PawaPay, pas juste
// enregistré) depuis le dashboard admin, vers n'importe quel numéro — pas
// forcément un vendeur DIARRA (ex: payer un prestataire externe). Distinct de
// ManualPayoutInput (qui ne fait que consigner un versement déjà fait à la
// main ailleurs) : ici de l'argent part réellement, d'où l'OTPCode (step-up,
// voir OTPPurposeAdminPayout) exigé juste avant l'envoi.
type DirectPayoutInput struct {
	AmountCFA int    `json:"amount_cfa"`
	Country   string `json:"country"`  // ISO 3166-1 alpha-3
	Operator  string `json:"operator"` // code PawaPay, ex "WAVE_SEN" — voir payment.XOFOperators
	Phone     string `json:"phone"`
	Note      string `json:"note,omitempty"`
	OTPCode   string `json:"otp_code"`
}

// CreatePayoutInput — le montant à verser. Le compte mobile money destinataire
// est celui enregistré au préalable via SetPayoutMethodInput (voir plus bas),
// pas resaisi à chaque demande.
type CreatePayoutInput struct {
	AmountCFA int `json:"amount"`
}

// SetPayoutMethodInput — le vendeur enregistre/modifie son moyen de versement.
// Deux canaux possibles, cumulables (PayPal prioritaire quand les deux sont
// renseignés) :
//   - Channel "mobile_money" (défaut si vide) : Phone + Operator + Country
//   - Channel "paypal"                          : PayPalEmail
type SetPayoutMethodInput struct {
	Channel     string `json:"channel"` // "mobile_money" | "paypal"
	Phone       string `json:"phone"`
	Operator    string `json:"operator"`
	Country     string `json:"country"`
	PayPalEmail string `json:"paypal_email"`
}

// PayPalPayoutMinCFA — plancher DIARRA pour un versement PayPal (~1 USD au
// taux XOF de USDRates, arrondi). En dessous, les frais PayPal dépassent
// l'intérêt du versement — l'admin utilise alors le mobile money ou le manuel.
const PayPalPayoutMinCFA = 600
