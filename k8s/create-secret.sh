#!/bin/sh
# Crée/met à jour le Secret diarra-secrets à partir du .env déjà présent à
# la racine du dépôt sur le VPS (le même que docker-compose.yml lit).
# Ne jamais commiter les valeurs réelles — ce script les lit et les envoie
# directement à l'API Kubernetes, sans jamais les afficher.
set -eu

export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

if [ ! -f .env ]; then
  echo "ERREUR : .env introuvable à la racine du dépôt." >&2
  exit 1
fi

set -a
. ./.env
set +a

kubectl create namespace diarra --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic diarra-secrets \
  --namespace diarra \
  --from-literal=JWT_SECRET="${JWT_SECRET}" \
  --from-literal=REFRESH_SECRET="${REFRESH_SECRET}" \
  --from-literal=DATABASE_URL="postgres://diarra:${POSTGRES_PASSWORD}@postgres:5432/diarra?sslmode=disable" \
  --from-literal=POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" \
  --from-literal=MINIO_ROOT_USER="${MINIO_ROOT_USER}" \
  --from-literal=MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD}" \
  --from-literal=S3_ACCESS_KEY_ID="${S3_ACCESS_KEY_ID:-${MINIO_ROOT_USER}}" \
  --from-literal=S3_SECRET_ACCESS_KEY="${S3_SECRET_ACCESS_KEY:-${MINIO_ROOT_PASSWORD}}" \
  --from-literal=RESEND_API_KEY="${RESEND_API_KEY:-}" \
  --from-literal=RESEND_FROM="${RESEND_FROM:-}" \
  --from-literal=SMTP_HOST="${SMTP_HOST:-}" \
  --from-literal=SMTP_USER="${SMTP_USER:-}" \
  --from-literal=SMTP_PASS="${SMTP_PASS:-}" \
  --from-literal=SMTP_FROM="${SMTP_FROM:-}" \
  --from-literal=MAILTRAP_API_KEY="${MAILTRAP_API_KEY:-}" \
  --from-literal=MAILTRAP_FROM="${MAILTRAP_FROM:-}" \
  --from-literal=MAILTRAP_SANDBOX_ID="${MAILTRAP_SANDBOX_ID:-}" \
  --from-literal=PAWAPAY_API_KEY="${PAWAPAY_API_KEY:-}" \
  --from-literal=PAWAPAY_CALLBACK_URL="${PAWAPAY_CALLBACK_URL:-}" \
  --from-literal=PAWAPAY_CALLBACK_IPS="${PAWAPAY_CALLBACK_IPS:-}" \
  --from-literal=FIREBASE_PROJECT_ID="${FIREBASE_PROJECT_ID:-}" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "Secret diarra-secrets créé/mis à jour."
