#!/bin/sh
# Résumé quotidien du trafic (diarra-vps, k3s), envoyé par email — même
# mécanisme que monitor.sh (Resend, pas de dépendance à un service de mail
# supplémentaire). Lancé par cron une fois par jour (voir
# install-traffic-report.sh) : utilise traffic.sh sur une fenêtre de 24h,
# tirée des logs encore retenus par le pod ingress-nginx-controller.
#
# Limite assumée : si le pod ingress-nginx a redémarré dans les dernières
# 24h (mise à jour, OOM...), la fenêtre réelle couverte est plus courte —
# le rapport reste correct sur ce qu'il a pu lire, juste incomplet ce
# jour-là. Pas de stockage long terme des logs bruts (coût jugé disproportionné
# pour la taille actuelle du trafic — à revoir si le volume augmente
# nettement, voir docs/PAIEMENTS.md pour le contexte du choix "pas de
# Prometheus/Grafana").

set -eu

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
ALERT_EMAIL="abmcompanysn@gmail.com"
RESEND_API_KEY=$(grep '^RESEND_API_KEY=' /opt/diarra/.env | cut -d= -f2-)
FROM="DIARRA Monitoring <noreply@abmcy.com>"

REPORT=$(sh "$SCRIPT_DIR/traffic.sh" 24h 2>&1)

# Échappement minimal pour tenir dans le JSON de l'API Resend (guillemets et
# retours à la ligne) — le rapport est du texte simple généré par nous, pas
# une entrée utilisateur, donc pas besoin d'échappement HTML.
BODY_JSON=$(printf '%s' "$REPORT" | sed 's/\\/\\\\/g; s/"/\\"/g' | awk '{printf "%s\\n", $0}')

if [ -z "$RESEND_API_KEY" ]; then
	echo "$(date): rapport trafic (email désactivé, RESEND_API_KEY absent):" >>/var/log/diarra-traffic.log
	echo "$REPORT" >>/var/log/diarra-traffic.log
	exit 0
fi

curl -s -X POST https://api.resend.com/emails \
	-H "Authorization: Bearer $RESEND_API_KEY" \
	-H "Content-Type: application/json" \
	-d "{\"from\":\"$FROM\",\"to\":\"$ALERT_EMAIL\",\"subject\":\"📊 DIARRA — trafic des dernières 24h\",\"html\":\"<pre style='font-family:monospace;white-space:pre-wrap'>${BODY_JSON}</pre>\"}" \
	>/dev/null 2>&1 || true

echo "$(date): rapport trafic envoyé" >>/var/log/diarra-traffic.log
