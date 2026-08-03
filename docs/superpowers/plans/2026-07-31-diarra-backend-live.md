# DIARRA — Plan de mise en service (Backend live)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Faire tourner DIARRA V1 de bout en bout : installer Go, connecter la base Neon, compiler/exécuter le backend, corriger les bugs connus, sécuriser l'API et valider le frontend contre le backend réel.

**Architecture:** Le backend Go/Chi (port 8080) parle à PostgreSQL (Neon). Le frontend Next.js (export statique, port 3000) consomme l'API via `NEXT_PUBLIC_API_URL`. PayDunya/R2/Resend sont désactivés tant que les clés ne sont pas fournies (dégradé, mais fonctionnel).

**Tech Stack:** Go 1.22+, Chi v5, pgx v5, Neon PostgreSQL, Next.js 15, cURL (tests), PowerShell (scripts locaux).

---

## Contexte et contraintes (à lire avant de commencer)

- **Aucun outil de base n'est installé** : Go absent, Docker absent, psql absent, pas de repo git. `winget` disponible.
- **Backend JAMAIS compilé** : erreur certaine de compilation connue → collision d'imports `middleware` dans `backend/cmd/server/main.go` (lignes 15 et 22). D'autres erreurs de compilation sont probables ; le Task 4 les traite.
- **BDD** : schéma écrit (`backend/migrations/001_init.sql`) mais jamais appliqué. Aucun `DATABASE_URL` fourni → entrée utilisateur **obligatoire** (Task 2).
- **Disque C** : ~2,25 GB libres après cleanup. Go zip portable ≈ 700 MB une fois extrait — suffisant, mais vérifier l'espace avant chaque téléchargement.
- **Sécurité** : tous les secrets vont dans `backend/.env.local` (jamais commité, fichier déjà ignoré par le `.gitignore` créé en Task 2).

---

## File Structure

| Fichier | Rôle |
|---|---|
| `backend/go.mod` | Module Go (existe, `go 1.22`) |
| `backend/cmd/server/main.go` | Serveur Chi (existe — **à corriger** : imports `middleware`) |
| `backend/cmd/migrate/main.go` | **Nouveau** : applique `migrations/*.sql` (tracké via `schema_migrations`) |
| `backend/start-dev.ps1` | **Nouveau** : charge `.env.local` + lance migrations (`-Migrate`) ou serveur |
| `backend/.env.local` | **Nouveau** : secrets (jamais commité) |
| `backend/.gitignore` | **Nouveau** : ignore `.env.local`, binaires |
| `backend/internal/middleware/rate_limit.go` | **Nouveau** : limiteur à seau (token bucket) par IP |
| `backend/internal/middleware/security_headers.go` | **Nouveau** : headers de sécurité |
| `backend/internal/repository/sale_repo.go` | Modifier : ajouter `UpdatePaymentReference` |
| `backend/internal/handler/sale_handler.go` | Modifier : enregistrer le token PayDunya après `createInvoice` |
| `backend/internal/handler/webhook_handler.go` | Modifier : lookup par token **et** invoice_token (+ custom_data) |
| `frontend/src/app/auth/forgot-password/page.tsx` | **Nouveau** : page mot de passe oublié |
| `frontend/src/lib/api.ts` | Modifier : ajouter `forgotPassword` |
| `frontend/src/app/auth/login/page.tsx` | Modifier : câbler le formulaire (2e étape) |

---

### Task 1 : Installer Go

**Files:**
- Aucun fichier modifié (outillage système).

- [ ] **Step 1 : Vérifier l'espace disque**

Run: `wmic logicaldisk get freespace,caption | findstr C`
Expected: FreeSpace ≥ 1 500 000 000 (1,5 GB). Sinon, nettoyer `%TEMP%` et `node_modules/.cache` avant de continuer.

- [ ] **Step 2 : Installer Go via winget**

Run: `winget install -e --id GoLang.Go --accept-source-agreements --accept-package-agreements`
Expected: exit code 0.

- [ ] **Step 3 : Vérifier l'installation**

Run: `go version`
Expected: `go version go1.2x.y windows/amd64`. Si la commande n'est pas trouvée, relancer le terminal (PATH).

