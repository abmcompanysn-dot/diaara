package email

import "context"

// Attachment — pièce jointe d'un email (ex: le fichier acheté, joint à
// l'email de confirmation de commande quand sa taille le permet).
type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string
}

// MailSender est le contrat commun à tous les fournisseurs d'envoi
// (Resend, SMTP, Postmark...). La v2 rend le système multi-fournisseurs :
// le NotificationService ne dépend plus d'un fournisseur précis.
type MailSender interface {
	Send(ctx context.Context, to, subject, html string, attachments ...Attachment) error
}
