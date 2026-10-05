-- L'inscription au DIARRA Summit devient payante (5000 FCFA, même tarif que
-- le produit "Tier Essentiel" déjà existant) — décision du 2026-10-05.
-- L'inscription est désormais créée au moment de la commande (SaleHandler.
-- Create), liée au checkout_token de la vente, avec statut "pending" tant
-- que le paiement n'est pas confirmé ; WebhookHandler.ConfirmPaidSale la
-- passe à "confirmed" et envoie l'email de confirmation à ce moment-là
-- (avant, Register créait l'inscription directement et gratuitement).

ALTER TABLE summit_registrations
  ADD COLUMN sale_checkout_token TEXT,
  ADD COLUMN status TEXT NOT NULL DEFAULT 'confirmed'
    CHECK (status IN ('pending', 'confirmed'));

-- Les inscriptions historiques (gratuites, avant ce changement) restent
-- "confirmed" par défaut — seules les nouvelles inscriptions payantes
-- démarrent "pending" (voir SaleHandler.Create).
CREATE UNIQUE INDEX idx_summit_registrations_checkout_token
  ON summit_registrations(sale_checkout_token) WHERE sale_checkout_token IS NOT NULL;
