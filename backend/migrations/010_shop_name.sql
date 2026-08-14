-- Identité affichée d'un utilisateur : nom (personne) et nom de boutique
-- (vendeur). Renseignés lors de l'inscription ou du passage au rôle vendeur.
ALTER TABLE users ADD COLUMN display_name TEXT;
ALTER TABLE users ADD COLUMN shop_name TEXT;
