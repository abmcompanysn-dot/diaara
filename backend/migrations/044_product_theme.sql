-- Thème (sujet du contenu) d'un produit, distinct de category (type
-- technique de fichier : ebook/pdf/service/...). Permet au catalogue de
-- filtrer par sujet plutôt que par simple format de fichier.
ALTER TABLE products ADD COLUMN theme TEXT;

CREATE INDEX idx_products_theme ON products (theme);
