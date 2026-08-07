-- Migration 004: OTP — purpose + attempts
-- Permet de distinguer plusieurs usages de code simultanés pour un même canal
-- (vérification email/phone, step-up, etc.) et de limiter le brute-force.

ALTER TABLE otp_codes
  ADD COLUMN IF NOT EXISTS purpose   TEXT    NOT NULL DEFAULT 'verify',
  ADD COLUMN IF NOT EXISTS attempts  INTEGER NOT NULL DEFAULT 0;

-- Un seul code valide (non utilisé, non expiré) par (user, channel, purpose).
CREATE INDEX IF NOT EXISTS idx_otp_user_channel_purpose
  ON otp_codes (user_id, channel, purpose);