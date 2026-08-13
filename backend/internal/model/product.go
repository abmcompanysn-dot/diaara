package model

import "time"

type Product struct {
	ID                     string    `json:"id"`
	VendorID               string    `json:"vendor_id"`
	Title                  string    `json:"title"`
	Description            *string   `json:"description,omitempty"`
	PriceCFA               int       `json:"price_cfa"`
	Category               string    `json:"category"`
	FileKey                string    `json:"file_key"`
	CoverImageKey          *string   `json:"cover_image_key,omitempty"`
	ModerationStatus       string    `json:"moderation_status"`
	ModerationNote         *string   `json:"moderation_note,omitempty"`
	AffiliateEnabled       bool      `json:"affiliate_enabled"`
	MaxCloserCommissionPct float64   `json:"max_closer_commission_pct"`
	// PreviewKeys : clés objet des aperçus filigranés générés automatiquement
	// (1 à 3 selon le type de fichier : pages PDF, image réduite, extrait
	// audio/vidéo). Vide tant que preview_status vaut "pending".
	PreviewKeys   []string  `json:"preview_keys"`
	PreviewStatus string    `json:"preview_status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateProductInput struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	PriceCFA    int     `json:"price_cfa"`
	Category    string  `json:"category"`
	FileKey     string  `json:"file_key"`
	// CoverImageKey : clé objet de l'image de couverture (optionnel).
	CoverImageKey string `json:"cover_image_key,omitempty"`
	// Affiliation (closer) : si AffiliateEnabled, les closers peuvent générer
	// un lien avec une commission ≤ MaxCloserCommissionPct.
	AffiliateEnabled       bool    `json:"affiliate_enabled"`
	MaxCloserCommissionPct float64 `json:"max_closer_commission_pct"`
}

type UpdateProductInput struct {
	Title                  *string  `json:"title,omitempty"`
	Description            *string  `json:"description,omitempty"`
	PriceCFA               *int     `json:"price_cfa,omitempty"`
	Category               *string  `json:"category,omitempty"`
	CoverImageKey          *string  `json:"cover_image_key,omitempty"`
	AffiliateEnabled       *bool    `json:"affiliate_enabled,omitempty"`
	MaxCloserCommissionPct *float64 `json:"max_closer_commission_pct,omitempty"`
}

type ModerateProductInput struct {
	Status string  `json:"status"`
	Note   *string `json:"note,omitempty"`
}
