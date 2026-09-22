package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"time"
)

const YesClientIDKey contextKey = "yes_client_id"

// YesClientLookup — même principe que GatewayClientLookup (gateway.go), pour
// l'intégration YES Messaging (migrations/037_yes_integration.sql). Séparé du
// système gateway_clients existant : protocole différent (session
// conversationnelle avec ses propres statuts, pas une passerelle de paiement
// générique), voir le commentaire de tête de la migration 037.
type YesClientLookup struct {
	ID             string
	APISecretHash  string
}

// RequireYesClient authentifie YES Messaging en deux temps, comme
// RequireGatewayClient : X-API-Key identifie le client (clé en clair,
// résolue directement — pas de hash sur celle-ci, contrairement à
// X-Gateway-Key, car ici le vrai secret est ailleurs : X-Signature-SHA256),
// puis la signature HMAC prouve que CE client a signé CE corps précis.
func RequireYesClient(lookupClient func(ctx context.Context, apiKey string) (YesClientLookup, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				http.Error(w, `{"error":"missing_api_key"}`, http.StatusUnauthorized)
				return
			}
			client, err := lookupClient(r.Context(), apiKey)
			if err != nil || client.ID == "" {
				http.Error(w, `{"error":"invalid_api_key"}`, http.StatusUnauthorized)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, `{"error":"read_failed"}`, http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body)) // le handler doit encore pouvoir le lire

			if !verifyYesSignature(client.APISecretHash, r, body) {
				http.Error(w, `{"error":"invalid_signature"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), YesClientIDKey, client.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// yesSignatureMaxAge — même fenêtre anti-rejeu que gatewaySignatureMaxAge
// (5 min), même raisonnement (tolère une horloge client désynchronisée sans
// affaiblir la protection contre le rejeu d'une requête interceptée).
const yesSignatureMaxAge = 5 * time.Minute

// verifyYesSignature vérifie X-Signature-SHA256 = HMAC-SHA256(secret,
// timestamp + "." + body), comparaison en temps constant — même construction
// que verifyGatewaySignature (gateway.go), voir ce fichier pour le
// raisonnement complet sur le choix de signer le hash du secret plutôt que
// le secret brut (jamais stocké en clair côté DIARRA).
func verifyYesSignature(secretHash string, r *http.Request, body []byte) bool {
	if secretHash == "" {
		return false
	}
	sigHex := r.Header.Get("X-Signature-SHA256")
	tsHeader := r.Header.Get("X-Timestamp")
	if sigHex == "" || tsHeader == "" {
		return false
	}
	ts, err := strconv.ParseInt(tsHeader, 10, 64)
	if err != nil {
		return false
	}
	age := time.Since(time.Unix(ts, 0))
	if age < 0 {
		age = -age
	}
	if age > yesSignatureMaxAge {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secretHash))
	mac.Write([]byte(tsHeader + "." + string(body)))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sigHex))
}

func GetYesClientID(ctx context.Context) string {
	if v, ok := ctx.Value(YesClientIDKey).(string); ok {
		return v
	}
	return ""
}
