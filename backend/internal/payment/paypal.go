package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// PayPal — paiement carte bancaire/PayPal, en remplacement de KPay pour ce
// flux (KPay désactivé au checkout depuis le 2026-09-03, voir
// CARD_PAYMENT_ENABLED côté frontend et resolveCheckoutProvider côté
// sale_handler.go). Orders API v2 (Checkout) : flux "hosted redirect", même
// principe que PawaPay/KPay — on crée une commande, on redirige l'acheteur
// vers le lien d'approbation PayPal, il revient sur ReturnUrl, on capture
// alors la commande (voir GetDepositStatus, appelé par checkout/return via
// SaleHandler.CheckoutStatus).
//
// PayPal ne supporte pas le XOF (catalogue DIARRA tarifé en FCFA) : les
// montants sont convertis en USD via USDRates (même table que la conversion
// PawaPay, voir ConvertFromXOF dans pawapay.go) — voir convertXOFToUSD.
type PayPalConfig struct {
	ClientID     string
	ClientSecret string
	BaseURL      string // défaut: https://api-m.sandbox.paypal.com — https://api-m.paypal.com en prod
	WebhookID    string // ID du webhook PayPal (Developer Dashboard > Webhooks), pour VerifyWebhookSignature
}

type PayPalClient struct {
	cfg    PayPalConfig
	client *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewPayPalClient(cfg PayPalConfig) *PayPalClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api-m.sandbox.paypal.com"
	}
	cfg.BaseURL = strings.TrimSuffix(baseURL, "/")
	return &PayPalClient{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}
}

func (c *PayPalClient) Name() string { return "paypal" }

// IsSandbox — utilisé côté admin/diagnostic pour afficher clairement
// l'environnement actif (même besoin que pour PawaPay/KPay).
func (c *PayPalClient) IsSandbox() bool {
	return strings.Contains(c.cfg.BaseURL, "sandbox")
}

// convertXOFToUSD convertit un montant FCFA vers USD avec 2 décimales
// (format string exigé par l'API PayPal, ex "10.00") — même table de taux
// que PawaPay (USDRates, pawapay.go), qui elle arrondit à l'entier (adaptée
// aux devises locales africaines, pas à l'USD).
func convertXOFToUSD(amountXOF int) string {
	rate := USDRates["XOF"]
	usd := float64(amountXOF) / rate
	return fmt.Sprintf("%.2f", usd)
}

// getAccessToken — OAuth2 client_credentials (POST /v1/oauth2/token), mis en
// cache jusqu'à expiration (marge de 60s) pour éviter un aller-retour à
// chaque appel — PayPal limite le taux d'émission de tokens.
func (c *PayPalClient) getAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/v1/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.cfg.ClientID, c.cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: oauth status %d: %s", ErrPaymentFailed, resp.StatusCode, string(body))
	}

	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	c.accessToken = out.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(out.ExpiresIn-60) * time.Second)
	return c.accessToken, nil
}

func (c *PayPalClient) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d: %s", ErrPaymentFailed, resp.StatusCode, string(respBody))
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return err
		}
	}
	return nil
}

// --- Création / capture de commande (dépôt) ---------------------------------

type PayPalOrderRequest struct {
	SaleID      string
	AmountXOF   int
	ReturnURL   string
	CancelURL   string
	Description string
}

type PayPalOrderResponse struct {
	ID         string
	Status     string
	ApproveURL string
}

type paypalOrderPayload struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	PurchaseUnits []struct {
		Payments struct {
			Captures []struct {
				ID string `json:"id"`
			} `json:"captures"`
		} `json:"payments"`
	} `json:"purchase_units"`
	Links []struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	} `json:"links"`
}

