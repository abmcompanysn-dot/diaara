# DIARRA — Marketplace de biens numériques

Plateforme de vente de fichiers numériques (PDF, formations, templates…) avec
achat en mobile money (PawaPay), affiliation (« closer »), modération admin et
livraison sécurisée par URL signée.

**Domaines** : `diarra.app` (frontend, Cloudflare Worker) et `abmcy.com` /
`diarra.abmcy.com` / `origin.abmcy.com` (backend + mail Mailcow, VPS) sont
des propriétés de l'entreprise **MAHU** — à ne pas confondre avec `mahu.app`,
un autre produit de la même entreprise, dans un dépôt séparé.

## Architecture

**Déploiement actuel : 100 % auto-hébergé sur VPS (Docker + Caddy), sans Cloudflare.**

```
┌────────────────────────────┐   HTTPS (Let's Encrypt via Caddy)
│  Navigateur / client       │ ───────────────────────────────────┐
└────────────────────────────┘                                    │
        │  /                          /api/*, /ws/*               │  redirect 302
        ▼                                   ▼                     ▼ (téléchargement)
┌──────────────────┐            ┌──────────────────┐   ┌───────────────────────┐
│ diarra-frontend   │            │ diarra-backend    │   │ diarra-minio          │
│ (Next.js export,  │            │ (Go, Chi, WS)     │──►│ S3-compatible         │
│  nginx, port 3001)│            │ port 8080         │   │ (fichiers produits)   │
└──────────────────┘            └─────────┬─────────┘   └───────────────────────┘
                                            ▼
                                  ┌───────────────────┐
                                  │ diarra-postgres    │
                                  │ (réseau Docker     │
                                  │  interne uniquement)│
                                  └───────────────────┘
```

- **Frontend** : Next.js (`output: export`) → build statique servi par nginx dans le
  conteneur `frontend`. Reverse-proxifié par Caddy sur le domaine principal.
- **Edge / TLS** : **Caddy** (installé sur le VPS) fait le reverse proxy et obtient les
  certificats **Let's Encrypt automatiquement** — `/api/*` et `/ws/*` vers le backend,
  le reste vers le frontend. Plus de Worker Cloudflare pour Diarra.
- **Backend** : Go 1.25, API REST + WebSocket temps réel (via `LISTEN/NOTIFY`
  PostgreSQL), email (SMTP / Resend / Mailtrap), upload S3, OTP (email + SMS),
  livraison par URL signée (3 téléchargements max). Tourne dans Docker sur le VPS.
- **Base de données** : PostgreSQL en conteneur Docker sur le VPS (réseau interne
  uniquement, pas de port publié sur l'hôte).
- **Stockage** : **MinIO auto-hébergé** (S3-compatible, bucket `diarra-files`) sur le
  VPS, exposé publiquement sur un sous-domaine dédié (`S3_ENDPOINT`) car les liens de
  téléchargement sont des redirections signées vers cet endpoint (le navigateur du
  client y accède directement).

> Ancien déploiement (Cloudflare Worker + Render + Neon + Tigris) : voir l'historique
> git. Le dossier `worker/` reste dans le repo mais n'est plus déployé.

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

Comptes (tous utilisables **sans carte bancaire** sur les tiers gratuits) :

| Service    | Usage             | Lien                        |
|------------|-------------------|-----------------------------|
| Neon       | Base PostgreSQL   | https://neon.tech           |
| Tigris     | Stockage S3       | https://console.storage.dev |
| Render     | Backend Go        | https://render.com          |
| Cloudflare | Worker + KV       | https://dash.cloudflare.com |
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

Variables d'environnement (dans le dashboard Render, un `.env` VPS, ou
`backend/.env.local` en local) :

```env
DATABASE_URL=<chaîne Neon>          # obligatoire
JWT_SECRET=<openssl rand -base64 48>      # obligatoire, ≥ 32 car.
REFRESH_SECRET=<openssl rand -base64 48>  # obligatoire, distinct
FRONTEND_URL=https://diarra-worker.sabel.workers.dev   # liens emails + /r/
PORT=8080
# Origines autorisées par CORS (séparées par des virgules)
CORS_ALLOWED_ORIGINS=https://diarra-worker.sabel.workers.dev,http://localhost:3000,http://localhost:3001,http://localhost:3002
# Stockage (fichiers produits) — si absent, l'upload S3 est désactivé
S3_ENDPOINT=https://fly.storage.tigris.dev
S3_ACCESS_KEY_ID=…
S3_SECRET_ACCESS_KEY=…
S3_BUCKET=diarra-files
S3_REGION=auto
# Emails — un seul fournisseur
SMTP_HOST=…  SMTP_PORT=…  SMTP_USER=…  SMTP_PASS=…  SMTP_FROM=…        # prod
# (alternative) RESEND_API_KEY=…  RESEND_FROM=…
# (développement) MAILTRAP_API_KEY=… MAILTRAP_SANDBOX_ID=… MAILTRAP_FROM=…
# Paiement mobile money (optionnel en dev)
PAWAPAY_API_KEY=…
PAWAPAY_BASE_URL=https://api.sandbox.pawapay.io   # sandbox
PAWAPAY_CALLBACK_URL=https://diarra-backend.onrender.com/api/webhooks/pawapay
PAWAPAY_CALLBACK_IPS=…   # IP PawaPay autorisées (sécurité)
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

## 4. Frontend + Worker (Cloudflare)

Les variables `NEXT_PUBLIC_*` sont **embarquées au build** : toute modification
d'URL impose un rebuild avant redéploiement du worker.

```bash
# 1. Builder le frontend avec les URL de production
cd frontend
NEXT_PUBLIC_API_URL=https://diarra-worker.sabel.workers.dev \
NEXT_PUBLIC_WS_URL=wss://diarra-worker.sabel.workers.dev \
npm run build          # → out/ (statique)

