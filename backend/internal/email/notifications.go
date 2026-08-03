package email

import (
	"context"
	"fmt"
)

// NotificationService regroupe l'envoi des emails transactionnels.
type NotificationService struct {
	client      *ResendClient
	frontendURL string
}

func NewNotificationService(client *ResendClient, frontendURL string) *NotificationService {
	return &NotificationService{
		client:      client,
		frontendURL: frontendURL,
	}
}

func (n *NotificationService) SendWelcome(ctx context.Context, to string) error {
	return n.client.Send(ctx, to, "Bienvenue sur DIARRA", `
		<h2>Bienvenue sur DIARRA !</h2>
		<p>Votre compte a été créé avec succès.</p>
		<p>Parcourez le catalogue et découvrez des produits numériques.</p>`)
}

func (n *NotificationService) SendEmailVerification(ctx context.Context, to, token string) error {
	link := fmt.Sprintf("%s/auth/verify-email?token=%s", n.frontendURL, token)
	return n.client.Send(ctx, to, "Vérifiez votre email", fmt.Sprintf(`
		<h2>Vérification de votre email</h2>
		<p>Cliquez sur le lien ci-dessous pour vérifier votre adresse email :</p>
		<p><a href="%s">Vérifier mon email</a></p>`, link))
}

func (n *NotificationService) SendOrderConfirmed(ctx context.Context, to, orderID string) error {
	link := fmt.Sprintf("%s/orders/%s", n.frontendURL, orderID)
	return n.client.Send(ctx, to, "Commande confirmée", fmt.Sprintf(`
		<h2>Paiement confirmé !</h2>
		<p>Votre commande <strong>%s</strong> a été payée avec succès.</p>
		<p>Vous serez notifié dès que le fichier sera livré.</p>
		<p><a href="%s">Suivre ma commande</a></p>`, orderID, link))
}

func (n *NotificationService) SendDeliveryReady(ctx context.Context, to, orderID string) error {
	link := fmt.Sprintf("%s/orders/%s", n.frontendURL, orderID)
	return n.client.Send(ctx, to, "Votre fichier est disponible", fmt.Sprintf(`
		<h2>Votre fichier est disponible !</h2>
		<p>Le fichier que vous avez acheté est prêt à être téléchargé.</p>
		<p><a href="%s">Télécharger mon fichier</a></p>`, link))
}

func (n *NotificationService) SendVendorSale(ctx context.Context, to, productTitle string, amount int) error {
	return n.client.Send(ctx, to, "Nouvelle vente !", fmt.Sprintf(`
		<h2>Nouvelle vente sur votre boutique</h2>
		<p>Le produit <strong>%s</strong> vient d'être vendu.</p>
		<p>Montant (après commission plateforme de 15%%) : <strong>%d FCFA</strong></p>`,
		productTitle, amount))
}

func (n *NotificationService) SendPayoutConfirmed(ctx context.Context, to string, amount int) error {
	return n.client.Send(ctx, to, "Versement confirmé", fmt.Sprintf(`
		<h2>Versement effectué</h2>
		<p>Votre versement de <strong>%d FCFA</strong> a été traité avec succès.</p>`, amount))
}