// CreateOrder — POST /v2/checkout/orders (intent=CAPTURE). Retourne l'URL
// d'approbation PayPal (lien rel="approve") vers laquelle rediriger
// l'acheteur — équivalent de GatewayUrl (KPay) / RedirectUrl (PawaPay).
func (c *PayPalClient) CreateOrder(ctx context.Context, req PayPalOrderRequest) (*PayPalOrderResponse, error) {
	body := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"reference_id": req.SaleID,
				"custom_id":    req.SaleID,
				"description":  req.Description,
				"amount": map[string]string{
					"currency_code": "USD",
					"value":         convertXOFToUSD(req.AmountXOF),
				},
			},
		},
		"application_context": map[string]interface{}{
			"brand_name":  "DIARRA",
			"user_action": "PAY_NOW",
			"return_url":  req.ReturnURL,
			"cancel_url":  req.CancelURL,
		},
	}

	var resp paypalOrderPayload
	if err := c.do(ctx, http.MethodPost, "/v2/checkout/orders", body, &resp); err != nil {
		return nil, err
	}

	out := &PayPalOrderResponse{ID: resp.ID, Status: resp.Status}
	for _, l := range resp.Links {
		if l.Rel == "approve" {
			out.ApproveURL = l.Href
		}
	}
	if out.ApproveURL == "" {
		return nil, fmt.Errorf("%w: pas de lien d'approbation dans la réponse PayPal", ErrPaymentFailed)
	}
	return out, nil
}

// PayPalOrderStatus — statut brut PayPal (CREATED | APPROVED | COMPLETED |
// VOIDED) + l'ID de capture une fois disponible.
type PayPalOrderStatus struct {
	ID        string
	Status    string
	CaptureID string
}

func extractCaptureID(p paypalOrderPayload) string {
	if len(p.PurchaseUnits) > 0 && len(p.PurchaseUnits[0].Payments.Captures) > 0 {
		return p.PurchaseUnits[0].Payments.Captures[0].ID
	}
	return ""
}

// GetOrder — GET /v2/checkout/orders/{id}.
func (c *PayPalClient) GetOrder(ctx context.Context, orderID string) (*PayPalOrderStatus, error) {
	var resp paypalOrderPayload
	if err := c.do(ctx, http.MethodGet, "/v2/checkout/orders/"+orderID, nil, &resp); err != nil {
		return nil, err
	}
	return &PayPalOrderStatus{ID: resp.ID, Status: resp.Status, CaptureID: extractCaptureID(resp)}, nil
}

// CaptureOrder — POST /v2/checkout/orders/{id}/capture : encaisse une
// commande APPROVED. PayPal renvoie une erreur ORDER_ALREADY_CAPTURED (422)
// si rappelée après capture — voir GetDepositStatus qui n'appelle Capture
// que si le statut est encore APPROVED, jamais après COMPLETED.
func (c *PayPalClient) CaptureOrder(ctx context.Context, orderID string) (*PayPalOrderStatus, error) {
	var resp paypalOrderPayload
	if err := c.do(ctx, http.MethodPost, "/v2/checkout/orders/"+orderID+"/capture", map[string]interface{}{}, &resp); err != nil {
		return nil, err
	}
	return &PayPalOrderStatus{ID: resp.ID, Status: resp.Status, CaptureID: extractCaptureID(resp)}, nil
}

// --- Remboursement -----------------------------------------------------------

type PayPalRefundResponse struct {
	ID     string
	Status string
}

// RefundCapture — POST /v2/payments/captures/{capture_id}/refund.
// Remboursement toujours intégral (pas de montant fourni) — même choix que
// KPay (voir kpay.go, InitiateRefund).
func (c *PayPalClient) RefundCapture(ctx context.Context, captureID string) (*PayPalRefundResponse, error) {
	var resp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := c.do(ctx, http.MethodPost, "/v2/payments/captures/"+captureID+"/refund", map[string]interface{}{}, &resp); err != nil {
		return nil, err
	}
	return &PayPalRefundResponse{ID: resp.ID, Status: resp.Status}, nil
}

// --- Versements vendeur (Payouts API v1) ------------------------------------

// PayPalPayoutRequest — un versement unitaire vers l'email PayPal d'un vendeur.
// SenderItemID doit être unique et stable (on passe l'ID du versement DIARRA)
// pour l'idempotence et le rapprochement.
type PayPalPayoutRequest struct {
	SenderBatchID string // unique par lot — ex "diarra-<payoutID>"
	SenderItemID  string // = ID du versement DIARRA
	ReceiverEmail string
	AmountXOF     int
	Note          string
}

// PayPalPayoutResponse — retour de POST /v1/payments/payouts.
type PayPalPayoutResponse struct {
	BatchID     string // payout_batch_id — à stocker pour interroger le statut
	BatchStatus string // PENDING | PROCESSING | SUCCESS | DENIED | ...
}

