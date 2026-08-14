-- Le pays de l'acheteur était déjà collecté à la commande (CreateOrderInput)
-- mais jamais enregistré sur la vente — nécessaire pour que le vendeur voit
-- d'où vient son client.
ALTER TABLE sales ADD COLUMN country TEXT;
