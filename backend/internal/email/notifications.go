package email

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// NotificationService regroupe l'envoi des emails transactionnels.
// Il ne dépend plus d'un fournisseur précis : n'importe quel MailSender
// (Resend, SMTP, Postmark...) peut être injecté.
type NotificationService struct {
	client      MailSender
	frontendURL string
}

func NewNotificationService(client MailSender, frontendURL string) *NotificationService {
	return &NotificationService{
		client:      client,
		frontendURL: frontendURL,
	}
}

// emailLayout est le gabarit HTML commun : header dégradé DIARRA, carte
// blanche, pied de page. {{INNER}} est remplacé par le contenu de chaque mail.
// CSS inline uniquement (fiabilité Gmail / Outlook / mobile).
const emailLayout = `<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body style="margin:0;padding:0;background-color:#f2f7f4;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f2f7f4;">
<tr><td align="center" style="padding:32px 16px;">
<table role="presentation" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;">

<tr>
<td style="background:linear-gradient(135deg,#0e4431 0%,#0f7a50 55%,#10a05f 100%);border-radius:16px 16px 0 0;padding:28px 32px;">
<img src="{{LOGO_URL}}" alt="DIARRA" height="28" style="height:28px;width:auto;display:block;border:0;">
<span style="display:block;font-family:Consolas,'Courier New',monospace;font-size:12px;color:#c9f22e;margin-top:8px;letter-spacing:1px;">// produits numériques en FCFA</span>
</td>
</tr>

<tr>
<td style="background-color:#ffffff;border-radius:0 0 16px 16px;padding:36px 32px;">
{{INNER}}
</td>
</tr>

<tr>
<td style="padding:20px 32px 0;">
<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:1.7;color:#6b7c74;text-align:center;">
&copy; DIARRA &middot; marketplace de produits numériques<br>
Si vous n&rsquo;êtes pas à l&rsquo;origine de cet email, ignorez-le simplement.<br>
Vous ne trouvez pas cet email&nbsp;? Vérifiez votre dossier spam, ou écrivez-nous à <a href="mailto:support@abmcy.com" style="color:#0f7a50;">support@abmcy.com</a>.
</p>
</td>
</tr>

</table>
</td></tr>
</table>
</body>
</html>`

// ctaBlock est le bouton d'action vert signature + lien de repli.
const ctaBlock = `<table role="presentation" cellpadding="0" cellspacing="0" style="margin:24px 0 0;">
<tr>
<td style="border-radius:999px;background:linear-gradient(135deg,#0e4431 0%,#0f7a50 55%,#10a05f 100%);">
<a href="{{URL}}" style="display:inline-block;padding:14px 32px;font-family:Arial,Helvetica,sans-serif;font-size:15px;font-weight:700;color:#ffffff;text-decoration:none;">{{TEXT}}</a>
</td>
</tr>
</table>
<p style="margin:8px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:13px;color:#6b7c74;">
<a href="{{URL}}" style="color:#0f7a50;text-decoration:underline;">Si le bouton ne s&rsquo;affiche pas, cliquez ici.</a>
</p>`

// renderEmail enveloppe le contenu dans le gabarit commun, logo DIARRA inclus
// (URL absolue — obligatoire en email, les clients ne chargent pas de
// fichiers locaux).
func (n *NotificationService) renderEmail(inner string) string {
	out := strings.Replace(emailLayout, "{{INNER}}", inner, 1)
	logoURL := strings.TrimSuffix(n.frontendURL, "/") + "/brand/diarra-logo-white-text.png"
	return strings.Replace(out, "{{LOGO_URL}}", logoURL, 1)
}

// contentHTML assemble titre + corps + (optionnel) bouton d'action.
func contentHTML(heading, body, ctaText, ctaURL string) string {
	cta := ""
	if ctaURL != "" && ctaText != "" {
		cta = strings.ReplaceAll(ctaBlock, "{{URL}}", ctaURL)
		cta = strings.ReplaceAll(cta, "{{TEXT}}", ctaText)
	}
	return fmt.Sprintf(`
<h1 style="margin:0 0 16px;font-family:Arial,Helvetica,sans-serif;font-size:22px;line-height:1.35;color:#052018;">%s</h1>
%s%s`, heading, body, cta)
}

