-- KPay utilise son propre identifiant (retourné à l'initiation) pour les
-- appels GET statut / remboursement ultérieurs — contrairement à PawaPay,
-- où depositId est généré côté DIARRA et sert déjà de clé (sales.payment_reference).
-- Reste NULL pour les ventes PawaPay (non nécessaire).
ALTER TABLE sales ADD COLUMN provider_transaction_id TEXT;
CREATE UNIQUE INDEX idx_sales_provider_transaction_id ON sales (provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;
