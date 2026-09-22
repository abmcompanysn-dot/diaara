package model

import "time"

// YesAPIClient — application autorisée à appeler /api/yes/* (voir
// migrations/037_yes_integration.sql). Authentification par signature HMAC
// (X-Signature-SHA256), pas par clé seule — voir middleware.RequireYesClient.
type YesAPIClient struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	APIKey        string    `json:"api_key"`
	APISecretHash string    `json:"-"` // jamais renvoyé au client
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

type ConversationalSessionStatus string

const (
	SessionMicroTicketPaid ConversationalSessionStatus = "micro_ticket_paid"
	SessionCompleted       ConversationalSessionStatus = "completed"
	SessionCancelled       ConversationalSessionStatus = "cancelled"
)

// ConversationalSession — pont léger entre une conversation YES et les
// paiements DIARRA (voir migrations/037_yes_integration.sql pour le
// raisonnement complet : YES possède l'état de la conversation, DIARRA ne le
// duplique pas).
type ConversationalSession struct {
	ID                string                      `json:"id"`
	YesConversationID string                      `json:"yes_conversation_id"`
	ProductID         string                      `json:"product_id"`
	BuyerID           string                      `json:"buyer_id"`
	SellerID          string                      `json:"seller_id"`
	// ReferralLinkID — lien d'affiliation déjà validé à l'initiation, réutilisé
	// tel quel pour la vente du solde (voir migrations/037_yes_integration.sql).
	ReferralLinkID    *string                     `json:"referral_link_id,omitempty"`
	MicroTicketSaleID *string                     `json:"micro_ticket_sale_id,omitempty"`
	SaleID            *string                     `json:"sale_id,omitempty"`
	Status            ConversationalSessionStatus `json:"status"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
}

// MicroTicketAmountCFA — montant fixe du "ticket d'entrée" qui ouvre la
// conversation avec le vendeur (équivalent ~1 USD, mais facturé en XOF fixe
// comme le reste de DIARRA — voir décision du 2026-09-22). Même plancher que
// PayPalPayoutMinCFA : en dessous, les frais prestataire dépassent l'intérêt
// du micro-paiement.
const MicroTicketAmountCFA = 600

// InitiateSessionInput — POST /api/yes/session/initiate (YES -> DIARRA).
type InitiateSessionInput struct {
	YesConversationID string  `json:"yes_conversation_id"`
	ProductID         string  `json:"product_id"`
	BuyerID           string  `json:"buyer_id"` // ID utilisateur DIARRA déjà connu de YES (compte lié)
	ReferralLinkID    *string `json:"referral_link_id,omitempty"`
	PaymentMethod     string  `json:"payment_method,omitempty"` // "mobile_money" | "card" | "paypal"
	// Coordonnées mobile money (requises si payment_method mobile_money) —
	// mêmes contraintes que SaleHandler.initiatePaymentPage.
	Phone   string `json:"phone,omitempty"`
	Country string `json:"country,omitempty"`
}

// InChatCheckoutInput — POST /api/yes/checkout/in-chat (YES -> DIARRA).
type InChatCheckoutInput struct {
	SessionID     string `json:"session_id"` // ID ConversationalSession (pas yes_conversation_id)
	PaymentMethod string `json:"payment_method,omitempty"`
	Phone         string `json:"phone,omitempty"`
}

// VendorReviewInput — POST /api/yes/vendors/review (YES -> DIARRA).
type VendorReviewInput struct {
	SessionID string `json:"session_id"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment,omitempty"`
}

// VendorReview — avis laissé par un acheteur à la fin d'une conversation.
type VendorReview struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	VendorID  string    `json:"vendor_id"`
	BuyerID   string    `json:"buyer_id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
