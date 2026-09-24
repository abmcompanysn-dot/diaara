package model

import "time"

// Announcement — message admin visible sur le dashboard vendeur ("Dernières
// mises à jour"), voir migrations/040_announcements.sql. Déclenche une
// notification push à tous les vendeurs (voir migrations/043, handler.Create
// — h.pushSvc), avec image optionnelle affichée dans la notification.
type Announcement struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	ImageKey  *string   `json:"image_key,omitempty"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateAnnouncementInput struct {
	Title    string  `json:"title"`
	Body     string  `json:"body"`
	ImageKey *string `json:"image_key,omitempty"`
}
