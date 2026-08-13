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

// MetadataItem est un objet à clé dynamique, ex: {"saleId": "abc"} ou
// {"customerId": "x@y.com", "isPII": true}. PawaPay exige ce format précis
// (pas de structure {"key":..,"value":..}, qui déclenche DUPLICATE_METADATA_FIELD
// car "key" et "value" seraient alors vus comme des noms de champ dupliqués
// entre plusieurs entrées du tableau).
type MetadataItem map[string]interface{}

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

// --- Page de paiement hébergée (Payment Page) --------------------------------
//
// Alternative à InitiateDeposit : au lieu de lui demander pays/opérateur/téléphone
// dans notre propre formulaire, on redirige l'acheteur vers une page hébergée par
// PawaPay où il choisit lui-même son opérateur mobile money et confirme le
// paiement. PawaPay le redirige ensuite vers ReturnUrl, et le webhook habituel
// (voir webhook_handler.go) confirme le statut final du dépôt.
//
// Schéma vérifié contre docs.pawapay.io/v2/api-reference/payment-page (2026-08-13).

type AmountDetails struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type PaymentPageRequest struct {
	DepositId       string         `json:"depositId"` // même UUID que pour un dépôt classique
	ReturnUrl       string         `json:"returnUrl"`
	AmountDetails   AmountDetails  `json:"amountDetails"`
	Country         string         `json:"country,omitempty"`         // optionnel : présélectionne le pays sur la page
	Reason          string         `json:"reason,omitempty"`          // 1-50 caractères, affiché à l'acheteur
	CustomerMessage string         `json:"customerMessage,omitempty"` // 4-22 caractères
	Language        string         `json:"language,omitempty"`        // "EN" ou "FR"
	Metadata        []MetadataItem `json:"metadata,omitempty"`
}

type PaymentPageResponse struct {
	DepositId     string         `json:"depositId,omitempty"`
	RedirectUrl   string         `json:"redirectUrl"`
	Status        string         `json:"status,omitempty"` // présent uniquement en cas de rejet : "REJECTED"
	FailureReason *FailureReason `json:"failureReason,omitempty"`
}

// CreatePaymentPage — POST /v2/paymentpage
func (c *PawaPayClient) CreatePaymentPage(ctx context.Context, req PaymentPageRequest) (*PaymentPageResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.BaseURL+"/v2/paymentpage", bytes.NewReader(body))
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("%w: status %d: %s", ErrPaymentFailed, resp.StatusCode, string(respBody))
	}

	var result PaymentPageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	if result.Status == "REJECTED" {
		code := "rejected"
		if result.FailureReason != nil {
			code = result.FailureReason.FailureCode
		}
		return nil, fmt.Errorf("%w: rejected: %s: %s", ErrPaymentFailed, code, string(respBody))
	}
	return &result, nil
}

// --- Versements vendeurs (payouts) ------------------------------------------

type PayoutRequest struct {
	PayoutId          string         `json:"payoutId"`
	Recipient         Payer          `json:"recipient"`
	Amount            string         `json:"amount"`
	Currency          string         `json:"currency"`
	ClientReferenceId string         `json:"clientReferenceId,omitempty"`
	CustomerMessage   string         `json:"customerMessage,omitempty"`
	Metadata          []MetadataItem `json:"metadata,omitempty"`
}

type PayoutInitiationResponse struct {
	PayoutId      string         `json:"payoutId"`
	Status        string         `json:"status"` // ACCEPTED | REJECTED | DUPLICATE_IGNORED
	Created       string         `json:"created"`
	FailureReason *FailureReason `json:"failureReason,omitempty"`
}

type PayoutStatusResponse struct {
	Status string      `json:"status"` // FOUND | NOT_FOUND
	Data   *PayoutData `json:"data,omitempty"`
}

type PayoutData struct {
	PayoutId              string         `json:"payoutId"`
	Status                string         `json:"status"` // ACCEPTED | ENQUEUED | PROCESSING | IN_RECONCILIATION | COMPLETED | FAILED
	Amount                string         `json:"amount"`
	Currency               string        `json:"currency"`
	Country                string        `json:"country"`
	ProviderTransactionId string         `json:"providerTransactionId"`
	FailureReason          *FailureReason `json:"failureReason,omitempty"`
}

// InitiatePayout — POST /v2/payouts
func (c *PawaPayClient) InitiatePayout(ctx context.Context, req PayoutRequest) (*PayoutInitiationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.BaseURL+"/v2/payouts", bytes.NewReader(body))
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

	var result PayoutInitiationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPayoutStatus — GET /v2/payouts/{payoutId}
func (c *PawaPayClient) GetPayoutStatus(ctx context.Context, payoutId string) (*PayoutStatusResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.cfg.BaseURL+"/v2/payouts/"+payoutId, nil)
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

	var result PayoutStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Remboursements (refunds) -----------------------------------------------

type RefundRequest struct {
	RefundId          string         `json:"refundId"`
	DepositId         string         `json:"depositId"`
	Amount            string         `json:"amount,omitempty"` // vide = remboursement total
	Currency          string         `json:"currency"`
	ClientReferenceId string         `json:"clientReferenceId,omitempty"`
	Metadata          []MetadataItem `json:"metadata,omitempty"`
}

type RefundInitiationResponse struct {
	RefundId      string         `json:"refundId"`
	Status        string         `json:"status"` // ACCEPTED | REJECTED | DUPLICATE_IGNORED
	Created       string         `json:"created"`
	FailureReason *FailureReason `json:"failureReason,omitempty"`
}

