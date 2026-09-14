-- Ajoute le statut "refunded" aux versements vendeur (payouts.status), pour
-- tracer un versement déjà payé (paid) mais dont l'argent a ensuite été rendu
-- au vendeur/à la plateforme (erreur, litige, double versement...). Sans ce
-- statut, un payout remboursé restait bloqué en "paid" et continuait de
-- compter dans PayoutHandler.totalEarned()/requested pour toujours — le solde
-- disponible du vendeur ne redescendait jamais, sans jamais afficher de
-- négatif non plus (PayoutHandler.Earnings clampe déjà "available" à 0),
-- juste un solde gelé à tort. Incident : versement remboursé côté
-- abmcompanysn, solde vendeur resté figé (2026-09-14).
--
-- refunded_at + refunded_by tracent qui a fait la reconnaissance du
-- remboursement et quand (même schéma que settled_by pour SettleManually).
ALTER TABLE payouts DROP CONSTRAINT payouts_status_check;
ALTER TABLE payouts ADD CONSTRAINT payouts_status_check
  CHECK (status IN ('requested', 'processing', 'paid', 'failed', 'refunded'));

ALTER TABLE payouts ADD COLUMN refunded_at TIMESTAMPTZ;
ALTER TABLE payouts ADD COLUMN refunded_by UUID REFERENCES users(id);