- [ ] **Step 4 (Fallback si winget échoue) : install portable zip**

Run (PowerShell) :
```powershell
$latest = (Invoke-RestMethod "https://go.dev/dl/?mode=json")[0]
$file = ($latest.files | Where-Object { $_.os -eq "windows" -and $_.arch -eq "amd64" -and $_.kind -eq "archive" }).filename
Invoke-WebRequest "https://go.dev/dl/$file" -OutFile "$env:TEMP\go.zip"
Expand-Archive "$env:TEMP\go.zip" -DestinationPath "$env:LOCALAPPDATA\GoDist"
[Environment]::SetEnvironmentVariable("Path", "$env:LOCALAPPDATA\GoDist\go\bin;$env:Path", "User")
```
Run: `go version` (nouveau terminal)
Expected: `go version go1.2x.y windows/amd64`

---

### Task 2 : Environnement local — `.env.local`, `start-dev.ps1`, `.gitignore`

**Files:**
- Create: `backend/.env.local`
- Create: `backend/start-dev.ps1`
- Create: `backend/.gitignore`

**Prérequis utilisateur :** le `DATABASE_URL` d'un projet Neon (console Neon → Settings → Connection string). Sans lui, le backend ne démarre pas (`log.Fatal`).

- [ ] **Step 1 : Créer `backend/.env.local`**

```bash
# DATABASE_URL fourni par l'utilisateur (Neon)
DATABASE_URL=postgres://USER:PASSWORD@HOST.neon.tech/dbname?sslmode=require
JWT_SECRET=<généré>
REFRESH_SECRET=<généré>

# Optionnel — le backend démarre sans, mais désactive la fonctionnalité
# PAYDUNYA_MASTER_KEY=
# PAYDUNYA_PRIVATE_KEY=
# PAYDUNYA_TOKEN=
# PAYDUNYA_RETURN_URL=
# PAYDUNYA_CANCEL_URL=
# PAYDUNYA_WEBHOOK_URL=
# PAYDUNYA_WEBHOOK_SECRET=
# RESEND_API_KEY=
# RESEND_FROM=DIARRA <no-reply@votre-domaine.com>
# FRONTEND_URL=http://localhost:3000
# R2_ACCOUNT_ID=
# R2_ACCESS_KEY_ID=
# R2_SECRET_ACCESS_KEY=
# R2_BUCKET=
# R2_ENDPOINT=
```

Générer les deux secrets (remplacer `<généré>`), depuis `frontend/` :
Run: `node -e "console.log('JWT_SECRET='+require('crypto').randomBytes(48).toString('hex'))"` et idem pour `REFRESH_SECRET`.
Expected: deux lignes `JWT_SECRET=…` / `REFRESH_SECRET=…` à coller dans `.env.local`.

- [ ] **Step 2 : Créer `backend/start-dev.ps1`**

```powershell
param([switch]$Migrate)

$envFile = Join-Path $PSScriptRoot ".env.local"
if (-not (Test-Path $envFile)) { Write-Error ".env.local introuvable"; exit 1 }

Get-Content $envFile | ForEach-Object {
  if ($_ -match '^\s*([A-Za-z0-9_]+)\s*=\s*(.*)$') {
    [Environment]::SetEnvironmentVariable($matches[1], $matches[2].Trim(), 'Process')
  }
}

if ($Migrate) {
  Write-Host "== Application des migrations =="
  go run ./cmd/migrate
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Write-Host "== Serveur : http://localhost:8080 =="
go run ./cmd/server
```

- [ ] **Step 3 : Créer `backend/.gitignore`**

```bash
.env.local
bin/
*.exe
```

---

### Task 3 : Migrateur Go + application du schéma Neon

**Files:**
- Create: `backend/cmd/migrate/main.go`

