# DIARRA — Spécification finale du projet

*Document de démarrage — v3, Juillet 2026. Ce fichier remplace toutes les versions précédentes et sert de référence unique pour commencer le développement.*

---

## 0. Vue d'ensemble

Diarra est une marketplace dédiée exclusivement aux produits et services numériques (clés d'abonnement, comptes, ebooks, fichiers PDF). Quatre rôles : **client**, **vendeur numérique**, **closer numérique** (affilié à commission libre), **admin**. Commission plateforme : **15 %** sur chaque transaction.

**Changement majeur v3 : PostgreSQL sans Supabase.** Le projet n'utilise plus aucun service Supabase (ni base de données, ni Auth, ni Storage). Toute la logique — y compris le temps réel — est portée par le backend Go et une base PostgreSQL gérée indépendamment.

---

## 1. Stack technique finale

| Couche | Choix retenu | Détail |
|---|---|---|
| Frontend | Next.js 15, export statique | Servi par le Worker Cloudflare |
| Edge / routage | Cloudflare Worker | Sert le frontend, route `/api/*` et `/ws/*` vers le Container, applique le rate limiting edge |
| Backend | Go, dans un **Cloudflare Container** | Accès TCP complet — connexion directe à Postgres |
| Framework Go | **Chi** | Proche de `net/http`, middlewares matures (auth, rate limit, logs) |
| Base de données | **PostgreSQL managé via Neon** (recommandé) ou Postgres auto-hébergé (VPS) | Neon : zéro Supabase, pairing natif avec Cloudflare, branching pour dev/staging, sans ops. Auto-hébergé : contrôle total mais backups/HA à gérer soi-même. Connexion via `pgx` + `pgxpool`. |
| Stockage & livraison | Cloudflare R2 | Liens signés à expiration courte, watermarking dynamique des PDF |
| Async / orchestration | Cloudflare Queues | Paiement confirmé → livraison → commission → notifications |
| Temps réel | PostgreSQL `LISTEN/NOTIFY` + WebSocket (voir section 3) | Remplace Supabase Realtime, sans nouvelle dépendance externe |
| Paiement | **PawaPay** (Orange Money, Wave, MTN MoMo, Free, Moov) | Dépôt mobile money asynchrone + webhook, zone XOF — voir section 9 |
| Email transactionnel | Resend ou Postmark | Appelé en HTTP direct depuis le Go |
| Authentification | JWT signé côté Go, hashage bcrypt/argon2 | Entièrement custom, aucune dépendance Auth externe |
| Rate limiting | WAF Cloudflare (edge) + compteurs en base Postgres | Jamais en mémoire locale du process (angle mort #18) |
| Hébergement | Cloudflare de bout en bout | Worker + Container + R2 + Queues |

---

## 2. Architecture

**Trois zones, quatre couches :**

1. **Edge (Worker Cloudflare)** — sert le frontend, route les appels API et WebSocket vers le Container, applique le rate limiting avant que la requête n'atteigne le Go.
2. **Backend (Container Go)** — toute la logique métier : commandes, commissions, rôles, appels à l'agrégateur de paiement, connexion directe TCP à Postgres, gestion des connexions WebSocket temps réel.
3. **Données (Postgres via Neon, R2, Queues)** — Postgres porte le schéma relationnel (section 4). R2 stocke les fichiers vendus. Queues absorbe l'asynchrone du paiement.
4. **Services externes** — agrégateur de paiement mobile money, fournisseur d'email transactionnel.

**Flux d'une transaction :** achat initié côté client → commande créée en base par le Go → paiement traité par l'agrégateur → webhook de confirmation reçu par le Worker → message publié sur Queue → consumer traite la livraison (lien R2 signé) et le calcul de commission (15 % plateforme, reste vendeur/closer) → notifications envoyées (email + push temps réel si l'utilisateur est connecté).

---

## 3. Temps réel des données

Sans Supabase Realtime, le temps réel est reconstruit avec deux briques déjà présentes dans la stack, sans nouvelle dépendance :

### 3.1 — Mécanisme

- **PostgreSQL `LISTEN`/`NOTIFY`** : des triggers SQL sur les tables `sales`, `referral_links` (clics) et `products` (statut de modération) appellent `pg_notify()` à chaque changement pertinent.
- **Chaque instance du Container Go maintient une connexion `LISTEN` permanente** vers Postgres. Comme Postgres diffuse la notification à *tous* les listeners, chaque instance reçoit l'événement quel que soit celui qui a traité l'écriture initiale — ça résout directement l'angle mort #18 (compteurs/état non partagés entre instances), sans avoir besoin d'un store externe type Redis.
- **WebSocket** : le Container Go expose des endpoints `/ws/vendor/:id`, `/ws/closer/:id`, `/ws/admin`, `/ws/order/:id`. Le Worker proxy bidirectionnellement les connexions WebSocket vers le Container (fonctionnalité native du package `@cloudflare/containers`, pas de bricolage nécessaire). Chaque instance Go broadcast aux clients WebSocket connectés localement les événements reçus via `LISTEN`.

### 3.2 — Cas d'usage concrets

| Événement temps réel | Qui le voit | Effet UI |
|---|---|---|
| Statut de commande change (`pending` → `paid` → `delivered`) | Client | La page d'achat se met à jour sans rechargement pendant l'attente de confirmation mobile money |
| Nouvelle vente enregistrée | Vendeur concerné | Compteur de ventes et revenu mis à jour en direct sur le dashboard |
| Clic ou conversion sur un lien | Closer concerné | Statistiques du lien mises à jour en direct |
| Nouveau produit soumis / transaction suspecte | Admin | Flux de modération et alertes fraude en direct |

### 3.3 — Ce qui n'a pas besoin de temps réel

Catalogue, fiches produits, historique d'achats : de simples requêtes HTTP classiques suffisent. Le temps réel est réservé aux écrans où l'attente d'un rafraîchissement dégraderait l'expérience (paiement en cours, dashboards vendeur/closer/admin).

---

## 4. Schéma de base de données (PostgreSQL)

```sql
-- Utilisateurs & authentification
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  phone TEXT UNIQUE,
  password_hash TEXT NOT NULL,
  email_verified_at TIMESTAMPTZ,
  phone_verified_at TIMESTAMPTZ,
  is_admin BOOLEAN NOT NULL DEFAULT FALSE,
  failed_login_attempts INTEGER NOT NULL DEFAULT 0,
  locked_until TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Rôles cumulables (client = implicite pour tous)
CREATE TABLE user_roles (
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('vendeur','closer')),
  PRIMARY KEY (user_id, role)
);

CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE email_verifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE otp_codes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  channel TEXT NOT NULL DEFAULT 'sms',
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE password_resets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Catalogue
CREATE TABLE products (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vendor_id UUID NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  description TEXT,
  price_cfa INTEGER NOT NULL CHECK (price_cfa >= 0),
  category TEXT NOT NULL,
  file_key TEXT NOT NULL,                        -- clé objet R2
  moderation_status TEXT NOT NULL DEFAULT 'pending'
    CHECK (moderation_status IN ('pending','approved','rejected')),
  moderation_note TEXT,
  affiliate_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  max_closer_commission_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Affiliation (closer numérique)
CREATE TABLE referral_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id UUID NOT NULL REFERENCES products(id),
  closer_id UUID NOT NULL REFERENCES users(id),
  slug TEXT UNIQUE NOT NULL,
  commission_pct NUMERIC(5,2) NOT NULL CHECK (commission_pct BETWEEN 0 AND 100),
  clicks INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ventes
CREATE TABLE sales (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  product_id UUID NOT NULL REFERENCES products(id),
  buyer_id UUID NOT NULL REFERENCES users(id),
  referral_link_id UUID REFERENCES referral_links(id),
  amount_cfa INTEGER NOT NULL,
  platform_fee_cfa INTEGER NOT NULL,
  closer_commission_cfa INTEGER NOT NULL DEFAULT 0,
  vendor_amount_cfa INTEGER NOT NULL,
  payment_provider TEXT NOT NULL,
  payment_reference TEXT UNIQUE NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending','paid','failed','refunded','delivered')),
  delivered_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Livraison sécurisée
CREATE TABLE delivery_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sale_id UUID NOT NULL REFERENCES sales(id),
  signed_url_token TEXT UNIQUE NOT NULL,
  max_downloads INTEGER NOT NULL DEFAULT 3,
  download_count INTEGER NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Versements
CREATE TABLE payouts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  amount_cfa INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'requested'
    CHECK (status IN ('requested','processing','paid','failed')),
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  paid_at TIMESTAMPTZ
);

-- Support & litiges
CREATE TABLE support_tickets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  sale_id UUID REFERENCES sales(id),
  subject TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','answered','closed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ticket_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ticket_id UUID NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
  author_id UUID NOT NULL REFERENCES users(id),
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Rate limiting applicatif (compteurs partagés entre instances)
CREATE TABLE rate_limit_counters (
  key TEXT PRIMARY KEY,              -- ex: 'referral_link_create:<user_id>:<date>'
  count INTEGER NOT NULL DEFAULT 1,
  window_start TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Temps réel : notification à chaque nouvelle vente ou changement de statut
CREATE OR REPLACE FUNCTION notify_sale_change() RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify('sales_channel', row_to_json(NEW)::text);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sales_notify
AFTER INSERT OR UPDATE ON sales
FOR EACH ROW EXECUTE FUNCTION notify_sale_change();

-- Temps réel : notification à chaque clic sur un lien d'affiliation
CREATE OR REPLACE FUNCTION notify_referral_click() RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify('referral_channel', row_to_json(NEW)::text);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER referral_notify
AFTER UPDATE OF clicks ON referral_links
FOR EACH ROW EXECUTE FUNCTION notify_referral_click();

-- Temps réel : notification à chaque changement de statut de modération
CREATE OR REPLACE FUNCTION notify_moderation_change() RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify('moderation_channel', row_to_json(NEW)::text);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER moderation_notify
AFTER UPDATE OF moderation_status ON products
FOR EACH ROW EXECUTE FUNCTION notify_moderation_change();
```

---

## 5. Fonctionnalités par domaine

### 5.1 — Authentification & comptes
Inscription (email + mot de passe, hashage bcrypt/argon2) · connexion/déconnexion · vérification d'email (lien) · vérification téléphone (OTP SMS) · mot de passe oublié/réinitialisation · changement de mot de passe · sessions (access + refresh token, révocation) · rôles cumulables · verrouillage temporaire après échecs de connexion · suppression de compte / export RGPD-like.

### 5.2 — Catalogue & produits
Dépôt produit (fichier, titre, description, prix, catégorie) · modification/retrait · statut de modération · recherche/filtres · fiche détaillée · marquage "ouvert à l'affiliation".

### 5.3 — Achat & paiement
Création de commande · paiement mobile money · webhook de confirmation · gestion des échecs (réessai) · historique d'achats · **statut de commande en temps réel** (section 3).

### 5.4 — Livraison de fichiers
Lien signé à expiration courte · watermarking dynamique des PDF · limite de téléchargements · re-téléchargement dans la limite autorisée.

### 5.5 — Affiliation (closer numérique)
Génération de lien unique · commission libre plafonnée · page de présentation personnalisée · statistiques en temps réel (clics, conversions, revenu) · détection anti-fraude (closer = acheteur).

### 5.6 — Vendeur — boutique & revenus
Création/paramétrage boutique · plafond de commission autorisé aux closers · statistiques de vente en temps réel · liste des liens générés.

### 5.7 — Commissions & versements
Calcul automatique (15 % plateforme) · répartition vendeur/closer · demande de retrait · historique des versements · export du relevé de gains.

### 5.8 — Notifications
Email de bienvenue et vérification · confirmation d'achat · livraison du produit · notification de vente (vendeur/closer) · confirmation de retrait · WhatsApp (V2). Fournisseur : Resend ou Postmark, appelé en HTTP direct depuis le Go.

### 5.9 — Support & litiges
Ouverture de ticket · réponse vendeur/admin · demande de remboursement.

### 5.10 — Modération & administration
Validation/refus des produits · suspension de compte · détection de transactions suspectes (dashboard temps réel) · configuration des paramètres globaux · tableau de bord global.

### 5.11 — Sécurité & anti-fraude
Rate limiting edge (WAF Cloudflare) sur les routes sensibles · rate limiting applicatif (table `rate_limit_counters`, jamais en mémoire) · KYC léger avant tout retrait · plafond de commission closer.

---

## 6. Angles morts à garder en tête

| Catégorie | Points clés |
|---|---|
| Légal | Revente de clés d'abonnement tierces (CGU tiers) · closer numérique = affiliation, jamais du recrutement (risque de requalification pyramidale) · statut juridique d'intermédiaire de paiement à valider · CGU/CGV à rédiger avant le premier utilisateur payant |
| Fraude | Auto-référencement (closer = acheteur) · blanchiment (KYC + seuils) · chargebacks mobile money (pas de dispute standardisée) |
| Produit | Anti-piratage des fichiers livrés · modération du catalogue avant publication · MVP trop large si le closer est inclus dès le V1 · concurrence (Selar, Gumroad, Payhip) |
| Stack | Maturité de Cloudflare Containers (produit récent) · cold start sur pic de charge (webhook de paiement) · compteurs et sessions temps réel non partagés entre instances (résolu par `LISTEN/NOTIFY`, voir section 3) · vendor lock-in Cloudflare |

---

## 7. Roadmap

**V1 — MVP (cœur transactionnel)**
Client + vendeur + admin, sans closer numérique. Authentification complète. Rate limiting edge dès le jour 1. Statut de commande en temps réel (le cas d'usage temps réel le plus critique, car l'attente de confirmation mobile money est le moment le plus frustrant sans feedback live).

**V2 — Croissance**
Closer numérique avec commission plafonnée · dashboards temps réel vendeur/closer/admin complets · watermarking dynamique · notifications WhatsApp · export fiscal · 2FA · extension multi-pays paiement.

---

## 8. Décisions ouvertes avant de coder

1. **Base de données** : Neon (managé, recommandé) vs Postgres auto-hébergé (VPS) — arbitrage ops vs coût long terme.
2. **Agrégateur de paiement principal** : **résolu → PawaPay** (dépôt mobile money asynchrone, webhook, zone XOF : Sénégal, Côte d'Ivoire, Bénin, Burkina Faso). Mise en production : créer le compte, obtenir une clé API, configurer le callback URL et lister les IP de callback dans `PAWAPAY_CALLBACK_IPS`.
3. **Plafond de commission closer** : fixer un maximum (ex. 50 %).
4. **Politique de remboursement** : à formaliser dans les CGV.
5. **Statut juridique** : à faire valider par un juriste local.

---

*Prochaine étape : initialiser le repo Go (module, structure `cmd/`, `internal/`), écrire le premier `Dockerfile` pour le Container Cloudflare, et provisionner la base Postgres (Neon ou VPS) à partir du schéma de la section 4.*
