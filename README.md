# DIARRA — Marketplace de biens numériques

Plateforme de vente de fichiers numériques (PDF, formations, templates…) avec
achat en mobile money (PawaPay), affiliation (« closer »), modération admin et
livraison sécurisée par URL signée.

**Domaines de production** :

| Rôle | Domaine | Servi par |
|------|---------|-----------|
| Site (frontend Next.js statique) | `https://diarra.app` | Cloudflare Worker `diaara` (assets + proxy) |
| API + WebSocket (`/api/*`, `/ws/*`, `/p/*`, `/feed/*`, `/sitemap.xml`) | `https://api.diarra.app` | Backend Go sur le VPS (`169.58.214.115`), derrière Caddy |
| Webhooks entrants (PawaPay, PayPal) | `https://api.diarra.app/api/webhooks/*` | Backend Go directement (ne passe pas par le Worker) |
| Stockage MinIO (liens de téléchargement signés) | `https://files.diarra.app` | MinIO auto-hébergé sur le VPS |

Chemin d'une requête navigateur : `diarra.app` → le Worker route `/api/*`,
`/ws/*`, `/p/*`, `/feed/*`, `/sitemap.xml` vers `BACKEND_URL`
(`https://api.diarra.app`, sous-domaine Cloudflare *proxied*) et sert le reste
depuis les assets statiques. Les webhooks des prestataires appellent
`api.diarra.app` en direct.

`diarra.app` est une propriété de l'entreprise **MAHU** — à ne pas confondre
avec `mahu.app`, un autre produit de la même entreprise, dans un dépôt séparé.
`diarra.abmcy.com` / `origin.abmcy.com` sont d'anciens points d'entrée, hors
service (le mail Mailcow reste sur `abmcy.com`, inchangé).

## Architecture

**Déploiement actuel (vérifié en direct le 2026-09-13) : backend sur un VPS
sous Kubernetes (k3s), frontend servi par un Worker Cloudflare.** La stack
Docker Compose (§7) a existé sur ce même VPS mais a été remplacée par k3s ;
elle n'est plus déployée nulle part — ne pas la considérer comme un
environnement de secours actif.

```
┌────────────────────────────┐   HTTPS (Cloudflare)
│  Navigateur / client       │ ───────────────────────────────────┐
└────────────────────────────┘                                    │
        │  /  (assets statiques)            /api/*, /ws/*, /p/*,  │  redirect 302
        ▼                                   /feed/*, /sitemap.xml ▼ (téléchargement)
┌──────────────────────┐                          ▼      ┌───────────────────────┐
│ Cloudflare Worker      │            ┌──────────────────┐ │ MinIO                 │
│ "diaara" (diarra.app)  │──proxy───► │ Backend Go (k3s)  │►│ S3-compatible         │
│ env.ASSETS = build     │            │ api.diarra.app    │ │ (fichiers produits)   │
│ Next.js statique       │            │ VPS 169.58.214.115│ │ files.diarra.app      │
└──────────────────────┘            └─────────┬─────────┘ └───────────────────────┘
   ▲ déployé par le build Git natif             ▼
   │ de Cloudflare (Workers Builds),   ┌───────────────────┐
   │ pas par ce dépôt/CI               │ PostgreSQL (k3s)   │
                                       │ StatefulSet, réseau │
                                       │ interne au cluster  │
                                       └───────────────────┘
```

- **Frontend** : Next.js (`output: export`) → build statique déployé sur un **Worker
  Cloudflare** (`worker/`, nom `diaara`) via l'intégration Git native de Cloudflare
  (Workers Builds) — se redéploie automatiquement à chaque push sur `master`,
  **indépendamment** de la CI de ce dépôt (`.github/workflows/deploy.yml`, qui ne
  touche que le backend). Le Worker route `/api/*`, `/ws/*`, `/p/*`, `/feed/*`,
  `/sitemap.xml` vers `BACKEND_URL` et sert le reste depuis `env.ASSETS`.
