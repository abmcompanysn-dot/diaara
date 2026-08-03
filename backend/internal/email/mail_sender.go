package email

import "context"

// MailSender est le contrat commun à tous les fournisseurs d'envoi
// (Resend, SMTP, Postmark...). La v2 rend le système multi-fournisseurs :
// le NotificationService ne dépend plus d'un fournisseur précis.
type MailSender interface {
	Send(ctx context.Context, to, subject, html string) error
}
