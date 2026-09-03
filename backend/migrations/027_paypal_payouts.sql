-- Migration 027 : versements vendeur via PayPal (Payouts API v1)
--
-- Le vendeur peut enregistrer, EN PLUS de son mobile money (payout_phone /
-- payout_operator / payout_country, migration 009), une adresse email PayPal.
-- Les deux canaux coexistent ; PayPal est prioritaire quand il est renseigné
-- (voir PayoutHandler.GetPayoutMethod / SettlePayoutAuto côté admin).
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS payout_paypal_email TEXT;

-- Sur le versement lui-même : l'email PayPal destinataire (copié depuis le
-- vendeur à la création de la demande, jamais relu après — même principe que
-- phone_number/operator pour le mobile money) et l'ID du batch PayPal renvoyé
-- par POST /v1/payments/payouts (pour interroger le statut ensuite).
ALTER TABLE payouts
  ADD COLUMN IF NOT EXISTS paypal_email    TEXT,
  ADD COLUMN IF NOT EXISTS paypal_batch_id TEXT;
