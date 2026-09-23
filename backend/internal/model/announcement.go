package model

import "time"

// Announcement — message admin visible sur le dashboard vendeur ("Dernières
// mises à jour"), voir migrations/040_announcements.sql. Pas de notification
// email : lecture passive au prochain chargement du dashboard.
type Announcement struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateAnnouncementInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
