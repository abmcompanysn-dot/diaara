// Package otp gère la génération, la vérification et le rate-limiting des codes
// de validation à usage unique (vérification email/téléphone, step-up...).
package otp

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/diarra/backend/internal/auth"
)

// CodeRepo abstraction du stockage des codes (pour faciliter le test).
type CodeRepo interface {
	CreateCode(ctx context.Context, userID, channel, purpose, codeHash string, expiresAt time.Time) error
	LatestCode(ctx context.Context, userID, channel, purpose string) (*OTPCode, error)
	IncAttempts(ctx context.Context, id string) (int, error)
	MarkUsed(ctx context.Context, id string) error
	LastIssuedAt(ctx context.Context, userID, channel, purpose string) (time.Time, error)
}

// OTPCode porté dans le package otp pour découpler du modèle.
type OTPCode struct {
	ID       string
	CodeHash string
	Attempts int
}

var (
	ErrInvalidCode     = errors.New("invalid code")
	ErrExpired         = errors.New("code expired")
	ErrTooManyAttempts = errors.New("too many attempts")
	ErrResendTooSoon   = errors.New("resend too soon")
)

const (
	codeLength     = 6
	codeTTL        = 10 * time.Minute
	resendCooldown = 60 * time.Second
	maxAttempts    = 5
)

// Service de génération/vérification OTP. Stateless (pas de pool stocké) :
// les codes sont persistés via CodeRepo.
type Service struct {
	repo CodeRepo
}

func NewService(repo CodeRepo) *Service {
	return &Service{repo: repo}
}

// Issue génère un code à 6 chiffres, le persiste (hashé) pour (user, channel,
// purpose) et renvoie le code en clair (à envoyer par le canal approprié).
// Renvoie ErrResendTooSoon si un code a été émis il y a moins de resendCooldown.
func (s *Service) Issue(ctx context.Context, userID, channel, purpose string) (string, error) {
	last, err := s.repo.LastIssuedAt(ctx, userID, channel, purpose)
	if err != nil {
		return "", err
	}
	if !last.IsZero() && time.Since(last) < resendCooldown {
		return "", ErrResendTooSoon
	}

	code, err := generateNumeric(codeLength)
	if err != nil {
		return "", err
	}

	if err := s.repo.CreateCode(ctx, userID, channel, purpose, auth.HashToken(code), time.Now().Add(codeTTL)); err != nil {
		return "", err
	}
	return code, nil
}

// Verify valide un code pour (user, channel, purpose). Sur succès, le code est
// marqué utilisé. Sur échec (code faux), les tentatives sont incrémentées et
// ErrTooManyAttempts est renvoyé à partir de maxAttempts (le code reste
// invérifiable).
func (s *Service) Verify(ctx context.Context, userID, channel, purpose, code string) error {
	c, err := s.repo.LatestCode(ctx, userID, channel, purpose)
	if err != nil || c == nil {
		return ErrInvalidCode
	}
	if c.Attempts >= maxAttempts {
		return ErrTooManyAttempts
	}
	if auth.HashToken(code) != c.CodeHash {
		att, _ := s.repo.IncAttempts(ctx, c.ID)
		if att >= maxAttempts {
			return ErrTooManyAttempts
		}
		return ErrInvalidCode
	}
	return s.repo.MarkUsed(ctx, c.ID)
}

// generateNumeric renvoie une chaîne à n chiffres zéro-paddedée aléatoire
// (cryptographiquement sûre). 6 chiffres → 1 000 000 codes possibles.
func generateNumeric(n int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", n, num), nil
}

// CodeTTL expose la durée de validité (utile pour les messages frontend).
func CodeTTL() time.Duration { return codeTTL }
