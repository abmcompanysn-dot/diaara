package model

import "time"

// yes_integration.go — achat conversationnel "in-chat" via YES.abmcy
// Business (voir doc d'intégration fournie par YES le 2026-09-22,
// JOURNAL-MODIFICATIONS.md). DIARRA est le SEUL appelant des 4 endpoints
// YES Business (session/initiate, send-offer, review, status) ; YES
// n'appelle DIARRA que pour un webhook entrant, delivery/fulfill (voir
// middleware.RequireYesDeliveryWebhook).
//
// Flux, du clic acheteur à la livraison :
//  1. Acheteur clique "Discuter avec le vendeur" sur une fiche produit DIARRA
//  2. DIARRA encaisse le micro-ticket (PawaPay, comme un achat classique)
//  3. Une fois confirmé payé, DIARRA appelle YES session/initiate — YES crée
//     la conversation + provisionne un compte YES pour l'acheteur si besoin,
//     renvoie session_id + chat_url
//  4. DIARRA redirige l'acheteur vers chat_url ; la session est "ouverte"
//     côté DIARRA (ConversationalSession.Status = SessionOpened)
//  5. Le vendeur négocie avec l'acheteur dans YES ; quand l'acheteur est prêt,
//     DIARRA (via son propre déclencheur, ex. le vendeur clique "proposer
//     l'offre" côté DIARRA) appelle YES send-offer — YES affiche le checkout
//     in-chat, session passe MICRO_TICKET_PAID -> OFFER_SENT côté YES
//  6. L'acheteur clique payer DANS LE CHAT ; DIARRA encaisse le solde
//     (PawaPay) via son propre flux normal, DÉCLENCHÉ CÔTÉ DIARRA (pas par
//     un webhook YES — voir doc : "un paiement échoué en amont ne produit
//     aucun webhook")
//  7. Solde confirmé payé -> DIARRA génère l'URL signée MinIO, appelle YES
//     delivery/fulfill -> session COMPLETED côté YES
//  8. L'acheteur note le vendeur DANS YES ; YES relaie via... en réalité
//     c'est DIARRA qui appelle session/review pour RELAYER une note déjà
//     recueillie côté DIARRA (pas YES qui pousse la note à DIARRA) — voir
//     doc : "Relaie une note vendeur" du point de vue de l'appelant (DIARRA).

type ConversationalSessionStatus string

const (
	// SessionPending — ligne créée à l'ouverture (OpenConversation), micro-
	// ticket pas encore confirmé payé. yes_session_id/chat_url encore vides.
	SessionPending ConversationalSessionStatus = "pending"
	// SessionOpened — micro-ticket confirmé payé, YES session/initiate
	// appelé avec succès (chat_url connue).
	SessionOpened ConversationalSessionStatus = "opened"
	// SessionOfferSent — DIARRA a appelé send-offer, en attente du paiement
	// du solde par l'acheteur.
	SessionOfferSent ConversationalSessionStatus = "offer_sent"
	SessionCompleted ConversationalSessionStatus = "completed"
	SessionCancelled ConversationalSessionStatus = "cancelled"
)

// ConversationalSession — pont entre une session YES Business et les
// paiements DIARRA. YesSessionID est l'ID CÔTÉ YES (renvoyé par
// session/initiate, connu seulement une fois le micro-ticket payé — voir
// SessionPending) ; toutes les autres requêtes YES Business
// (status/send-offer/review/delivery) s'y réfèrent.
type ConversationalSession struct {
	ID           string  `json:"id"`
	YesSessionID *string `json:"yes_session_id,omitempty"`
	ChatURL      *string `json:"chat_url,omitempty"`
	ProductID    string  `json:"product_id"`
	BuyerID      string  `json:"buyer_id"`
	SellerID     string  `json:"seller_id"`
	// ReferralLinkID — lien d'affiliation déjà validé à l'ouverture,
	// réutilisé tel quel pour la vente du solde.
	ReferralLinkID    *string                     `json:"referral_link_id,omitempty"`
	MicroTicketSaleID string                      `json:"micro_ticket_sale_id"`
	SaleID            *string                     `json:"sale_id,omitempty"`
	Status            ConversationalSessionStatus `json:"status"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
}

// DefaultYesMicroTicketAmountCFA — valeur par défaut si l'admin n'a jamais
// touché au réglage (voir model.SettingYesMicroTicketAmountCFA) : équivalent
// ~1 USD, facturé en XOF fixe comme le reste de DIARRA. Même plancher que
// PayPalPayoutMinCFA : en dessous, les frais prestataire dépassent l'intérêt
// du micro-paiement.
const DefaultYesMicroTicketAmountCFA = 600

// OpenConversationInput — POST /api/vendor-chat/open (acheteur DIARRA ->
// DIARRA, PAS vers YES directement) : le clic "Discuter avec le vendeur"
// sur une fiche produit. Authentification normale DIARRA (JWT), comme
// n'importe quelle route acheteur — ce n'est qu'ensuite que DIARRA appelle
// YES Business en interne, une fois le micro-ticket payé.
type OpenConversationInput struct {
	ProductID      string  `json:"product_id"`
	ReferralLinkID *string `json:"referral_link_id,omitempty"`
	Country        string  `json:"country"`
	// Phone/Operator : dépôt PawaPay direct (POST /v2/deposits), voir
	// SaleHandler.initiateDirectDeposit — mêmes conventions.
	Phone    string `json:"phone"`
	Operator string `json:"operator"`
	// OTP : voir model.CreateOrderInput.OTP — même convention, requis
	// uniquement pour un opérateur PayDunya avec RequiresOTP.
	OTP string `json:"otp,omitempty"`
}

// SendOfferInput — POST /api/vendor-chat/{session_id}/send-offer (vendeur ->
// DIARRA), déclenche l'appel YES send-offer.
type SendOfferInput struct{}

// VendorReviewInput — POST /api/vendor-chat/{session_id}/review (acheteur ->
// DIARRA), relayé ensuite vers YES session/review.
type VendorReviewInput struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment,omitempty"`
}

// VendorReview — avis laissé par un acheteur à la fin d'une conversation,
// stocké côté DIARRA ET relayé à YES (voir YesHandler.SubmitVendorReview).
type VendorReview struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	VendorID  string    `json:"vendor_id"`
	BuyerID   string    `json:"buyer_id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