- [ ] **Step 1 : Écrire `backend/cmd/migrate/main.go`**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		log.Fatalf("schema_migrations: %v", err)
	}

	dir := "migrations"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		log.Fatalf("glob: %v", err)
	}
	sort.Strings(files)

	for _, f := range files {
		name := filepath.Base(f)
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)`, name).Scan(&exists); err != nil {
			log.Fatalf("check %s: %v", name, err)
		}
		if exists {
			fmt.Printf("skip %s\n", name)
			continue
		}
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			log.Fatalf("read %s: %v", f, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Fatalf("begin %s: %v", name, err)
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			tx.Rollback(ctx)
			log.Fatalf("apply %s: %v", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
			tx.Rollback(ctx)
			log.Fatalf("record %s: %v", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			log.Fatalf("commit %s: %v", name, err)
		}
		fmt.Printf("applied %s\n", name)
	}
}
```

- [ ] **Step 2 : Appliquer les migrations**

Run (depuis `backend/`) : `powershell -ExecutionPolicy Bypass -File ./start-dev.ps1 -Migrate`
Expected: `applied 001_init.sql` puis la sortie du serveur démarre. Si `migrations` est introuvable (CWD), lancer depuis `backend/` ou passer le chemin : `go run ./cmd/migrate migrations`.

- [ ] **Step 3 : Vérifier les tables**

Run: `curl -s http://localhost:8080/health`
Expected: `{"status":"ok"}` (prouve que la connexion DB + ping fonctionnent). Arrêter le serveur (Ctrl+C) avant de continuer.

---

### Task 4 : Compiler et corriger le backend (jamais compilé)

**Files:**
- Modify: `backend/cmd/server/main.go` (imports + câblage sécurité)

- [ ] **Step 1 : Tentative de compilation pour lister les erreurs**

Run (depuis `backend/`): `go build ./...`
Expected: erreurs. La collision connue est `middleware redeclared in this block` (main.go importe `go-chi/chi/v5/middleware` **et** `internal/middleware` sans alias, lignes 15 et 22).

- [ ] **Step 2 : Corriger la collision d'imports `middleware`**

Dans `backend/cmd/server/main.go`, aliasser le middleware Chi :

```go
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
```
(importer `internal/middleware` sous son nom `middleware`, inchangé.)

- [ ] **Step 3 : Remplacer les usages Chi**

Dans `main.go`, `r.Use(middleware.Logger)` et `r.Use(middleware.Recoverer)` (lignes ~141-142) deviennent :
```go
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
```

- [ ] **Step 4 : Corriger les autres erreurs de compilation signalées par le build**

Run: `go build ./...` puis `go vet ./...`
Pour chaque erreur : la corriger dans le fichier concerné, relancer le build, répéter jusqu'à `go build ./...` silencieux. Ne pas commenter de logique métier pour masquer une erreur.

- [ ] **Step 5 : `go mod tidy`**

Run: `go mod tidy`
Expected: aucun message d'erreur réseau (dépendances déjà dans `go.mod`, `go.sum` généré).

---

### Task 5 : Démarrer le backend et smoke-tester les endpoints

**Files:** aucun (validation).

- [ ] **Step 1 : Lancer le serveur en arrière-plan**

Run (depuis `backend/`) : `nohup powershell -ExecutionPolicy Bypass -File ./start-dev.ps1 > /tmp/diarra-backend.log 2>&1 &`
Puis attendre ~10 s : `curl -s http://localhost:8080/health`
Expected: `{"status":"ok"}`.

- [ ] **Step 2 : Inscrire un utilisateur**

Run:
```bash
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@diarra.sn","password":"Password123!","phone":"770000001"}'
```
Expected: JSON avec `user`, `access_token`, `refresh_token`. Extraire `access_token` → `$TOKEN`.

- [ ] **Step 3 : Se connecter**

Run:
```bash
curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@diarra.sn","password":"Password123!"}'
```
Expected: JSON avec `access_token`, `refresh_token`. Utiliser ce token comme `$TOKEN`.

- [ ] **Step 4 : Lister le catalogue (vide au départ)**

Run: `curl -s http://localhost:8080/api/products`
Expected: `{"products":[]}`.

- [ ] **Step 5 : Créer un produit (sans R2 — vérifier la tolérance)**

Run:
```bash
curl -s -X POST http://localhost:8080/api/vendor/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Clé Canva Pro 1 an","description":"Compte premium","price_cfa":12500,"category":"subscription"}'
```
Expected: produit créé avec `moderation_status:"pending"` OU erreur claire indiquant qu'un `file_key`/upload est requis. **Noter le comportement** : si un fichier est requis sans R2, soit `file_key` est optionnel (on continue), soit il faut passer par `/upload` (bloqué sans R2) — à trancher en Task 5-Step 8.

