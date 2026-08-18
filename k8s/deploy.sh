#!/bin/sh
# Build l'image backend avec Docker (comme docker-compose.yml) puis
# l'importe dans le containerd de k3s — pas de registre, cluster à un seul
# noeud. À lancer depuis la racine du dépôt, sur le VPS.
# Le frontend est déployé séparément sur Cloudflare Worker (voir worker/).
set -eu

export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

echo "== Build backend =="
# timeout : un build Docker qui reste accroché (observé plusieurs fois sur
# ce VPS partagé, typiquement sur l'installation de paquets système via
# apk sous forte charge disque/réseau) sature le nœud pendant des dizaines
# de minutes et fait planter tout le cluster (CoreDNS, ingress, etc.) en
# cascade. Un échec propre au bout de 8 min vaut largement mieux qu'un
# blocage silencieux qui coupe le site.
timeout 480 docker build -t diarra-backend:latest ./backend

# Le frontend n'est plus construit/servi ici : il est déployé sur Cloudflare
# Worker (voir worker/), déclenché séparément par le build Git Cloudflare.
echo "== Import dans containerd (k3s) =="
docker save diarra-backend:latest | k3s ctr images import -

echo "== Application des manifests =="
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/headers-configmap.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/minio.yaml
kubectl apply -f k8s/backend.yaml
kubectl apply -f k8s/ingress.yaml

# Le tag d'image reste toujours `:latest` (pas de registre, pas de digest
# par déploiement) : si seul le CODE a changé (pas le fichier YAML), `apply`
# ne voit aucune différence sur le Deployment et ne redémarre rien tout
# seul — on force donc explicitement un rollout à chaque déploiement.
echo "== Redémarrage forcé (nouvelle image, même tag) =="
kubectl -n diarra rollout restart deployment/backend

echo "== Attente que tout soit prêt =="
kubectl -n diarra rollout status statefulset/postgres --timeout=120s
kubectl -n diarra rollout status statefulset/minio --timeout=120s
kubectl -n diarra rollout status deployment/redis --timeout=60s
kubectl -n diarra rollout status deployment/backend --timeout=120s

echo "== Terminé =="
kubectl -n diarra get pods
