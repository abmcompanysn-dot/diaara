#!/bin/sh
# Copie les données de la stack Docker Compose existante vers la nouvelle
# stack k3s. Lecture seule côté source : la stack Docker Compose n'est
# jamais modifiée ni arrêtée par ce script.
set -eu

export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

echo "== Postgres : dump depuis Docker Compose, restore dans k3s =="
docker exec diarra-postgres-1 pg_dump -U diarra diarra \
  | kubectl exec -i -n diarra postgres-0 -- psql -U diarra diarra

echo "== MinIO : mc mirror (ancien -> nouveau, via port-forward temporaire) =="
kubectl -n diarra port-forward svc/minio 19000:9000 >/tmp/pf-minio.log 2>&1 &
PF_PID=$!
sleep 2
trap 'kill $PF_PID 2>/dev/null || true' EXIT

MINIO_ROOT_USER=$(kubectl -n diarra get secret diarra-secrets -o jsonpath='{.data.MINIO_ROOT_USER}' | base64 -d)
MINIO_ROOT_PASSWORD=$(kubectl -n diarra get secret diarra-secrets -o jsonpath='{.data.MINIO_ROOT_PASSWORD}' | base64 -d)

docker run --rm --network host \
  -e MC_HOST_old="http://${MINIO_ROOT_USER}:${MINIO_ROOT_PASSWORD}@127.0.0.1:9000" \
  -e MC_HOST_new="http://${MINIO_ROOT_USER}:${MINIO_ROOT_PASSWORD}@127.0.0.1:19000" \
  minio/mc mb --ignore-existing new/diarra-files

docker run --rm --network host \
  -e MC_HOST_old="http://${MINIO_ROOT_USER}:${MINIO_ROOT_PASSWORD}@127.0.0.1:9000" \
  -e MC_HOST_new="http://${MINIO_ROOT_USER}:${MINIO_ROOT_PASSWORD}@127.0.0.1:19000" \
  minio/mc mirror old/diarra-files new/diarra-files

kill $PF_PID 2>/dev/null || true
trap - EXIT

echo "== Vérification du schéma (migrations Go, idempotent) =="
kubectl -n diarra exec deploy/backend -- ./migrate || true

echo "== Terminé =="
