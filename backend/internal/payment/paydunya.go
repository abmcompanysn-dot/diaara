package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PayDunya — dépôt mobile money direct via l'API PayDunya (2 étapes,
// contrairement à PawaPay en une seule) :
//  1. Création d'une "facture" (checkout-invoice/create) qui renvoie un
//     token.
//  2. Appel SoftPay pour cet opérateur précis (endpoint ET noms de champs
//     différents par opérateur — pas d'API uniforme comme PawaPay) avec ce
//     token + numéro de téléphone de l'acheteur, qui déclenche la demande
//     d'autorisation sur son téléphone.
//
// Statut final : callback (callback_url posé à la création de facture) ou
// polling de checkout-invoice/confirm/{token} — voir GetInvoiceStatus.
// Docs : https://developers.paydunya.com/doc/FR/http_json (facture),
// https://developers.paydunya.com/doc/FR/softpay (softpay par opérateur).

type PayDunyaConfig struct {
	MasterKey   string // PAYDUNYA-MASTER-KEY
	PrivateKey  string // PAYDUNYA-PRIVATE-KEY
	Token       string // PAYDUNYA-TOKEN
	BaseURL     string // défaut sandbox: https://app.paydunya.com/sandbox-api/v1
	CallbackURL string // URL publique du webhook IPN (déclarée dans le dashboard ET envoyée par requête)
}

type PayDunyaClient struct {
	cfg    PayDunyaConfig
	client *http.Client
}

func NewPayDunyaClient(cfg PayDunyaConfig) *PayDunyaClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://app.paydunya.com/sandbox-api/v1"
	}
	cfg.BaseURL = baseURL
	return &PayDunyaClient{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *PayDunyaClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PAYDUNYA-MASTER-KEY", c.cfg.MasterKey)
	req.Header.Set("PAYDUNYA-PRIVATE-KEY", c.cfg.PrivateKey)
	req.Header.Set("PAYDUNYA-TOKEN", c.cfg.Token)
}

// --- Création de facture -----------------------------------------------------

type InvoiceItem struct {
	Name        string `json:"name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   string `json:"unit_price"`
	TotalPrice  string `json:"total_price"`
	Description string `json:"description,omitempty"`
}

type InvoiceCustomer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone,omitempty"`
}

type Invoice struct {
	Items       map[string]InvoiceItem `json:"items"`
	Customer    InvoiceCustomer        `json:"customer"`
	TotalAmount int                    `json:"total_amount"`
	Description string                 `json:"description"`
}

type InvoiceStore struct {
	Name string `json:"name"`
}

