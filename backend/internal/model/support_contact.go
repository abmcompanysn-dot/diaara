package model

import "time"

// SupportAgent — membre de l'équipe support qui reçoit un email à chaque
// nouveau contact via le widget public du site (bouton "Contacter le support").
type SupportAgent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateSupportAgentInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateSupportAgentInput struct {
	Name   *string `json:"name,omitempty"`
	Email  *string `json:"email,omitempty"`
	Active *bool   `json:"active,omitempty"`
}

// SupportContactRequest — message envoyé par un visiteur (non authentifié)
// depuis le widget de contact public. contact_method détermine comment les
// agents support peuvent répondre (lien mailto ou WhatsApp).
type SupportContactRequest struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	ContactMethod string    `json:"contact_method"`
	ContactValue  string    `json:"contact_value"`
	Message       string    `json:"message"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateSupportContactInput struct {
	Name          string `json:"name"`
	ContactMethod string `json:"contact_method"`
	ContactValue  string `json:"contact_value"`
	Message       string `json:"message"`
}
