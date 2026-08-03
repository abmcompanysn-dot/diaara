'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

interface Product {
  id: string;
  title: string;
  description: string;
  price_cfa: number;
  category: string;
  vendor_id: string;
  moderation_status: string;
  created_at: string;
}

export default function AdminProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadProducts();
  }, []);

  const loadProducts = async () => {
    setLoading(true);
    try {
      const result = await api.getPendingProducts();
      setProducts(result.products);
    } catch (err: any) {
      setError(err.message || 'Impossible de charger les produits');
    } finally {
      setLoading(false);
    }
  };

  const handleModerate = async (id: string, status: string) => {
    try {
      await api.moderateProduct(id, { status });
      setProducts(products.filter((p) => p.id !== id));
    } catch (err: any) {
      alert(err.message || 'Action échouée');
    }
  };

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-5xl mx-auto">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Modération</h1>
          <p className="text-green-700">Produits en attente de validation</p>
        </div>
        <Button variant="outline" size="sm" render={<Link href="/admin" />}>
          ← Dashboard
        </Button>
      </header>

      {error && <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded">{error}</div>}

      {products.length === 0 ? (
        <div className="text-center py-12 border rounded-lg">
          <p className="text-green-900/50">Aucun produit en attente de modération.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {products.map((product) => (
            <div key={product.id} className="p-6 border rounded-lg">
              <div className="flex justify-between items-start mb-2">
                <h2 className="font-semibold text-lg">{product.title}</h2>
                <span className="text-primary font-bold">
                  {product.price_cfa.toLocaleString()} FCFA
                </span>
              </div>
              <p className="text-muted-foreground text-sm mb-2">
                {product.description || 'Pas de description'}
              </p>
              <div className="flex items-center gap-4 text-sm text-muted-foreground mb-4">
                <Badge variant="outline">{product.category}</Badge>
                <span>Vendeur : {product.vendor_id.slice(0, 8)}...</span>
                <span>{new Date(product.created_at).toLocaleDateString('fr-FR')}</span>
              </div>
              <div className="flex gap-3">
                <Button onClick={() => handleModerate(product.id, 'approved')}>
                  Approuver
                </Button>
                <Button
                  variant="destructive"
                  onClick={() => handleModerate(product.id, 'rejected')}
                >
                  Refuser
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </main>
  );
}
