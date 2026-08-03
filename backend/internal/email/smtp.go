package email

import (
	"context"
	"fmt"
	"time"

	"github.com/wneessen/go-mail"
)

// SMTPConfig paramètre un serveur SMTP (Mailtrap en dev, fournisseur en prod).
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string // ex: "DIARRA <no-reply@diarra.example>"
}

// SMTPClient envoie les emails via un serveur SMTP (STARTTLS/TLS).
type SMTPClient struct {
	from   string
	client *mail.Client
}

// NewSMTPClient construit un client SMTP. En dev, Mailtrap : port 587 + STARTTLS.
func NewSMTPClient(cfg SMTPConfig) (*SMTPClient, error) {
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	client, err := mail.NewClient(cfg.Host,
		mail.WithPort(cfg.Port),
		mail.WithTLSPortPolicy(mail.TLSMandatory),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(cfg.Username),
		mail.WithPassword(cfg.Password),
		mail.WithTimeout(15*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("smtp: client: %w", err)
	}
	return &SMTPClient{from: cfg.From, client: client}, nil
}

func (c *SMTPClient) Send(ctx context.Context, to, subject, html string) error {
	m := mail.NewMsg()
	if err := m.From(c.from); err != nil {
		return fmt.Errorf("smtp: from: %w", err)
	}
	if err := m.To(to); err != nil {
		return fmt.Errorf("smtp: to: %w", err)
	}
	m.Subject(subject)
	m.SetBodyString(mail.TypeTextHTML, html)

	if err := c.client.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("smtp: send: %w", err)
	}
	return nil
}
