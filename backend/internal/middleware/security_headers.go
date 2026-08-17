package middleware

import "net/http"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'")
		// HSTS : n'a d'effet que sur une connexion déjà HTTPS (ignoré par les
		// navigateurs en clair) — sûr à envoyer inconditionnellement, y
		// compris en dev local. Force le HTTPS pour toute requête future,
		// y compris la toute première si le domaine est sur la HSTS preload
		// list du navigateur.
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}
