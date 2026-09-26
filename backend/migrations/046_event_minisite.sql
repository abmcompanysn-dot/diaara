-- Lot 2 mini-site événement : logo propre à l'événement (remplace le logo
-- DIARRA sur sa page publique, qui n'affiche déjà plus le header/footer du
-- site — voir Header/Footer côté frontend), texte de présentation de
-- l'organisateur, et un carrousel hero dédié (distinct de la galerie photo
-- de la migration 045).

ALTER TABLE events
  ADD COLUMN logo_key TEXT,
  ADD COLUMN about_organizer TEXT;

-- event_hero_images : jusqu'à 5 images dédiées au carrousel du bandeau hero
-- (limite appliquée côté application, voir EventHandler.AddHeroImage) —
-- distinctes de event_gallery_images (affichée plus bas sur la page, sans
-- limite stricte). Même mécanisme de service (proxy backend, jamais d'URL
-- de stockage publique directe).
CREATE TABLE event_hero_images (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  file_key TEXT NOT NULL,
  sort_order SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_event_hero_images_event ON event_hero_images(event_id, sort_order);