- **Backend** : Go 1.25, API REST + WebSocket temps réel (via `LISTEN/NOTIFY`
  PostgreSQL), email (SMTP / Resend / Mailtrap), upload S3, OTP (email + SMS),
  livraison par URL signée. Tourne en pods Kubernetes (k3s, namespace `diarra`,
  manifests dans `k8s/`) sur le VPS `169.58.214.115`, source synchronisée dans
  `/opt/diarra` (rsync, pas un clone git) et déployée par `k8s/deploy.sh`.
- **Edge / TLS côté VPS** : ingress-nginx (NodePort) derrière **Caddy**, qui obtient
  les certificats Let's Encrypt et route `api.diarra.app`/`files.diarra.app` vers le
  cluster. Caddy est partagé avec d'autres apps du même VPS (non concernées par k3s).
- **Base de données** : PostgreSQL en StatefulSet k3s (réseau interne au cluster
  uniquement, pas de port publié sur l'hôte).
- **Stockage** : **MinIO auto-hébergé** (S3-compatible, bucket `diarra-files`),
  exposé publiquement sur un sous-domaine dédié (`S3_ENDPOINT`) car les liens de
  téléchargement sont des redirections signées vers cet endpoint (le navigateur du
  client y accède directement).

> Ancien déploiement (Render + Neon + Tigris) : voir l'historique git — abandonné,
> plus aucune trace dans la config actuelle.

## Règle d'or : aucune clé en dur

**Aucune clé secrète n'est écrite dans le code ni commitée dans git.**

- Les secrets (Jetons JWT, chaîne de connexion DB, clés S3/PawaPay, SMTP) sont
  uniquement dans des **variables d'environnement** (dashboard Render, `.env` du VPS,
  ou `backend/.env.local` pour le dev local).
- Les fichiers `.env.example` ne contiennent que des placeholders. Le vrai fichier
  local `backend/.env.local` est **gitignoré** (cf. `.gitignore`).
- Tout est lu via `os.Getenv(...)` dans `backend/cmd/server/main.go`.

### Vérifier qu'aucun secret n'est tracé

```bash
git ls-files | grep -iE "\.env($|\.)"          # ne doit lister que les .env.example
git grep -nE "npg_|AKIA|tsec_|tkey_"            # doit être vide
```

## Prérequis

> La table ci-dessous liste des tiers gratuits utiles pour développer/tester
> en local sans rien auto-héberger. **La production actuelle n'utilise ni
> Neon ni Render** : PostgreSQL et le stockage S3 (MinIO) tournent
> auto-hébergés sur le VPS sous k3s (voir Architecture ci-dessus) ; seul
> Cloudflare (frontend) et PawaPay (paiement) sont des tiers réellement
> utilisés en prod.

Comptes (tous utilisables **sans carte bancaire** sur les tiers gratuits) :

| Service    | Usage             | Lien                        |
|------------|-------------------|-----------------------------|
| Neon       | Base PostgreSQL *(dev local uniquement — la prod utilise Postgres auto-hébergé)* | https://neon.tech           |
| Tigris     | Stockage S3 *(dev local uniquement — la prod utilise MinIO auto-hébergé)*     | https://console.storage.dev |
| Render     | Backend Go *(non utilisé en prod actuellement)* | https://render.com          |
| Cloudflare | Worker (frontend, utilisé en prod)    | https://dash.cloudflare.com |
| Mailtrap   | Emails sandbox    | https://mailtrap.io         |
| PawaPay    | Mobile money      | https://pawapay.io *(sandbox d'abord)* |

Outils locaux : Node.js ≥ 20, Go ≥ 1.25, Docker, et les CLIs `neonctl`,
`wrangler`, `render`.

---

## 1. Base de données (Neon)

1. Créer un projet Neon (ex. `diarra`), copier la chaîne `DATABASE_URL`
   (`postgresql://…?sslmode=require`).
2. Migrations exécutées automatiquement au démarrage du backend (voir plus bas).
   Pour les appliquer manuellement : `go run ./cmd/migrate` dans `backend/`.

## 2. Stockage (Tigris)

