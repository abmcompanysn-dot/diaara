# DIARRA — Marketplace de biens numériques

Plateforme de vente de fichiers numériques (PDF, formations, templates…) avec
achat en mobile money, affiliation (« closer »), modération admin et livraison
sécurisée par URL signée.

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
│  Backend Go (Render) │                     │  /api/* , /ws/* proxifiés  │
│  diarra-backend       │                     │  → backend Render          │
└──────────┬───────────┘                     └────────────────────────────┘
           │
   ┌───────┴────────┬──────────────┐
   ▼                ▼              ▼
┌─────────┐   ┌──────────┐   ┌──────────┐
│ Neon DB │   │ Tigris   │   │ PayDunya │  (à configurer plus tard)
│ Postgres│   │ fichiers │   │ mobile   │
└─────────┘   └──────────┘   └──────────┘
```

- **Frontend** : Next.js (`output: export`) → dossier `frontend/out` (100 % statique).
- **Edge** : Worker Cloudflare sert les fichiers statiques **et** proxifie l'API
  (`/api/*`) et le WebSocket (`/ws/*`) vers le backend. Un cron (toutes les 5 min)
  garde l'instance Render gratuite éveillée. Un KV Cloudflare limite les appels
  sur `/api/auth/login`.
- **Backend** : Go 1.25, API REST + WebSocket temps réel (via `LISTEN/NOTIFY`
  PostgreSQL), email, upload S3, livraison par URL signée (3 téléchargements max).
- **Base de données** : PostgreSQL managé sur **Neon**.
- **Stockage** : **Tigris** (S3-compatible, bucket `diarra-files`) pour les fichiers
  vendus et les aperçus produits.

## Règle d'or : aucune clé en dur

**Aucune clé secrète n'est écrite dans le code ni commitée dans git.**

- Les secrets (Jetons JWT, chaîne de connexion DB, clés S3/PayDunya, SMTP) sont
  uniquement dans des **variables d'environnement du dashboard Render**.
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

| Service   | Usage            | Lien                      |
|-----------|------------------|---------------------------|
| Neon      | Base PostgreSQL  | https://neon.tech         |
| Tigris    | Stockage S3      | https://console.storage.dev |
| Render    | Backend Go       | https://render.com        |
| Cloudflare| Worker + KV      | https://dash.cloudflare.com |
| PayDunya  | Paiement mobile money | https://paydunya.com  *(plus tard)* |

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

## 3. Backend Go (Render)

Le repo inclut `backend/Dockerfile` : il construit le binaire, applique les
migrations (via `docker-entrypoint.sh`) puis démarre le serveur. Health check :
`/health`.

Créer un service **Web Service** sur Render (Blueprints Docker) pointant vers
`backend/`, instance gratuite, et renseigner **dans le dashboard Render**
(secret env vars, jamais dans le code) :

```env
DATABASE_URL=<chaîne Neon>          # obligatoire
JWT_SECRET=<openssl rand -base64 48>      # obligatoire, ≥ 32 car.
REFRESH_SECRET=<openssl rand -base64 48>  # obligatoire, distinct
FRONTEND_URL=https://diarra-worker.idrissoualanni0.workers.dev   # liens emails + /r/
PORT=8080
# Stockage (fichiers produits) — si absent, l'upload S3 est désactivé
S3_ENDPOINT=https://fly.storage.tigris.dev
S3_ACCESS_KEY_ID=…
S3_SECRET_ACCESS_KEY=…
S3_BUCKET=diarra-files
S3_REGION=auto
# Emails — un seul fournisseur (SMTP recommandé)
SMTP_HOST=…  SMTP_PORT=…  SMTP_USER=…  SMTP_PASS=…  SMTP_FROM=…
# (alternative) RESEND_API_KEY=…  RESEND_FROM=…
```

Génération des secrets :

```bash
openssl rand -base64 48
```

Le backend démarre **sans** S3 ni email (fonctionnalités désactivées), mais
**ne démarre pas** sans `JWT_SECRET`, `REFRESH_SECRET` et `DATABASE_URL`.

## 4. Frontend + Worker (Cloudflare)

Les variables `NEXT_PUBLIC_*` sont **embarquées au build** : toute modification
d'URL impose un rebuild avant redéploiement du worker.

```bash
# 1. Builder le frontend avec les URL de production
cd frontend
NEXT_PUBLIC_API_URL=https://diarra-worker.idrissoualanni0.workers.dev \
NEXT_PUBLIC_WS_URL=wss://diarra-worker.idrissoualanni0.workers.dev \
npm run build          # → out/ (statique)

# 2. Déployer le worker (sert out/ + proxy API/WS + cron anti-sleep)
cd ../worker
npx wrangler login
npx wrangler deploy
```

`worker/wrangler.jsonc` : `assets` → `../frontend/out` (avec `run_worker_first`),
`BACKEND_URL` → URL Render, binding KV `RATE_LIMITS`, cron `*/5 * * * *`.

## 5. Paiement PayDunya (à configurer plus tard)

Aujourd'hui la plateforme fonctionne sans : une commande peut être créée mais le
paiement est en attente (`payment_init_failed` si PayDunya absent). Pour activer
le paiement mobile money plus tard :

1. Créer le compte marchand PayDunya, récupérer les clés Master/Private/Token.
2. Ajouter au dashboard Render :

```env
PAYDUNYA_MASTER_KEY=…
PAYDUNYA_PRIVATE_KEY=…
PAYDUNYA_TOKEN=…
PAYDUNYA_RETURN_URL=https://diarra-worker.idrissoualanni0.workers.dev/order/checkout
PAYDUNYA_CANCEL_URL=https://diarra-worker.idrissoualanni0.workers.dev/order/cancel
PAYDUNYA_WEBHOOK_URL=https://diarra-backend.onrender.com/api/webhooks/paydunya
PAYDUNYA_WEBHOOK_SECRET=…   # optionnel : vérification HMAC (header PAYDUNYA-SIGNATURE)
```

3. Redéployer le backend. Le mode est affiché au démarrage
   (`Emails via SMTP/Resend/Mailtrap`, `paydunya=nil` sinon).

## 6. Compte administrateur

Le backend n'a pas de seed : le flag `is_admin` se positionne en base.

```bash
# 1. Créer le compte par l'API (ou le formulaire d'inscription)
curl -X POST https://diarra-worker.idrissoualanni0.workers.dev/api/auth/register \
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

