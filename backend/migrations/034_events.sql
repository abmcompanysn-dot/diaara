-- Événements vendeur : tout vendeur peut créer un événement (webinaire,
-- atelier, lancement...) avec 1 à 3 offres, chacune payante (liée à un
-- Product du catalogue, catégorie "event" — même mécanique que les billets
-- DIARRA Summit) ou gratuite (juste une inscription + lien de visio envoyé
-- par email, pas de paiement).
--
-- Le DIARRA Summit (voir migration 033 + cmd/seed_summit_tickets) devient à
-- terme un événement comme un autre sous ce système, plutôt qu'un cas
-- spécial câblé en dur.

CREATE TABLE events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vendor_id UUID NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  description TEXT,
  cover_image_key TEXT,
  -- event_date : informatif (affiché sur la fiche), pas utilisé pour bloquer
  -- l'inscription — un événement passé reste visible/consultable.
  event_date TIMESTAMPTZ,
  -- meeting_link : lien de visio (Zoom/Meet/...) envoyé aux inscrits des
  -- offres gratuites de cet événement. Un événement 100% payant peut le
  -- laisser vide et le communiquer autrement (voir Product.FileKey livré).
  meeting_link TEXT,
  moderation_status TEXT NOT NULL DEFAULT 'pending'
    CHECK (moderation_status IN ('pending', 'approved', 'rejected')),
  moderation_note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_events_vendor ON events(vendor_id);
CREATE INDEX idx_events_moderation_status ON events(moderation_status);

-- event_offers : 1 à 3 paliers par événement (appliqué côté application,
-- pas en contrainte SQL — voir EventRepo.CreateOffer). is_free XOR product_id
-- : une offre gratuite n'a pas de Product (pas de paiement, pas de fichier à
-- livrer) ; une offre payante référence le Product catalogue qui porte son
-- prix et son fichier de confirmation généré (voir buildTicketHTML-like).
CREATE TABLE event_offers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  is_free BOOLEAN NOT NULL DEFAULT false,
  product_id UUID REFERENCES products(id),
  sort_order SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (is_free = (product_id IS NULL))
);

CREATE INDEX idx_event_offers_event ON event_offers(event_id);

-- event_registrations : inscriptions aux offres GRATUITES uniquement (une
-- offre payante passe par sales/checkout, pas par cette table — voir
-- EventHandler.RegisterFree vs le checkout standard). Générique : sert pour
-- n'importe quel événement, pas seulement le DIARRA Summit.
CREATE TABLE event_registrations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_offer_id UUID NOT NULL REFERENCES event_offers(id) ON DELETE CASCADE,
  full_name TEXT NOT NULL,
  email TEXT NOT NULL,
  phone_number TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_event_registrations_offer_email ON event_registrations(event_offer_id, lower(email));
CREATE INDEX idx_event_registrations_offer ON event_registrations(event_offer_id);