// formatCFA formate un montant en FCFA avec séparateur de milliers ("12 500").
func formatCFA(n int) string {
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func (n *NotificationService) SendWelcome(ctx context.Context, to string) error {
	inner := contentHTML(
		"Bienvenue sur DIARRA !",
		`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Votre compte a été créé avec succès. Parcourez le catalogue et découvrez des produits numériques vendus en FCFA, en toute sécurité.</p>`,
		"Parcourir le catalogue",
		n.frontendURL+"/catalog",
	)
	return n.client.Send(ctx, to, "Bienvenue sur DIARRA", n.renderEmail(inner))
}

func (n *NotificationService) SendEmailVerification(ctx context.Context, to, token string) error {
	link := fmt.Sprintf("%s/auth/verify-email?token=%s", n.frontendURL, token)
	inner := contentHTML(
		"Vérifiez votre email",
		`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Cliquez sur le bouton ci-dessous pour confirmer votre adresse email. Votre compte ne sera pleinement actif qu&rsquo;après cette vérification.</p>`,
		"Vérifier mon email",
		link,
	)
	return n.client.Send(ctx, to, "Vérifiez votre email", n.renderEmail(inner))
}

func (n *NotificationService) SendPasswordReset(ctx context.Context, to, token string) error {
	link := fmt.Sprintf("%s/auth/reset-password?token=%s", n.frontendURL, token)
	inner := contentHTML(
		"Réinitialisation du mot de passe",
		`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Cliquez sur le bouton ci-dessous pour choisir un nouveau mot de passe.</p>
<p style="margin:16px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:13px;line-height:1.6;color:#6b7c74;">Ce lien expire dans 1 heure. Si vous n&rsquo;êtes pas à l&rsquo;origine de cette demande, ignorez cet email.</p>`,
		"Choisir un nouveau mot de passe",
		link,
	)
	return n.client.Send(ctx, to, "Réinitialisation de votre mot de passe", n.renderEmail(inner))
}

// MaxEmailAttachmentBytes — au-delà, le fichier n'est plus joint à l'email de
// confirmation (seul le lien de téléchargement reste) : la plupart des
// fournisseurs d'envoi transactionnel rejettent l'email entier au-delà de
// 10-25 Mo selon le fournisseur — mieux vaut un email sans pièce jointe
// qu'un email qui échoue silencieusement pour un gros fichier.
const MaxEmailAttachmentBytes = 10 * 1024 * 1024 // 10 Mo

// SendOrderConfirmed — le lien pointe vers la page de suivi publique (par
// checkout_token) plutôt que /orders/{id}, car un acheteur invité n'a pas de
// session pour accéder à son espace connecté. Si le fichier acheté est fourni
// (attachment non nil) et sous la limite de taille, il est aussi joint
// directement à l'email — le lien reste affiché dans tous les cas.
func (n *NotificationService) SendOrderConfirmed(ctx context.Context, to, buyerName, productTitle string, amountCFA int, checkoutToken string, attachment *Attachment) error {
	link := fmt.Sprintf("%s/checkout/return?token=%s", n.frontendURL, checkoutToken)
	note := ""
	if attachment != nil {
		note = " Il est aussi joint à cet email."
	}
	body := fmt.Sprintf(`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Bonjour %s, votre commande a été payée avec succès. Téléchargez votre fichier depuis le bouton ci-dessous.%s</p>
<div style="margin:16px 0 0;padding:16px 18px;background-color:#f2f7f4;border-left:4px solid #0f7a50;border-radius:8px;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:1.6;color:#0a3225;">
<strong>%s</strong><br>
Montant payé : <strong>%s FCFA</strong>
</div>`, buyerName, note, productTitle, formatCFA(amountCFA))
	inner := contentHTML("Paiement confirmé !", body, "Télécharger mon fichier", link)
	if attachment != nil && len(attachment.Content) <= MaxEmailAttachmentBytes {
		return n.client.Send(ctx, to, "Commande confirmée", n.renderEmail(inner), *attachment)
	}
	return n.client.Send(ctx, to, "Commande confirmée", n.renderEmail(inner))
}

// SendPaymentFailed — envoyé quand PawaPay confirme l'échec d'un paiement
// (fonds insuffisants, opération annulée par le client, etc.). Le lien
// renvoie vers la fiche produit pour retenter l'achat facilement.
func (n *NotificationService) SendPaymentFailed(ctx context.Context, to, buyerName, productTitle, productID string) error {
	link := fmt.Sprintf("%s/product?id=%s", n.frontendURL, productID)
	body := fmt.Sprintf(`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Bonjour %s, votre paiement pour <strong>%s</strong> n&rsquo;a pas abouti.</p>
<p style="margin:16px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:1.6;color:#6b7c74;">Ça arrive quand le solde mobile money est insuffisant, ou si l&rsquo;opération a été annulée pendant la validation. Aucun montant n&rsquo;a été débité pour cette tentative. Vous pouvez réessayer directement depuis le bouton ci-dessous.</p>`, buyerName, productTitle)
	inner := contentHTML("Le paiement n'a pas abouti", body, "Réessayer le paiement", link)
	return n.client.Send(ctx, to, "Votre paiement n'a pas abouti", n.renderEmail(inner))
}

func (n *NotificationService) SendDeliveryReady(ctx context.Context, to, orderID string) error {
	link := fmt.Sprintf("%s/orders/%s", n.frontendURL, orderID)
	inner := contentHTML(
		"Votre fichier est disponible !",
		`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Le fichier que vous avez acheté est prêt à être téléchargé. Bonne utilisation !</p>`,
		"Télécharger mon fichier",
		link,
	)
	return n.client.Send(ctx, to, "Votre fichier est disponible", n.renderEmail(inner))
}

func (n *NotificationService) SendRefundConfirmed(ctx context.Context, to, buyerName, productTitle string, amountCFA int) error {
	body := fmt.Sprintf(`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Bonjour %s, votre remboursement a été traité avec succès.</p>
<div style="margin:16px 0 0;padding:16px 18px;background-color:#f2f7f4;border-left:4px solid #0f7a50;border-radius:8px;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:1.6;color:#0a3225;">
<strong>%s</strong><br>
Montant remboursé : <strong>%s FCFA</strong>
</div>
<p style="margin:16px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:13px;line-height:1.6;color:#6b7c74;">Le délai de réception dépend de votre opérateur mobile money.</p>`, buyerName, productTitle, formatCFA(amountCFA))
	inner := contentHTML("Remboursement confirmé", body, "", "")
	return n.client.Send(ctx, to, "Votre remboursement a été traité", n.renderEmail(inner))
}

func (n *NotificationService) SendVendorSale(ctx context.Context, to, productTitle string, amount int) error {
	body := fmt.Sprintf(`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Le produit <strong>%s</strong> vient d&rsquo;être vendu.</p>
<p style="margin:16px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Montant après commission plateforme de 15&nbsp;%% : <strong style="color:#0f7a50;">%s FCFA</strong></p>`, productTitle, formatCFA(amount))
	inner := contentHTML("Nouvelle vente sur votre boutique !", body, "Voir mes revenus", n.frontendURL+"/vendor/earnings")
	return n.client.Send(ctx, to, "Nouvelle vente !", n.renderEmail(inner))
}

func (n *NotificationService) SendPayoutConfirmed(ctx context.Context, to string, amount int) error {
	body := fmt.Sprintf(`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Votre versement de <strong style="color:#0f7a50;">%s FCFA</strong> a été traité avec succès.</p>`, formatCFA(amount))
	inner := contentHTML("Versement effectué", body, "Voir mes revenus", n.frontendURL+"/vendor/earnings")
	return n.client.Send(ctx, to, "Versement confirmé", n.renderEmail(inner))
}

// SendSupportContact notifie un agent support d'un nouveau message envoyé
// depuis le widget de contact public du site (visiteur non authentifié).
// Le bouton d'action pointe directement vers une réponse par email (mailto:)
// ou WhatsApp (wa.me) selon le canal choisi par le visiteur. name/message/
// contactValue sont fournis par un visiteur anonyme : échappés avant
// insertion dans le HTML pour éviter toute injection dans l'email.
func (n *NotificationService) SendSupportContact(ctx context.Context, to, name, contactMethod, contactValue, message string) error {
	safeName := html.EscapeString(name)
	safeMessage := strings.ReplaceAll(html.EscapeString(message), "\n", "<br>")
	safeContactValue := html.EscapeString(contactValue)

	var ctaLabel, ctaURL string
	if contactMethod == "whatsapp" {
		ctaLabel = "Répondre sur WhatsApp"
		ctaURL = "https://wa.me/" + nonDigits.ReplaceAllString(contactValue, "") +
			"?text=" + url.QueryEscape("Bonjour "+name+", ici le support DIARRA. ")
	} else {
		ctaLabel = "Répondre par email"
		ctaURL = "mailto:" + contactValue + "?subject=" + url.QueryEscape("Re: votre message à DIARRA")
	}

	body := fmt.Sprintf(`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Nouveau message depuis le site, de <strong>%s</strong> (%s : %s).</p>
<div style="margin:16px 0 0;padding:16px 18px;background-color:#f2f7f4;border-left:4px solid #0f7a50;border-radius:8px;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:1.6;color:#0a3225;">%s</div>`,
		safeName, contactMethod, safeContactValue, safeMessage)
	inner := contentHTML("Nouveau contact support", body, ctaLabel, ctaURL)
	return n.client.Send(ctx, to, "Nouveau message support : "+name, n.renderEmail(inner))
}

var nonDigits = regexp.MustCompile(`\D`)

func (n *NotificationService) SendOTP(ctx context.Context, to, code, purpose string) error {
	subject := "Votre code de vérification DIARRA"
	body := fmt.Sprintf(`<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.7;color:#0a3225;">Votre code de vérification pour %s est : <strong style="font-size:20px;color:#0f7a50;">%s</strong></p>
<p style="margin:16px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:13px;line-height:1.6;color:#6b7c74;">Ce code expire dans 10 minutes. Ne le partagez avec personne.</p>`, purpose, code)
	inner := contentHTML("Vérification du compte", body, "", "")
	return n.client.Send(ctx, to, subject, n.renderEmail(inner))
}
