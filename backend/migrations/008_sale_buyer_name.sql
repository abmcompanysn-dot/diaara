-- Nom complet de l'acheteur, saisi au checkout (aucun champ nom n'existe
-- sur le compte utilisateur, donc on le garde au niveau de la commande).
ALTER TABLE sales ADD COLUMN buyer_name TEXT NOT NULL DEFAULT '';
