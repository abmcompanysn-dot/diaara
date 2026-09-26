-- Personnalisation de la page événement publique : couleur d'accent, lieu
-- physique, galerie photo, programme/planning horaire. Objectif : que la
-- fiche événement se lise comme un mini-site dédié plutôt qu'une fiche
-- catalogue générique (voir /event/event-detail.tsx).

ALTER TABLE events
  -- accent_color : couleur hex (#RRGGBB) choisie par le vendeur, remplace le
  -- dégradé vert DIARRA fixe sur SA page événement uniquement.
  ADD COLUMN accent_color TEXT,
  -- Lieu physique, indépendant de meeting_link (un événement peut avoir les
  -- deux : présentiel + suivi en visio). venue_map_url : lien Google Maps ou
  -- équivalent, jamais généré côté serveur (l'utilisateur colle son propre lien).
  ADD COLUMN venue_name TEXT,
  ADD COLUMN venue_address TEXT,
  ADD COLUMN venue_map_url TEXT;

-- Galerie photo : plusieurs images en plus de cover_image_key (ex: photos
-- d'éditions précédentes). Servies via le même mécanisme que cover_image_key
-- (proxy backend, jamais d'URL de stockage publique directe).
CREATE TABLE event_gallery_images (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  file_key TEXT NOT NULL,
  sort_order SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_event_gallery_images_event ON event_gallery_images(event_id, sort_order);

-- Programme / planning : une ligne = un créneau horaire (heure libre en
-- texte, ex "14h00", pas un TIMESTAMPTZ — un programme reste correct même si
-- la date exacte de l'événement change ou n'est pas encore fixée).
CREATE TABLE event_schedule_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  time_label TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT,
  sort_order SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_event_schedule_items_event ON event_schedule_items(event_id, sort_order);
