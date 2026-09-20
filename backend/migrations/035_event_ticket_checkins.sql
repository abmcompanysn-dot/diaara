-- Vérification à l'entrée des billets d'événement (QR code sur le document
-- de confirmation, voir internal/eventfile). Table séparée plutôt que des
-- colonnes sur `sales` : ne concerne que les ventes d'offres d'événement
-- payantes (product.category = 'event'), pas la marketplace en général —
-- éviter d'alourdir le schéma `sales` pour un usage minoritaire.
CREATE TABLE event_ticket_checkins (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sale_id UUID NOT NULL UNIQUE REFERENCES sales(id),
  -- check_in_token : valeur imprévisible encodée dans le QR (pas l'UUID de
  -- la vente en clair, pour qu'un billet ne puisse pas être deviné/forgé à
  -- partir d'un autre identifiant déjà exposé côté acheteur, ex: checkout_token).
  check_in_token TEXT UNIQUE NOT NULL,
  checked_in_at TIMESTAMPTZ,
  checked_in_by UUID REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_event_ticket_checkins_token ON event_ticket_checkins(check_in_token);