1. Créer un bucket `diarra-files`.
2. Générer une paire de clés (Access Key / Secret) sur `console.storage.dev`.
   Endpoint : `https://fly.storage.tigris.dev` (région `auto`).

## 3. Backend Go

Le repo inclut `backend/Dockerfile` : il construit le binaire, applique les
migrations (via `docker-entrypoint.sh`) puis démarre le serveur. Health check :
`/health`.

Variables d'environnement (dans le `.env` du VPS, ou `backend/.env.local` en
local) :

```env
DATABASE_URL=postgres://…              # obligatoire
JWT_SECRET=<openssl rand -base64 48>      # obligatoire, ≥ 32 car.
REFRESH_SECRET=<openssl rand -base64 48>  # obligatoire, distinct
FRONTEND_URL=https://diarra.app        # liens emails + /r/ + return_url PayPal
PORT=8080
# Origines autorisées par CORS (séparées par des virgules)
CORS_ALLOWED_ORIGINS=https://diarra.app,http://localhost:3000,http://localhost:3001,http://localhost:3002
# Stockage (fichiers produits) — si absent, l'upload S3 est désactivé
S3_ENDPOINT=https://files.diarra.app
S3_ACCESS_KEY_ID=…
S3_SECRET_ACCESS_KEY=…
S3_BUCKET=diarra-files
S3_REGION=us-east-1
# Emails — un seul fournisseur
SMTP_HOST=…  SMTP_PORT=…  SMTP_USER=…  SMTP_PASS=…  SMTP_FROM=…        # prod
# (alternative) RESEND_API_KEY=…  RESEND_FROM=…
# (développement) MAILTRAP_API_KEY=… MAILTRAP_SANDBOX_ID=… MAILTRAP_FROM=…
# Paiement mobile money (optionnel en dev)
PAWAPAY_API_KEY=…
PAWAPAY_BASE_URL=https://api.sandbox.pawapay.io   # sandbox
PAWAPAY_CALLBACK_URL=https://api.diarra.app/api/webhooks/pawapay
PAWAPAY_CALLBACK_IPS=…   # IP PawaPay autorisées (sécurité)
# Paiement carte / PayPal (checkout + versements vendeur)
PAYPAL_CLIENT_ID=…
PAYPAL_CLIENT_SECRET=…
PAYPAL_BASE_URL=https://api-m.sandbox.paypal.com  # sandbox ; api-m.paypal.com en prod
PAYPAL_WEBHOOK_ID=…   # ID du webhook PayPal → https://api.diarra.app/api/webhooks/paypal
```

Génération des secrets :

```bash
openssl rand -base64 48
```

Le backend démarre **sans** S3, email ni PawaPay (fonctionnalités désactivées,
log `WARNING`), mais **ne démarre pas** sans `JWT_SECRET`, `REFRESH_SECRET` et
`DATABASE_URL`.

### Authentification par OTP

Le backend implémente une vérification d'identité à deux canaux :

| Endpoint                          | Accès | Rôle |
|-----------------------------------|-------|------|
| `POST /api/auth/register`         | public | Création du compte → renvoie `access_token`, `pending_verifications` et `dev_email_otp`/`dev_phone_otp` en dev |
| `POST /api/auth/login`            | public | Login email+password → cookie `refresh_token` httpOnly + `access_token` |
| `POST /api/auth/refresh`          | public | Renouvelle l'`access_token` via le cookie httpOnly |
| `POST /api/auth/logout`           | public | Invalide le refresh token |
| `POST /api/auth/send-otp`         | authentifié | Envoie un OTP 6 chiffres sur le canal demandé (`email` ou `phone`) |
| `POST /api/auth/verify-otp`       | authentifié | Valide le code → marque `email_verified_at` / `phone_verified_at` |
| `POST /api/auth/verify-email`     | public | Ancien flux par lien — conservé |
| `POST /api/auth/forgot-password` / `reset-password` | public | Réinitialisation du mot de passe |
| `GET  /api/auth/me`               | authentifié | Profil + statuts de vérification |

