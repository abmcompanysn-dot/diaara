-- Sponsorisation de produits sur les régies publicitaires (Meta d'abord :
-- Facebook + Instagram), demandée le 2026-10-01 : le vendeur choisit un
-- produit, un budget, une durée et des pays depuis DIARRA ; DIARRA crée la
-- campagne sur SON compte publicitaire (voir payment/meta_ads.go) et la pub
-- renvoie vers la page produit DIARRA.
--
-- Montants (FCFA) :
--   amount_cfa     = ce que le vendeur paie (débité de son solde de gains) ;
--   commission_cfa = part DIARRA (réglage ads_commission_pct) ;
--   ad_budget_cfa  = amount_cfa - commission_cfa, budget réellement confié
--                    à la régie (converti dans la devise du compte pub).
--
-- refunded = true : la somme est rendue au solde du vendeur (pub refusée par
-- Meta, échec de création) — exclue du calcul du solde disponible, voir
-- AdCampaignRepo.balanceDebitsSQL.
--
-- status : launching (payé, création en cours chez la régie) | in_review
-- (en vérification chez Meta) | active | completed | stopped (arrêtée par un
-- admin) | rejected (refusée par Meta, remboursée) | failed (création
-- impossible, remboursée).

CREATE TABLE IF NOT EXISTS ad_campaigns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vendor_id UUID NOT NULL REFERENCES users(id),
  product_id UUID NOT NULL REFERENCES products(id),
  platform TEXT NOT NULL DEFAULT 'meta',
  status TEXT NOT NULL DEFAULT 'launching',
  amount_cfa INTEGER NOT NULL CHECK (amount_cfa > 0),
  commission_cfa INTEGER NOT NULL DEFAULT 0,
  ad_budget_cfa INTEGER NOT NULL CHECK (ad_budget_cfa > 0),
  duration_days INTEGER NOT NULL CHECK (duration_days > 0),
  countries TEXT[] NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  payment_method TEXT NOT NULL DEFAULT 'balance',
  refunded BOOLEAN NOT NULL DEFAULT false,
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
