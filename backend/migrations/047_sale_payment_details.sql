-- Détails de paiement saisis par l'acheteur au checkout, jusqu'ici transmis
-- au prestataire puis perdus : impossible pour le vendeur/l'admin de savoir
-- quel numéro et quel opérateur avaient été utilisés, ni pourquoi un
-- paiement avait échoué — donc impossible de relancer le client autrement
-- que par email (signalé 2026-09-27, client ivoirien bloqué au checkout).
--
-- payer_phone      : MSISDN international réellement débité (ex "2250546968556"),
--                    distinct de users.phone (numéro du compte, souvent vide).
-- payment_operator : code opérateur LOGIQUE choisi (ex "WAVE_SEN", voir
--                    payment.LogicalOperator.Code) — NULL pour carte/PayPal.
-- failure_reason   : code/message d'échec renvoyé par le prestataire, ou
--                    "payment_init_failed" si la demande n'a même pas pu partir.

ALTER TABLE sales
  ADD COLUMN IF NOT EXISTS payer_phone TEXT,
  ADD COLUMN IF NOT EXISTS payment_operator TEXT,
  ADD COLUMN IF NOT EXISTS failure_reason TEXT;
