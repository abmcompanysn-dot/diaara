'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';

interface Product {
  id: string;
  title: string;
  price_cfa: number;
  category: string;
  moderation_status: string;
}

export default function VendorProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadProducts();
  }, []);

  const loadProducts = async () => {
    setLoading(true);
    try {
      const result = await api.getVendorProducts();
      setProducts(result.products);
    } catch (err: any) {
      setError(err.message || 'Impossible de charger les produits');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Supprimer ce produit ?')) return;
    try {
      await api.deleteProduct(id);
      setProducts(products.filter((p) => p.id !== id));
    } catch (err: any) {
      alert(err.message || 'Suppression échouée');
    }
  };

  const statusBadge = (status: string) => {
    const colors: Record<string, string> = {
      pending: 'bg-yellow-100 text-yellow-700',
      approved: 'bg-green-100 text-green-700',
      rejected: 'bg-red-100 text-red-700',
    };
    const labels: Record<string, string> = {
      pending: 'En attente',
      approved: 'Approuvé',
      rejected: 'Refusé',
    };
    return (
      <span className={`px-2 py-1 rounded text-xs ${colors[status] || 'bg-green-100'}`}>
        {labels[status] || status}
      </span>
    );
  };

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-4xl mx-auto">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Mes produits</h1>
          <p className="text-green-700">{products.length} produit(s) en boutique</p>
        </div>
        <div className="flex gap-3">
          <Link href="/vendor/earnings" className="px-4 py-2 border rounded hover:bg-green-50">
            Revenus
          </Link>
          <Link href="/vendor/products/new" className="px-4 py-2 gradient-green text-white rounded">
            + Nouveau produit
          </Link>
        </div>
      </header>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-600 rounded">{error}</div>}

      {products.length === 0 ? (
        <div className="text-center py-12 border rounded-lg">
          <p className="text-green-900/50 mb-4">Aucun produit pour le moment.</p>
          <Link href="/vendor/products/new" className="text-green-600">
            Déposer mon premier produit
          </Link>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border rounded">
            <thead className="bg-green-50">
              <tr>
                <th className="p-3 text-left">Produit</th>
                <th className="p-3 text-left">Prix</th>
                <th className="p-3 text-left">Catégorie</th>
                <th className="p-3 text-left">Statut</th>
                <th className="p-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {products.map((product) => (
                <tr key={product.id} className="border-t">
                  <td className="p-3">
                    <Link href={`/product?id=${product.id}`} className="font-medium text-green-600">
                      {product.title}
                    </Link>
                  </td>
                  <td className="p-3">{product.price_cfa.toLocaleString()} FCFA</td>
                  <td className="p-3">{product.category}</td>
                  <td className="p-3">{statusBadge(product.moderation_status)}</td>
                  <td className="p-3 text-right">
                    <button
                      onClick={() => handleDelete(product.id)}
                      className="text-red-600 hover:underline text-sm"
                    >
                      Supprimer
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </main>
  );
}
