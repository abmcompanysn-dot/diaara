package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// yes_business.go — client HTTP DIARRA -> YES.abmcy Business (achat
// conversationnel "in-chat"). Contrairement à PawaPay/PayPal/PayDunya,
// DIARRA est ici le SEUL appelant : les 5 endpoints (session/initiate,
// session/{id}/status, session/{id}/send-offer, session/{id}/review,
// delivery/fulfill) sont tous côté YES Business — DIARRA n'expose jamais de
// route à YES en retour. Voir doc d'intégration fournie par YES le
// 2026-09-22 (JOURNAL-MODIFICATIONS.md) : le paiement (micro-ticket + solde)
// reste entièrement géré par DIARRA (PawaPay), YES ne fait que piloter la
// conversation et confirmer/relayer un avis vendeur.
//
// Signature HMAC-SHA256 : message = METHOD + "\n" + path + "\n" + timestamp
// + "\n" + body, en-têtes X-API-Key / X-Timestamp / X-Signature — format
// propre à YES Business, DISTINCT de celui du reste de DIARRA (gateway.go,
// yes_client.go legacy) qui signe timestamp+"."+body sous
// X-Diarra-Signature/X-Signature-SHA256. Ne pas réutiliser verifyGatewaySignature
// ici : les deux protocoles ne sont pas interchangeables.

type YesBusinessConfig struct {
	BaseURL string // ex "https://yes-api.mahu.cards/business" — pas de slash final
	APIKey  string // "bizkey_..."
	Secret  string // affiché une seule fois côté YES à la création de la clé
}

type YesBusinessClient struct {
	cfg    YesBusinessConfig
	client *http.Client
}

func NewYesBusinessClient(cfg YesBusinessConfig) *YesBusinessClient {
	cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")
	return &YesBusinessClient{cfg: cfg, client: &http.Client{Timeout: 15 * time.Second}}
}

// SessionStatus — valeurs possibles du champ "status" (voir doc : cycle de
// vie MICRO_TICKET_PAID -> OFFER_SENT -> COMPLETED, CANCELLED à tout moment
// avant COMPLETED, expiration automatique après 72h sans confirmation).
type YesSessionStatus string

const (
	YesSessionMicroTicketPaid YesSessionStatus = "MICRO_TICKET_PAID"
	YesSessionOfferSent       YesSessionStatus = "OFFER_SENT"
	YesSessionCompleted       YesSessionStatus = "COMPLETED"
	YesSessionCancelled       YesSessionStatus = "CANCELLED"
)

type InitiateSessionRequest struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	// ProductImageURL — URL publique et directement chargeable dans un <img>
	// (pas d'auth requise côté DIARRA pour la charger, voir doc YES Business
	// 2026-09-23) : alimente le widget produit épinglé en haut du fil de
	// conversation. Optionnel — champ vide si le produit n'a pas de
	// couverture (voir GetCoverURL).
	ProductImageURL  string `json:"product_image_url,omitempty"`
	SellerHandle     string `json:"seller_handle"`
	BuyerExternalID  string `json:"buyer_external_id"`
	BuyerDisplayName string `json:"buyer_display_name"`
	// BuyerContactEmail — email réel de l'acheteur (distinct de l'email
	// synthétique interne diarra-<id>@partners.yes.abmcy que YES provisionne
	// pour son compte) : sert à YES pour lui envoyer les emails de
	// notification (conversation démarrée, offre reçue) — voir doc YES
	// Business du 2026-09-23. Optionnel côté YES (best-effort), mais sans
	// lui l'acheteur ne reçoit aucun email de leur part.
	BuyerContactEmail string `json:"buyer_contact_email,omitempty"`
	MicroTicketAmount int    `json:"micro_ticket_amount"`
	FinalAmount       int    `json:"final_amount"`
	Currency          string `json:"currency"`
}

type InitiateSessionResponse struct {
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
	ChatURL        string `json:"chat_url"`
}

// InitiateSession — POST /api/v1/yes/session/initiate. Idempotent côté YES
// (rappelable sans dupliquer la session) — DIARRA n'a pas besoin de son
// propre garde-fou d'idempotence supplémentaire ici.
func (c *YesBusinessClient) InitiateSession(ctx context.Context, req InitiateSessionRequest) (*InitiateSessionResponse, error) {
	var out InitiateSessionResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/yes/session/initiate", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type SendOfferResponse struct {
	SessionID   string `json:"session_id"`
	Status      string `json:"status"`
	CheckoutURL string `json:"checkout_url"`
}

