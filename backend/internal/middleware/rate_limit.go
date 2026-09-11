package middleware

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
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
	return l.allowKey(ctx, key, l.burst)
}

// allowKey applique le même mécanisme que allow (fenêtre glissante de
// l.window, dérivée du rate/burst de CE limiter) mais sous une clé Redis et
// un plafond différents — sert à donner à une route spécifique (voir
// "/api/orders/status" dans Middleware) un quota bien plus large que le
// reste du trafic de la même IP, sans instancier un second *RateLimiter.
func (l *RateLimiter) allowKey(ctx context.Context, key string, burst float64) bool {
	count, err := l.cache.IncrWithExpire(ctx, "ratelimit:"+key, l.window)
	if err != nil {
		log.Printf("WARNING: rate limiter Redis indisponible, requête laissée passer: %v", err)
		return true
	}
	// count == 0 : cache no-op (REDIS_URL absent) — limitation désactivée.
	if count == 0 {
		return true
	}
	return count <= int64(burst)
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
		// Garde-fou : si l'IP résolue n'identifie pas un vrai visiteur mais un
		// maillon de l'infra (ingress-nginx, Caddy... tous en réseau privé),
		// c'est que les en-têtes de proxy ne remontent pas l'IP d'origine.
		// Rate-limiter sur cette IP reviendrait à compter TOUS les visiteurs
		// dans un seul compteur et à écrouler tout le monde d'un coup
		// (incident 2026-09-07 : 429 généralisés sur /api/auth/* après la
		// migration k3s). Dans ce cas on laisse passer — mieux vaut pas de
		// limitation qu'une limitation qui bloque tout le site.
		if isInfraIP(ip) {
			next.ServeHTTP(w, r)
			return
		}
		// /api/orders/status : suivi de paiement, interrogé toutes les 3s
		// depuis checkout/return pendant que l'acheteur attend la
		// confirmation. Partageait le même quota (40 req/4s par IP) que
		// TOUT le reste du trafic de cette IP — plusieurs onglets ouverts en
		// parallèle (ou une IP mutualisée, courante derrière un NAT
		// opérateur mobile) suffisaient à le vider et à bloquer l'acheteur
		// en plein paiement avec des 429 (incident 2026-09-04). Clé Redis
		// dédiée, quota bien plus large, pour ne plus jamais entrer en
		// compétition avec le reste du trafic de cette IP.
		if r.URL.Path == "/api/orders/status" {
			if !l.allowKey(r.Context(), "checkout:"+ip, 100) {
				w.Header().Set("X-RateLimit-Limit", "100")
				http.Error(w, `{"error":"too_many_requests"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if !l.allow(r.Context(), ip) {
			w.Header().Set("X-RateLimit-Limit", "40")
			http.Error(w, `{"error":"too_many_requests"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// trustedProxySecret — si posé (variable TRUSTED_PROXY_SECRET), les en-têtes
// CF-Connecting-IP / X-Real-IP / X-Forwarded-For ne sont honorés que si la
// requête porte aussi X-Diarra-Origin-Secret avec cette valeur. Sans ce
// garde-fou, N'IMPORTE QUI peut poser lui-même CF-Connecting-IP et se faire
// passer pour une IP différente à chaque requête — ce qui annule
// complètement le rate limiting (dont la limite stricte anti-brute-force sur
// /api/auth/login) : voir audit sécurité 2026-09-11. Le secret doit être posé
// par le proxy de confiance (Caddy) sur toute requête qu'il transmet, jamais
// accessible depuis l'extérieur — variable vide = comportement inchangé
// (rétrocompatible tant que l'infra n'a pas été mise à jour côté Caddy).
var trustedProxySecret = os.Getenv("TRUSTED_PROXY_SECRET")

// clientIP tente de retrouver l'IP réelle du visiteur à travers la chaîne de
// proxys de production : Cloudflare -> Caddy -> ingress-nginx -> backend.
//
// Ordre de préférence :
//  1. CF-Connecting-IP : posé par Cloudflare, contient l'IP du visiteur et
//     rien d'autre (pas une liste). C'est la source la plus fiable ici.
//  2. X-Real-IP : posé par ingress-nginx, une seule IP.
//  3. X-Forwarded-For : liste "client, proxy1, proxy2..." — on prend le
//     PREMIER élément (le client d'origine). Selon la config des proxys
//     intermédiaires cet élément peut être réécrit, d'où sa position après
//     CF-Connecting-IP.
//  4. r.RemoteAddr : dernier recours (= IP de l'ingress en prod k3s, donc
//     partagée entre tous les visiteurs — à éviter pour le rate limiting,
//     mais mieux que rien en dev/local sans proxy).
//
// Chaque candidat est nettoyé d'un éventuel ":port" et validé comme IP ; un
// candidat invalide est ignoré au profit du suivant. Voir trustedProxySecret
// ci-dessus : ces en-têtes ne sont des candidats valables QUE si ce garde-fou
// est satisfait (ou désactivé).
func clientIP(r *http.Request) string {
	if trustedProxySecret != "" && r.Header.Get("X-Diarra-Origin-Secret") != trustedProxySecret {
		if ip := parseIPMaybePort(r.RemoteAddr); ip != "" {
			return ip
		}
		return r.RemoteAddr
	}
	candidates := []string{
		r.Header.Get("CF-Connecting-IP"),
		r.Header.Get("X-Real-IP"),
		firstForwardedFor(r.Header.Get("X-Forwarded-For")),
	}
	for _, c := range candidates {
		if ip := parseIPMaybePort(c); ip != "" {
			return ip
		}
	}
	if ip := parseIPMaybePort(r.RemoteAddr); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// isInfraIP indique si l'IP correspond à un maillon d'infrastructure plutôt
// qu'à un visiteur : loopback, ou réseau privé (RFC 1918 / unique-local IPv6).
// En prod k3s, l'ingress-nginx et Caddy sont tous deux sur 10.42.x/127.0.0.1 ;
// une requête vue avec une telle IP signifie que l'IP réelle du visiteur n'a
// pas été propagée. En dev local (sans proxy), r.RemoteAddr vaut aussi
// 127.0.0.1 — la limitation y est donc de fait désactivée, ce qui est sans
// conséquence pour un poste de développement.
func isInfraIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return true // non résolue : on ne peut pas limiter dessus de façon fiable
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// firstForwardedFor retourne le premier élément d'un en-tête X-Forwarded-For
// ("client, proxy1, proxy2") — l'IP du client d'origine.
func firstForwardedFor(xff string) string {
	if xff == "" {
		return ""
	}
	if i := strings.IndexByte(xff, ','); i >= 0 {
		return strings.TrimSpace(xff[:i])
	}
	return strings.TrimSpace(xff)
}

// parseIPMaybePort accepte "1.2.3.4", "1.2.3.4:5678", "[::1]" ou "[::1]:5678"
// et renvoie l'IP seule si elle est valide, sinon "".
func parseIPMaybePort(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if net.ParseIP(s) != nil {
		return s
	}
	if host, _, err := net.SplitHostPort(s); err == nil && net.ParseIP(host) != nil {
		return host
	}
	return ""
}
