-- Migration 037 : liaison DIARRA <-> YES Messaging — vente conversationnelle
-- "in-chat" (micro-ticket 1$/600 FCFA pour ouvrir la discussion avec le
-- vendeur, solde payé plus tard une fois le client convaincu) + avis vendeur
-- transmis depuis YES à la fin de la conversation.
--
-- YES gère lui-même tout l'état de la conversation (messages, présence,
-- etc.) — DIARRA ne duplique PAS cet état. conversational_sessions n'est
-- qu'un pont léger entre l'identifiant de session côté YES et les
-- enregistrements de paiement/vente côté DIARRA (nécessaire pour retrouver
-- la bonne vente au moment du paiement du solde, et pour rattacher l'avis
-- vendeur à la bonne transaction).
--
-- Authentification des appels YES -> DIARRA : HMAC dédié (X-Signature-SHA256
-- sur clé/secret d'yes_api_clients), volontairement séparé de
-- gateway_clients (029_gateway_clients.sql) qui sert un usage différent
-- (passerelle de paiement générique pour un client externe type ABMCY Core,
-- pas un protocole de session conversationnelle avec ses propres statuts).

-- Client API autorisé à appeler les endpoints /api/yes/*. api_secret_hash :
-- jamais le secret en clair en base, même principe que
-- gateway_clients.api_key_hash — comparé en temps constant à la vérification
-- de signature HMAC (voir middleware.RequireYesClient).
CREATE TABLE IF NOT EXISTS yes_api_clients (
	id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name             TEXT NOT NULL,
	api_key          TEXT NOT NULL UNIQUE, -- identifiant clair (header X-API-Key), pas secret en soi
	api_secret_hash  TEXT NOT NULL,        -- clé HMAC — jamais stocké en clair
	is_active        BOOLEAN NOT NULL DEFAULT TRUE,
	created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Session conversationnelle : un pont entre yes_conversation_id (chez YES) et
-- les paiements DIARRA. sale_id reste NULL tant que seul le micro-ticket est
-- payé — posé au paiement du solde (in-chat checkout), voir CommissionEngine.
CREATE TYPE conversational_session_status AS ENUM (
	'micro_ticket_paid', 'completed', 'cancelled'
);

CREATE TABLE IF NOT EXISTS conversational_sessions (
	id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	yes_conversation_id  TEXT NOT NULL UNIQUE,
	product_id           UUID NOT NULL REFERENCES products(id),
	buyer_id             UUID NOT NULL REFERENCES users(id),
	seller_id            UUID NOT NULL REFERENCES users(id),
	-- Lien d'affiliation déjà validé (produit correspondant, pas d'auto-
	-- référencement) à l'initiation — reréférencé tel quel pour la vente du
	-- solde (in-chat checkout) plutôt que revalidé une seconde fois, pour
	-- qu'un closer révoqué entre les deux étapes ne casse pas un achat déjà
	-- engagé par le micro-ticket.
	referral_link_id     UUID REFERENCES referral_links(id),
	-- Vente(s) DIARRA rattachée(s) à cette session — le micro-ticket est
	-- toujours créé comme Sale à part (payment_reference propre, comptabilisé
	-- séparément) ; sale_id référence la vente du SOLDE, créée seulement au
	-- paiement final (in-chat checkout), pas à l'initiation.
	micro_ticket_sale_id UUID REFERENCES sales(id),
	sale_id              UUID REFERENCES sales(id),
	status               conversational_session_status NOT NULL DEFAULT 'micro_ticket_paid',
	created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_conv_sessions_buyer ON conversational_sessions (buyer_id);
CREATE INDEX IF NOT EXISTS idx_conv_sessions_seller ON conversational_sessions (seller_id);

-- Avis vendeur transmis par YES à la fin d'une conversation aboutie — un
-- seul avis par session (UNIQUE sur session_id), jamais modifiable après
-- coup (pas d'UPDATE prévu, cohérent avec un avis d'achat classique).
CREATE TABLE IF NOT EXISTS vendor_reviews (
	id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	session_id UUID UNIQUE NOT NULL REFERENCES conversational_sessions(id),
	vendor_id  UUID NOT NULL REFERENCES users(id),
	buyer_id   UUID NOT NULL REFERENCES users(id),
	rating     INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
	comment    TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vendor_reviews_vendor ON vendor_reviews (vendor_id);