- L'OTP dure 10 min, 3 tentatives max, envoi limité (cooldown 60 s côté frontend).
- En dev, le code est renvoyé dans la réponse (`dev_email_otp`) — jamais en prod.
- L'inscription **redirige vers la vérification d'email obligatoire** avant de
  pouvoir créer un produit (middleware `RequireVerifiedEmail` sur les routes vendeur).

## 4. Frontend (build statique)

Les variables `NEXT_PUBLIC_*` sont **embarquées au build** : toute modification
d'URL impose un rebuild du conteneur `frontend` (voir `frontend/Dockerfile`,
qui reçoit `NEXT_PUBLIC_API_URL` et `NEXT_PUBLIC_WS_URL` en build args).

```bash
cd frontend
NEXT_PUBLIC_API_URL=https://api.diarra.app \
NEXT_PUBLIC_WS_URL=wss://api.diarra.app \
NEXT_PUBLIC_SITE_URL=https://diarra.app \
npm run build          # → out/ (statique, servi par nginx dans le conteneur)
```

En production, ces valeurs sont passées via le `.env` du VPS et
`docker compose build frontend` (voir §7). Le dossier `worker/` (ancien Worker
Cloudflare) n'est plus déployé.

## 5. Paiements (PawaPay + PayPal)

**PawaPay** (mobile money, encaissement + versements vendeur) :

1. Créer un compte marchand PawaPay, récupérer la clé API et les IP de callback.
2. Renseigner les variables `PAWAPAY_*` dans le `.env` du VPS.
3. Webhook de confirmation : `POST https://api.diarra.app/api/webhooks/pawapay`
   (vérifié par Content-Digest + IP autorisées). `PAWAPAY_CALLBACK_URL` doit
   pointer exactement dessus, sinon PawaPay ne pousse jamais le statut final.

**PayPal** (carte bancaire + compte PayPal au checkout ; versements vendeur vers
un email PayPal — KPay a été suspendu de ce flux le 2026-09-03) :

1. Créer une app sur developer.paypal.com (Sandbox puis Live), récupérer
   `PAYPAL_CLIENT_ID` / `PAYPAL_CLIENT_SECRET`.
2. Créer un webhook pointant vers
   `POST https://api.diarra.app/api/webhooks/paypal`, écoutant :
   `CHECKOUT.ORDER.APPROVED`, `PAYMENT.CAPTURE.COMPLETED`,
   `PAYMENT.CAPTURE.DECLINED`, `PAYMENT.CAPTURE.REFUNDED`,
   `PAYMENT.PAYOUTS-ITEM.SUCCEEDED`, `PAYMENT.PAYOUTS-ITEM.FAILED`,
   `PAYMENT.PAYOUTS-ITEM.BLOCKED`.
3. Copier l'**ID du webhook** créé dans `PAYPAL_WEBHOOK_ID` (sert à vérifier les
   webhooks entrants auprès de l'API PayPal — pas de HMAC local). Les IDs
   sandbox et Live sont distincts : recréer le webhook au passage en prod.
4. Activer les **Payouts** sur le compte PayPal Business (peut nécessiter une
   demande selon le pays du compte).

## 6. Compte administrateur

Le backend n'a pas de seed : le flag `is_admin` se positionne en base.

```bash
# 1. Créer le compte par l'API (ou le formulaire d'inscription)
curl -X POST https://api.diarra.app/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@votre-domaine.com","password":"MotDePasseFort!","phone":"+221700000000"}'

# 2. Le promouvoir admin (via neonctl psql ou psql)
UPDATE users SET is_admin = TRUE WHERE email = 'admin@votre-domaine.com';
```

Périmètre admin (route `/api/admin`, protégée par `RequireAdmin`) :

