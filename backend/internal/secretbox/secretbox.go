// Package secretbox chiffre de petits secrets stockés en base (ex. jetons
// d'accès Meta des vendeurs) avec AES-256-GCM.
//
// Format chiffré : "v1:" + base64(nonce || texte chiffré || tag). Le préfixe
// de version permettra de changer d'algorithme ou de clé plus tard sans
// casser les valeurs déjà stockées.
//
// La clé (32 octets) vient de l'environnement, en base64 (standard ou URL,
// avec ou sans padding) ou en hexadécimal (64 caractères). Exemple pour en
// générer une : `openssl rand -base64 32`.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

const prefix = "v1:"

var (
	ErrInvalidKey    = errors.New("secretbox: la clé doit faire 32 octets (base64 ou hex)")
	ErrInvalidCipher = errors.New("secretbox: valeur chiffrée invalide ou clé incorrecte")
)

// ParseKey décode une clé de 32 octets fournie en hexadécimal ou en base64.
func ParseKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrInvalidKey
	}
	if len(s) == 64 {
		if k, err := hex.DecodeString(s); err == nil {
			return k, nil
		}
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if k, err := enc.DecodeString(s); err == nil && len(k) == 32 {
			return k, nil
		}
	}
	return nil, ErrInvalidKey
}

// Box chiffre/déchiffre avec une clé fixe. Sûr pour un usage concurrent.
type Box struct {
	aead cipher.AEAD
}

func New(key []byte) (*Box, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

// NewFromString = ParseKey + New.
func NewFromString(key string) (*Box, error) {
	k, err := ParseKey(key)
	if err != nil {
		return nil, err
	}
	return New(k)
}

// Encrypt chiffre plain avec un nonce aléatoire (deux chiffrements du même
// texte donnent deux valeurs différentes).
func (b *Box) Encrypt(plain string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(plain), nil)
	return prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt déchiffre une valeur produite par Encrypt. Toute altération (ou une
// autre clé) donne ErrInvalidCipher.
func (b *Box) Decrypt(value string) (string, error) {
	if !strings.HasPrefix(value, prefix) {
		return "", ErrInvalidCipher
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil {
		return "", ErrInvalidCipher
	}
	ns := b.aead.NonceSize()
	if len(raw) < ns+b.aead.Overhead() {
		return "", ErrInvalidCipher
	}
	plain, err := b.aead.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", ErrInvalidCipher
	}
	return string(plain), nil
}
