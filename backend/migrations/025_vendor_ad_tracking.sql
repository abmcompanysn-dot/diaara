-- Permet à chaque vendeur de renseigner son propre Facebook Pixel / Google
-- Tag pour suivre ses visites et conversions sur sa boutique et ses pages
-- produit (publicité gérée par le vendeur lui-même, pas par DIARRA).
ALTER TABLE users ADD COLUMN IF NOT EXISTS facebook_pixel_id TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_tag_id TEXT;