| Endpoint                                  | Rôle                     |
|-------------------------------------------|--------------------------|
| `GET  /api/admin/products/pending`        | Modération — produits à valider |
| `PUT  /api/admin/products/{id}/moderate`  | Approuver / rejeter (`approved`/`rejected`) |
| `GET  /api/admin/users`                   | Liste des utilisateurs   |
| `PUT  /api/admin/users/{id}/role`         | Accorder/retirer `vendeur` ou `closer` |
| `PUT  /api/admin/users/{id}/suspend`      | Suspendre un compte (30 j) |
| `GET  /api/admin/stats`                   | Tableau de bord (ventes, revenus, modération) |
| `GET  /api/admin/sales`                   | Toutes les ventes        |
| `WS   /ws/admin`                          | Alertes modération en direct |

## 7. Déploiement sur un VPS

Le dépôt contient deux façons de faire tourner le backend sur un VPS : la
stack Kubernetes (**k3s, celle réellement utilisée en production**, voir
plus bas) et un `docker-compose.yml` plus ancien, conservé pour référence
mais **plus déployé nulle part**. Le frontend n'est déployé par aucun des
deux : il l'est par le build Git natif de Cloudflare (§4).

### Déploiement historique (Docker Compose) — remplacé par k3s, non actif

1. Copier le projet sur le VPS (`git clone`, ou `scp` si le VPS n'a pas d'accès au
   dépôt privé) et créer un fichier `.env` **à la racine du repo** (à côté de
   `docker-compose.yml`, pas dans `backend/`) avec :
   - `APP_ENV=production` — **important** : sans ça, l'API renvoie les codes
     OTP/reset de mot de passe en clair dans ses réponses (utile en dev sans
     fournisseur email configuré, dangereux en production) et les cookies de
     session ne sont pas marqués `Secure`.
   - `POSTGRES_PASSWORD`, `JWT_SECRET`, `REFRESH_SECRET` (générés avec
     `openssl rand -base64 48`).
   - `FRONTEND_URL=https://diarra.app`, `CORS_ALLOWED_ORIGINS=https://diarra.app`.
   - `NEXT_PUBLIC_API_URL=https://api.diarra.app`,
     `NEXT_PUBLIC_WS_URL=wss://api.diarra.app` (build args du frontend — le
     navigateur passe par le Worker `diarra.app`, qui proxifie vers
     `api.diarra.app`).
   - `NEXT_PUBLIC_SITE_URL=https://diarra.app` — URL publique du site, utilisée
     pour générer `sitemap.xml`, `robots.ts` et les URLs canoniques (SEO).
   - `BACKEND_HOST_PORT` / `BACKEND2_HOST_PORT` / `FRONTEND_HOST_PORT` /
     `MINIO_API_HOST_PORT` / `MINIO_CONSOLE_HOST_PORT` : uniquement si les
     ports par défaut (`8080`, `8082`, `3000`, `9000`, `9001`) sont déjà pris
     par une autre app sur le même VPS.
   - `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` (identifiants MinIO, générés
     aléatoirement) + `S3_ENDPOINT=https://<sous-domaine-stockage>`,
     `S3_ACCESS_KEY_ID=$MINIO_ROOT_USER`, `S3_SECRET_ACCESS_KEY=$MINIO_ROOT_PASSWORD`,
     `S3_BUCKET=diarra-files`, `S3_REGION=us-east-1`.
   - `FIREBASE_PROJECT_ID` (backend, connexion Google) + `NEXT_PUBLIC_FIREBASE_API_KEY`,
     `NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN`, `NEXT_PUBLIC_FIREBASE_PROJECT_ID`,
     `NEXT_PUBLIC_FIREBASE_APP_ID` (frontend, build-time — voir Firebase Console >
     Paramètres du projet) — sans ça le bouton "Continuer avec Google" reste masqué.
     `NEXT_PUBLIC_TIKTOK_URL` optionnel (lien affiché dans le footer).
2. `docker compose build backend backend2 frontend && docker compose up -d` →
   lance Postgres (réseau interne uniquement), MinIO, deux instances du
   backend (`backend`/`backend2`, équilibrées par Caddy — voir étape 4) et le
   frontend, tous bindés sur `127.0.0.1` (jamais exposés directement — la
   convention sur un VPS partagé entre plusieurs apps est de tout faire
   passer par le reverse proxy).