# 2. Déployer le worker (sert out/ + proxy API/WS + cron anti-sleep)
cd ../worker
npx wrangler login
npx wrangler deploy
```

`worker/wrangler.jsonc` : `assets` → `../frontend/out` (avec `run_worker_first`),
`BACKEND_URL` → URL du backend (Render ou VPS), binding KV `RATE_LIMITS`,
cron `*/5 * * * *`.

## 5. Paiement PawaPay (à configurer)

Aujourd'hui la plateforme fonctionne sans : une commande peut être créée mais le
paiement reste en attente si `PAWAPAY_API_KEY` est absent.

1. Créer un compte marchand PawaPay, récupérer la clé API (et les IP de callback).
2. Ajouter au backend (Render ou VPS) les variables `PAWAPAY_*` vues plus haut.
3. Le webhook de confirmation est `POST /api/webhooks/pawapay` (vérifié par
   Content-Digest + IP autorisées).

## 6. Compte administrateur

Le backend n'a pas de seed : le flag `is_admin` se positionne en base.

```bash
# 1. Créer le compte par l'API (ou le formulaire d'inscription)
curl -X POST https://diarra-worker.sabel.workers.dev/api/auth/register \
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

## 7. Déploiement sur un VPS (Docker)

Le dépôt contient un `docker-compose.yml` qui lance **Postgres + backend +
frontend** sur un VPS. Deux options :

### Déploiement actuel — tout sur le VPS (frontend + backend + Postgres + MinIO)

1. Copier le projet sur le VPS (`git clone`, ou `scp` si le VPS n'a pas d'accès au
   dépôt privé) et créer un fichier `.env` **à la racine du repo** (à côté de
   `docker-compose.yml`, pas dans `backend/`) avec :
   - `APP_ENV=production` — **important** : sans ça, l'API renvoie les codes
     OTP/reset de mot de passe en clair dans ses réponses (utile en dev sans
     fournisseur email configuré, dangereux en production) et les cookies de
     session ne sont pas marqués `Secure`.
   - `POSTGRES_PASSWORD`, `JWT_SECRET`, `REFRESH_SECRET` (générés avec
     `openssl rand -base64 48`).
   - `FRONTEND_URL`, `CORS_ALLOWED_ORIGINS`, `NEXT_PUBLIC_API_URL`,
     `NEXT_PUBLIC_WS_URL` → votre domaine (ex. `https://diarra.abmcy.com`,
     `wss://diarra.abmcy.com`).
   - `NEXT_PUBLIC_SITE_URL=https://diarra.abmcy.com` — URL publique du site,
     utilisée pour générer `sitemap.xml`, `robots.ts` et les URLs canoniques
     (SEO). Sans elle, ces fichiers retombent sur `https://diarra.abmcy.com`
     par défaut.
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

   diarra.abmcy.com {
       handle /api/* { import backend_lb }
       handle /ws/*  { import backend_lb }
       # Liens de partage produit, redirection d'affiliation, flux produits
       # (Google Merchant / Facebook, sitemap) : servis par le backend Go,
       # pas le build statique du frontend.
       handle /p/*           { import backend_lb }
       handle /r/*           { import backend_lb }
       handle /feed/*        { import backend_lb }
       handle /sitemap.xml   { import backend_lb }
       handle                { reverse_proxy 127.0.0.1:3001 }
   }

   s3.diarra.abmcy.com {
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

### Déploiement Kubernetes (k3s) — alternative à Docker Compose

Le dossier `k8s/` contient une stack Kubernetes complète pour DIARRA (k3s à
un seul noeud, ingress nginx, tout en interne — Caddy reste l'unique point
d'entrée TLS du VPS, partagé avec les autres apps). **Mailcow et
miadtalent ne sont pas concernés** : ils continuent de tourner en Docker
Compose, inchangés.

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
5. `sh k8s/migrate-data.sh` — copie les données de la stack Docker Compose
   existante (Postgres + MinIO) vers la nouvelle stack k3s, **sans jamais
   modifier la source**. À ne lancer qu'une fois la stack k3s vérifiée
   saine (`kubectl -n diarra get pods`).
6. Bascule : modifier le Caddyfile pour pointer `diarra.abmcy.com` vers
   `127.0.0.1:30080` (le NodePort nginx) au lieu des ports Docker Compose —
   même méthode que d'habitude (sauvegarde, `diff`, `caddy validate`,
   `reload`).
7. **Garder l'ancienne stack Docker Compose arrêtée mais pas supprimée**
   (`docker compose stop`, jamais `down -v`) pendant la période
   d'observation : retour arrière en une commande si besoin.

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
curl -s https://diarra.abmcy.com/api/products

# Frontend servi
curl -s -o /dev/null -w "%{http_code}\n" https://diarra.abmcy.com/

# Stockage MinIO joignable publiquement
curl -s https://s3.diarra.abmcy.com/minio/health/live

# WebSocket temps réel (token d'un utilisateur connecté)
node -e "new WebSocket('wss://diarra.abmcy.com/ws/order/<sale_id>?token=<JWT>').onopen=()=>console.log('OK')"
```

## URLs actuelles

- Application : https://diarra.abmcy.com
- Stockage (MinIO) : https://s3.diarra.abmcy.com
- Dépôt : https://github.com/idrissoualanni/diarra (privé, branche `master`)

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
