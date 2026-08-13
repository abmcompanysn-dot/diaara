-- Moyen de versement mobile money du vendeur (enregistré une fois, réutilisé
-- pour chaque demande de versement). payout_phone est déjà normalisé (MSISDN)
-- et payout_operator est le code PawaPay ("ORANGE_SEN" par ex.), tous deux
-- validés au moment de l'enregistrement.
ALTER TABLE users ADD COLUMN payout_phone TEXT;
ALTER TABLE users ADD COLUMN payout_operator TEXT;
ALTER TABLE users ADD COLUMN payout_country TEXT;
