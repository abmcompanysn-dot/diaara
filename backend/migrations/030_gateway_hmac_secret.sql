-- Migration 030 : signature HMAC sur les requêtes entrantes de la passerelle
-- de paiement (en plus de la clé API X-Gateway-Key déjà en place, migration
-- 029) — un client externe (ex. ABMCY Core) signe chaque requête avec un
-- secret dédié, distinct de sa clé API, même principe que
-- payment.VerifyKPaySignature (X-KPAY-Signature) déjà utilisé pour les
-- webhooks entrants KPay.
ALTER TABLE gateway_clients
  ADD COLUMN IF NOT EXISTS hmac_secret_hash TEXT;
