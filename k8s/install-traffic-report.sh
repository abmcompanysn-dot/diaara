#!/bin/sh
# Installe traffic-report.sh en tâche cron quotidienne (8h du matin, heure
# du VPS/UTC). À lancer une fois sur le VPS après avoir copié traffic.sh et
# traffic-report.sh dans /opt/diarra/k8s/.
set -eu

chmod +x /opt/diarra/k8s/traffic.sh /opt/diarra/k8s/traffic-report.sh
touch /var/log/diarra-traffic.log

CRON_LINE="0 8 * * * /opt/diarra/k8s/traffic-report.sh"
# `grep -v` renvoie un code non-nul quand rien ne correspond (cas normal au
# premier install) — sous `set -e`, ça arrêterait le sous-shell avant le
# `echo` suivant et viderait le crontab. `|| true` neutralise ça.
(crontab -l 2>/dev/null | grep -v 'diarra/k8s/traffic-report.sh' || true; echo "$CRON_LINE") | crontab -

echo "Installé. Vérifier avec : crontab -l"
echo "Logs : tail -f /var/log/diarra-traffic.log"
echo "Test immédiat : sh /opt/diarra/k8s/traffic-report.sh"
echo "Suivi à la demande : sh /opt/diarra/k8s/traffic.sh [1h|24h|...]"
