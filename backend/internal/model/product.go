package model

import "time"

type Product struct {
	ID          string  `json:"id"`
	VendorID    string  `json:"vendor_id"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	PriceCFA    int     `json:"price_cfa"`
	// PriceMode : "fixed" (défaut) ou "flexible" ("paye ce que tu veux").
	// MinPriceCFA n'a de sens qu'en mode flexible.
	PriceMode     string  `json:"price_mode"`
	MinPriceCFA   *int    `json:"min_price_cfa,omitempty"`
	Category      string  `json:"category"`
	FileKey       string  `json:"file_key"`
	CoverImageKey *string `json:"cover_image_key,omitempty"`
	// ImagePrompt : prompt de génération d'image renseigné par une création
	// automatisée (IA/script) qui n'a pas encore de fichier — un modérateur
	// génère l'image à partir de ce prompt puis l'attache avant d'approuver.
	ImagePrompt            *string `json:"image_prompt,omitempty"`
	ModerationStatus       string  `json:"moderation_status"`
	ModerationNote         *string `json:"moderation_note,omitempty"`
	AffiliateEnabled       bool    `json:"affiliate_enabled"`
	MaxCloserCommissionPct float64 `json:"max_closer_commission_pct"`
	// PreviewKeys : clés objet des aperçus filigranés générés automatiquement
	// (1 à 3 selon le type de fichier : pages PDF, image réduite, extrait
	// audio/vidéo). Vide tant que preview_status vaut "pending".
	PreviewKeys   []string  `json:"preview_keys"`
	PreviewStatus string    `json:"preview_status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AdminProduct — un produit avec l'email du vendeur, pour l'écran de
// modération admin (évite d'afficher un UUID brut à la place du vendeur).
type AdminProduct struct {
	Product
	VendorEmail string `json:"vendor_email"`
}

type CreateProductInput struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	PriceCFA    int     `json:"price_cfa"`
	// PriceMode : "fixed" (défaut si vide) ou "flexible". MinPriceCFA requis
	// si "flexible".
	PriceMode   string `json:"price_mode,omitempty"`
	MinPriceCFA *int   `json:"min_price_cfa,omitempty"`
	Category    string `json:"category"`
	FileKey     string `json:"file_key"`
	// CoverImageKey : clé objet de l'image de couverture (optionnel).
	CoverImageKey string `json:"cover_image_key,omitempty"`
	// ImagePrompt : voir Product.ImagePrompt.
	ImagePrompt *string `json:"image_prompt,omitempty"`
	// Affiliation (closer) : si AffiliateEnabled, les closers peuvent générer
	// un lien avec une commission ≤ MaxCloserCommissionPct.
	AffiliateEnabled       bool    `json:"affiliate_enabled"`
	MaxCloserCommissionPct float64 `json:"max_closer_commission_pct"`
}

type UpdateProductInput struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	PriceCFA    *int    `json:"price_cfa,omitempty"`
	PriceMode   *string `json:"price_mode,omitempty"`
	MinPriceCFA *int    `json:"min_price_cfa,omitempty"`
	Category    *string `json:"category,omitempty"`
	// FileKey : remplace le fichier livré au client (le vendeur ré-uploade
	// via /api/vendor/products/upload puis passe la nouvelle clé ici).
	FileKey                *string  `json:"file_key,omitempty"`
	CoverImageKey          *string  `json:"cover_image_key,omitempty"`
	AffiliateEnabled       *bool    `json:"affiliate_enabled,omitempty"`
	MaxCloserCommissionPct *float64 `json:"max_closer_commission_pct,omitempty"`
}

type ModerateProductInput struct {
	Status string  `json:"status"`
	Note   *string `json:"note,omitempty"`
}
