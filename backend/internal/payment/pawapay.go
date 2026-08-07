package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// PawaPay — dépôt mobile money via l'API PawaPay v2.
// Docs : https://docs.pawapay.io
// Le flux est asynchrone : on initie un dépôt, PawaPay pousse une demande
// d'autorisation sur le téléphone de l'acheteur, puis nous envoie un callback
// (webhook) avec le statut final.

var ErrPaymentFailed = errors.New("payment request failed")

type PawaPayConfig struct {
	APIKey      string // jeton généré dans le dashboard PawaPay (PAWAPAY_API_KEY)
	BaseURL     string // défaut sandbox: https://api.sandbox.pawapay.io
	CallbackURL string // URL publique du webhook (déclaré dans le dashboard)
}

type PawaPayClient struct {
	cfg    PawaPayConfig
	client *http.Client
}

func NewPawaPayClient(cfg PawaPayConfig) *PawaPayClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.sandbox.pawapay.io"
	}
	cfg.BaseURL = strings.TrimSuffix(baseURL, "/")
	return &PawaPayClient{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// --- Requête de dépôt -------------------------------------------------------

type DepositRequest struct {
	DepositId         string         `json:"depositId"` // UUID v4 fourni par nous (servira de payment_reference)
	Payer             Payer          `json:"payer"`
	Amount            string         `json:"amount"` // montant en string, ex "1500"
	Currency          string         `json:"currency"`
	ClientReferenceId string         `json:"clientReferenceId,omitempty"`
	CustomerMessage   string         `json:"customerMessage,omitempty"`
	Metadata          []MetadataItem `json:"metadata,omitempty"`
	CallbackUrl       string         `json:"callbackUrl,omitempty"`
}

type Payer struct {
	Type          string        `json:"type"` // "MMO"
	AccountDetails AccountDetails `json:"accountDetails"`
}

type AccountDetails struct {
	PhoneNumber string `json:"phoneNumber"` // MSISDN, chiffres seuls, code pays obligatoire
	Provider    string `json:"provider"`    // ex "ORANGE_SEN"
}

type MetadataItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type FailureReason struct {
	FailureCode    string `json:"failureCode"`
	FailureMessage string `json:"failureMessage"`
}

type DepositInitiationResponse struct {
	DepositId     string         `json:"depositId"`
	Status        string         `json:"status"` // ACCEPTED | REJECTED | DUPLICATE_IGNORED
	Created       string         `json:"created"`
	FailureReason *FailureReason `json:"failureReason,omitempty"`
}

// --- Statut d'un dépôt -------------------------------------------------------

type DepositStatusResponse struct {
	Status string       `json:"status"` // FOUND | NOT_FOUND
	Data   *DepositData `json:"data,omitempty"`
}

type DepositData struct {
	DepositId            string         `json:"depositId"`
	Status               string         `json:"status"` // ACCEPTED | PROCESSING | IN_RECONCILIATION | COMPLETED | FAILED
	Amount               string         `json:"amount"`
	Currency             string         `json:"currency"`
	Country              string         `json:"country"`
	ProviderTransactionId string        `json:"providerTransactionId"`
	FailureReason        *FailureReason `json:"failureReason,omitempty"`
}

// InitiateDeposit — POST /v2/deposits
func (c *PawaPayClient) InitiateDeposit(ctx context.Context, req DepositRequest) (*DepositInitiationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.BaseURL+"/v2/deposits", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d: %s", ErrPaymentFailed, resp.StatusCode, string(respBody))
	}

	var result DepositInitiationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDepositStatus — GET /v2/deposits/{depositId}
func (c *PawaPayClient) GetDepositStatus(ctx context.Context, depositId string) (*DepositStatusResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.cfg.BaseURL+"/v2/deposits/"+depositId, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d: %s", ErrPaymentFailed, resp.StatusCode, string(respBody))
	}

	var result DepositStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *PawaPayClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
}

// --- Opérateurs mobile money (pays de la zone CFA) ---------------------------

// Operator définit un opérateur mobile money supporté par PawaPay.
type Operator struct {
	Label      string // affiché à l'acheteur
	Provider   string // code PawaPay
	Country    string // ISO 3166-1 alpha-3
	DialCode   string // indicatif téléphonique du pays
}

var XOFOperators = []Operator{
	// Sénégal
	{Label: "Orange Money", Provider: "ORANGE_SEN", Country: "SEN", DialCode: "221"},
	{Label: "Wave", Provider: "WAVE_SEN", Country: "SEN", DialCode: "221"},
	{Label: "Free Money", Provider: "FREE_SEN", Country: "SEN", DialCode: "221"},
	// Côte d'Ivoire
	{Label: "MTN MoMo", Provider: "MTN_MOMO_CIV", Country: "CIV", DialCode: "225"},
	{Label: "Orange Money", Provider: "ORANGE_CIV", Country: "CIV", DialCode: "225"},
	{Label: "Wave", Provider: "WAVE_CIV", Country: "CIV", DialCode: "225"},
	// Bénin
	{Label: "MTN MoMo", Provider: "MTN_MOMO_BEN", Country: "BEN", DialCode: "229"},
	{Label: "Moov Money", Provider: "MOOV_BEN", Country: "BEN", DialCode: "229"},
	// Burkina Faso
	{Label: "Moov Money", Provider: "MOOV_BFA", Country: "BFA", DialCode: "226"},
	{Label: "Orange Money", Provider: "ORANGE_BFA", Country: "BFA", DialCode: "226"},
}

// ResolveOperator retourne l'opérateur correspondant au couple (pays, opérateur).
func ResolveOperator(country, operator string) (*Operator, error) {
	for i := range XOFOperators {
		o := &XOFOperators[i]
		if strings.EqualFold(o.Country, country) && strings.EqualFold(o.Label, operator) {
			return o, nil
		}
	}
	return nil, fmt.Errorf("opérateur non supporté pour ce pays: %s / %s", country, operator)
}

var nonDigits = regexp.MustCompile(`\D`)

// NormalizePhone convertit un numéro local (ex "77 123 45 67") en MSISDN
// international exigé par PawaPay (chiffres seuls, code pays, pas de zéro initial).
func NormalizePhone(dialCode, phone string) (string, error) {
	digits := nonDigits.ReplaceAllString(phone, "")
	if digits == "" {
		return "", errors.New("numéro de téléphone invalide")
	}
	// Déjà au format international (code pays présent) ?
	if len(digits) >= len(dialCode)+6 && strings.HasPrefix(digits, dialCode) {
		return digits, nil
	}
	// Numéro local commençant par 0 : on retire le 0 puis on préfixe l'indicatif.
	digits = strings.TrimPrefix(digits, "0")
	if len(digits) < 6 {
		return "", errors.New("numéro de téléphone invalide")
	}
	return dialCode + digits, nil
}
