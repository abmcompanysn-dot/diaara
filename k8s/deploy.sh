#!/bin/sh
# Build les images Diarra avec Docker (comme docker-compose.yml) puis les
# importe dans le containerd de k3s — pas de registre, cluster à un seul
# noeud. À lancer depuis la racine du dépôt, sur le VPS.
set -eu

export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

echo "== Build backend =="
docker build -t diarra-backend:latest ./backend

echo "== Build frontend =="
docker build -t diarra-frontend:latest ./frontend \
  --build-arg NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-https://diarra.abmcy.com}" \
  --build-arg NEXT_PUBLIC_WS_URL="${NEXT_PUBLIC_WS_URL:-wss://diarra.abmcy.com}" \
  --build-arg NEXT_PUBLIC_SITE_URL="${NEXT_PUBLIC_SITE_URL:-https://diarra.abmcy.com}"

echo "== Import dans containerd (k3s) =="
docker save diarra-backend:latest | k3s ctr images import -
docker save diarra-frontend:latest | k3s ctr images import -

echo "== Application des manifests =="
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/minio.yaml
kubectl apply -f k8s/backend.yaml
kubectl apply -f k8s/frontend.yaml
kubectl apply -f k8s/ingress.yaml

echo "== Attente que tout soit prêt =="
kubectl -n diarra rollout status statefulset/postgres --timeout=120s
kubectl -n diarra rollout status statefulset/minio --timeout=120s
kubectl -n diarra rollout status deployment/redis --timeout=60s
kubectl -n diarra rollout status deployment/backend --timeout=120s
kubectl -n diarra rollout status deployment/frontend --timeout=60s

echo "== Terminé =="
kubectl -n diarra get pods
