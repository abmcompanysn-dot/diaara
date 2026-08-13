-- Aperçus filigranés générés automatiquement à la publication d'un produit
-- (pages PDF, image réduite, extrait audio/vidéo). preview_status permet au
-- frontend d'afficher "génération en cours" pendant le traitement en tâche
-- de fond (peut prendre quelques secondes à ~1 min selon le fichier).
ALTER TABLE products ADD COLUMN preview_keys TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE products ADD COLUMN preview_status TEXT NOT NULL DEFAULT 'pending'
  CHECK (preview_status IN ('pending', 'ready', 'unsupported', 'failed'));
