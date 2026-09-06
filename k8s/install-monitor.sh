#!/bin/sh
# Installe monitor.sh en tâche cron toutes les 5 minutes. À lancer une fois
# sur le VPS après avoir copié monitor.sh dans /opt/diarra/k8s/.
set -eu

chmod +x /opt/diarra/k8s/monitor.sh
touch /var/log/diarra-monitor.log

CRON_LINE="*/5 * * * * /opt/diarra/k8s/monitor.sh"
# `grep -v` renvoie un code non-nul quand rien ne correspond (cas normal au
# premier install, crontab vide) — sous `set -e`, ça arrêterait le sous-shell
# avant le `echo` suivant et viderait le crontab. `|| true` neutralise ça.
(crontab -l 2>/dev/null | grep -v 'diarra/k8s/monitor.sh' || true; echo "$CRON_LINE") | crontab -

echo "Installé. Vérifier avec : crontab -l"
echo "Logs : tail -f /var/log/diarra-monitor.log"