## 7. Vérification du déploiement

```bash
# Backend vivant
curl -s https://diarra-backend.onrender.com/health

# API via le worker (proxy)
curl -s https://diarra-worker.idrissoualanni0.workers.dev/api/products

# Frontend servi
curl -s -o /dev/null -w "%{http_code}\n" https://diarra-worker.idrissoualanni0.workers.dev/

# WebSocket temps réel (token d'un utilisateur connecté)
node -e "new WebSocket('wss://diarra-worker.idrissoualanni0.workers.dev/ws/order/<sale_id>?token=<JWT>').onopen=()=>console.log('OK')"
```

## URLs actuelles (déploiement en cours)

- Application : https://diarra-worker.idrissoualanni0.workers.dev
- Backend : https://diarra-backend.onrender.com
- Dépôt : https://github.com/idrissoualanni/diarra (privé, branche `master`)

## Références clés dans le repo

- `backend/cmd/server/main.go` — config serveur, routes, mode emails/paydunya.
- `backend/internal/handler/admin_handler.go` — contrôle admin.
- `backend/internal/handler/webhook_handler.go` — callback PayDunya (HMAC).
- `backend/internal/handler/delivery_handler.go` — URL signée de livraison.
- `worker/src/index.ts` — proxy API/WS, rate-limit, cron anti-sleep.
- `worker/wrangler.jsonc` — assets frontend, `BACKEND_URL`, KV, cron.
- `frontend/src/lib/api.ts` / `use-websocket.ts` — URL build depuis `NEXT_PUBLIC_*`.
