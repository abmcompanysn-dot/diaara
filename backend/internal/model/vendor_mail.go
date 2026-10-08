package model

import "time"

type VendorMailThread struct {
	ID            string    `json:"id"`
	VendorID      string    `json:"vendor_id"`
	Subject       string    `json:"subject"`
	LastMessageID *string   `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Champs de jointure pour l'affichage admin (non stockés sur cette table).
	VendorEmail string `json:"vendor_email,omitempty"`
	VendorShop  string `json:"vendor_shop,omitempty"`
	// Statut calculé pour le tableau de bord de campagne (voir ListThreads) :
	// HasSent = au moins un message sortant réellement envoyé (status='sent') ;
	// HasReplied = au moins un message entrant reçu de ce vendeur.
	HasSent    bool `json:"has_sent"`
	HasReplied bool `json:"has_replied"`
}

type VendorMailMessage struct {
	ID        string  `json:"id"`
	ThreadID  string  `json:"thread_id"`
	Direction string  `json:"direction"` // outbound | inbound
	Status    string  `json:"status"`    // draft | approved | sent | received | rejected
	Subject   string  `json:"subject"`
	Body      string  `json:"body"`
	MessageID *string `json:"-"`
	InReplyTo *string `json:"-"`
	// BannerURL : image remplaçant le bandeau "DIARRA" par défaut dans
	// l'email HTML (campagne ponctuelle, ex: Octobre Rose). nil = bandeau
	// texte par défaut.
	BannerURL  *string    `json:"banner_url,omitempty"`
	ApprovedBy *string    `json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	SentAt     *time.Time `json:"sent_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// VendorMailThreadWithMessages : un fil complet avec tous ses messages,
// renvoyé par la page admin pour afficher la conversation.
type VendorMailThreadWithMessages struct {
	Thread   VendorMailThread    `json:"thread"`
	Messages []VendorMailMessage `json:"messages"`
}

type StartVendorMailThreadInput struct {
	VendorID string `json:"vendor_id"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
}

type DraftVendorMailReplyInput struct {
	Body string `json:"body"`
}

// BroadcastVendorMailInput — crée un brouillon par destinataire pour chaque
// rôle listé dans Roles ("vendeur", "closer"). Body peut contenir le
// placeholder littéral "{{nom}}", remplacé par le nom (boutique/affichage)
// du destinataire, ou retiré proprement si ce destinataire n'en a pas.
type BroadcastVendorMailInput struct {
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Roles   []string `json:"roles"`
}
