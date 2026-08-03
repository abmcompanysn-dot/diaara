package model

import "time"

type DeliveryLink struct {
	ID            string     `json:"id"`
	SaleID        string     `json:"sale_id"`
	SignedURLToken string    `json:"signed_url_token"`
	MaxDownloads  int        `json:"max_downloads"`
	DownloadCount int        `json:"download_count"`
	ExpiresAt     time.Time  `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
}
