-- Migration 043 : image optionnelle sur les annonces admin -> vendeurs, et
-- envoi d'une notification push à chaque annonce créée (voir
-- migrations/040_announcements.sql — jusqu'ici lecture passive uniquement).
ALTER TABLE announcements ADD COLUMN image_key TEXT;
