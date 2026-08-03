package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateToken génère un token opaque de 32 octets (256 bits) utilisé pour
// la vérification d'email et la réinitialisation de mot de passe.
func GenerateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// HashToken calcule l'empreinte SHA-256 d'un token.
// Seule l'empreinte est stockée en base : un token volé dans la base ne peut
// pas être réutilisé.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