// SendOffer — POST /api/v1/yes/session/{id}/send-offer. Fait passer la
// session MICRO_TICKET_PAID -> OFFER_SENT côté YES ; DIARRA appelle ceci une
// fois le micro-ticket confirmé payé (voir YesHandler.InitiateSession).
func (c *YesBusinessClient) SendOffer(ctx context.Context, sessionID string) (*SendOfferResponse, error) {
	var out SendOfferResponse
	path := fmt.Sprintf("/api/v1/yes/session/%s/send-offer", sessionID)
	if err := c.do(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type SessionStatusResponse struct {
	SessionID         string `json:"session_id"`
	Status            string `json:"status"`
	ProductID         string `json:"product_id"`
	SellerHandle      string `json:"seller_handle"`
	BuyerHandle       string `json:"buyer_handle"`
	ConversationID    string `json:"conversation_id"`
	DeliveryURL       string `json:"delivery_url"`
	DeliveryExpiresAt string `json:"delivery_expires_at"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// GetSessionStatus — GET /api/v1/yes/session/{id}/status. Filet de sécurité
// de réconciliation (voir doc : "un paiement échoué en amont ne produit
// aucun webhook" — utilisé si DIARRA n'est pas certain de l'état d'une
// session après un délai raisonnable).
func (c *YesBusinessClient) GetSessionStatus(ctx context.Context, sessionID string) (*SessionStatusResponse, error) {
	var out SessionStatusResponse
	path := fmt.Sprintf("/api/v1/yes/session/%s/status", sessionID)
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type SubmitReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment,omitempty"`
}

// SubmitReview — POST /api/v1/yes/session/{id}/review. Appelable uniquement
// après COMPLETED côté YES (409 sinon) — DIARRA doit donc avoir déjà appelé
// delivery/fulfill (ou observé COMPLETED via GetSessionStatus) avant.
func (c *YesBusinessClient) SubmitReview(ctx context.Context, sessionID string, req SubmitReviewRequest) error {
	path := fmt.Sprintf("/api/v1/yes/session/%s/review", sessionID)
	return c.do(ctx, http.MethodPost, path, req, nil)
}

type FulfillDeliveryRequest struct {
	SessionID         string `json:"session_id"`
	Status            string `json:"status"` // toujours "COMPLETED" (voir doc)
	DeliveryURL       string `json:"delivery_url"`
	DeliveryExpiresAt string `json:"delivery_expires_at"` // RFC3339
}

// FulfillDelivery — POST /api/v1/yes/delivery/fulfill. Idempotent côté YES
// (rejouable sans danger, voir doc) — appelé par DIARRA une fois la vente du
// solde confirmée payée et l'URL signée générée (voir
// YesHandler.NotifyDelivery, branché dans webhook_handler.go ConfirmPaidSale).
func (c *YesBusinessClient) FulfillDelivery(ctx context.Context, req FulfillDeliveryRequest) error {
	return c.do(ctx, http.MethodPost, "/api/v1/yes/delivery/fulfill", req, nil)
}

// do — construit, signe et envoie une requête YES Business. body nil = corps
// vide (GET, ou POST sans payload comme SendOffer) ; body non-nil est
// marshalé en JSON. La signature porte sur le path EXACT vu par le service
// YES Business (voir doc : jamais un préfixe de reverse-proxy côté DIARRA,
// qui n'en a de toute façon aucun ici — BaseURL pointe déjà sur
// yes-api.mahu.cards/business).
func (c *YesBusinessClient) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var rawBody []byte
	if body != nil {
		var err error
		rawBody, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	var reader io.Reader
	if rawBody != nil {
		reader = bytes.NewReader(rawBody)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, reader)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	timestamp := time.Now().UTC().Format(time.RFC3339)
	httpReq.Header.Set("X-API-Key", c.cfg.APIKey)
	httpReq.Header.Set("X-Timestamp", timestamp)
	httpReq.Header.Set("X-Signature", c.sign(method, path, timestamp, rawBody))

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: yes business status %d on %s %s: %s", ErrPaymentFailed, resp.StatusCode, method, path, string(respBody))
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return err
		}
	}
	return nil
}

// sign — HMAC-SHA256(secret, method + "\n" + path + "\n" + timestamp + "\n" + body),
// hex minuscules. body vide (GET, ou POST sans payload) = chaîne vide, comme
// documenté par YES ("chaîne vide pour un GET sans corps").
func (c *YesBusinessClient) sign(method, path, timestamp string, body []byte) string {
	message := method + "\n" + path + "\n" + timestamp + "\n" + string(body)
	mac := hmac.New(sha256.New, []byte(c.cfg.Secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
