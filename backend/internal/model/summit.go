package model

import "time"

// SummitRegistration — inscription à un événement DIARRA Summit (ex: celui
// du 26 octobre 2026, en ligne, sur la digitalisation par l'IA et les
// produits numériques). Public : ne requiert pas de compte DIARRA.
type SummitRegistration struct {
	ID          string    `json:"id"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	Profile     string    `json:"profile"`
	CreatedAt   time.Time `json:"created_at"`
}

// SummitProfiles — valeurs acceptées pour SummitRegistration.Profile,
// alignées sur la contrainte CHECK de la migration 033.
var SummitProfiles = map[string]bool{
	"vendeur":      true,
	"acheteur":     true,
	"entrepreneur": true,
	"curieux":      true,
}

type CreateSummitRegistrationInput struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Profile  string `json:"profile"`
}
