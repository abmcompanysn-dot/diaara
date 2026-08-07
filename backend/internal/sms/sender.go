// Package sms fournit une abstraction d'envoi de SMS (OTP téléphone essentiellement).
// En dev, un stub logge sans rien envoyer ; le code est renvoyé dans la réponse
// API pour permettre de tester le flux. Un gateway réel (Termii/Twilio) se
// branche en implémentant Sender.
package sms

import (
	"context"
	"log"
)

// Sender envoie un SMS à un destinataire. Le message est libre (code OTP,
// notification...).
type Sender interface {
	Send(ctx context.Context, to, message string) error
}

// NoopSender ne fait rien (utilisé quand aucun gateway n'est configuré).
type NoopSender struct{}

func (NoopSender) Send(ctx context.Context, to, message string) error { return nil }

// LogSender (dev) logge simplement le SMS.
type LogSender struct{}

func (LogSender) Send(_ context.Context, to, message string) error {
	log.Printf("[SMS][dev] to=%s msg=%s", to, message)
	return nil
}
