#!/bin/sh
# Supervision légère du cluster k3s (diarra-vps) : détecte exactement le
# genre de situation qui est passée inaperçue pendant des semaines sur
# l'ancien VPS (134% CPU en continu, pods qui redémarrent en boucle) et
# alerte par email — pas de stack Prometheus/Grafana, trop lourd pour ce
# VPS à 4 CPU/8 Go partagé avec toute la stack applicative. Lancé par cron
# (voir install-monitor.sh) toutes les 5 minutes.
#
# État persistant dans /root/.diarra-monitor-state pour ne pas ré-alerter à
# chaque passage tant que le problème n'est pas résolu (une seule alerte au
# déclenchement, une seule au retour à la normale).

set -eu

STATE_FILE=/root/.diarra-monitor-state
ALERT_EMAIL="abmcompanysn@gmail.com"
RESEND_API_KEY=$(grep '^RESEND_API_KEY=' /opt/diarra/.env | cut -d= -f2-)
FROM="DIARRA Monitoring <noreply@abmcy.com>"

send_alert() {
	subject="$1"
	body="$2"
	if [ -z "$RESEND_API_KEY" ]; then
		echo "$(date): ALERTE (email désactivé, RESEND_API_KEY absent): $subject — $body" >>/var/log/diarra-monitor.log
		return
	fi
	curl -s -X POST https://api.resend.com/emails \
		-H "Authorization: Bearer $RESEND_API_KEY" \
		-H "Content-Type: application/json" \
		-d "{\"from\":\"$FROM\",\"to\":\"$ALERT_EMAIL\",\"subject\":\"$subject\",\"html\":\"<pre>$body</pre>\"}" \
		>/dev/null 2>&1 || true
	echo "$(date): ALERTE envoyée: $subject" >>/var/log/diarra-monitor.log
}

was_alerting=0
[ -f "$STATE_FILE" ] && was_alerting=1

problems=""

# 1) Charge CPU soutenue anormale (load average 5 min > 2x le nombre de CPU)
# — c'est exactement le signal qui aurait détecté l'incident k3s de
# l'ancien VPS (load average 130 sur 4 coeurs) dès les premières minutes.
NPROC=$(nproc)
LOAD5=$(awk '{print $2}' /proc/loadavg)
THRESHOLD=$(awk -v n="$NPROC" 'BEGIN{print n*2}')
OVER=$(awk -v l="$LOAD5" -v t="$THRESHOLD" 'BEGIN{print (l>t)?1:0}')
if [ "$OVER" = "1" ]; then
	problems="${problems}Charge CPU anormale : load average 5min = $LOAD5 (seuil : $THRESHOLD sur $NPROC coeurs)\n"
fi

# 2) Pods en CrashLoopBackOff ou avec beaucoup de redémarrages récents
CRASHING=$(kubectl get pods -A --no-headers 2>/dev/null | awk '$4=="CrashLoopBackOff"{print $1"/"$2}')
if [ -n "$CRASHING" ]; then
	problems="${problems}Pods en CrashLoopBackOff :\n$CRASHING\n"
fi

# 3) Nodes non-Ready (le node lui-même en détresse)
NOTREADY=$(kubectl get nodes --no-headers 2>/dev/null | awk '$2!="Ready"{print $1}')
if [ -n "$NOTREADY" ]; then
	problems="${problems}Node(s) non-Ready :\n$NOTREADY\n"
fi

# 4) Espace disque (le disque plein a autant de conséquences qu'un CPU
# saturé — pods qui ne redémarrent plus, etcd/sqlite qui refuse d'écrire)
DISK_PCT=$(df / --output=pcent | tail -1 | tr -dc '0-9')
if [ "$DISK_PCT" -gt 85 ]; then
	problems="${problems}Disque : ${DISK_PCT}% utilisé (seuil 85%)\n"
fi

if [ -n "$problems" ]; then
	if [ "$was_alerting" = "0" ]; then
		send_alert "🔴 DIARRA — problème détecté sur diarra-vps" "$problems"
	fi
	echo "$problems" >"$STATE_FILE"
else
	if [ "$was_alerting" = "1" ]; then
		send_alert "✅ DIARRA — retour à la normale sur diarra-vps" "Tous les indicateurs surveillés sont revenus dans les seuils normaux."
		rm -f "$STATE_FILE"
	fi
fi