type InvoiceActions struct {
	CancelURL   string `json:"cancel_url,omitempty"`
	ReturnURL   string `json:"return_url,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type CreateInvoiceRequest struct {
	Invoice    Invoice           `json:"invoice"`
	Store      InvoiceStore      `json:"store"`
	CustomData map[string]string `json:"custom_data,omitempty"`
	Actions    InvoiceActions    `json:"actions"`
}

type CreateInvoiceResponse struct {
	ResponseCode string `json:"response_code"` // "00" = succès
	ResponseText string `json:"response_text"` // URL checkout hébergé (non utilisée, on reste sur SoftPay)
	Description  string `json:"description"`
	Token        string `json:"token"`
}

// CreateInvoice — POST /checkout-invoice/create. Le token renvoyé sert
// ensuite d'invoice_token/payment_token (nom variable selon l'opérateur —
// voir SoftpayEndpoint) pour InitiateSoftpay.
func (c *PayDunyaClient) CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*CreateInvoiceResponse, error) {
	if req.Actions.CallbackURL == "" {
		req.Actions.CallbackURL = c.cfg.CallbackURL
	}
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodPost, "/checkout-invoice/create", req, &raw); err != nil {
		return nil, err
	}
	var out CreateInvoiceResponse
	_ = json.Unmarshal(raw, &out)
	if out.ResponseCode != "00" {
		msg := out.Description
		if msg == "" {
			msg = string(raw)
		}
		return nil, fmt.Errorf("%w: %s", ErrPaymentFailed, msg)
	}
	return &out, nil
}

// --- SoftPay (déclenchement du paiement par opérateur) -----------------------

// SoftpayResponse — format commun à tous les opérateurs (voir doc : "success"
// booléen + "message"). Certains opérateurs (Orange Money SN, Wave SN/CI,
// Djamo — tout opérateur dont le message dit "Rediriger vers cette URL pour
// completer le paiement") renvoient en plus une URL vers laquelle
// l'acheteur DOIT être redirigé pour finaliser (page QR code, app Wave...) —
// sans cette redirection le paiement reste bloqué indéfiniment, l'acheteur
// ne reçoit jamais de demande de validation sur son téléphone (constaté
// 2026-09-25 en prod : commande restée "pending" sans jamais d'USSD/prompt
// côté Orange Money SN). OMUrl (spécifique Orange Money SN, dans
// other_url.om_url) ouvre directement l'app Orange Money sur mobile,
// préférée à URL (page web QR code) quand disponible.
type SoftpayResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	URL      string `json:"url"`
	OtherURL struct {
		OMUrl    string `json:"om_url"`
		MaxitURL string `json:"maxit_url"`
	} `json:"other_url"`
}

// RedirectURL — URL vers laquelle rediriger l'acheteur pour finaliser le
// paiement, si l'opérateur en fournit une (voir SoftpayResponse). Préfère
// l'app mobile (om_url) à la page web (url) quand les deux existent.
func (r *SoftpayResponse) RedirectURL() string {
	if r.OtherURL.OMUrl != "" {
		return r.OtherURL.OMUrl
	}
	return r.URL
}

// InitiateSoftpay — POST vers l'endpoint SoftPay de l'opérateur donné (voir
// SoftpayEndpoint), avec le corps déjà construit par BuildSoftpayPayload
// (les noms de champs diffèrent par opérateur, impossible à unifier dans un
// struct Go commun).
func (c *PayDunyaClient) InitiateSoftpay(ctx context.Context, endpoint string, payload map[string]interface{}) (*SoftpayResponse, error) {
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodPost, "/softpay/"+endpoint, payload, &raw); err != nil {
		return nil, err
	}
	var out SoftpayResponse
	_ = json.Unmarshal(raw, &out)
	if !out.Success {
		msg := out.Message
		if msg == "" {
			// Le format de réponse SoftPay diffère par opérateur (pas de
			// "message" standard partout, ex un champ imbriqué ou un code
			// numérique) — le corps brut permet de diagnostiquer plutôt que
			// de renvoyer une erreur vide (constaté 2026-09-25 en prod).
			msg = string(raw)
		}
		return nil, fmt.Errorf("%w: %s", ErrPaymentFailed, msg)
	}
	return &out, nil
}

// --- Statut d'une facture -----------------------------------------------------

type InvoiceStatusResponse struct {
	ResponseCode string `json:"response_code"`
	ResponseText string `json:"response_text"`
	Status       string `json:"status"` // pending | completed | cancelled | failed
	Invoice      struct {
		Token       string `json:"token"`
		TotalAmount int    `json:"total_amount"`
	} `json:"invoice"`
}

// GetInvoiceStatus — GET /checkout-invoice/confirm/{token}.
func (c *PayDunyaClient) GetInvoiceStatus(ctx context.Context, token string) (*InvoiceStatusResponse, error) {
	var out InvoiceStatusResponse
	if err := c.do(ctx, http.MethodGet, "/checkout-invoice/confirm/"+token, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- Versements vendeur (API "disburse") --------------------------------------
//
// Flux en 2 étapes, distinct du dépôt : get-invoice (crée une facture de
// déboursement, renvoie disburse_token) puis submit-invoice (exécute le
// virement). InitiateDisbursement enchaîne les deux pour rester au même
// niveau d'abstraction que PawaPay.InitiatePayout (un seul appel côté
// PaymentProvider, voir payDunyaAdapter.InitiatePayout).

type getDisburseInvoiceRequest struct {
	AccountAlias string `json:"account_alias"`
	Amount       int    `json:"amount"`
	WithdrawMode string `json:"withdraw_mode"`
	CallbackURL  string `json:"callback_url,omitempty"`
}

type getDisburseInvoiceResponse struct {
	ResponseCode  string `json:"response_code"`
	DisburseToken string `json:"disburse_token"`
}

type submitDisburseInvoiceRequest struct {
	DisburseInvoice string `json:"disburse_invoice"`
	DisburseID      string `json:"disburse_id"`
}

type submitDisburseInvoiceResponse struct {
	ResponseCode  string `json:"response_code"`
	Status        string `json:"status"` // success | pending | failed
	ResponseText  string `json:"response_text"`
	TransactionID string `json:"transaction_id"`
}

// DisbursementRequest — versement vers un compte mobile money (AccountAlias
// = numéro local sans indicatif, WithdrawMode = code PayDunya ex
// "orange-money-senegal", voir PayDunyaOperators[].WithdrawMode).
type DisbursementRequest struct {
	AccountAlias string
	Amount       int
	WithdrawMode string
}

type DisbursementResponse struct {
	Success       bool
	DisburseID    string // = disburse_token de get-invoice, sert de référence pour check-status
	TransactionID string
	Message       string
}

// InitiateDisbursement enchaîne get-invoice + submit-invoice.
func (c *PayDunyaClient) InitiateDisbursement(ctx context.Context, req DisbursementRequest) (*DisbursementResponse, error) {
	var invoice getDisburseInvoiceResponse
	if err := c.do(ctx, http.MethodPost, "/disburse/get-invoice", getDisburseInvoiceRequest{
		AccountAlias: req.AccountAlias,
		Amount:       req.Amount,
		WithdrawMode: req.WithdrawMode,
		CallbackURL:  c.cfg.CallbackURL,
	}, &invoice); err != nil {
		return nil, err
	}
	if invoice.ResponseCode != "00" || invoice.DisburseToken == "" {
		return nil, fmt.Errorf("%w: création facture déboursement échouée (code %s)", ErrPaymentFailed, invoice.ResponseCode)
	}

	var submit submitDisburseInvoiceResponse
	if err := c.do(ctx, http.MethodPost, "/disburse/submit-invoice", submitDisburseInvoiceRequest{
		DisburseInvoice: invoice.DisburseToken,
	}, &submit); err != nil {
		return nil, err
	}
	return &DisbursementResponse{
		Success:       submit.ResponseCode == "00" && submit.Status != "failed",
		DisburseID:    invoice.DisburseToken,
		TransactionID: submit.TransactionID,
		Message:       submit.ResponseText,
	}, nil
}

type checkDisburseStatusRequest struct {
	DisburseInvoice string `json:"disburse_invoice"`
}

// DisbursementStatusResponse — réponse de check-status.
type DisbursementStatusResponse struct {
	ResponseCode string `json:"response_code"`
	Status       string `json:"status"` // success | pending | failed
	Message      string `json:"response_text"`
}

// GetDisbursementStatus — POST /disburse/check-status.
func (c *PayDunyaClient) GetDisbursementStatus(ctx context.Context, disburseToken string) (*DisbursementStatusResponse, error) {
	var out DisbursementStatusResponse
	if err := c.do(ctx, http.MethodPost, "/disburse/check-status", checkDisburseStatusRequest{
		DisburseInvoice: disburseToken,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DirectDepositRequest — infos nécessaires pour un dépôt PayDunya de bout en
// bout (création de facture + softpay), niveau d'abstraction équivalent à
// PawaPayClient.InitiateDirectDeposit — appelable sans dépendre de
// *model.Sale/*model.Product (réutilisable par SaleHandler et YesHandler).
type DirectDepositRequest struct {
	ProductTitle string
	BuyerName    string
	BuyerEmail   string
	AmountCFA    int
	Operator     PayDunyaOperator // voir FindPayDunyaOperator
	Phone        string           // numéro local, sans indicatif (ajouté via Operator.DialCode)
	ReturnURL    string
	// OTP — code obtenu par l'acheteur AVANT l'appel, requis uniquement si
	// Operator.RequiresOTP (voir ce champ). Ignoré sinon.
	OTP string
}

// InitiateDirectDeposit enchaîne CreateInvoice + InitiateSoftpay pour
// l'opérateur donné. Retourne le token de facture (à persister comme
// payment_reference, pour le polling/webhook ultérieur) et l'URL vers
// laquelle rediriger l'acheteur.
func (c *PayDunyaClient) InitiateDirectDeposit(ctx context.Context, req DirectDepositRequest) (token string, redirectURL string, err error) {
	if req.Operator.RequiresOTP && req.OTP == "" {
		return "", "", fmt.Errorf("%w: code OTP requis pour %s", ErrPaymentFailed, req.Operator.Label)
	}
	msisdn, err := NormalizePhone(req.Operator.DialCode, req.Phone)
	if err != nil {
		return "", "", err
	}
	// Les endpoints SoftPay attendent le numéro LOCAL (sans indicatif pays,
	// ex "777587999"), contrairement au champ Customer.Phone de la facture
	// qui accepte le format international. Envoyer l'indicatif dans le
	// payload softpay déclenche "numéro invalide"/"numéro non valide du
	// pays X" côté PayDunya (constaté 2026-09-25 sur Orange Money SN et
	// Wave SN en prod, avec un vrai numéro).
	localPhone := msisdn
	if len(msisdn) > len(req.Operator.DialCode) && msisdn[:len(req.Operator.DialCode)] == req.Operator.DialCode {
		localPhone = msisdn[len(req.Operator.DialCode):]
	}

	amountStr := fmt.Sprintf("%d", req.AmountCFA)
	invoice, err := c.CreateInvoice(ctx, CreateInvoiceRequest{
		Invoice: Invoice{
			Items: map[string]InvoiceItem{
				"item_0": {Name: req.ProductTitle, Quantity: 1, UnitPrice: amountStr, TotalPrice: amountStr},
			},
			Customer:    InvoiceCustomer{Name: req.BuyerName, Email: req.BuyerEmail, Phone: msisdn},
			TotalAmount: req.AmountCFA,
			Description: req.ProductTitle,
		},
		Store:   InvoiceStore{Name: "DIARRA"},
		Actions: InvoiceActions{ReturnURL: req.ReturnURL, CancelURL: req.ReturnURL},
	})
	if err != nil {
		return "", "", err
	}

	payload := req.Operator.BuildPayload(req.BuyerName, req.BuyerEmail, localPhone, invoice.Token, req.OTP)
	softpay, err := c.InitiateSoftpay(ctx, req.Operator.Endpoint, payload)
	if err != nil {
		return invoice.Token, "", err
	}
	// Certains opérateurs (voir SoftpayResponse.RedirectURL) exigent que
	// l'acheteur soit redirigé vers une URL précise pour finaliser — sans
	// ça le paiement reste bloqué en attente indéfiniment (aucun USSD/prompt
	// n'est jamais envoyé). Les autres opérateurs (push USSD/SMS direct sur
	// le téléphone, ex Free Money, Expresso, MTN, Moov...) n'en ont pas
	// besoin : on retombe sur ReturnURL, l'acheteur valide directement sur
	// son téléphone pendant que le frontend poll le statut de la commande.
	if redirect := softpay.RedirectURL(); redirect != "" {
		return invoice.Token, redirect, nil
	}
	return invoice.Token, req.ReturnURL, nil
}

// --- HTTP interne -------------------------------------------------------------

func (c *PayDunyaClient) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
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
	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: status %d: %s", ErrPaymentFailed, resp.StatusCode, string(respBody))
	}
	// PayDunya renvoie souvent 200 même en échec métier (success: false /
	// response_code != "00"), à charge de l'appelant de vérifier ces champs
	// — voir CreateInvoice/InitiateSoftpay.
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("%w: réponse illisible: %s", ErrPaymentFailed, string(respBody))
	}
	return nil
}
