'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';

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
        <Link href="/admin" className="text-green-600">← Dashboard</Link>
      </header>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-600 rounded">{error}</div>}

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
                <span className="text-green-600 font-bold">{product.price_cfa.toLocaleString()} FCFA</span>
              </div>
              <p className="text-green-700 text-sm mb-2">
                {product.description || 'Pas de description'}
              </p>
              <div className="flex items-center gap-4 text-sm text-green-900/50 mb-4">
                <span>{product.category}</span>
                <span>Vendeur : {product.vendor_id.slice(0, 8)}...</span>
                <span>{new Date(product.created_at).toLocaleDateString('fr-FR')}</span>
              </div>
              <div className="flex gap-3">
                <button
                  onClick={() => handleModerate(product.id, 'approved')}
                  className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
                >
                  Approuver
                </button>
                <button
                  onClick={() => handleModerate(product.id, 'rejected')}
                  className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
                >
                  Refuser
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </main>
  );
}