3. Créer le bucket MinIO une fois le conteneur `minio` démarré :

   ```bash
   docker run --rm --network <projet>_default --entrypoint sh minio/mc -c \
     "mc alias set local http://minio:9000 \$MINIO_ROOT_USER \$MINIO_ROOT_PASSWORD \
      && mc mb local/diarra-files"
   ```

4. Reverse proxy **Caddy** (déjà installé) — ajouter au `Caddyfile` (TLS Let's
   Encrypt automatique, aucun certbot à gérer) :

   ```caddyfile
   # Équilibre entre les deux instances du backend (backend/backend2) avec
   # bascule automatique : si l'une échoue son /health, Caddy arrête de lui
   # envoyer du trafic jusqu'à ce qu'elle redevienne saine.
   (backend_lb) {
       reverse_proxy 127.0.0.1:8080 127.0.0.1:8082 {
           lb_policy round_robin
           health_uri /health
           health_interval 10s
           health_timeout 3s
       }
   }

   # api.diarra.app : point d'entrée du backend Go. Le navigateur passe par le
   # Worker Cloudflare (diarra.app), qui proxifie /api/*, /ws/*, /p/*, /feed/*,
   # /sitemap.xml ici ; les webhooks PawaPay/PayPal appellent ce domaine en
   # direct. Le frontend statique est servi par le Worker, pas par Caddy.
   api.diarra.app {
       handle /api/*         { import backend_lb }
       handle /ws/*          { import backend_lb }
       handle /p/*           { import backend_lb }
       handle /r/*           { import backend_lb }
       handle /feed/*        { import backend_lb }
       handle /sitemap.xml   { import backend_lb }
       handle                { import backend_lb }
   }

   files.diarra.app {
       reverse_proxy 127.0.0.1:9000
   }
   ```

   Puis `systemctl reload caddy`. **Important** : le endpoint MinIO doit être
   accessible publiquement (pas seulement en réseau Docker interne) car
   `delivery_handler.go` redirige (302) le navigateur du client directement vers
   une URL pré-signée sur cet endpoint — ce n'est pas le backend qui sert le
   fichier.
5. Ajouter les enregistrements DNS (A) des deux domaines vers l'IP du VPS avant
   l'étape 4, sinon Caddy ne pourra pas obtenir les certificats.

### Déploiement actuel — Kubernetes (k3s)

Le dossier `k8s/` contient la stack Kubernetes réellement utilisée en
production pour DIARRA (k3s à un seul noeud, ingress nginx, tout en interne —
Caddy reste l'unique point d'entrée TLS du VPS, partagé avec les autres apps).
**Mailcow et les autres apps du même VPS ne sont pas concernés** : ils
continuent de tourner en Docker Compose, inchangés — seul DIARRA est passé
sous k3s.

**Déploiement continu** : `.github/workflows/deploy.yml` se déclenche sur
chaque push `master` touchant `backend/**` ou `k8s/**` — il `rsync` le dépôt
vers `/opt/diarra` sur le VPS (sans toucher `.env`) puis lance `k8s/deploy.sh`
à distance. `VPS_HOST` y est fixé en IP (`169.58.214.115`) plutôt qu'un
hostname Hostinger `srvNNNNNNN.hstgr.cloud`, qui se casse silencieusement à
chaque réinstallation/migration de VPS côté Hostinger (vécu le 2026-09-13 :
plusieurs jours de pipeline mort sans alerte, déploiements faits à la main en
attendant — voir `JOURNAL-MODIFICATIONS.md`). Si le pipeline échoue à
nouveau, vérifier dans cet ordre : (1) le secret GitHub `VPS_SSH_KEY`
correspond-il à une clé listée dans `~/.ssh/authorized_keys` sur le VPS
(`ssh diarra-vps "cat ~/.ssh/authorized_keys"`) ; (2) `169.58.214.115` est-il
toujours l'IP du VPS ; (3) `/opt/diarra/.env` existe-t-il encore.

Installation initiale (déjà faite, gardée pour référence) :

