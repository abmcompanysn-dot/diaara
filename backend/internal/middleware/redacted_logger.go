package middleware

import (
	"log"
	"net/url"
	"time"

	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// RequestLogger — remplace chimw.Logger : même ligne de log (méthode, chemin,
// statut, taille, durée), mais le jeton d'accès JWT porté par
// /ws/*?token=... (voir RequireAuth : un navigateur ne peut pas poser
// d'en-tête Authorization sur un upgrade WebSocket) n'est jamais écrit en
// clair — sinon chaque connexion WebSocket loguait l'access token complet
// sur stdout, potentiellement retenu par un agrégateur de logs (audit
// sécurité 2026-09-11). La requête réellement transmise au handler suivant
// n'est jamais modifiée : seule la valeur utilisée POUR CETTE LIGNE DE LOG
// est expurgée.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		log.Printf("%s %s from %s - %d %dB in %s",
			r.Method, redactedRequestURI(r), r.RemoteAddr, ww.Status(), ww.BytesWritten(), time.Since(start))
	})
}

// redactedQueryParams — paramètres jamais écrits en clair dans les logs :
// "token" (JWT des WebSockets) ; "code" et "state" du retour Facebook Login
// (/api/meta/oauth/callback, voir handler/ad_meta_connect.go).
var redactedQueryParams = map[string]bool{"token": true, "code": true, "state": true}

// redactedRequestURI reproduit r.RequestURI mais remplace la valeur de tout
// paramètre sensible (redactedQueryParams) par "***" avant de la renvoyer —
// jamais appliqué à la requête elle-même, seulement à la chaîne loguée.
func redactedRequestURI(r *http.Request) string {
	q := r.URL.Query()
	sensitive := false
	for k := range q {
		if redactedQueryParams[k] {
			sensitive = true
			break
		}
	}
	if !sensitive {
		return r.RequestURI
	}
	redacted := make(url.Values, len(q))
	for k, v := range q {
		if redactedQueryParams[k] {
			redacted[k] = []string{"***"}
			continue
		}
		redacted[k] = v
	}
	path := r.URL.Path
	if raw := redacted.Encode(); raw != "" {
		path += "?" + raw
	}
	return path
}