- [ ] **Step 6 : Promouvoir l'utilisateur en admin**

Run (SQL direct via le pool) — depuis `backend/` :
```bash
# récupérer l'ID, puis
# via un one-liner Go temporaire ou la console Neon :
```
Si pas de console pratique, créer `backend/cmd/seed/main.go` :
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	email := os.Args[1]
	cmd, err := pool.Exec(ctx, `UPDATE users SET is_admin = TRUE WHERE email = $1`, email)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("rows updated: %d\n", cmd.RowsAffected())
}
```
Run: `DATABASE_URL=<du .env.local> go run ./cmd/seed test@diarra.sn` (ou via `start-dev.ps1` étendu — au choix de l'exécuteur, mais le plus simple est la console Neon → SQL editor : `UPDATE users SET is_admin = TRUE WHERE email = 'test@diarra.sn';`).

- [ ] **Step 7 : Reconnecter l'admin et approuver le produit**

Run:
```bash
# re-login pour un token avec is_admin=true (le JWT embarque le flag)
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" \
  -d '{"email":"test@diarra.sn","password":"Password123!"}' | node -e "let d='';process.stdin.on('data',c=>d+=c).on('end',()=>console.log(JSON.parse(d).access_token))")
curl -s http://localhost:8080/api/admin/products/pending -H "Authorization: Bearer $TOKEN"
# puis PUT /api/admin/products/{id}/moderate {"status":"approved"}
```
Expected: liste des produits pending, puis le produit passe à `approved`.

- [ ] **Step 8 : Créer une commande (paiement désactivé — observer)**

Run:
```bash
curl -s -X POST http://localhost:8080/api/orders \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"product_id":"<id du produit>"}'
```
Expected (sans clés PayDunya) : soit `payment_url` vide avec `order.status:"pending"`, soit erreur `payment_init_failed`. **Noter le résultat** — c'est le comportement attendu en mode dégradé ; à vérifier en Task 6 que le statut reste cohérent.

- [ ] **Step 9 : Tickets support**

Run:
```bash
curl -s -X POST http://localhost:8080/api/tickets \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"subject":"Test","message":"Bonjour"}'
curl -s http://localhost:8080/api/tickets -H "Authorization: Bearer $TOKEN"
```
Expected: ticket créé puis listé.

- [ ] **Step 10 : Vérifier l'interdiction admin sans rôle**

Run: `curl -s http://localhost:8080/api/admin/users -H "Authorization: Bearer $BADTOKEN"` avec un token non-admin.
Expected: `{"error":"forbidden"}`.

---

### Task 6 : Corriger l'incohérence `payment_reference` / token PayDunya

> Le bug : `Create` enregistre un UUID provisoire dans `sales.payment_reference` mais **ne met jamais** le token de facture PayDunya. Le webhook cherche ensuite la vente par `payload.Token` → introuvable → paiement jamais confirmé.

**Files:**
- Modify: `backend/internal/repository/sale_repo.go`
- Modify: `backend/internal/handler/sale_handler.go`
- Modify: `backend/internal/handler/webhook_handler.go`

- [ ] **Step 1 : Ajouter `UpdatePaymentReference` au repo**

Dans `backend/internal/repository/sale_repo.go`, après `UpdateStatus` :
```go
func (r *SaleRepo) UpdatePaymentReference(ctx context.Context, id, ref string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sales SET payment_reference = $2 WHERE id = $1`, id, ref)
	return err
}
```

- [ ] **Step 2 : Enregistrer le token PayDunya après création de facture**

Dans `backend/internal/handler/sale_handler.go`, remplacer le bloc `createInvoice` de `Create` :
```go
	// Initier la facture PayDunya
	invoice, err := h.createInvoice(r.Context(), created, product)
	if err != nil || invoice == nil {
		h.saleRepo.UpdateStatus(r.Context(), created.ID, string(model.SaleFailed))
		http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
		return
	}

	// Enregistrer le token PayDunya comme référence de paiement (unique)
	if invoice.Token != "" {
		if err := h.saleRepo.UpdatePaymentReference(r.Context(), created.ID, invoice.Token); err != nil {
			h.saleRepo.UpdateStatus(r.Context(), created.ID, string(model.SaleFailed))
			http.Error(w, `{"error":"payment_init_failed"}`, http.StatusBadGateway)
			return
		}
		created.PaymentReference = invoice.Token
	}