type RefundStatusResponse struct {
	Status string      `json:"status"` // FOUND | NOT_FOUND
	Data   *RefundData `json:"data,omitempty"`
}

type RefundData struct {
	RefundId               string         `json:"refundId"`
	Status                 string         `json:"status"` // ACCEPTED | ENQUEUED | PROCESSING | IN_RECONCILIATION | COMPLETED | FAILED
	Amount                 string         `json:"amount"`
	Currency               string         `json:"currency"`
	Country                string         `json:"country"`
	ProviderTransactionId  string         `json:"providerTransactionId"`
	FailureReason          *FailureReason `json:"failureReason,omitempty"`
}

// InitiateRefund — POST /v2/refunds
func (c *PawaPayClient) InitiateRefund(ctx context.Context, req RefundRequest) (*RefundInitiationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.BaseURL+"/v2/refunds", bytes.NewReader(body))
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

	var result RefundInitiationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRefundStatus — GET /v2/refunds/{refundId}
func (c *PawaPayClient) GetRefundStatus(ctx context.Context, refundId string) (*RefundStatusResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.cfg.BaseURL+"/v2/refunds/"+refundId, nil)
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

	var result RefundStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
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

// GatewaySettingKey retourne la clé settings ("gateway_...") correspondant à un
// provider PawaPay (ex: "ORANGE_SEN" -> "gateway_orange_money"), pour vérifier
// si l'admin a désactivé cette passerelle. Chaîne vide si non reconnu (jamais
// bloqué dans ce cas, pour ne pas casser un nouvel opérateur pas encore mappé).
func GatewaySettingKey(provider string) string {
	switch {
	case strings.HasPrefix(provider, "ORANGE_"):
		return "gateway_orange_money"
	case strings.HasPrefix(provider, "WAVE_"):
		return "gateway_wave"
	case strings.HasPrefix(provider, "MTN_MOMO"):
		return "gateway_mtn_momo"
	case strings.HasPrefix(provider, "FREE_"):
		return "gateway_free_money"
	case strings.HasPrefix(provider, "MOOV_"):
		return "gateway_moov_money"
	default:
		return ""
	}
}

// ResolveOperator retourne l'opérateur correspondant au couple (pays, opérateur).
// "operator" peut être le libellé ("Orange Money") ou le code PawaPay
// ("ORANGE_SEN") — le frontend envoie le code depuis la sélection par cartes.
func ResolveOperator(country, operator string) (*Operator, error) {
	for i := range XOFOperators {
		o := &XOFOperators[i]
		if !strings.EqualFold(o.Country, country) {
			continue
		}
		if strings.EqualFold(o.Label, operator) || strings.EqualFold(o.Provider, operator) {
			return o, nil
		}
	}
	return nil, fmt.Errorf("opérateur non supporté pour ce pays: %s / %s", country, operator)
}

// --- Pays PawaPay et conversion de devise (Payment Page uniquement) ---------
//
// Liste complète des 20 pays couverts par PawaPay (docs.pawapay.io/v2/docs/providers).
// Utilisée uniquement côté checkout (Payment Page) : le catalogue DIARRA est
// tarifé en FCFA (XOF), donc pour tout pays hors zone XOF/XAF (parité fixe),
// le montant doit être converti dans la devise locale avant d'être envoyé à
// PawaPay. Les versements vendeur restent en XOF (XOFOperators ci-dessus) —
// les convertir nécessiterait une seconde conversion (devise acheteur → XOF
// → devise versement), hors périmètre pour l'instant.
var CountryCurrency = map[string]string{
	"BEN": "XOF", "BFA": "XOF", "CMR": "XAF", "CIV": "XOF", "COD": "CDF",
	"ETH": "ETB", "GAB": "XAF", "GHA": "GHS", "KEN": "KES", "LSO": "LSL",
	"MWI": "MWK", "MOZ": "MZN", "NGA": "NGN", "COG": "XAF", "RWA": "RWF",
	"SEN": "XOF", "SLE": "SLE", "TZA": "TZS", "UGA": "UGX", "ZMB": "ZMW",
}

// USDRates — unités de devise pour 1 USD. Table statique fixée manuellement
// (source : exchangerate-api.com, relevé le 2026-08-13), pas d'appel API
// externe en temps réel. Les taux fluctuent — à remettre à jour périodiquement,
// en particulier NGN, GHS et TZS qui bougent vite.
var USDRates = map[string]float64{
	"XOF": 568.76, "XAF": 568.76, "GHS": 11.58, "KES": 129.24, "NGN": 1360.25,
	"RWF": 1474.52, "UGX": 3696.31, "TZS": 2644.86, "ZMW": 18.79, "MWK": 1739.93,
	"MZN": 63.75, "LSL": 16.14, "ETB": 161.34, "SLE": 24.65, "CDF": 2281.76,
}

// ConvertFromXOF convertit un montant en FCFA (XOF) vers la devise cible, en
// passant par le dollar US comme pivot (table USDRates). Retourne le montant
// arrondi en string, au format attendu par l'API PawaPay.
func ConvertFromXOF(amountXOF int, targetCurrency string) (string, error) {
	if targetCurrency == "" || targetCurrency == "XOF" || targetCurrency == "XAF" {
		return fmt.Sprintf("%d", amountXOF), nil
	}
	xofRate, ok := USDRates["XOF"]
	if !ok {
		return "", errors.New("taux XOF introuvable")
	}
	targetRate, ok := USDRates[targetCurrency]
	if !ok {
		return "", fmt.Errorf("devise non supportée: %s", targetCurrency)
	}
	usd := float64(amountXOF) / xofRate
	converted := usd * targetRate
	return fmt.Sprintf("%.0f", converted), nil
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
