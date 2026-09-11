-- Migration 031 : plafonds de versement par client passerelle (audit
-- sécurité 2026-09-11) — jusqu'ici POST /api/gateway/v1/payouts n'imposait
-- aucune limite : une clé API + secret HMAC compromis pouvait vider tout le
-- solde PawaPay de DIARRA vers un numéro arbitraire. Défauts alignés sur le
-- plafond KYC "non vérifié" déjà en usage côté ABMCY Core (200 000 FCFA) ;
-- ajustable par client depuis /admin/gateway.
ALTER TABLE gateway_clients
  ADD COLUMN IF NOT EXISTS max_payout_cfa INTEGER NOT NULL DEFAULT 200000,
  ADD COLUMN IF NOT EXISTS daily_payout_cap_cfa INTEGER NOT NULL DEFAULT 500000;
