-- Paramètres globaux configurables depuis l'interface admin (taux de
-- commission, activation des passerelles de paiement mobile money).
-- Clé/valeur simple : flexible pour ajouter d'autres réglages plus tard.
CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO settings (key, value) VALUES
  ('commission_rate_pct', '15'),
  ('gateway_orange_money', 'true'),
  ('gateway_wave', 'true'),
  ('gateway_mtn_momo', 'true'),
  ('gateway_free_money', 'true'),
  ('gateway_moov_money', 'true');
