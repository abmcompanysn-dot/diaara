-- Création de produit automatisée (IA/script) sans fichier immédiat : le
-- produit part avec un prompt de génération d'image, à charge pour un
-- modérateur de générer l'image et de l'attacher avant d'approuver
-- (voir ProductHandler.AutoCreate et AttachFile).
ALTER TABLE products ADD COLUMN image_prompt TEXT;
