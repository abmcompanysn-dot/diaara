package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// firebaseCertsURL sert les certificats X.509 utilisés pour signer les ID
// tokens Firebase Auth — endpoint officiel documenté par Google pour une
// vérification manuelle (sans le SDK Admin complet, plus léger et suffisant
// ici puisque le backend a déjà sa propre gestion JWT/sessions).
const firebaseCertsURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

// FirebaseVerifier vérifie les ID tokens émis par Firebase Auth (ex: après
// une connexion Google côté frontend) et en extrait l'identité (email, UID).
// Les certificats sont mis en cache en mémoire et rafraîchis périodiquement
// (ils tournent côté Google environ une fois par jour).
type FirebaseVerifier struct {
	projectID string

	mu        sync.RWMutex
	certs     map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func NewFirebaseVerifier(projectID string) *FirebaseVerifier {
	return &FirebaseVerifier{projectID: projectID}
}

type FirebaseClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	// PhoneNumber — présent (format E.164, ex "+221771234567") quand l'ID
	// token vient d'une connexion par téléphone (signInWithPhoneNumber côté
	// frontend), absent pour les autres méthodes (Google, etc.).
	PhoneNumber string `json:"phone_number"`
	jwt.RegisteredClaims
}

var (
	ErrFirebaseNotConfigured = errors.New("firebase auth non configuré")
	ErrInvalidFirebaseToken  = errors.New("jeton firebase invalide")
)

// VerifyIDToken valide la signature RS256, l'audience (project ID), l'émetteur
// et l'expiration d'un ID token Firebase, puis retourne les claims (email, UID).
func (v *FirebaseVerifier) VerifyIDToken(idToken string) (*FirebaseClaims, error) {
	if v == nil || v.projectID == "" {
		return nil, ErrFirebaseNotConfigured
	}

	claims := &FirebaseClaims{}
	token, err := jwt.ParseWithClaims(idToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("kid manquant")
		}
		return v.publicKey(kid)
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidFirebaseToken
	}

	wantIssuer := "https://securetoken.google.com/" + v.projectID
	if claims.Issuer != wantIssuer {
		return nil, ErrInvalidFirebaseToken
	}
	aud, err := claims.GetAudience()
	if err != nil || len(aud) == 0 || aud[0] != v.projectID {
		return nil, ErrInvalidFirebaseToken
	}
	if claims.Subject == "" || claims.Email == "" {
		return nil, ErrInvalidFirebaseToken
	}

	return claims, nil
}

// publicKey retrouve la clé correspondant au kid, en rafraîchissant le cache
// si nécessaire (kid inconnu, ou cache absent/périmé).
func (v *FirebaseVerifier) publicKey(kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.certs[kid]
	stale := time.Since(v.fetchedAt) > time.Hour
	v.mu.RUnlock()
	if ok && !stale {
		return key, nil
	}

	if err := v.refreshCerts(); err != nil {
		if ok {
			// Rafraîchissement échoué mais on a déjà une clé en cache : on
			// continue avec l'ancienne plutôt que de bloquer toutes les connexions.
			return key, nil
		}
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.certs[kid]
	if !ok {
		return nil, errors.New("clé de signature inconnue")
	}
	return key, nil
}

func (v *FirebaseVerifier) refreshCerts() error {
	resp, err := http.Get(firebaseCertsURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("firebase certs: statut %d", resp.StatusCode)
	}

	var raw map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return err
	}

	certs := make(map[string]*rsa.PublicKey, len(raw))
	for kid, pemCert := range raw {
		block, _ := pem.Decode([]byte(pemCert))
		if block == nil {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			continue
		}
		certs[kid] = pubKey
	}
	if len(certs) == 0 {
		return errors.New("aucun certificat firebase valide")
	}

	v.mu.Lock()
	v.certs = certs
	v.fetchedAt = time.Now()
	v.mu.Unlock()
	return nil
}
