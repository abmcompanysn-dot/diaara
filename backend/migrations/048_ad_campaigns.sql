-- Sponsorisation de produits sur Facebook + Instagram (Meta), demandée le
-- 2026-10-01. Modèle retenu : le vendeur fait la pub SUR SA PROPRE PAGE
-- Facebook, avec SON PROPRE compte publicitaire Meta, et c'est LUI qui paie
-- Meta (moyen de paiement de son compte pub). DIARRA ne sert que d'interface
-- pour créer et suivre la pub (voir payment/meta_ads.go, meta_oauth.go et
-- handler/ad_handler.go) : aucun argent ne transite par DIARRA.

-- Connexion Facebook du vendeur (Facebook Login). Une seule par vendeur.
--   access_token     : jeton utilisateur longue durée, CHIFFRÉ (AES-256-GCM,
--                      clé META_TOKEN_ENCRYPTION_KEY, voir internal/secretbox) ;
--                      jamais renvoyé au frontend ni logué ;
--   page_* / ad_account_* / currency : choix du vendeur (NULL tant qu'il n'a
--                      pas choisi), currency = devise du compte publicitaire ;
--   needs_reconnect  : jeton expiré ou révoqué (erreur Meta 190) — le vendeur
--                      doit reconnecter son compte ; remis à false à la
--                      reconnexion.
CREATE TABLE IF NOT EXISTS vendor_meta_connections (
  vendor_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  fb_user_id TEXT NOT NULL,
  fb_user_name TEXT NOT NULL DEFAULT '',
  access_token TEXT NOT NULL,
  token_expires_at TIMESTAMPTZ,
  page_id TEXT,
  page_name TEXT,
  ad_account_id TEXT,
  ad_account_name TEXT,
  currency TEXT,
  needs_reconnect BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Campagnes lancées depuis DIARRA sur le compte publicitaire du vendeur.
--   budget_cfa    : budget total saisi par le vendeur (FCFA), converti dans
--                   la devise du compte pub (currency) au lancement ;
--   ad_account_id / page_id / currency : compte, page et devise utilisés
--                   pour CETTE campagne (le vendeur peut en changer ensuite) ;
--   spend_cfa     : dépensé selon Meta, reconverti en FCFA (indicatif).
--
-- status : launching (création en cours chez Meta) | in_review (en
-- vérification chez Meta) | active | completed | stopped (arrêtée par le
-- vendeur ou un admin) | rejected (refusée par Meta) | failed (création
-- impossible).
CREATE TABLE IF NOT EXISTS ad_campaigns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vendor_id UUID NOT NULL REFERENCES users(id),
  product_id UUID NOT NULL REFERENCES products(id),
  platform TEXT NOT NULL DEFAULT 'meta',
  status TEXT NOT NULL DEFAULT 'launching',
  budget_cfa INTEGER NOT NULL CHECK (budget_cfa > 0),
  ad_account_id TEXT NOT NULL,
  page_id TEXT NOT NULL,
  currency TEXT NOT NULL,
  duration_days INTEGER NOT NULL CHECK (duration_days > 0),
  countries TEXT[] NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ,
  external_campaign_id TEXT,
  external_adset_id TEXT,
  external_creative_id TEXT,
  external_ad_id TEXT,
  failure_reason TEXT,
  impressions BIGINT NOT NULL DEFAULT 0,
  reach BIGINT NOT NULL DEFAULT 0,
  clicks BIGINT NOT NULL DEFAULT 0,
  spend_cfa INTEGER NOT NULL DEFAULT 0,
  stats_updated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ad_campaigns_vendor ON ad_campaigns (vendor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ad_campaigns_status ON ad_campaigns (status);
