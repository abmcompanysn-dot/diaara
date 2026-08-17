-- Registre des agents support (nom + email) qui reçoivent une notification
-- à chaque nouveau contact via le widget public du site, et journal de ces
-- demandes de contact (visiteur non authentifié, pas de compte requis).
CREATE TABLE IF NOT EXISTS support_agents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS support_contact_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  contact_method TEXT NOT NULL CHECK (contact_method IN ('email', 'whatsapp')),
  contact_value TEXT NOT NULL,
  message TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_support_contact_requests_created_at
  ON support_contact_requests (created_at DESC);
