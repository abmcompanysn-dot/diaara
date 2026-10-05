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
	"html"
	"strings"
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

// Send envoie le message (texte brut + alternative HTML habillée DIARRA) et
// renvoie son Message-ID (à stocker pour chaîner une éventuelle réponse via
// In-Reply-To/References). La personnalisation (nom du vendeur) est déjà
// dans textBody — voir le gabarit de brouillon généré par VendorMailHandler.
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
	m.AddAlternativeString(mail.TypeTextHTML, renderHTML(textBody))
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

// renderHTML habille un corps texte brut (tel qu'édité dans /admin/vendor-mail)
// dans un gabarit HTML DIARRA minimal. Les sauts de ligne du texte source
// deviennent des paragraphes ; aucun markup n'est interprété (texte
// entièrement échappé) pour qu'un brouillon édité en texte simple reste
// fidèle une fois rendu en HTML.
func renderHTML(textBody string) string {
	var bodyHTML strings.Builder
	for _, para := range strings.Split(strings.TrimSpace(textBody), "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		escaped := strings.ReplaceAll(html.EscapeString(para), "\n", "<br>")
		bodyHTML.WriteString(fmt.Sprintf(`<p style="margin:0 0 14px;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">%s</p>`, escaped))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="fr">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"></head>
<body style="margin:0;padding:0;background-color:#f2f7f4;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f2f7f4;">
<tr><td align="center" style="padding:32px 16px;">
<table role="presentation" cellpadding="0" cellspacing="0" style="max-width:560px;width:100%%;">
<tr><td style="background:linear-gradient(135deg,#0e4431 0%%,#0f7a50 55%%,#10a05f 100%%);border-radius:16px 16px 0 0;padding:24px 28px;">
<span style="display:block;font-family:Arial,Helvetica,sans-serif;font-size:16px;font-weight:700;color:#ffffff;">DIARRA</span>
<span style="display:block;font-family:Consolas,'Courier New',monospace;font-size:11px;color:#c9f22e;margin-top:6px;">// un message de l'équipe</span>
</td></tr>
<tr><td style="background-color:#ffffff;border-radius:0 0 16px 16px;padding:28px;">
%s
</td></tr>
<tr><td style="padding:18px 28px 0;">
<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:1.6;color:#6b7c74;text-align:center;">
Message personnel de l&rsquo;équipe DIARRA. Vous pouvez y répondre directement.
</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`, bodyHTML.String())
}
