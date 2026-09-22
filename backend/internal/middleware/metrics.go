package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics — instrumentation HTTP pour Prometheus/Grafana (voir k8s-monitoring/
// à la racine du dépôt et JOURNAL-MODIFICATIONS.md, 2026-09-22). Alimente le
// dashboard "DIARRA — API" : taux de requêtes (rate(...)), p95/p99
// (histogram_quantile sur RequestDuration) et taux d'erreur
// (rate(...{status=~"5.."}) / rate(...)).
//
// Cardinalité des labels volontairement limitée à method + route (le motif
// de route chi, ex "/api/products/{id}", jamais l'URL brute avec IDs) +
// status : un ID de vente/produit dans un label exploserait le nombre de
// séries stockées par Prometheus (cardinality explosion) sur un VPS déjà
// contraint en RAM.
var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "diarra_http_requests_total",
			Help: "Nombre total de requêtes HTTP traitées, par méthode/route/statut.",
		},
		[]string{"method", "route", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "diarra_http_request_duration_seconds",
			Help: "Durée des requêtes HTTP en secondes, par méthode/route.",
			// Bornes adaptées à une API web classique (de 1ms à 10s) — assez
			// fines autour de 50-500ms pour un p95/p99 lisible sans multiplier
			// le nombre de buckets stockés.
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "route"},
	)
)

// Metrics enregistre chaque requête dans requestsTotal/requestDuration. À
// poser tôt dans la chaîne (avant RequireAuth etc.) pour couvrir aussi les
// 401/403/429 — le taux d'erreur doit refléter tout ce que voit un client,
// pas seulement les requêtes qui atteignent un handler métier.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		// RouteContext donne le motif de route chi ("/api/products/{id}"), pas
		// l'URL avec un ID réel — posé par chi APRÈS le routage, donc lisible
		// seulement une fois next.ServeHTTP revenu.
		route := "unmatched" // 404 sur un chemin inconnu : évite une série par URL aléatoire
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			if pattern := rctx.RoutePattern(); pattern != "" {
				route = pattern
			}
		}

		status := strconv.Itoa(ww.Status())
		requestsTotal.WithLabelValues(r.Method, route, status).Inc()
		requestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}
