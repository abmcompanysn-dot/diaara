package model

import "time"

// AdminActivityLog — une ligne du journal d'activité admin (backoffice 360°).
// AdminID nullable : certaines actions sont automatiques (cron de
// réconciliation) sans admin humain à l'origine — voir migration 028.
type AdminActivityLog struct {
	ID          string    `json:"id"`
	AdminID     *string   `json:"admin_id,omitempty"`
	AdminEmail  string    `json:"admin_email,omitempty"` // jointure users, vide si AdminID nil
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    *string   `json:"target_id,omitempty"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
