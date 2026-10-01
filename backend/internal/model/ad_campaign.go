package model

import "time"

// AdCampaign — sponsorisation d'un produit sur une régie publicitaire
// (Meta : Facebook + Instagram), voir migration 048 pour le détail des
// montants et des statuts.
type AdCampaign struct {
	ID                 string     `json:"id"`
	VendorID           string     `json:"vendor_id"`
	ProductID          string     `json:"product_id"`
	Platform           string     `json:"platform"`
	Status             string     `json:"status"`
	AmountCFA          int        `json:"amount_cfa"`
	CommissionCFA      int        `json:"commission_cfa"`
	AdBudgetCFA        int        `json:"ad_budget_cfa"`
	DurationDays       int        `json:"duration_days"`
	Countries          []string   `json:"countries"`
	Message            string     `json:"message"`
	PaymentMethod      string     `json:"payment_method"`
	Refunded           bool       `json:"refunded"`
	StartsAt           *time.Time `json:"starts_at,omitempty"`
	EndsAt             *time.Time `json:"ends_at,omitempty"`
	ExternalCampaignID *string    `json:"-"`
	ExternalAdSetID    *string    `json:"-"`
	ExternalCreativeID *string    `json:"-"`
	ExternalAdID       *string    `json:"-"`
	FailureReason      *string    `json:"failure_reason,omitempty"`
	Impressions        int64      `json:"impressions"`
	Reach              int64      `json:"reach"`
	Clicks             int64      `json:"clicks"`
	SpendCFA           int        `json:"spend_cfa"`
	StatsUpdatedAt     *time.Time `json:"stats_updated_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// Jointures d'affichage (listes vendeur/admin), vides sinon.
	ProductTitle string `json:"product_title,omitempty"`
	VendorEmail  string `json:"vendor_email,omitempty"`
}

const (
	AdStatusLaunching = "launching"
	AdStatusInReview  = "in_review"
	AdStatusActive    = "active"
	AdStatusCompleted = "completed"
	AdStatusStopped   = "stopped"
	AdStatusRejected  = "rejected"
	AdStatusFailed    = "failed"
)

type CreateAdCampaignInput struct {
	ProductID    string   `json:"product_id"`
	AmountCFA    int      `json:"amount_cfa"`
	DurationDays int      `json:"duration_days"`
	Countries    []string `json:"countries"` // ISO 3166-1 alpha-2 (format Meta), ex "SN"
	Message      string   `json:"message"`
}

// Réglages admin (table settings) de la sponsorisation.
const (
	// SettingAdsEnabled — interrupteur général ("true"/"false", "true" par
	// défaut) : coupe la création de nouvelles campagnes sans toucher au
	// serveur. Sans configuration Meta (META_ADS_*), la fonctionnalité reste
	// de toute façon indisponible.
	SettingAdsEnabled = "ads_enabled"
	// SettingAdsCommissionPct — part DIARRA prélevée sur le montant payé par
	// le vendeur (couvre le change, les frais de carte et la marge).
	SettingAdsCommissionPct = "ads_commission_pct"
	// SettingAdsMinDailyCFA — budget publicitaire minimum PAR JOUR (après
	// commission), pour rester au-dessus du minimum journalier exigé par Meta.
	SettingAdsMinDailyCFA = "ads_min_daily_cfa"
)

const (
	DefaultAdsCommissionPct = 20.0
	DefaultAdsMinDailyCFA   = 1000
	AdsMaxDurationDays      = 30
	AdsMaxAmountCFA         = 2_000_000
	AdsMessageMaxLen        = 500
)

// AdCountries — pays ciblables (ISO2 Meta -> libellé), alignés sur les pays
// où DIARRA vend. Miroir de AD_COUNTRIES côté frontend (lib/ads.ts).
var AdCountries = map[string]string{
	"SN": "Sénégal", "CI": "Côte d'Ivoire", "BJ": "Bénin", "TG": "Togo",
	"ML": "Mali", "BF": "Burkina Faso", "NE": "Niger", "GN": "Guinée",
	"CM": "Cameroun", "GA": "Gabon", "CG": "Congo-Brazzaville", "CD": "RD Congo",
	"KE": "Kenya", "RW": "Rwanda", "UG": "Ouganda", "ZM": "Zambie",
	"SL": "Sierra Leone", "MA": "Maroc", "FR": "France", "BE": "Belgique",
	"CA": "Canada", "US": "États-Unis",
}
