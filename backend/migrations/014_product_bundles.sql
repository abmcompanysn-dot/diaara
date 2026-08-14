-- Packs de produits : un vendeur regroupe plusieurs de ses produits déjà
-- publiés dans un article unique vendu à un prix fixé pour le pack.
CREATE TABLE product_bundles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vendor_id UUID NOT NULL REFERENCES users(id),
  title TEXT NOT NULL,
  description TEXT,
  price_cfa INTEGER NOT NULL,
  cover_image_key TEXT,
  moderation_status TEXT NOT NULL DEFAULT 'approved'
    CHECK (moderation_status IN ('pending', 'approved', 'rejected')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE product_bundle_items (
  bundle_id UUID NOT NULL REFERENCES product_bundles(id) ON DELETE CASCADE,
  product_id UUID NOT NULL REFERENCES products(id),
  PRIMARY KEY (bundle_id, product_id)
);

CREATE INDEX idx_bundles_vendor ON product_bundles(vendor_id);
CREATE INDEX idx_bundle_items_product ON product_bundle_items(product_id);

-- Note : le paiement/livraison d'un pack n'est pas encore branché sur la
-- table `sales` (product_id y est NOT NULL et le reste du pipeline —
-- versements, livraison, affiliation — suppose un produit unique). Câbler
-- l'achat d'un pack nécessite une décision de modélisation à part (voir
-- discussion) plutôt qu'un changement précipité de ce schéma central.
