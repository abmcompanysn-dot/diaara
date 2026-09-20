-- DIARRA Summit (26 novembre 2026) : événement en ligne sur la
-- digitalisation par l'IA et les produits numériques intelligents.
-- Table d'inscriptions publique (page /summit du frontend), indépendante
-- des comptes utilisateurs — un inscrit n'a pas besoin de créer de compte
-- DIARRA pour participer.

CREATE TABLE summit_registrations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  full_name TEXT NOT NULL,
  email TEXT NOT NULL,
  phone_number TEXT NOT NULL,
  profile TEXT NOT NULL
    CHECK (profile IN ('vendeur', 'acheteur', 'entrepreneur', 'curieux')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Un email ne s'inscrit qu'une fois (évite les doublons de confirmation).
CREATE UNIQUE INDEX idx_summit_registrations_email ON summit_registrations(lower(email));
CREATE INDEX idx_summit_registrations_created_at ON summit_registrations(created_at DESC);
