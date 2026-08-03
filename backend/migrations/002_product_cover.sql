-- Migration 002: Couverture d'image pour les produits
-- Clé objet dans le stockage (Tigris) ; servie via GET /api/products/{id}/cover.

ALTER TABLE products ADD COLUMN cover_image_key TEXT;
