-- Migration 040 : annonces admin -> vendeurs ("Dernières mises à jour" sur
-- le dashboard vendeur). Pas d'email — visible uniquement côté dashboard
-- (voir handler.AnnouncementHandler).
CREATE TABLE announcements (
	id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	title      TEXT NOT NULL,
	body       TEXT NOT NULL,
	created_by UUID NOT NULL REFERENCES users(id),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_announcements_created_at ON announcements (created_at DESC);
