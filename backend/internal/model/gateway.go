package model

import "time"

// GatewayClient — une application externe autorisée à utiliser DIARRA comme
// passerelle de paiement (ex. ABMCY Core) au lieu d'intégrer PawaPay/KPay/
// PayPal directement de son côté. Voir migration 029 et
// internal/payment/provider.go (interface PaymentProvider déjà unifiée entre
// les 3 agrégateurs, réutilisée telle quelle ici).
type GatewayClient struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	APIKeyHash      string    `json:"-"`
	// HMACSecretHash : jamais le secret en clair, même principe que
	// APIKeyHash. Sert à vérifier X-Diarra-Signature sur chaque requête
	// entrante (voir middleware.RequireGatewayClient et
	// migration 030_gateway_hmac_secret.sql) — une clé API seule suffit à
	// authentifier QUI appelle, la signature garantit en plus que le CORPS
	// n'a pas été altéré en chemin (même sur TLS, une seconde ligne de
	// défense contre un secret de clé API qui fuirait sans le secret HMAC).
	HMACSecretHash  string    `json:"-"`
	DefaultCallbackURL *string   `json:"default_callback_url,omitempty"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Types et statuts d'une transaction de passerelle — vocabulaire commun déjà
// utilisé par payment.PaymentProvider (DepositOutcome.Status, etc.), repris
// tel quel pour ne pas introduire un second vocabulaire à traduire.
const (
	GatewayTxTypeDeposit = "deposit"
	GatewayTxTypePayout  = "payout"
	GatewayTxTypeRefund  = "refund"

	GatewayTxPending    = "pending"
	GatewayTxProcessing = "processing"
	GatewayTxCompleted  = "completed"
	GatewayTxFailed     = "failed"
	GatewayTxCancelled  = "cancelled"
)

// GatewayTransaction — un dépôt, versement ou remboursement initié par un
// client externe via la passerelle. client_ref est l'identifiant CHEZ LE
// CLIENT (ex. une commande ABMCY Core), jamais réutilisé côté DIARRA.
type GatewayTransaction struct {
	ID                 string    `json:"id"`
	ClientID           string    `json:"-"` // jamais exposé au client : implicite (son propre token)
	ClientRef          string    `json:"client_ref"`
	Type               string    `json:"type"`
	Provider           string    `json:"provider"`
	ProviderRef        *string   `json:"provider_ref,omitempty"`
	RelatedDepositRef  *string   `json:"-"`
	Status             string    `json:"status"`
	FailureReason      *string   `json:"failure_reason,omitempty"`
	AmountCFA          int       `json:"amount_cfa"`
	Currency           string    `json:"currency"`
	RecipientPhone     *string   `json:"recipient_phone,omitempty"`
	RecipientOperator  *string   `json:"recipient_operator,omitempty"`
	Country            *string   `json:"country,omitempty"`
	Description        *string   `json:"description,omitempty"`
	CallbackURL         *string   `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	// RedirectURL n'est jamais stocké : renvoyé une seule fois, à la création
	// d'un dépôt (page de paiement hébergée PawaPay/KPay). Rempli par le
	// handler, pas par le repo.
	RedirectURL string `json:"redirect_url,omitempty"`
}

// GatewayCreateDepositInput — POST /api/gateway/v1/deposits.
type GatewayCreateDepositInput struct {
	ClientRef   string `json:"client_ref"`
	AmountCFA   int    `json:"amount_cfa"`
	Country     string `json:"country"` // ISO 3166-1 alpha-3, requis (page hébergée PawaPay)
	Description string `json:"description,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"` // surcharge default_callback_url du client
	// ReturnURL : où renvoyer le NAVIGATEUR de l'utilisateur après la page de
	// paiement hébergée. Le client (ex. ABMCY Core) passe sa propre page de
	// suivi ; sinon on retombe sur /gateway/return du frontend DIARRA.
	ReturnURL string `json:"return_url,omitempty"`
}

// CreatePayoutInput — POST /api/gateway/v1/payouts.
type GatewayCreatePayoutInput struct {
	ClientRef       string `json:"client_ref"`
	AmountCFA       int    `json:"amount_cfa"`
	RecipientPhone  string `json:"recipient_phone"`
	RecipientOperator string `json:"recipient_operator"` // code PawaPay, ex "WAVE_SEN" — voir payment.XOFOperators
	Country         string `json:"country"`
	CallbackURL     string `json:"callback_url,omitempty"`
}

// GatewayCreateRefundInput — POST /api/gateway/v1/refunds.
type GatewayCreateRefundInput struct {
	ClientRef      string `json:"client_ref"`
	DepositClientRef string `json:"deposit_client_ref"` // client_ref du dépôt à rembourser
	CallbackURL    string `json:"callback_url,omitempty"`
}