1. Installer k3s (Traefik désactivé, Caddy garde les ports 80/443) :
   `curl -sfL https://get.k3s.io | sh -s - --disable traefik --write-kubeconfig-mode 644`
2. Installer l'ingress nginx en NodePort (jamais en LoadBalancer/80/443,
   pour ne pas entrer en conflit avec Caddy) :
   ```bash
   helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
   helm install ingress-nginx ingress-nginx/ingress-nginx \
     --namespace ingress-nginx --create-namespace \
     --set controller.service.type=NodePort \
     --set controller.service.nodePorts.http=30080 \
     --set controller.service.nodePorts.https=30443
   ```
3. `sh k8s/create-secret.sh` — crée le Secret `diarra-secrets` à partir du
   `.env` déjà présent à la racine (jamais commité, jamais affiché).
4. `sh k8s/deploy.sh` — build les images, les importe dans containerd (pas
   de registre, cluster à un seul noeud) et applique tous les manifests.
   C'est ce même script que la CI relance à chaque déploiement automatique,
   et qu'on peut relancer à la main en cas de panne du pipeline :
   `ssh diarra-vps "cd /opt/diarra && BACKEND_CHANGED=true sh k8s/deploy.sh"`
   (après y avoir resynchronisé le code le plus récent).
5. Caddyfile côté VPS : `api.diarra.app` route vers `127.0.0.1:30080` (le
   NodePort nginx), `files.diarra.app` vers MinIO.

Vérifier l'état du cluster : `ssh diarra-vps "kubectl -n diarra get pods"`.

> Notes Docker :
> - `frontend/Dockerfile` reçoit `NEXT_PUBLIC_API_URL` **et** `NEXT_PUBLIC_WS_URL`
>   en build arg (embarqués au build, donc tout changement d'URL impose un rebuild).
> - La variable `CORS_ALLOWED_ORIGINS` du backend doit inclure l'origine du front.
> - Ne jamais commiter le fichier `.env` du VPS (déjà dans `.gitignore`).
> - Sur un VPS partagé avec d'autres apps, vérifier les ports déjà utilisés
>   (`ss -ltnp`) avant de lancer `docker compose up` — les valeurs par défaut de
>   ce repo (`8080`, `3000`→`3001` recommandé, `9000`, `9001`) peuvent entrer en
>   conflit.

## 8. Vérification du déploiement

```bash
# Backend vivant
curl -s https://api.diarra.app/api/products

# Frontend servi
curl -s -o /dev/null -w "%{http_code}\n" https://diarra.app/

# Stockage MinIO joignable publiquement
curl -s https://files.diarra.app/minio/health/live

# WebSocket temps réel (token d'un utilisateur connecté)
node -e "new WebSocket('wss://api.diarra.app/ws/order/<sale_id>?token=<JWT>').onopen=()=>console.log('OK')"
```

## URLs actuelles

- Application : https://diarra.app
- Stockage (MinIO) : https://files.diarra.app
- Dépôt : https://github.com/abmcompanysn-dot/diaara (privé, branche `master`)

## Références clés dans le repo

- `backend/cmd/server/main.go` — config serveur, routes, mode emails/paydunya.
- `backend/internal/handler/auth_handler.go` — inscription, login, OTP, cookies httpOnly.
- `backend/internal/handler/admin_handler.go` — contrôle admin.
- `backend/internal/handler/webhook_handler.go` — callback PawaPay (Content-Digest + IP).
- `backend/internal/handler/delivery_handler.go` — URL signée de livraison.
- `worker/src/index.ts` — proxy API/WS, rate-limit, cron anti-sleep.
- `worker/wrangler.jsonc` — assets frontend, `BACKEND_URL`, KV, cron.
- `frontend/src/lib/api.ts` / `frontend/src/lib/auth.tsx` — contrat API et session
  (refresh token en cookie httpOnly).
- `frontend/src/components/otp-form.tsx` — composant OTP 6 chiffres réutilisable.
- `docker-compose.yml` — stack Postgres + backend + frontend pour VPS/dev.
