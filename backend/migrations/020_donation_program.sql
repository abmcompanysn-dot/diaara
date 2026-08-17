-- Programme de reversement automatique ("Fidélisation") : une part de la
-- commission DIARRA sur chaque vente alimente une cagnotte ; au-delà d'un
-- seuil (réglage admin), elle est répartie automatiquement entre des
-- destinataires mobile money.

-- Cagnotte : une seule ligne (singleton), mise à jour de façon atomique
-- (UPDATE ... RETURNING côté application) — jamais en mémoire, le backend
-- tourne en plusieurs replicas.
CREATE TABLE donation_pool (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  balance_cfa INTEGER NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO donation_pool (id, balance_cfa) VALUES (1, 0);

CREATE TABLE donation_recipients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  phone_number TEXT NOT NULL,
  operator TEXT NOT NULL,
  country TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE donation_payouts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  recipient_id UUID NOT NULL REFERENCES donation_recipients(id),
  amount_cfa INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'requested'
    CHECK (status IN ('requested', 'processing', 'paid', 'failed')),
  pawapay_payout_id TEXT,
  failure_reason TEXT,
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  paid_at TIMESTAMPTZ
);

CREATE INDEX idx_donation_payouts_recipient ON donation_payouts(recipient_id);
CREATE INDEX idx_donation_payouts_status ON donation_payouts(status);