// SendPayout — POST /v1/payments/payouts (un seul item par lot : DIARRA règle
// les vendeurs à l'unité, pas en masse). Montant converti FCFA → USD (PayPal
// ne fait pas le XOF), même table que l'encaissement (convertXOFToUSD).
func (c *PayPalClient) SendPayout(ctx context.Context, req PayPalPayoutRequest) (*PayPalPayoutResponse, error) {
	note := req.Note
	if note == "" {
		note = "Versement DIARRA"
	}
	body := map[string]interface{}{
		"sender_batch_header": map[string]interface{}{
			"sender_batch_id": req.SenderBatchID,
			"email_subject":   "Vous avez reçu un versement DIARRA",
			"email_message":   "Votre versement a été envoyé sur votre compte PayPal.",
		},
		"items": []map[string]interface{}{
			{
				"recipient_type": "EMAIL",
				"receiver":       req.ReceiverEmail,
				"amount": map[string]string{
					"value":    convertXOFToUSD(req.AmountXOF),
					"currency": "USD",
				},
				"note":          note,
				"sender_item_id": req.SenderItemID,
			},
		},
	}

	var resp struct {
		BatchHeader struct {
			PayoutBatchID string `json:"payout_batch_id"`
			BatchStatus   string `json:"batch_status"`
		} `json:"batch_header"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/payments/payouts", body, &resp); err != nil {
		return nil, err
	}
	if resp.BatchHeader.PayoutBatchID == "" {
		return nil, fmt.Errorf("%w: pas de payout_batch_id dans la réponse PayPal", ErrPaymentFailed)
	}
	return &PayPalPayoutResponse{
		BatchID:     resp.BatchHeader.PayoutBatchID,
		BatchStatus: resp.BatchHeader.BatchStatus,
	}, nil
}

// PayPalPayoutStatus — statut consolidé d'un versement (lu sur l'item unique
// du lot). ItemStatus : SUCCESS | FAILED | UNCLAIMED | RETURNED | ONHOLD |
// BLOCKED | REFUNDED | PENDING.
type PayPalPayoutStatus struct {
	BatchStatus string
	ItemStatus  string
	Reason      string
}

// GetPayoutBatch — GET /v1/payments/payouts/{payout_batch_id}. Renvoie le
// statut du lot et de son unique item.
func (c *PayPalClient) GetPayoutBatch(ctx context.Context, batchID string) (*PayPalPayoutStatus, error) {
	var resp struct {
		BatchHeader struct {
			BatchStatus string `json:"batch_status"`
		} `json:"batch_header"`
		Items []struct {
			TransactionStatus string `json:"transaction_status"`
			Errors            struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"items"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/payments/payouts/"+batchID, nil, &resp); err != nil {
		return nil, err
	}
	out := &PayPalPayoutStatus{BatchStatus: resp.BatchHeader.BatchStatus}
	if len(resp.Items) > 0 {
		out.ItemStatus = resp.Items[0].TransactionStatus
		out.Reason = resp.Items[0].Errors.Message
	}
	return out, nil
}

// --- Vérification des webhooks ------------------------------------------------

// VerifyWebhookSignature — POST /v1/notifications/verify-webhook-signature :
// contrairement à KPay (HMAC simple vérifiable localement), PayPal exige de
// leur renvoyer les en-têtes de signature + le corps brut pour validation
// côté serveur PayPal (pas de vérification locale possible sans dupliquer
// leur logique de certificat).
func (c *PayPalClient) VerifyWebhookSignature(ctx context.Context, headers http.Header, rawBody []byte) (bool, error) {
	if c.cfg.WebhookID == "" {
		return false, fmt.Errorf("PAYPAL_WEBHOOK_ID non configuré")
	}
	var event interface{}
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return false, err
	}
	body := map[string]interface{}{
		"auth_algo":         headers.Get("Paypal-Auth-Algo"),
		"cert_url":          headers.Get("Paypal-Cert-Url"),
		"transmission_id":   headers.Get("Paypal-Transmission-Id"),
		"transmission_sig":  headers.Get("Paypal-Transmission-Sig"),
		"transmission_time": headers.Get("Paypal-Transmission-Time"),
		"webhook_id":        c.cfg.WebhookID,
		"webhook_event":     event,
	}
	var resp struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := c.do(ctx, http.MethodPost, "/v1/notifications/verify-webhook-signature", body, &resp); err != nil {
		return false, err
	}
	return resp.VerificationStatus == "SUCCESS", nil
}
