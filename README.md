# DIARRA — Marketplace de biens numériques

Plateforme de vente de fichiers numériques (PDF, formations, templates…) avec
achat en mobile money (PawaPay), affiliation (« closer »), modération admin et
livraison sécurisée par URL signée.

## Architecture

```
┌────────────────────────────┐   assets statiques   ┌───────────────────────┐
│  Navigateur / client       │ ───────────────────► │  Cloudflare Worker    │
└────────────────────────────┘                      │  (diarra-worker)      │
        │                                           └───────────────────────┘
        │                                              │          │
        │  /api/* , /ws/* (proxy)                      │          │ cron
        ▼                                              ▼          ▼
┌──────────────────────┐                     ┌────────────────────────────┐
│  Backend Go (Render  │                     │  /api/* , /ws/* proxifiés  │
│  ou VPS via Docker)  │                     │  → backend                 │
└──────────┬───────────┘                     └────────────────────────────┘
           │
   ┌───────┴────────┬──────────────┐
   ▼                ▼              ▼
┌─────────┐   ┌──────────┐   ┌──────────┐
│ Neon DB │   │ Tigris   │   │ PawaPay  │
│ Postgres│   │ fichiers │   │ mobile   │
└─────────┘   └──────────┘   └──────────┘
```

- **Frontend** : Next.js (`output: export`) → dossier `frontend/out` (100 % statique).
- **Edge** : Worker Cloudflare sert les fichiers statiques **et** proxifie l'API
  (`/api/*`) et le WebSocket (`/ws/*`) vers le backend. Un cron (toutes les 5 min)
  garde l'instance éveillée. Un KV Cloudflare limite les appels sur `/api/auth/login`.
- **Backend** : Go 1.25, API REST + WebSocket temps réel (via `LISTEN/NOTIFY`
  PostgreSQL), email (SMTP / Resend / Mailtrap), upload S3, OTP (email + SMS),
  livraison par URL signée (3 téléchargements max). Dockerfile fourni pour Render
  **ou** tout VPS.
- **Base de données** : PostgreSQL managé sur **Neon** (ou Postgres Docker sur VPS).
- **Stockage** : **Tigris** (S3-compatible, bucket `diarra-files`) pour les fichiers
  vendus et les aperçus produits.

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

### Option A — Backend sur VPS, frontend toujours servi par Cloudflare

1. Installer Docker + Docker Compose sur le VPS.
2. Copier le projet, créer un fichier `backend/.env` avec les variables de la
   section 3 (`DATABASE_URL` peut pointer vers Postgres local du VPS ou Neon).
3. Lancer uniquement le backend :

   ```bash
   docker compose up -d --build backend
   ```

4. Exposer le port `8080` (UFW : `sudo ufw allow 8080`). La `BACKEND_URL` du
   worker pointe alors vers `https://<ip-ou-domaine-du-vps>:8080` (avec TLS en
   reverse proxy nginx/caddy devant).
5. Redéployer le worker : `cd worker && npx wrangler deploy`.

### Option B — Tout sur le VPS (frontend + backend + Postgres)

1. Copier le projet sur le VPS et remplir `.env` (voir section 3).
2. Lancer la stack complète :

   ```bash
   docker compose up -d --build
   ```

   → frontend sur `:3000`, backend sur `:8080`, Postgres interne.
3. Mettre un reverse proxy (Caddy ou nginx) avec HTTPS et rediriger le domaine
   vers le conteneur frontend. `FRONTEND_URL` et les URLs build pointent alors
   vers votre domaine au lieu du worker Cloudflare.

> Notes Docker :
> - `frontend/Dockerfile` reçoit `NEXT_PUBLIC_API_URL` en build arg.
> - La variable `CORS_ALLOWED_ORIGINS` du backend doit inclure l'origine du front.
> - Ne jamais commiter le fichier `.env` du VPS.

## 8. Vérification du déploiement

```bash
# Backend vivant
curl -s https://diarra-backend.onrender.com/health

# API via le worker (proxy)
curl -s https://diarra-worker.sabel.workers.dev/api/products

# Frontend servi
curl -s -o /dev/null -w "%{http_code}\n" https://diarra-worker.sabel.workers.dev/

# WebSocket temps réel (token d'un utilisateur connecté)
node -e "new WebSocket('wss://diarra-worker.sabel.workers.dev/ws/order/<sale_id>?token=<JWT>').onopen=()=>console.log('OK')"
```

## URLs actuelles (déploiement en cours)

- Application : https://diarra-worker.sabel.workers.dev
- Backend : https://diarra-backend.onrender.com
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
