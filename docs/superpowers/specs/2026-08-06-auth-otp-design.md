# DIARRA — Étape 1 : Authentification complète + OTP + vérifications

Date : 2026-08-06
Statut : Validé (design approuvé par Rodin)

## Contexte
L'authentification existante couvre register/login/refresh/verify-email(lien)/forgot/reset,
mais :
- la table `otp_codes` existe (`channel='sms'`) mais n'est **utilisée nulle part** ;
- aucun provider email n'est configuré en dev → la vérif email n'est jamais opérationnelle ;
- l'email n'est **pas obligatoire** (comptes non vérifiés peuvent vendre/affilier) ;
- le téléphone n'est **jamais vérifié**, alors qu'il est central pour le mobile money ;
- CORS est `["*"]` + credentials (dangereux) ;
- refresh token en `localStorage` (vulnérable XSS).

## Objectifs étape 1
1. Terminer le flux d'authentification (OTP email + téléphone).
2. Configurer un provider email (Mailtrap sandbox pour le dev).
3. Ajouter les pages frontend OTP.
4. Rendre la vérification email **immédiatement obligatoire** à l'inscription (gate dashboard).
5. Durcir : CORS restrictif + refresh token en cookie httpOnly.
6. Tester tous les parcours d'inscription + analyser la sécurité d'auth par rôle.

## Décisions
- **OTP email** : code 6 chiffres (remplace le lien token). Provider **Mailtrap sandbox**.
- **OTP téléphone** : SMS ; en **dev, stub** renvoyant le code en clair dans la réponse
  (interface `SMSSender` abstraite, gateway réel — Termii/Twilio — à brancher plus tard).
- **Gates** : email vérifié requis pour vendre (créer/modifier produit) et affilier
  (créer lien) ; téléphone vérifié requis pour demander un versement (`/payouts`).
- Achat (guest checkout) : aucun gate.
- **Email vérif immédiate** : l'inscription impose la saisie du code email avant l'accès
  au dashboard (interstitiel de vérif).
- **Refresh token → cookie httpOnly** + `SameSite=Lax`, `Secure` en prod.
- Codes OTP **jamais renvoyés en clair en prod** (uniquement via stub/dev).

## Architecture

### Backend
- `internal/otp` — `OTPService` :
  - `Issue(ctx, userID, channel, purpose) (plaintext string, error)` — génère 6 chiffres,
    hash stocké, TTL 10 min, expire l'ancien du même (channel,purpose).
  - `Verify(ctx, userID, channel, purpose, code) error` — vérifie, incrémente `attempts`,
    bloque après 5, marque `used_at`.
  - `ResendCooldown` : 1/min par (userID,channel,purpose).
- `internal/sms` — interface `SMSSender { Send(ctx, to, msg) error }` :
  - `stubSender` (dev) : log + renvoie le code via callback pour le debug.
- `email.NotificationService.SendOTP(email, code)` (ajout).
- Migration `004_otp_purpose.sql` :
  - `ALTER TABLE otp_codes ADD COLUMN purpose TEXT NOT NULL DEFAULT 'verify';`
  - `ALTER TABLE otp_codes ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0;`
  - index `(user_id, channel, purpose)`.
- Endpoints :
  - `POST /api/auth/send-otp` `{channel}` (auth, resend).
  - `POST /api/auth/verify-otp` `{channel, code}` (auth).
  - Supprime `POST /api/auth/verify-email` (token) → remplacé par code.
- `register` : émet code email immédiatement (+ code SMS si téléphone fourni).
  Réponse inclut `pending: ["email"]` (et `"sms"` si phone). **Pas de token en clair en prod.**
- À l'`register`, on **n'accorde l'accès** au dashboard que si email vérifié — sinon le
  frontend reste sur l'interstitiel de vérif (le backend reste connecté pour allow verify).
- Middleware :
  - `RequireVerifiedEmail` sur `/api/vendor/products` (POST/PUT/DELETE) + `/api/closer/links` (POST).
  - `RequireVerifiedPhone` sur `/api/vendor/payouts` (POST).
- CORS : `CORS_ALLOWED_ORIGINS` (env, défaut `http://localhost:3000`), `AllowCredentials: true`.
- Refresh cookie : `refresh` lit le cookie `refresh_token` (ou body pour compat ? non,
  cookie only), `logout` vide le cookie, `login`/`register`/`refresh` posent le cookie httpOnly.

### Frontend
- `/auth/verify-email` → page « entrer le code email » (resend, redirect dashboard si vérifié).
- `/auth/verify-phone` → idem SMS (si téléphone).
- Flux `register` : après inscription → redirect `/auth/verify-email` (email obligatoire),
  puis si téléphone → `/auth/verify-phone` (optionnel pour closer/vendeur).
- `lib/api.ts` : `sendOtp`, `verifyOtp` ; supprimer `verifyEmail(token)`.
- `lib/auth.tsx` : 
  - arrête de stocker refresh en localStorage (le cookie est géré par le navigateur via
    `credentials: 'include'`) ; ne stocke que l'access token (ou access aussi en cookie ?
    non — access reste en mémoire/localStorage, refresh en cookie).
  - `getMe` renvoie `email_verified`, `phone_verified` → bannière de vérif.
- Toutes les requêtes `fetch` passent `credentials: 'include'`.
- Bannière « Email non vérifié » dans le dashboard/sidebar.

### Sécurité — analyse par rôle
- **Client (rôle implicite, pas d'inscription vendeur/closer)** : peut s'inscrire, doit
  vérifier email immédiatement, peut acheter (guest aussi). Sans rôle vendeur/closer, n'a
  accès à aucun endpoint `/vendor/*` ni `/closer/*` (403).
- **Vendeur** : email vérifié requis pour créer/modifier/supprimer un produit. Téléphone
  vérifié requis pour les versements. `RequireAuth + RequireRole(vendeur)` + gates.
- **Closer** : email vérifié requis pour créer un lien d'affiliation. Ne peut pas promouvoir
  son propre produit (`cannot_promote_own_product` déjà présent).
- **Admin** : `is_admin` en base uniquement ; aucun endpoint register ne peut donner admin.
  Accès `/api/admin/*` (`RequireAdmin`). Peut modérer, gérer rôles/suspensions, voir stats/ventes.

## Tests
- Go : `OTPService` (gen/verify/expire/bruteforce/resend cooldown), gates middleware,
  refresh-cookie flow.
- curl E2E : register → send-otp → verify-otp → vendor product bloqué puis débloqué ;
  payout bloqué sans phone vérifié.
- Frontend : parcours register→verify→dashboard pour les 3 rôles.

## Limites / hors-scope (étapes suivantes)
- Pas de 2FA à la connexion (OTP uniquement pour vérification initiale + plus tard step-up).
- Gateway SMS réel (Termii/Twilio) câblé plus tard (interface prête).
- Étape 2 : analyse des flows par utilisateur + règle produit (publication directe sans
  approbation admin, messagerie admin→auteur) + audit de cohérence global.