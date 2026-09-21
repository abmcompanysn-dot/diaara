-- Paliers de sponsoring du DIARRA Summit (Bronze/Argent/Or...), éditables
-- depuis l'admin plutôt que codés en dur dans /summit/sponsors — voir la
-- demande de l'utilisateur : les montants ont déjà changé deux fois en
-- session (100k/700k/1M puis 100k/250k/500k), donc un admin doit pouvoir
-- les ajuster sans redéploiement.
CREATE TABLE summit_sponsor_tiers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  price_cfa INTEGER NOT NULL,
  -- perks : liste de lignes d'avantages, une par ligne (pas de table à part,
  -- ce n'est jamais interrogé indépendamment du palier qui le porte).
  perks TEXT[] NOT NULL DEFAULT '{}',
  highlight BOOLEAN NOT NULL DEFAULT false,
  sort_order SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Entreprises sponsors réelles (logo + lien), rattachées à un palier.
-- Séparé de summit_sponsor_tiers : une entreprise a un logo à uploader et
-- un statut de publication propres, pas juste une ligne de texte.
CREATE TABLE summit_sponsors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tier_id UUID REFERENCES summit_sponsor_tiers(id),
  name TEXT NOT NULL,
  logo_key TEXT,
  website_url TEXT,
  -- published : un admin doit valider l'affichage public (une demande
  -- reçue via SummitSponsorForm ne doit pas apparaître automatiquement
  -- sur la page tant qu'elle n'est pas confirmée/payée).
  published BOOLEAN NOT NULL DEFAULT false,
  sort_order SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_summit_sponsors_published ON summit_sponsors(published);
CREATE INDEX idx_summit_sponsors_tier ON summit_sponsors(tier_id);

-- Seed des 3 paliers actuels (Bronze/Argent/Or — 100k/250k/500k FCFA),
-- pour que /summit/sponsors ait immédiatement du contenu après migration.
INSERT INTO summit_sponsor_tiers (name, price_cfa, perks, highlight, sort_order) VALUES
  ('Bronze', 100000, ARRAY[
    'Logo affiché dans la section partenaires de diarra.app/summit',
    'Mention dans la page de remerciements post-événement'
  ], false, 0),
  ('Argent', 250000, ARRAY[
    'Tout le palier Bronze',
    'Logo dans les emails de confirmation envoyés à chaque inscrit',
    'Stand dédié le jour de l''événement (UCAD)'
  ], false, 1),
  ('Or', 500000, ARRAY[
    'Tout le palier Argent',
    'Logo en tête d''affiche sur la page de l''événement',
    'Mention orale à l''ouverture du Summit',
    'Publication dédiée sur les réseaux DIARRA / ABMCY'
  ], true, 2);
