package model

import "time"

// ReferralLink est un lien d'affiliation généré par un closer pour un produit.
type ReferralLink struct {
	ID            string    `json:"id"`
	ProductID     string    `json:"product_id"`
	ProductTitle  string    `json:"product_title,omitempty"`
	CloserID      string    `json:"closer_id"`
	Slug          string    `json:"slug"`
	CommissionPct float64   `json:"commission_pct"`
	Clicks        int       `json:"clicks"`
	CreatedAt     time.Time `json:"created_at"`
	// Stats agrégées (remplies par ListByCloser).
	Sales       int `json:"sales,omitempty"`
	RevenueCFA  int `json:"revenue_cfa,omitempty"`
	Commissions int `json:"commissions_cfa,omitempty"`
}

type CreateReferralLinkInput struct {
	ProductID     string  `json:"product_id"`
	CommissionPct float64 `json:"commission_pct"`
	// Slug est généré par le backend avant l'insertion (champ interne).
	Slug string `json:"-"`
}
