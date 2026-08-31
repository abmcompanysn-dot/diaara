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

// KPay — dépôts, versements, remboursements mobile money (12 pays) + carte
// bancaire/PayPal, en complément de PawaPay (qui ne fait pas carte/PayPal).
// Docs fournies par KPay (contexte agent IA, relevé 2026-08-19). Flux
// asynchrone comme PawaPay : on initie, KPay pousse un webhook au statut
// final — voir webhook_handler.go pour la vérification de signature.
//
// Différences structurelles avec PawaPay à garder en tête :
//   - Auth par deux en-têtes statiques (X-API-Key/X-Secret-Key), pas de
//     Authorization: Bearer.
//   - Même URL en sandbox et production : seul le préfixe des clés diffère
//     (kpay_test_/sk_test_ vs kpay_live_/sk_live_).
//   - Webhooks signés par HMAC-SHA256 réel (X-KPAY-Signature) avec un
//     secret DÉDIÉ, distinct de SecretKey — pas de schéma Content-Digest+IP
//     comme PawaPay.
//   - Remboursement toujours intégral (pas de champ "amount" sur l'appel).

type KPayConfig struct {
	APIKey        string // X-API-Key (KPAY_API_KEY), préfixe kpay_test_/kpay_live_
	SecretKey     string // X-Secret-Key (KPAY_SECRET_KEY), préfixe sk_test_/sk_live_
	BaseURL       string // défaut: https://admin.kpay.site (identique sandbox/prod)
	WebhookSecret string // KPAY_WEBHOOK_SECRET — vérifie X-KPAY-Signature, distinct de SecretKey
}

type KPayClient struct {
	cfg    KPayConfig
	client *http.Client
}

func NewKPayClient(cfg KPayConfig) *KPayClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://admin.kpay.site"
	}
	cfg.BaseURL = strings.TrimSuffix(baseURL, "/")
	return &KPayClient{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *KPayClient) Name() string { return "kpay" }

func (c *KPayClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	req.Header.Set("X-Secret-Key", c.cfg.SecretKey)
}

func (c *KPayClient) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, reader)
	if err != nil {
		return err
	}
	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("%w: status %d: %s", ErrPaymentFailed, resp.StatusCode, string(respBody))
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return err
		}
	}
	return nil
}

// --- Paiements (dépôts) ------------------------------------------------------

