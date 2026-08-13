-- Champs nécessaires pour déclencher un versement PawaPay réel (numéro mobile
-- money + opérateur du vendeur) et pour suivre l'ID PawaPay associé.
ALTER TABLE payouts ADD COLUMN phone_number TEXT NOT NULL DEFAULT '';
ALTER TABLE payouts ADD COLUMN operator TEXT NOT NULL DEFAULT '';
ALTER TABLE payouts ADD COLUMN pawapay_payout_id TEXT;
ALTER TABLE payouts ADD COLUMN failure_reason TEXT;
CREATE UNIQUE INDEX idx_payouts_pawapay_id ON payouts (pawapay_payout_id) WHERE pawapay_payout_id IS NOT NULL;

-- Remboursements PawaPay : état intermédiaire pendant le traitement asynchrone
-- + référence de remboursement pour retrouver la vente depuis le webhook.
ALTER TABLE sales DROP CONSTRAINT sales_status_check;
ALTER TABLE sales ADD CONSTRAINT sales_status_check
  CHECK (status IN ('pending','paid','failed','refund_pending','refunded','delivered'));
ALTER TABLE sales ADD COLUMN refund_reference TEXT;
CREATE UNIQUE INDEX idx_sales_refund_reference ON sales (refund_reference) WHERE refund_reference IS NOT NULL;
