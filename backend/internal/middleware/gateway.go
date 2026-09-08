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

const GatewayClientIDKey contextKey = "gateway_client_id"

// GatewayClientLookup — ce que RequireGatewayClient a besoin de savoir sur un
// client résolu par sa clé API : son ID (isolation des transactions, voir
// GatewayRepo.FindByClientRef) et le hash de son secret HMAC (vérification de
// signature ci-dessous). HMACSecretHash vide = client sans secret configuré
// (créé avant la migration 030, ou pas encore tourné) : la requête est
// refusée plutôt que de laisser passer une signature vide comme valide.
type GatewayClientLookup struct {
	ID             string
	HMACSecretHash string
}

// RequireGatewayClient authentifie une application externe (ex. ABMCY Core)
// en DEUX temps, tous deux nécessaires — pas juste la clé API :
//  1. X-Gateway-Key : identifie QUI appelle (hashée en SHA-256, résolue vers
//     un client_id via lookup — même algorithme que auth.HashToken, pas de
//     dépendance croisée pour éviter un import cycle).
//  2. X-Diarra-Signature : prouve que CE client précis a signé CE corps
//     précis avec son secret HMAC dédié (jamais le même secret que la clé
//     API — une fuite de l'un ne compromet pas l'autre), accompagné de
//     X-Diarra-Timestamp pour empêcher un rejeu (une requête interceptée
//     valide ne peut pas être renvoyée telle quelle indéfiniment).
//
// Même principe défensif que le relais de callback DIARRA -> client
// (gateway_relay.go, X-Diarra-Gateway-Signature) mais dans l'autre sens.
func RequireGatewayClient(lookupClient func(ctx context.Context, apiKeyHash string) (GatewayClientLookup, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Gateway-Key")
			if key == "" {
				http.Error(w, `{"error":"missing_gateway_key"}`, http.StatusUnauthorized)
				return
			}
			sum := sha256.Sum256([]byte(key))
			keyHash := hex.EncodeToString(sum[:])
			client, err := lookupClient(r.Context(), keyHash)
			if err != nil || client.ID == "" {
				http.Error(w, `{"error":"invalid_gateway_key"}`, http.StatusUnauthorized)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, `{"error":"read_failed"}`, http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body)) // le handler doit encore pouvoir le lire

			if !verifyGatewaySignature(client.HMACSecretHash, r, body) {
				http.Error(w, `{"error":"invalid_signature"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), GatewayClientIDKey, client.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// gatewaySignatureMaxAge — au-delà de cette fenêtre, une requête signée est
// refusée même si la signature est mathématiquement valide (protection
// anti-rejeu). Généreux (5 min) pour tolérer une horloge client légèrement
// désynchronisée sans obliger ABMCY Core (ou un futur client) à du NTP strict.
const gatewaySignatureMaxAge = 5 * time.Minute

// verifyGatewaySignature vérifie X-Diarra-Signature = HMAC-SHA256(secret,
// timestamp + "." + body), comparaison en temps constant (même principe que
// payment.VerifyKPaySignature). Le timestamp entre dans la signature ET dans
// la fenêtre de validité : un attaquant qui rejoue une requête interceptée ne
// peut ni la faire accepter après expiration, ni changer le timestamp sans
// invalider la signature.
func verifyGatewaySignature(hmacSecretHash string, r *http.Request, body []byte) bool {
	if hmacSecretHash == "" {
		return false
	}
	sigHex := r.Header.Get("X-Diarra-Signature")
	tsHeader := r.Header.Get("X-Diarra-Timestamp")
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
	if age > gatewaySignatureMaxAge {
		return false
	}

	// Le secret HMAC lui-même n'est jamais en clair côté DIARRA (voir
	// model.GatewayClient.HMACSecretHash) : on vérifie donc la signature
	// contre le HASH du secret comme clé HMAC, pas contre le secret brut.
	// Le client doit signer avec ce même hash (voir docs ABMCY Core /
	// scripts/gen-gateway-signature) — équivalent en sécurité (le hash
	// SHA-256 d'un secret aléatoire de 256 bits reste un secret de 256
	// bits), et évite à DIARRA de jamais stocker/manipuler le secret brut.
	mac := hmac.New(sha256.New, []byte(hmacSecretHash))
	mac.Write([]byte(tsHeader + "." + string(body)))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sigHex))
}

func GetGatewayClientID(ctx context.Context) string {
	if v, ok := ctx.Value(GatewayClientIDKey).(string); ok {
		return v
	}
	return ""
}