// PaymentInitRequest — POST /api/v1/payments/init. Mode GATEWAY (utilisé par
// DIARRA, voir plan) : Provider/PhoneNumber omis, ReturnUrl requis — KPay
// héberge la page où l'acheteur choisit lui-même son opérateur/carte/PayPal.
// PaymentMethod "CARD"/"PAYPAL" force ce mode même si un jour un mode USSD
// par défaut était utilisé ailleurs.
type PaymentInitRequest struct {
	Amount        string            `json:"amount"`
	Currency      string            `json:"currency,omitempty"`
	ExternalId    string            `json:"externalId"`
	Provider      string            `json:"provider,omitempty"`      // mode USSD uniquement (non utilisé côté checkout DIARRA)
	PhoneNumber   string            `json:"phoneNumber,omitempty"`   // mode USSD uniquement
	ReturnUrl     string            `json:"returnUrl,omitempty"`     // requis en mode GATEWAY
	CancelUrl     string            `json:"cancelUrl,omitempty"`     // optionnel en mode GATEWAY
	PaymentMethod string            `json:"paymentMethod,omitempty"` // "CARD" | "PAYPAL" — force la passerelle
	Description   string            `json:"description,omitempty"`
	CustomerEmail string            `json:"customerEmail,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type PaymentInitResponse struct {
	ID         string `json:"id"` // ID KPay — nécessaire pour GET/remboursement ultérieurs (voir sales.provider_transaction_id)
	Reference  string `json:"reference"`
	Status     string `json:"status"` // PENDING
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	Mode       string `json:"mode,omitempty"`
	GatewayUrl string `json:"gatewayUrl,omitempty"`
	Message    string `json:"message,omitempty"`
}

// InitiatePayment — POST /api/v1/payments/init
func (c *KPayClient) InitiatePayment(ctx context.Context, req PaymentInitRequest) (*PaymentInitResponse, error) {
	var result PaymentInitResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/payments/init", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type PaymentStatusResponse struct {
	ID            string `json:"id"`
	Status        string `json:"status"` // PENDING | PROCESSING | COMPLETED | FAILED | CANCELLED
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	ExternalId    string `json:"externalId"`
	FailureReason string `json:"failureReason,omitempty"`
	CompletedAt   string `json:"completedAt,omitempty"`
}

// GetPaymentStatus — GET /api/v1/payments/:id
func (c *KPayClient) GetPaymentStatus(ctx context.Context, id string) (*PaymentStatusResponse, error) {
	var result PaymentStatusResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/payments/"+id, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Remboursements -----------------------------------------------------------
//
// Toujours intégral : pas de champ "amount" sur cet appel (contrairement à
// PawaPay). Un seul remboursement actif par paiement, fenêtre de 7 jours,
// idempotent via ExternalId.

type RefundInitRequest struct {
	Reason     string `json:"reason,omitempty"`
	ExternalId string `json:"externalId,omitempty"`
}

type RefundInitResponse struct {
	ID                     string `json:"id"`
	Status                 string `json:"status"` // PENDING à l'initiation, issue finale par webhook
	Amount                 string `json:"amount"`
	Currency               string `json:"currency"`
	OriginalPaymentId      string `json:"originalPaymentId"`
	OriginalPaymentStatus  string `json:"originalPaymentStatus"`
	Message                string `json:"message,omitempty"`
}

// InitiateRefund — POST /api/v1/payments/:id/refund (id = ID KPay du paiement,
// voir PaymentInitResponse.ID / sales.provider_transaction_id).
func (c *KPayClient) InitiateRefund(ctx context.Context, paymentID string, req RefundInitRequest) (*RefundInitResponse, error) {
	var result RefundInitResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/payments/"+paymentID+"/refund", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Versements vendeurs (payouts) --------------------------------------------
//
// Mode USSD (Provider+PhoneNumber) : DIARRA connaît déjà l'opérateur exact
// du vendeur (enregistré via SetPayoutMethod), contrairement au checkout.

type PayoutInitRequest struct {
	Amount      string            `json:"amount"`
	Provider    string            `json:"provider"`
	PhoneNumber string            `json:"phoneNumber"`
	ExternalId  string            `json:"externalId,omitempty"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type PayoutInitResponse struct {
	ID         string `json:"id"`
	Reference  string `json:"reference"`
	Status     string `json:"status"` // PENDING
	Amount     string `json:"amount"`
	NetAmount  string `json:"netAmount"`
	FeeAmount  string `json:"feeAmount"`
	Currency   string `json:"currency"`
	ExternalId string `json:"externalId"`
	Message    string `json:"message,omitempty"`
}

// InitiatePayout — POST /api/v1/payments/withdraw
func (c *KPayClient) InitiatePayout(ctx context.Context, req PayoutInitRequest) (*PayoutInitResponse, error) {
	var result PayoutInitResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/payments/withdraw", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// KPayPayoutStatusResponse — préfixé KPay pour éviter la collision avec
// payment.PayoutStatusResponse (PawaPay), les deux providers cohabitant
// dans le même package.
type KPayPayoutStatusResponse struct {
	ID            string `json:"id"`
	Status        string `json:"status"` // PENDING | PROCESSING | COMPLETED | FAILED | CANCELLED
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	FailureReason string `json:"failureReason,omitempty"`
}

// GetPayoutStatus — GET /api/v1/payments/withdraw/:id
func (c *KPayClient) GetPayoutStatus(ctx context.Context, id string) (*KPayPayoutStatusResponse, error) {
	var result KPayPayoutStatusResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/payments/withdraw/"+id, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Vérification webhook ------------------------------------------------------

// VerifyKPaySignature vérifie l'en-tête X-KPAY-Signature : HMAC-SHA256 hex du
// corps BRUT reçu (non re-sérialisé), comparaison en temps constant.
func VerifyKPaySignature(secret string, body []byte, signatureHex string) bool {
	if secret == "" || signatureHex == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signatureHex)))
}

// --- Opérateurs supportés par KPay ---------------------------------------------

// KPayProviderCodes — sous-ensemble exact des codes opérateur (identiques aux
// codes PawaPay dans XOFOperators pour les corridors partagés) acceptés par
// l'API KPay en mode USSD, d'après leur doc (relevé 2026-08-19). WAVE_SEN et
// WAVE_CIV sont volontairement ABSENTS : suspendus côté KPay le temps de
// finaliser un nouveau protocole d'authentification — ne jamais permettre à
// l'admin d'assigner KPay à un opérateur Wave tant que ce n'est pas confirmé
// rétabli (voir validation dans admin_handler.go UpdateSettings).
var KPayProviderCodes = map[string]bool{
	"MTN_MOMO_BEN":      true,
	"MOOV_BEN":          true,
	"MTN_MOMO_CMR":      true,
	"ORANGE_CMR":        true,
	"MTN_MOMO_CIV":      true,
	"ORANGE_CIV":        true,
	"VODACOM_MPESA_COD": true,
	"AIRTEL_COD":        true,
	"ORANGE_COD":        true,
	"AIRTEL_GAB":        true,
	"MPESA_KEN":         true,
	"AIRTEL_COG":        true,
	"MTN_MOMO_COG":      true,
	"AIRTEL_RWA":        true,
	"MTN_MOMO_RWA":      true,
	"FREE_SEN":          true,
	"ORANGE_SEN":        true,
	"ORANGE_SLE":        true,
	"AIRTEL_OAPI_UGA":   true,
	"MTN_MOMO_UGA":      true,
	"AIRTEL_OAPI_ZMB":   true,
	"MTN_MOMO_ZMB":      true,
	"ZAMTEL_ZMB":        true,
}
