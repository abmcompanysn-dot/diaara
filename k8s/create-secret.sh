#!/bin/sh
# Crée/met à jour le Secret diarra-secrets à partir du .env déjà présent à
# la racine du dépôt sur le VPS (le même que docker-compose.yml lit).
# N'utilise jamais `. .env` (sourcing shell) : une valeur du .env peut
# contenir des caractères que sh interprète (ex: $, `, guillemets) — on lit
# donc chaque ligne en texte brut avec grep/cut, sans jamais l'évaluer.
# Ne jamais commiter les valeurs réelles — ce script les lit et les envoie
# directement à l'API Kubernetes, sans jamais les afficher.
set -eu

export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

if [ ! -f .env ]; then
  echo "ERREUR : .env introuvable à la racine du dépôt." >&2
  exit 1
fi

# Lecture brute d'une clé (première correspondance, sans évaluation shell).
env_value() {
  grep -E "^$1=" .env | head -1 | cut -d= -f2-
}

kubectl create namespace diarra --dry-run=client -o yaml | kubectl apply -f -

POSTGRES_PASSWORD=$(env_value POSTGRES_PASSWORD)
MINIO_ROOT_USER=$(env_value MINIO_ROOT_USER)
MINIO_ROOT_PASSWORD=$(env_value MINIO_ROOT_PASSWORD)
S3_ACCESS_KEY_ID=$(env_value S3_ACCESS_KEY_ID)
S3_SECRET_ACCESS_KEY=$(env_value S3_SECRET_ACCESS_KEY)

# SMTP_HOST/USER/PASS/FROM (Mailcow) sont volontairement absents ici : le
# pod backend ne peut pas atteindre Mailcow depuis le réseau k3s (isolation
# réseau imposée par kube-router, pas un problème de configuration —
# vérifié : le trafic sortant du pod est bloqué vers n'importe quelle
# adresse locale au VPS, y compris la passerelle Docker et l'IP publique).
# Le backend bascule sur Resend (ou Mailtrap) — SMTP_HOST reste dans le
# .env pour un éventuel usage local/Docker Compose, mais n'est plus porté
# dans le Secret k3s : sa présence ferait passer le switch email de
# main.go sur la branche SMTP avant même de regarder RESEND_API_KEY.
kubectl create secret generic diarra-secrets \
  --namespace diarra \
  --from-literal=JWT_SECRET="$(env_value JWT_SECRET)" \
  --from-literal=REFRESH_SECRET="$(env_value REFRESH_SECRET)" \
  --from-literal=DATABASE_URL="postgres://diarra:${POSTGRES_PASSWORD}@postgres:5432/diarra?sslmode=disable" \
  --from-literal=POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" \
  --from-literal=MINIO_ROOT_USER="${MINIO_ROOT_USER}" \
  --from-literal=MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD}" \
  --from-literal=S3_ACCESS_KEY_ID="${S3_ACCESS_KEY_ID:-$MINIO_ROOT_USER}" \
  --from-literal=S3_SECRET_ACCESS_KEY="${S3_SECRET_ACCESS_KEY:-$MINIO_ROOT_PASSWORD}" \
  --from-literal=RESEND_API_KEY="$(env_value RESEND_API_KEY)" \
  --from-literal=RESEND_FROM="$(env_value RESEND_FROM)" \
  --from-literal=MAILTRAP_API_KEY="$(env_value MAILTRAP_API_KEY)" \
  --from-literal=MAILTRAP_FROM="$(env_value MAILTRAP_FROM)" \
  --from-literal=MAILTRAP_SANDBOX_ID="$(env_value MAILTRAP_SANDBOX_ID)" \
  --from-literal=PAWAPAY_API_KEY="$(env_value PAWAPAY_API_KEY)" \
  --from-literal=PAWAPAY_CALLBACK_URL="$(env_value PAWAPAY_CALLBACK_URL)" \
  --from-literal=PAWAPAY_CALLBACK_IPS="$(env_value PAWAPAY_CALLBACK_IPS)" \
  --from-literal=KPAY_API_KEY="$(env_value KPAY_API_KEY)" \
  --from-literal=KPAY_SECRET_KEY="$(env_value KPAY_SECRET_KEY)" \
  --from-literal=KPAY_WEBHOOK_SECRET="$(env_value KPAY_WEBHOOK_SECRET)" \
  --from-literal=KPAY_CALLBACK_URL="$(env_value KPAY_CALLBACK_URL)" \
  --from-literal=KPAY_PAYOUT_CALLBACK_URL="$(env_value KPAY_PAYOUT_CALLBACK_URL)" \
  --from-literal=KPAY_REFUND_CALLBACK_URL="$(env_value KPAY_REFUND_CALLBACK_URL)" \
  --from-literal=FIREBASE_PROJECT_ID="$(env_value FIREBASE_PROJECT_ID)" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "Secret diarra-secrets créé/mis à jour."
