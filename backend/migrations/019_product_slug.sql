-- Slug lisible pour les URLs produit (ex: "10-templates-n8n-automatisation")
-- à la place de l'UUID technique. NULL autorisé temporairement : les
-- produits existants sont rétro-remplis par ProductRepo.BackfillSlugs au
-- démarrage du serveur (nécessite le même slugify que la création, plus
-- simple à faire côté Go qu'en SQL pur pour la translittération des accents).
ALTER TABLE products ADD COLUMN slug TEXT UNIQUE;
