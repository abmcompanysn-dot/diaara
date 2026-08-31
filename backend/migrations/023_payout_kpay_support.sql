-- Ajoute le support KPay aux versements vendeur : le prestataire n'était pas
-- discriminé du tout sur payouts (contrairement à sales.payment_provider,
-- qui existait déjà mais n'était jamais lu) — pawapay_payout_id supposait
-- PawaPay comme seul prestataire possible.
ALTER TABLE payouts ADD COLUMN provider TEXT NOT NULL DEFAULT 'pawapay';
ALTER TABLE payouts ADD COLUMN provider_reference TEXT;
UPDATE payouts SET provider_reference = pawapay_payout_id WHERE pawapay_payout_id IS NOT NULL;
DROP INDEX IF EXISTS idx_payouts_pawapay_id;
ALTER TABLE payouts DROP COLUMN pawapay_payout_id;
CREATE UNIQUE INDEX idx_payouts_provider_reference ON payouts (provider, provider_reference) WHERE provider_reference IS NOT NULL;

-- Réglages par opérateur exact (remplace le regroupement par marque de
-- gateway_orange_money/gateway_wave/etc. — voir model.GatewayOperatorSettingKey).
-- Valeur à 3 états : "off" | "pawapay" | "kpay". Tous à "pawapay" par défaut
-- (comportement inchangé tant que l'admin ne réassigne pas explicitement).
INSERT INTO settings (key, value) VALUES
  ('gateway_op_orange_sen', 'pawapay'),
  ('gateway_op_wave_sen', 'pawapay'),
  ('gateway_op_free_sen', 'pawapay'),
  ('gateway_op_mtn_momo_civ', 'pawapay'),
  ('gateway_op_orange_civ', 'pawapay'),
  ('gateway_op_wave_civ', 'pawapay'),
  ('gateway_op_mtn_momo_ben', 'pawapay'),
  ('gateway_op_moov_ben', 'pawapay'),
  ('gateway_op_moov_bfa', 'pawapay'),
  ('gateway_op_orange_bfa', 'pawapay'),
  ('gateway_op_mtn_momo_cmr', 'pawapay'),
  ('gateway_op_orange_cmr', 'pawapay'),
  ('gateway_op_airtel_gab', 'pawapay'),
  ('gateway_op_airtel_cog', 'pawapay'),
  ('gateway_op_mtn_momo_cog', 'pawapay'),
  ('gateway_op_vodacom_mpesa_cod', 'pawapay'),
  ('gateway_op_airtel_cod', 'pawapay'),
  ('gateway_op_orange_cod', 'pawapay'),
  ('gateway_op_mtn_momo_gha', 'pawapay'),
  ('gateway_op_airteltigo_gha', 'pawapay'),
  ('gateway_op_vodafone_gha', 'pawapay'),
  ('gateway_op_airtel_nga', 'pawapay'),
  ('gateway_op_mtn_momo_nga', 'pawapay'),
  ('gateway_op_mpesa_ken', 'pawapay'),
  ('gateway_op_airtel_rwa', 'pawapay'),
  ('gateway_op_mtn_momo_rwa', 'pawapay'),
  ('gateway_op_airtel_oapi_uga', 'pawapay'),
  ('gateway_op_mtn_momo_uga', 'pawapay'),
  ('gateway_op_airtel_tza', 'pawapay'),
  ('gateway_op_vodacom_tza', 'pawapay'),
  ('gateway_op_tigo_tza', 'pawapay'),
  ('gateway_op_halotel_tza', 'pawapay'),
  ('gateway_op_airtel_oapi_zmb', 'pawapay'),
  ('gateway_op_mtn_momo_zmb', 'pawapay'),
  ('gateway_op_zamtel_zmb', 'pawapay'),
  ('gateway_op_airtel_mwi', 'pawapay'),
  ('gateway_op_tnm_mwi', 'pawapay'),
  ('gateway_op_movitel_moz', 'pawapay'),
  ('gateway_op_vodacom_moz', 'pawapay'),
  ('gateway_op_mpesa_lso', 'pawapay'),
  ('gateway_op_orange_sle', 'pawapay'),
  ('gateway_op_mpesa_eth', 'pawapay')
ON CONFLICT (key) DO NOTHING;

-- Réglages par pays pour le checkout (mode GATEWAY, page hébergée) — voir
-- model.CheckoutProviderSettingKey. Un par pays de payment.CountryCurrency.
INSERT INTO settings (key, value) VALUES
  ('checkout_provider_ben', 'pawapay'),
  ('checkout_provider_bfa', 'pawapay'),
  ('checkout_provider_cmr', 'pawapay'),
  ('checkout_provider_civ', 'pawapay'),
  ('checkout_provider_cod', 'pawapay'),
  ('checkout_provider_eth', 'pawapay'),
  ('checkout_provider_gab', 'pawapay'),
  ('checkout_provider_gha', 'pawapay'),
  ('checkout_provider_ken', 'pawapay'),
  ('checkout_provider_lso', 'pawapay'),
  ('checkout_provider_mwi', 'pawapay'),
  ('checkout_provider_moz', 'pawapay'),
  ('checkout_provider_nga', 'pawapay'),
  ('checkout_provider_cog', 'pawapay'),
  ('checkout_provider_rwa', 'pawapay'),
  ('checkout_provider_sen', 'pawapay'),
  ('checkout_provider_sle', 'pawapay'),
  ('checkout_provider_tza', 'pawapay'),
  ('checkout_provider_uga', 'pawapay'),
  ('checkout_provider_zmb', 'pawapay')
ON CONFLICT (key) DO NOTHING;