```

- [ ] **Step 3 : Webhook — lookup par token ET invoice_token ET custom_data**

Dans `backend/internal/handler/webhook_handler.go` :
1. Étendre le payload :
```go
type PayDunyaWebhookPayload struct {
	Token        string `json:"token"`
	Status       string `json:"status"`
	ResponseCode string `json:"response_code"`
	ResponseText string `json:"response_text"`
	InvoiceToken string `json:"invoice_token"`
	CustomData   struct {
		SaleID string `json:"sale_id"`
	} `json:"custom_data"`
}
```
2. Remplacer la recherche :
```go
	sale, err := h.saleRepo.FindByPaymentReference(r.Context(), payload.Token)
	if err != nil {
		sale, err = h.saleRepo.FindByPaymentReference(r.Context(), payload.InvoiceToken)
	}
	if err != nil && payload.CustomData.SaleID != "" {
		sale, err = h.saleRepo.FindByID(r.Context(), payload.CustomData.SaleID)
	}
	if err != nil {
		http.Error(w, `{"error":"sale_not_found"}`, http.StatusNotFound)
		return
	}
```

- [ ] **Step 4 : Vérifier**

Run: `go build ./...`
Expected: silencieux. Si PayDunya sandbox disponible plus tard, re-tester le webhook avec un payload `{"token":"…","invoice_token":"…","custom_data":{"sale_id":"…"},"status":"completed"}` et observer `{"status":"paid"}`.

---

### Task 7 : Page frontend `/auth/forgot-password` (404 actuellement)

**Files:**
- Create: `frontend/src/app/auth/forgot-password/page.tsx`
- Modify: `frontend/src/lib/api.ts`

- [ ] **Step 1 : Ajouter `forgotPassword` au client API**

Dans `frontend/src/lib/api.ts`, dans l'objet `api` (après `refreshToken`) :
```ts
  forgotPassword: (email: string) =>
    fetchApi<{ message: string }>('/api/auth/forgot-password', {
      method: 'POST',
      body: JSON.stringify({ email }),
      skipAuth: true,
    }),
```

- [ ] **Step 2 : Créer la page**

`frontend/src/app/auth/forgot-password/page.tsx` :
```tsx
'use client';

import { useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await api.forgotPassword(email);
      setSent(true);
    } catch (err: any) {
      setError(err.message || 'Échec de la demande');
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="min-h-screen flex items-center justify-center bg-paper px-4 py-16">
      <div className="max-w-md w-full p-8 bg-white rounded-2xl shadow-card border border-green-900/5">
        <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2 text-center">
          // mot de passe oublié
        </p>
        <h1 className="font-display text-3xl font-bold text-center text-green-950">
          Réinitialiser le mot de passe
        </h1>

        {sent ? (
          <div className="mt-6 p-4 bg-green-50 text-green-800 rounded-lg text-sm">
            Si un compte existe avec cet email, un lien de réinitialisation vient d&apos;être envoyé.
          </div>
        ) : (
          <>
            {error && (
              <div className="mt-6 mb-2 p-3 bg-red-50 text-red-600 rounded-lg text-sm">{error}</div>
            )}
            <form onSubmit={handleSubmit} className="mt-6 space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1 text-green-900/80">Email</label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full p-3 border border-green-900/10 rounded-lg focus:border-green-500 focus:shadow-glow transition-all"
                  required
                />
              </div>
              <button
                type="submit"
                disabled={loading}
                className="w-full py-3 gradient-green text-white font-semibold rounded-lg hover:opacity-95 transition-opacity disabled:opacity-50"
              >
                {loading ? 'Envoi...' : "Recevoir un lien de réinitialisation"}
              </button>
            </form>
          </>
        )}

        <p className="mt-4 text-center text-sm text-green-900/70">
          <Link href="/auth/login" className="text-green-600 font-semibold hover:text-green-500">
            ← Retour à la connexion
          </Link>
        </p>
      </div>
    </main>
  );
}
```

- [ ] **Step 3 : Vérifier**

Run (depuis `frontend/`): `npm run build`
Expected: build OK, route `/auth/forgot-password` présente. Vérifier le 200 : `curl -sL -o /dev/null -w "%{http_code}" http://localhost:3000/auth/forgot-password`

