package model

import "time"

type ProductBundle struct {
	ID               string    `json:"id"`
	VendorID         string    `json:"vendor_id"`
	Title            string    `json:"title"`
	Description      *string   `json:"description,omitempty"`
	PriceCFA         int       `json:"price_cfa"`
	CoverImageKey    *string   `json:"cover_image_key,omitempty"`
	ModerationStatus string    `json:"moderation_status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateBundleInput struct {
	Title         string   `json:"title"`
	Description   *string  `json:"description,omitempty"`
	PriceCFA      int      `json:"price_cfa"`
	CoverImageKey string   `json:"cover_image_key,omitempty"`
	ProductIDs    []string `json:"product_ids"`
}
