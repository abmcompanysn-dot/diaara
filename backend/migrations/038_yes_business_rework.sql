-- Migration 038 : refonte de l'intégration YES (voir 037_yes_integration.sql)
-- suite à la lecture de la vraie doc d'intégration YES.abmcy Business
-- (fournie le 2026-09-22, après coup) — le sens du flux était inversé dans
-- la version initiale (037) : on pensait YES appelant DIARRA avec ses
-- propres clients API (table yes_api_clients), en réalité c'est DIARRA qui
-- est CLIENT de YES Business avec une clé API/secret uniques (posés en
-- config serveur, pas en base — un seul partenaire YES, pas plusieurs
-- clients externes comme gateway_clients).
--
-- 037 n'a jamais eu de données réelles (déployée quelques heures, tables
-- vides) — DROP + CREATE plutôt qu'un ALTER incrémental, plus lisible que
-- de complexifier le schéma pour rien.

DROP TABLE IF EXISTS vendor_reviews;
DROP TABLE IF EXISTS conversational_sessions;
DROP TABLE IF EXISTS yes_api_clients;
DROP TYPE IF EXISTS conversational_session_status;

CREATE TYPE conversational_session_status AS ENUM (
	'pending', 'opened', 'offer_sent', 'completed', 'cancelled'
);

-- Pont léger entre une session YES Business et les paiements DIARRA — voir
-- le commentaire de tête de ce fichier et internal/model/yes_integration.go
-- pour le déroulé complet du flux (clic "Discuter avec le vendeur" -> micro-
-- ticket -> session/initiate -> ... -> delivery/fulfill).
--
-- La ligne est créée dès OpenConversation (statut "pending", AVANT même la
-- confirmation du paiement micro-ticket) — nécessaire pour que
-- ConfirmPaidSale retrouve la session via micro_ticket_sale_id et sache
-- QUAND appeler YES session/initiate (jamais avant confirmation réelle du
-- paiement, voir doc : "un paiement échoué en amont ne produit aucun
-- webhook"). yes_session_id/chat_url restent NULL tant que cet appel n'a pas
-- eu lieu (statut encore "pending").
CREATE TABLE conversational_sessions (
	id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	-- ID de session CÔTÉ YES (renvoyé par session/initiate), réutilisé dans
	-- tous les appels YES Business suivants — NULL tant que le micro-ticket
	-- n'est pas confirmé payé (statut "pending").
	yes_session_id       TEXT UNIQUE,
	chat_url             TEXT,
	product_id           UUID NOT NULL REFERENCES products(id),
	buyer_id             UUID NOT NULL REFERENCES users(id),
	seller_id            UUID NOT NULL REFERENCES users(id),
	referral_link_id     UUID REFERENCES referral_links(id),
	micro_ticket_sale_id UUID NOT NULL REFERENCES sales(id),
	sale_id              UUID REFERENCES sales(id),
	status               conversational_session_status NOT NULL DEFAULT 'pending',
	created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_conv_sessions_buyer ON conversational_sessions (buyer_id);
CREATE INDEX idx_conv_sessions_seller ON conversational_sessions (seller_id);
CREATE UNIQUE INDEX idx_conv_sessions_micro_ticket ON conversational_sessions (micro_ticket_sale_id);

-- Avis vendeur : recueilli côté DIARRA puis relayé à YES via session/review
-- (l'interface de chat YES n'a pas de composant de notation propre d'après
-- la doc). Un seul avis par session, jamais modifiable après coup.
CREATE TABLE vendor_reviews (
	id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	session_id UUID UNIQUE NOT NULL REFERENCES conversational_sessions(id),
	vendor_id  UUID NOT NULL REFERENCES users(id),
	buyer_id   UUID NOT NULL REFERENCES users(id),
	rating     INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
	comment    TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_vendor_reviews_vendor ON vendor_reviews (vendor_id);
