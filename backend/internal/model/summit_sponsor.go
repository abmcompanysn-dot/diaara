package model

import "time"

// SummitSponsorTier — palier de sponsoring du DIARRA Summit (Bronze/Argent/
// Or...), éditable depuis l'admin (voir migration 036_summit_sponsors.sql).
type SummitSponsorTier struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	PriceCFA  int       `json:"price_cfa"`
	Perks     []string  `json:"perks"`
	Highlight bool      `json:"highlight"`
	SortOrder int16     `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateSummitSponsorTierInput struct {
	Name      string   `json:"name"`
	PriceCFA  int      `json:"price_cfa"`
	Perks     []string `json:"perks"`
	Highlight bool     `json:"highlight"`
	SortOrder int16    `json:"sort_order"`
}

type UpdateSummitSponsorTierInput struct {
	Name      *string   `json:"name,omitempty"`
	PriceCFA  *int      `json:"price_cfa,omitempty"`
	Perks     *[]string `json:"perks,omitempty"`
	Highlight *bool     `json:"highlight,omitempty"`
	SortOrder *int16    `json:"sort_order,omitempty"`
}

// SummitSponsor — entreprise sponsor réelle, affichée sur /summit/sponsors
// une fois published=true (validation admin, voir le formulaire "Devenir
// sponsor" qui ne crée pas directement de ligne publiée).
type SummitSponsor struct {
	ID         string    `json:"id"`
	TierID     *string   `json:"tier_id,omitempty"`
	Name       string    `json:"name"`
	LogoKey    *string   `json:"logo_key,omitempty"`
	WebsiteURL *string   `json:"website_url,omitempty"`
	Published  bool      `json:"published"`
	SortOrder  int16     `json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateSummitSponsorInput struct {
	TierID     *string `json:"tier_id,omitempty"`
	Name       string  `json:"name"`
	LogoKey    *string `json:"logo_key,omitempty"`
	WebsiteURL *string `json:"website_url,omitempty"`
	Published  bool    `json:"published"`
	SortOrder  int16   `json:"sort_order,omitempty"`
}

type UpdateSummitSponsorInput struct {
	TierID     *string `json:"tier_id,omitempty"`
	Name       *string `json:"name,omitempty"`
	LogoKey    *string `json:"logo_key,omitempty"`
	WebsiteURL *string `json:"website_url,omitempty"`
	Published  *bool   `json:"published,omitempty"`
	SortOrder  *int16  `json:"sort_order,omitempty"`
}
