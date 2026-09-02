-- Migration 026 : relance des acheteurs en attente + opérations manuelles admin
--
-- 1. Relance panier abandonné (LOT 2) : on trace la dernière relance envoyée
--    pour une vente restée "pending", et le nombre de relances (cron 1h puis
--    24h, puis stop ; bouton manuel possible en plus).
-- 2. Confirmation manuelle d'un paiement (LOT 3) : quand un acheteur dit avoir
--    payé mais que le webhook n'est jamais arrivé, un admin peut basculer la
--    vente en "paid" à la main — on garde qui l'a fait et quand.
-- 3. Versement manuel (LOT 4) : quand l'argent est envoyé au vendeur hors
--    PawaPay/KPay (Wave perso, espèces, virement), un admin marque le
--    versement "paid" sans appel prestataire, avec une note libre + un
--    éventuel montant de frais/taxe retenu.

ALTER TABLE sales
  ADD COLUMN IF NOT EXISTS reminded_at            TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS reminder_count         INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS manually_confirmed_by  UUID REFERENCES users(id),
  ADD COLUMN IF NOT EXISTS manually_confirmed_at  TIMESTAMPTZ;

-- Le versement peut désormais être réglé à la main : on autorise une méthode
-- "manual" et on stocke la note + les frais retenus. provider_reference sert
-- alors de référence libre saisie par l'admin (ex. "Wave TX ABC123").
ALTER TABLE payouts
  ADD COLUMN IF NOT EXISTS is_manual      BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS manual_note    TEXT,
  ADD COLUMN IF NOT EXISTS fee_cfa        INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS settled_by     UUID REFERENCES users(id);

-- Retrouver vite les ventes à relancer (cron) : "pending" pas trop vieilles.
CREATE INDEX IF NOT EXISTS idx_sales_pending_reminder
  ON sales (created_at)
  WHERE status = 'pending';
