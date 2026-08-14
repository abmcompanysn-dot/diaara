-- Prix modulable ("paye ce que tu veux") : le vendeur active price_mode
-- 'flexible' et fixe un prix minimum ; l'acheteur choisit alors librement le
-- montant (>= min_price_cfa) sur la fiche produit. Pour price_mode 'fixed'
-- (défaut, comportement inchangé), le prix reste product.price_cfa et le
-- serveur ignore tout montant envoyé par le client.
ALTER TABLE products ADD COLUMN price_mode TEXT NOT NULL DEFAULT 'fixed' CHECK (price_mode IN ('fixed', 'flexible'));
ALTER TABLE products ADD COLUMN min_price_cfa INTEGER;
