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

type ResendClient struct {
	apiKey  string
	from    string
	baseURL string
	client  *http.Client
}

type ResendConfig struct {
	APIKey string
	From   string // ex: "DIARRA <no-reply@diarra.example>"
}

var _ MailSender = (*ResendClient)(nil)

func NewResendClient(cfg ResendConfig) *ResendClient {
	return &ResendClient{
		apiKey:  cfg.APIKey,
		from:    cfg.From,
		baseURL: "https://api.resend.com",
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

type resendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"` // base64
}

type sendRequest struct {
	From        string             `json:"from"`
	To          string             `json:"to"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

// Send envoie un email HTML transactionnel via l'API Resend.
func (c *ResendClient) Send(ctx context.Context, to, subject, html string, attachments ...Attachment) error {
	var atts []resendAttachment
	for _, a := range attachments {
		atts = append(atts, resendAttachment{
			Filename: a.Filename,
			Content:  base64.StdEncoding.EncodeToString(a.Content),
		})
	}
	body, err := json.Marshal(sendRequest{
		From:        c.from,
		To:          to,
		Subject:     subject,
		HTML:        html,
		Attachments: atts,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
