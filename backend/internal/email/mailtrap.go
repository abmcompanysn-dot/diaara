package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MailtrapConfig paramètre le client API Mailtrap.
// En mode sandbox (dev), les emails sont captés dans le bac de test :
// aucun domaine ni DNS requis. Sans SandboxID, on envoie via le domaine démo.
type MailtrapConfig struct {
	APIKey    string
	FromEmail string // ex: "hello@demomailtrap.co" (démo) ou "no-reply@ton-domaine.com"
	FromName  string // ex: "DIARRA"
	SandboxID string // ID du bac de test (visible dans l'URL du sandbox Mailtrap)
}

type mailtrapAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type mailtrapAttachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"` // base64
	Type        string `json:"type,omitempty"`
	Disposition string `json:"disposition,omitempty"`
}

type mailtrapSendRequest struct {
	From        mailtrapAddress      `json:"from"`
	To          []mailtrapAddress    `json:"to"`
	Subject     string               `json:"subject"`
	HTML        string               `json:"html"`
	Category    string               `json:"category,omitempty"`
	Attachments []mailtrapAttachment `json:"attachments,omitempty"`
}

// MailtrapClient envoie les emails via l'API Mailtrap.
type MailtrapClient struct {
	apiKey    string
	fromEmail string
	fromName  string
	sandboxID string
	baseURL   string
	client    *http.Client
}

var _ MailSender = (*MailtrapClient)(nil)

// NewMailtrapClient construit le client. Si SandboxID est renseigné,
// l'endpoint sandbox (https://sandbox.api.mailtrap.io) est utilisé,
// sinon l'endpoint d'envoi transactionnel (https://send.api.mailtrap.io).
func NewMailtrapClient(cfg MailtrapConfig) *MailtrapClient {
	baseURL := "https://send.api.mailtrap.io"
	if cfg.SandboxID != "" {
		baseURL = "https://sandbox.api.mailtrap.io"
	}
	return &MailtrapClient{
		apiKey:    cfg.APIKey,
		fromEmail: cfg.FromEmail,
		fromName:  cfg.FromName,
		sandboxID: cfg.SandboxID,
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *MailtrapClient) Send(ctx context.Context, to, subject, html string, attachments ...Attachment) error {
	var atts []mailtrapAttachment
	for _, a := range attachments {
		atts = append(atts, mailtrapAttachment{
			Filename:    a.Filename,
			Content:     base64.StdEncoding.EncodeToString(a.Content),
			Type:        a.ContentType,
			Disposition: "attachment",
		})
	}
	body, err := json.Marshal(mailtrapSendRequest{
		From:        mailtrapAddress{Email: c.fromEmail, Name: c.fromName},
		To:          []mailtrapAddress{{Email: to}},
		Subject:     subject,
		HTML:        html,
		Category:    "transactional",
		Attachments: atts,
	})
	if err != nil {
		return err
	}

	url := c.baseURL + "/api/send"
	if c.sandboxID != "" {
		url += "/" + c.sandboxID
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "diarra-backend/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mailtrap: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
