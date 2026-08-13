package model

import "time"

// Scopes de permission admin (fixes, pas un moteur de permissions générique).
// Un admin sans aucune ligne admin_permissions a l'accès complet (legacy).
const (
	AdminPermModeration = "moderation"
	AdminPermUsers      = "users"
	AdminPermFinance    = "finance"
	AdminPermInfra      = "infra"
)

// ValidAdminPermission retourne true si le scope fait partie des scopes attribuables.
func ValidAdminPermission(perm string) bool {
	switch perm {
	case AdminPermModeration, AdminPermUsers, AdminPermFinance, AdminPermInfra:
		return true
	default:
		return false
	}
}

type AdminPermissionInput struct {
	Permission string `json:"permission"`
	Action     string `json:"action"` // "grant" ou "revoke"
}

// AdminSummary représente un admin et ses scopes pour la page de gestion des accès.
type AdminSummary struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Permissions []string  `json:"permissions"` // vide = accès complet
	CreatedAt   time.Time `json:"created_at"`
}

type AdminStatusInput struct {
	Action string `json:"action"` // "grant" ou "revoke"
}
