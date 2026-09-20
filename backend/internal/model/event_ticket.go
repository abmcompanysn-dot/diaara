package model

import "time"

// EventTicketCheckin — état de vérification à l'entrée d'un billet
// d'événement payant (une ligne par Sale, voir migration 035). Créée à la
// demande, la première fois que le PDF du billet est généré (voir
// EventTicketHandler.PDF) — pas au moment du paiement, pour ne pas toucher
// au chemin critique du webhook.
type EventTicketCheckin struct {
	ID           string     `json:"id"`
	SaleID       string     `json:"sale_id"`
	CheckInToken string     `json:"check_in_token"`
	CheckedInAt  *time.Time `json:"checked_in_at,omitempty"`
	CheckedInBy  *string    `json:"checked_in_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
