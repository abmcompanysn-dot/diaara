package service

import (
	"context"
	"encoding/json"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/diarra/backend/internal/repository"
)

// PushService — envoi de notifications Web Push (RFC 8291), en plus des
// notifications in-app déjà existantes (voir NotificationRepo). Branché sur
// le même point d'accroche que les notifications in-app (WebhookHandler.notify,
// ProductHandler) — un push est envoyé chaque fois qu'une notification
// in-app est créée, sans distinction de type (demande du 2026-09-24).
//
// Nil-safe : si les clés VAPID ne sont pas configurées (dev local, ou avant
// que l'admin les génère en prod), PushService reste nil et
// WebhookHandler.notify continue de fonctionner normalement (juste sans
// push) — même principe que paypal/kpay/yesBusiness ailleurs dans ce repo.
type PushService struct {
	pushRepo *repository.PushRepo
	client   *webpush.Options
}

func NewPushService(pushRepo *repository.PushRepo, vapidPublicKey, vapidPrivateKey, vapidSubject string) *PushService {
	return &PushService{
		pushRepo: pushRepo,
		client: &webpush.Options{
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			// Subject : email ou URL de contact exigé par la spec VAPID (le
			// serveur push d'un navigateur peut s'en servir pour contacter
			// l'expéditeur en cas d'abus) — voir doc RFC 8292.
			Subscriber: vapidSubject,
			TTL:        60 * 60 * 24, // 24h : au-delà, une notification périmée (ex. "commande confirmée" de la veille) n'a plus d'intérêt à être livrée en retard.
		},
	}
}

type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
	Tag   string `json:"tag,omitempty"`
	// Image : grande illustration affichée dans le corps de la notification
	// (voir public/sw.js) — ex. photo du produit vendu/acheté. Vide = pas
	// d'image, dégradation silencieuse.
	Image string `json:"image,omitempty"`
}

// NotifyUser — envoie un push à TOUS les abonnements de cet utilisateur
// (plusieurs appareils possibles). Best-effort : une erreur d'envoi (clé
// périmée, navigateur qui a révoqué l'abonnement...) est ignorée pour cet
// abonnement précis, sans bloquer les autres ni l'appelant — jamais
// d'erreur remontée, cohérent avec le caractère "en plus" du push (le
// canal fiable reste la notification in-app, déjà persistée en base par
// l'appelant AVANT ce point).
func (s *PushService) NotifyUser(ctx context.Context, userID, title, body, link, tag string) {
	s.NotifyUserWithImage(ctx, userID, title, body, link, tag, "")
}

// NotifyUserWithImage — comme NotifyUser, avec une grande illustration
// (image, ex. photo du produit) affichée dans le corps de la notification.
func (s *PushService) NotifyUserWithImage(ctx context.Context, userID, title, body, link, tag, image string) {
	if s == nil || s.pushRepo == nil {
		return
	}
	subs, err := s.pushRepo.ListByUser(ctx, userID)
	if err != nil || len(subs) == 0 {
		return
	}

	payload, err := json.Marshal(pushPayload{Title: title, Body: body, URL: link, Tag: tag, Image: image})
	if err != nil {
		return
	}

	for _, sub := range subs {
		webpushSub := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}
		resp, err := webpush.SendNotification(payload, webpushSub, s.client)
		if err != nil {
			log.Printf("push: envoi échoué pour user=%s endpoint=%s: %v", userID, sub.Endpoint, err)
			continue
		}
		resp.Body.Close()
		// 404/410 : le navigateur a révoqué cet abonnement (désinstallation,
		// permission retirée) — le conserver en base enverrait dans le vide à
		// chaque notification future, on le nettoie dès qu'on le détecte.
		if resp.StatusCode == 404 || resp.StatusCode == 410 {
			_ = s.pushRepo.Delete(ctx, sub.Endpoint)
		}
	}
}
