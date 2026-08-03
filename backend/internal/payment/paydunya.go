package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var ErrPaymentFailed = errors.New("payment request failed")

type PayDunyaConfig struct {
	MasterKey   string
	PrivateKey  string
	Token       string
	BaseURL     string // défaut: https://app.paydunya.com
	ReturnURL   string
	CancelURL   string
	WebhookURL  string
}

type PayDunyaClient struct {
	cfg     PayDunyaConfig
	baseURL string
	client  *http.Client
}

func NewPayDunyaClient(cfg PayDunyaConfig) *PayDunyaClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://app.paydunya.com"
	}
	return &PayDunyaClient{
		cfg:     cfg,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type InvoiceRequest struct {
	Invoice struct {
		TotalAmount int    `json:"total_amount"`
		Description string `json:"description"`
	} `json:"invoice"`
	Store struct {
		Name string `json:"name"`
	} `json:"store"`
	Customer struct {
		Firstname string `json:"firstname"`
		Lastname  string `json:"lastname"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
	} `json:"customer"`
	CustomData map[string]string `json:"custom_data,omitempty"`
	Actions    struct {
		CancelURL string `json:"cancel_url"`
		ReturnURL string `json:"return_url"`
		CallbackURL string `json:"callback_url"`
	} `json:"actions"`
}

type InvoiceResponse struct {
	ResponseText  string `json:"response_text"`
	ResponseCode  string `json:"response_code"`
	Token         string `json:"token"`
	Response      string `json:"response"`
	InvoiceToken  string `json:"invoice_token"`
	InvoiceURL    string `json:"invoice_url"`
}

type CheckoutResponse struct {
	ResponseText string `json:"response_text"`
	ResponseCode string `json:"response_code"`
	Status       string `json:"status"`
	Token        string `json:"token"`
	Invoice      struct {
		TotalAmount int    `json:"total_amount"`
		Token       string `json:"token"`
	} `json:"invoice"`
}

func (c *PayDunyaClient) CreateInvoice(ctx context.Context, req InvoiceRequest) (*InvoiceResponse, error) {
	req.Store.Name = "DIARRA"
	req.Actions.ReturnURL = c.cfg.ReturnURL
	req.Actions.CancelURL = c.cfg.CancelURL
	req.Actions.CallbackURL = c.cfg.WebhookURL

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/checkout-invoice/create", bytes.NewReader(body))
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

	var result InvoiceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if result.ResponseCode != "00" {
		return nil, fmt.Errorf("%w: response_code %s: %s", ErrPaymentFailed, result.ResponseCode, result.ResponseText)
	}

	return &result, nil
}

func (c *PayDunyaClient) GetCheckoutStatus(ctx context.Context, token string) (*CheckoutResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/v1/checkout-invoice/confirm/"+token, nil)
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

	var result CheckoutResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *PayDunyaClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PAYDUNYA-MASTER-KEY", c.cfg.MasterKey)
	req.Header.Set("PAYDUNYA-PRIVATE-KEY", c.cfg.PrivateKey)
	req.Header.Set("PAYDUNYA-TOKEN", c.cfg.Token)
}
