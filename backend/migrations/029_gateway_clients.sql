-- Migration 029 : passerelle de paiement pour des applications externes
-- (ex. ABMCY Core) — DIARRA orchestre déjà PawaPay/KPay/PayPal (voir
-- internal/payment/provider.go, interface PaymentProvider unifiée) ; ces
-- deux tables permettent à d'AUTRES applications de passer par cette même
-- intégration au lieu de refaire leur propre connexion à chaque agrégateur.
--
-- Volontairement séparé de sales/payouts : une transaction gateway appartient
-- à un client externe, pas à un vendeur/produit DIARRA — les mélanger aurait
-- pollué les vraies données de vente (voir la règle "ne jamais perdre/mélanger
-- les données de ventes" dans les notes de session du 2026-09-07).

-- Un client externe autorisé à utiliser la passerelle (ex. "abmcy-core").
-- api_key_hash : jamais la clé en clair, même règle que le hash des tokens
-- de session (voir auth.HashToken) — comparée en temps constant à la
-- vérification (voir middleware.RequireGatewayClient).
CREATE TABLE IF NOT EXISTS gateway_clients (
	id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name             TEXT NOT NULL,
	api_key_hash     TEXT NOT NULL UNIQUE,
	default_callback_url TEXT,
	is_active        BOOLEAN NOT NULL DEFAULT TRUE,
	created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Une transaction initiée par un client externe via la passerelle : dépôt
-- (paiement entrant), versement sortant, ou remboursement. client_ref est
-- l'identifiant CHEZ LE CLIENT (ex. une commande ABMCY Core) — jamais
-- réutilisé côté DIARRA, sert juste à ce que le client retrouve sa
-- transaction dans son propre système sans avoir à stocker notre UUID
-- comme clé primaire chez lui.
CREATE TABLE IF NOT EXISTS gateway_transactions (
	id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	client_id                UUID NOT NULL REFERENCES gateway_clients(id),
	client_ref               TEXT NOT NULL,
	type                     TEXT NOT NULL CHECK (type IN ('deposit', 'payout', 'refund')),
	provider                 TEXT NOT NULL, -- 'pawapay' | 'kpay' | 'paypal'
	provider_ref             TEXT,          -- depositId/payoutId/refundId côté agrégateur
	-- Pour un refund : le provider_ref du dépôt remboursé (nécessaire pour
	-- rejouer/vérifier le remboursement auprès de l'agrégateur).
	related_deposit_ref      TEXT,
	status                   TEXT NOT NULL DEFAULT 'pending'
		CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
	failure_reason           TEXT,
	amount_cfa               INTEGER NOT NULL,
	currency                 TEXT NOT NULL DEFAULT 'XOF',
	-- Coordonnées destinataire pour un payout (jamais pour un deposit, où
	-- l'acheteur choisit lui-même son opérateur sur la page hébergée).
	recipient_phone          TEXT,
	recipient_operator       TEXT,
	country                  TEXT,
	description              TEXT,
	-- URL à notifier à chaque changement de statut (webhook sortant DIARRA ->
	-- client) — surcharge default_callback_url du client si fournie à la
	-- création de CETTE transaction précise.
	callback_url             TEXT,
	created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),

	UNIQUE (client_id, client_ref, type)
);

CREATE INDEX IF NOT EXISTS idx_gateway_tx_provider_ref ON gateway_transactions (provider, provider_ref);
CREATE INDEX IF NOT EXISTS idx_gateway_tx_client ON gateway_transactions (client_id, created_at DESC);
