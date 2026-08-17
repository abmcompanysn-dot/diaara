// Package notify regroupe les canaux de notification autres que l'email
// (aujourd'hui : WhatsApp via CallMeBot).
package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// SendCallMeBotWhatsApp envoie un message WhatsApp via l'API gratuite
// CallMeBot (callmebot.com) : chaque agent support crée sa propre clé API
// personnelle liée à son numéro WhatsApp (voir instructions sur callmebot.com
// — envoyer "I allow callmebot to send me messages" au numéro CallMeBot
// depuis son propre WhatsApp pour obtenir sa clé). Best-effort : l'appelant
// ignore l'erreur (ne bloque jamais l'envoi de l'email, canal principal).
func SendCallMeBotWhatsApp(ctx context.Context, phone, apiKey, text string) error {
	endpoint := "https://api.callmebot.com/whatsapp.php?phone=" + url.QueryEscape(phone) +
		"&text=" + url.QueryEscape(text) +
		"&apikey=" + url.QueryEscape(apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("callmebot: statut inattendu %d", resp.StatusCode)
	}
	return nil
}
