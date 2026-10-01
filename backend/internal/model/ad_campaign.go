package model

import "time"

// AdCampaign — sponsorisation d'un produit sur Meta (Facebook + Instagram),
// lancée depuis DIARRA sur le compte publicitaire DU VENDEUR, qui paie Meta
// directement. Voir migration 048 pour le détail des colonnes et statuts.
type AdCampaign struct {
	ID                 string     `json:"id"`
	VendorID           string     `json:"vendor_id"`
	ProductID          string     `json:"product_id"`
	Platform           string     `json:"platform"`
	Status             string     `json:"status"`
	BudgetCFA          int        `json:"budget_cfa"`
	AdAccountID        string     `json:"ad_account_id"`
	PageID             string     `json:"page_id"`
	Currency           string     `json:"currency"`
	DurationDays       int        `json:"duration_days"`
	Countries          []string   `json:"countries"`
	Message            string     `json:"message"`
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
	BudgetCFA    int      `json:"budget_cfa"` // budget total, facturé par Meta au vendeur
	DurationDays int      `json:"duration_days"`
	Countries    []string `json:"countries"` // ISO 3166-1 alpha-2 (format Meta), ex "SN"
	Message      string   `json:"message"`
}

// VendorMetaConnection — compte Facebook connecté par le vendeur (Facebook
// Login) et ses choix de page / compte publicitaire. Le jeton (chiffré) n'est
// jamais sérialisé.
type VendorMetaConnection struct {
	VendorID       string     `json:"-"`
	FBUserID       string     `json:"fb_user_id"`
	FBUserName     string     `json:"fb_user_name"`
	AccessTokenEnc string     `json:"-"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
	PageID         *string    `json:"page_id,omitempty"`
	PageName       *string    `json:"page_name,omitempty"`
	AdAccountID    *string    `json:"ad_account_id,omitempty"`
	AdAccountName  *string    `json:"ad_account_name,omitempty"`
	Currency       *string    `json:"currency,omitempty"`
	NeedsReconnect bool       `json:"needs_reconnect"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Ready — page et compte publicitaire choisis, jeton valide.
func (c *VendorMetaConnection) Ready() bool {
	return c != nil && !c.NeedsReconnect && c.PageID != nil && *c.PageID != "" &&
		c.AdAccountID != nil && *c.AdAccountID != "" && c.Currency != nil && *c.Currency != ""
}

// SettingAdsEnabled — interrupteur général (table settings, "true"/"false",
// "true" par défaut) : coupe la création de nouvelles campagnes sans toucher
// au serveur. Sans configuration Meta (META_APP_* et
// META_TOKEN_ENCRYPTION_KEY), la fonctionnalité reste de toute façon
// indisponible.
const SettingAdsEnabled = "ads_enabled"

const (
	// AdsMinDailyCFA — budget minimum PAR JOUR, pour rester au-dessus du
	// minimum journalier exigé par Meta (≈ 1 € / 1 $ par jour selon la devise).
	AdsMinDailyCFA     = 1000
	AdsMaxDurationDays = 30
	// AdsMaxBudgetCFA — garde-fou contre une faute de frappe (le vendeur paie
	// lui-même Meta, mais un zéro de trop coûterait cher).
	AdsMaxBudgetCFA  = 2_000_000
	AdsMessageMaxLen = 500
)

// AdCountries — pays ciblables (ISO2 Meta -> libellé), alignés sur les pays
// où DIARRA vend. Miroir côté frontend : renvoyé par /api/vendor/ads/config.
var AdCountries = map[string]string{
	"SN": "Sénégal", "CI": "Côte d'Ivoire", "BJ": "Bénin", "TG": "Togo",
	"ML": "Mali", "BF": "Burkina Faso", "NE": "Niger", "GN": "Guinée",
	"CM": "Cameroun", "GA": "Gabon", "CG": "Congo-Brazzaville", "CD": "RD Congo",
	"KE": "Kenya", "RW": "Rwanda", "UG": "Ouganda", "ZM": "Zambie",
	"SL": "Sierra Leone", "MA": "Maroc", "FR": "France", "BE": "Belgique",
	"CA": "Canada", "US": "États-Unis",
}