---

### Task 8 : Middlewares de sécurité (Phase 10)

**Files:**
- Create: `backend/internal/middleware/rate_limit.go`
- Create: `backend/internal/middleware/security_headers.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1 : Créer `rate_limit.go`**

```go
package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64
	burst   float64
}

func NewRateLimiter(rate, burst float64) *RateLimiter {
	return &RateLimiter{buckets: make(map[string]*bucket), rate: rate, burst: burst}
}

func (l *RateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(b.tokens+elapsed*l.rate, l.burst)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !l.allow(ip) {
			http.Error(w, `{"error":"too_many_requests"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		host, _, err := net.SplitHostPort(xff)
		if err == nil {
			return host
		}
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
```

- [ ] **Step 2 : Créer `security_headers.go`**

```go
package middleware

import "net/http"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'")
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 3 : Câbler dans `main.go`**

Dans `backend/cmd/server/main.go`, après `r.Use(chimw.Recoverer)` :
```go
	r.Use(middleware.NewRateLimiter(10, 40).Middleware)
	r.Use(middleware.SecurityHeaders)
```
(régler les valeurs si besoin : 10 req/s avec burst 40 pour le dev.)

- [ ] **Step 4 : Vérifier**

Run: `go build ./...`
Expected: silencieux.
Run: `curl -si http://localhost:8080/health | grep -i "x-content-type-options"`
Expected: `X-Content-Type-Options: nosniff`. Test rate limit : `for i in $(seq 1 60); do curl -s -o /dev/null -w "%{http_code} " http://localhost:8080/health; done` → au-delà de 40 req en rafale, des `429` apparaissent.

---

### Task 9 : Vérification finale de bout en bout

**Files:** aucun (validation).

- [ ] **Step 1 : Frontend pointé vers le backend**

Dans `frontend/`, créer `.env.local` :
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
```
Rebuild frontend : `npm run build` puis relancer `npm run dev`.

- [ ] **Step 2 : Parcours complet via l'UI**

Depuis http://localhost:3000 : inscrire → login → `/dashboard` → acheter via `/catalog` (mode dégradé : observer le retour de `/api/orders`) → `/support` ouvrir un ticket → `/vendor/products` créer un produit.

- [ ] **Step 3 : Régression**

Run: les 30 endpoints listés dans la réponse précédente — vérifier qu'aucun ne retourne 500 inattendu. `go vet ./...` silencieux.

- [ ] **Step 4 : Bilan**

Documenter dans `DIARRA_CLAUDE.md` (ou fichier `docs/STATUS.md`) : ce qui tourne (auth, catalogue, produits, tickets, admin, sécurité) et ce qui reste dépendant de clés (PayDunya, R2, Resend) + le déploiement Cloudflare.

---

## Self-Review

**Couverture spec (reste à faire après ce plan)** : déploiement Cloudflare (Container + Worker + R2 + Queues) — volontairement hors périmètre, nécessite les comptes/clés. Vérifications manuelles réelles PayDunya/R2/Resend — dépendantes de clés. La Phase 10 WAF côté Worker reste à faire dans le plan déploiement.

**Pas de placeholders** : chaque fichier à créer a son code complet. Les seules entrées utilisateur sont `DATABASE_URL` (Task 2) et les clés optionnelles — inévitables.

**Cohérence des types** : `UpdatePaymentReference(ctx, id, ref)` est déclaré dans le repo (Task 6-Step 1) et appelé depuis `sale_handler.go` avec `created.ID` (string) et `invoice.Token` (string) — aligné. `RateLimiter.Middleware` et `SecurityHeaders` sont des `func(http.Handler) http.Handler`, compatibles `r.Use(...)`.
