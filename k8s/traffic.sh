#!/bin/sh
# Suivi de trafic (diarra-vps, k3s) : agrège les logs d'accès du pod
# ingress-nginx-controller — point de passage unique de toute l'API/du
# frontend proxifié (voir k8s/Caddyfile.new : Caddy relaie tout /api/*,
# /ws/*, /p/*, /r/*, /feed/* vers 127.0.0.1:30080). Pas de Prometheus/Grafana
# (trop lourd pour ce VPS à 4 CPU/8 Go partagé, voir monitor.sh) : ce script
# lit directement les logs du pod avec kubectl logs.
#
# Usage :
#   sh traffic.sh           -> résumé sur la dernière heure
#   sh traffic.sh 24h       -> résumé sur les dernières 24h (limite kubectl)
#
# Limite connue : kubectl logs ne lit que les logs encore retenus par le
# pod (perdus s'il redémarre) — pour un historique fiable au-delà de la
# rétention du pod, voir le cron quotidien (install-traffic-report.sh) qui
# vide le log courant dans /var/log/diarra-traffic-YYYYMMDD.log chaque jour.

set -eu

WINDOW="${1:-1h}"
NS=ingress-nginx
DEPLOY=deploy/ingress-nginx-controller

RAW=$(kubectl -n "$NS" logs --since="$WINDOW" "$DEPLOY" 2>/dev/null || true)

if [ -z "$RAW" ]; then
	echo "Aucune requête trouvée sur la fenêtre $WINDOW (ou pod ingress-nginx injoignable)."
	exit 0
fi

# Ne garder que les vraies lignes d'accès (démarrent par une IPv4) : le
# stream du pod peut aussi contenir des lignes de démarrage/diagnostic
# nginx qui polluent les comptages sinon (ex. top IP, total de requêtes).
LOG=$(echo "$RAW" | grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+ ' || true)

if [ -z "$LOG" ]; then
	echo "Aucune requête d'accès trouvée sur la fenêtre $WINDOW (logs présents mais aucune ligne reconnue)."
	exit 0
fi

TOTAL=$(echo "$LOG" | wc -l)
MINUTES=$(case "$WINDOW" in
	*h) echo $(( ${WINDOW%h} * 60 )) ;;
	*m) echo "${WINDOW%m}" ;;
	*d) echo $(( ${WINDOW%d} * 1440 )) ;;
	*) echo 60 ;;
esac)
RATE=$(awk -v t="$TOTAL" -v m="$MINUTES" 'BEGIN{printf "%.1f", t/m}')

echo "=== Trafic diarra — fenêtre: $WINDOW ==="
echo "Requêtes totales : $TOTAL  (~$RATE req/min)"
echo

# Le statut HTTP suit immédiatement la ligne de requête ("... HTTP/1.x" 200 ...)
# — ancré sur HTTP/1.\d" précisément, car les champs referer/UA plus loin
# dans la ligne contiennent aussi des séquences "<espace>NNN<espace>" qui
# matcheraient un motif trop lâche comme '" [0-9]{3} '.
STATUSES=$(echo "$LOG" | grep -oE 'HTTP/1\.[01]" [0-9]{3} ' | grep -oE '[0-9]{3}')

echo "--- Codes HTTP ---"
echo "$STATUSES" | sort | uniq -c | sort -rn | \
	awk '{printf "%6d  %s\n", $1, $2}'
echo

ERR5XX=$(echo "$STATUSES" | grep -c '^5' || true)
ERR4XX=$(echo "$STATUSES" | grep -c '^4' || true)
if [ "$ERR5XX" -gt 0 ]; then
	echo "⚠️  $ERR5XX erreur(s) serveur (5xx) sur la fenêtre"
fi
echo "$ERR4XX erreur(s) client (4xx) sur la fenêtre"
echo

echo "--- Top 15 endpoints (méthode + chemin, sans query string) ---"
echo "$LOG" | grep -oE '"[A-Z]+ [^"?]+' | sed 's/^"//' | sort | uniq -c | sort -rn | head -15 | \
	awk '{count=$1; $1=""; printf "%6d  %s\n", count, $0}'
echo

# Note : $remote_addr vu par ingress-nginx est l'IP de Caddy sur ce même
# nœud (10.42.0.x, réseau interne k3s) — Caddy ne relaie pas encore
# X-Forwarded-For/X-Real-IP vers l'ingress (voir k8s/Caddyfile.new), donc
# cette section ne distingue pas les vrais visiteurs entre eux pour
# l'instant. Gardée pour détecter une IP interne qui se comporterait mal
# (poids anormal), pas pour du géo/anti-abus par IP visiteur.
echo "--- Top 10 IP sources (vues par ingress-nginx — voir note ci-dessus) ---"
echo "$LOG" | awk '{print $1}' | sort | uniq -c | sort -rn | head -10 | \
	awk '{printf "%6d  %s\n", $1, $2}'
echo

# request_time est l'avant-dernier champ numérique décimal juste avant
# "[upstream-name]" — on l'extrait par motif plutôt que par position de
# colonne (le nombre de champs varie légèrement selon les versions).
TIMES=$(echo "$LOG" | grep -oE ' [0-9]+\.[0-9]+ \[' | grep -oE '[0-9]+\.[0-9]+' || true)
if [ -n "$TIMES" ]; then
	echo "--- Latence upstream (secondes) ---"
	echo "$TIMES" | sort -n | awk '
		{ a[NR]=$1 }
		END {
			if (NR==0) { print "aucune donnée"; exit }
			p50=a[int(NR*0.50)+1]; p95=a[int(NR*0.95)+1]; p99=a[int(NR*0.99)+1]
			max=a[NR]
			printf "p50: %ss   p95: %ss   p99: %ss   max: %ss\n", p50, p95, p99, max
		}'
	echo
fi

echo "--- Répartition par heure (fenêtre $WINDOW) ---"
echo "$LOG" | grep -oE '\[[0-9]{2}/[A-Za-z]{3}/[0-9]{4}:[0-9]{2}' | sed 's/^\[//' | \
	awk -F: '{print $2}' | sort | uniq -c | \
	awk '{printf "%2sh  %s %d\n", $2, "▏", $1}' | sort -k1
