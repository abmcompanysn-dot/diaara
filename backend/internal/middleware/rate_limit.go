package middleware

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/diarra/backend/internal/cache"
)

// RateLimiter — compteur à fenêtre fixe dans Redis (INCR + EXPIRE au premier
// coup), plutôt qu'un token-bucket en mémoire du process : indispensable
// pour rester valable si le backend tourne un jour sur plusieurs instances
// (voir DIARRA_CLAUDE.md §5.11, "jamais en mémoire locale du process").
//
// Sans REDIS_URL configuré (cacheClient no-op, voir internal/cache), la
// limitation est simplement désactivée plutôt que de retomber sur un
// comportement local incohérent entre instances — comme le reste des
// intégrations optionnelles du projet (S3, email...).
//
// Fail-open explicite : si Redis répond une erreur (indisponible, timeout),
// la requête passe quand même — la disponibilité du site prime sur la
// stricte limitation en cas de panne du cache.
type RateLimiter struct {
	cache *cache.Client
	rate  float64
	burst float64
	// window — durée sur laquelle "burst" requêtes sont autorisées, dérivée
	// de rate/burst pour approximer le même débit soutenu qu'un token-bucket
	// continu (ex. 0.2 req/s, burst 8 -> fenêtre de 40s).
	window time.Duration
}

func NewRateLimiter(cacheClient *cache.Client, rate, burst float64) *RateLimiter {
	return &RateLimiter{
		cache:  cacheClient,
		rate:   rate,
		burst:  burst,
		window: time.Duration(burst / rate * float64(time.Second)),
	}
}

func (l *RateLimiter) allow(ctx context.Context, key string) bool {
	count, err := l.cache.IncrWithExpire(ctx, "ratelimit:"+key, l.window)
	if err != nil {
		log.Printf("WARNING: rate limiter Redis indisponible, requête laissée passer: %v", err)
		return true
	}
	// count == 0 : cache no-op (REDIS_URL absent) — limitation désactivée.
	if count == 0 {
		return true
	}
	return count <= int64(l.burst)
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// kubelet sonde /health en continu (liveness + readiness, sur les 2
		// pods) — un 429 ici fait échouer la liveness probe et tue le pod,
		// provoquant un CrashLoopBackOff (observé en prod le 19/08 : les deux
		// pods backend tombaient en boucle car /health finissait par
		// dépasser le quota partagé avec le reste du trafic). La santé du
		// pod ne doit jamais dépendre de la charge du reste du trafic.
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		ip := clientIP(r)
		if !l.allow(r.Context(), ip) {
			w.Header().Set("X-RateLimit-Limit", "40")
			http.Error(w, `{"error":"too_many_requests"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		host, _, err := net.SplitHostPort(xff)
		if err == nil {
			return host
		}
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
