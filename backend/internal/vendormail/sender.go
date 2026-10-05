// Package vendormail gère la boucle de discussion email avec les vendeurs
// depuis atekossibrunel@diarra.app : lecture IMAP des réponses, envoi SMTP
// des messages validés par l'admin. Volontairement séparé de internal/email
// (notifications transactionnelles automatiques) — ici RIEN ne part sans
// validation explicite d'un admin dans /admin/vendor-mail (voir
// VendorMailRepo et AdminHandler.ListVendorMailDrafts/ApproveVendorMailDraft
// pour le détail du flux).
package vendormail

import (
	"context"
	"fmt"
	"time"

	mail "github.com/wneessen/go-mail"
)

// SenderConfig pointe vers le même serveur Postfix que la boîte
// atekossibrunel@diarra.app (voir Config.FromAddress) — un compte SMTP
// distinct de celui utilisé pour les notifications transactionnelles
// (SMTP_* généralistes), pour que les messages partent bien sous cette
// identité et que les réponses arrivent dans la même boîte IMAP.
type SenderConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string // ex: "Brunel Atekossi <atekossibrunel@diarra.app>"
}

type Sender struct {
	from   string
	client *mail.Client
}

func NewSender(cfg SenderConfig) (*Sender, error) {
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
		return nil, fmt.Errorf("vendormail: smtp client: %w", err)
	}
	return &Sender{from: cfg.FromAddress, client: client}, nil
}

// Send envoie le message et renvoie son Message-ID (à stocker pour chaîner
// une éventuelle réponse via In-Reply-To/References).
func (s *Sender) Send(ctx context.Context, to, subject, textBody string, inReplyTo *string) (string, error) {
	m := mail.NewMsg()
	if err := m.From(s.from); err != nil {
		return "", fmt.Errorf("vendormail: from: %w", err)
	}
	if err := m.To(to); err != nil {
		return "", fmt.Errorf("vendormail: to: %w", err)
	}
	m.Subject(subject)
	m.SetBodyString(mail.TypeTextPlain, textBody)
	m.SetMessageID()

	if inReplyTo != nil && *inReplyTo != "" {
		m.SetGenHeader(mail.HeaderInReplyTo, *inReplyTo)
		m.SetGenHeader(mail.HeaderReferences, *inReplyTo)
	}

	if err := s.client.DialAndSendWithContext(ctx, m); err != nil {
		return "", fmt.Errorf("vendormail: send: %w", err)
	}
	return m.GetMessageID(), nil
}
