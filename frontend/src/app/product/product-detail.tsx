'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';

interface Product {
  id: string;
  title: string;
  description: string;
  price_cfa: number;
  category: string;
  moderation_status: string;
}

export default function ProductDetailPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const id = searchParams.get('id') || '';
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [buying, setBuying] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (id) loadProduct();
  }, [id]);

  const loadProduct = async () => {
    try {
      const result = await api.getProduct(id);
      setProduct(result.product);
    } catch (err) {
      setError('Produit introuvable');
    } finally {
      setLoading(false);
    }
  };

  const handleBuy = async () => {
    setBuying(true);
    setError('');
    try {
      const result = await api.createOrder({ product_id: id });
      // Si un lien de paiement existe, rediriger
      if (result.payment_url) {
        window.location.href = result.payment_url;
      } else {
        router.push(`/order?id=${result.order.id}`);
      }
    } catch (err: any) {
      setError(err.message || 'Échec de l\'achat');
      setBuying(false);
    }
  };

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  if (!product) {
    return (
      <main className="p-8 text-center">
        <p className="text-red-600">{error}</p>
        <Link href="/catalog" className="text-green-600">Retour au catalogue</Link>
      </main>
    );
  }

  const formatPrice = (price: number) => `${price.toLocaleString()} FCFA`;

  return (
    <main className="min-h-screen p-8 max-w-3xl mx-auto">
      <nav className="mb-6">
        <Link href="/catalog" className="text-green-600">← Catalogue</Link>
      </nav>

      <div className="border rounded-lg p-8">
        <div className="flex justify-between items-start mb-4">
          <h1 className="text-3xl font-bold">{product.title}</h1>
          <span className="text-2xl text-green-600 font-bold">{formatPrice(product.price_cfa)}</span>
        </div>

        <span className="inline-block mb-4 px-2 py-1 bg-green-100 rounded text-sm text-green-800">
          {product.category}
        </span>

        <p className="text-green-800 mb-8 whitespace-pre-line">
          {product.description || 'Pas de description pour ce produit.'}
        </p>

        {error && <div className="mb-4 p-3 bg-red-50 text-red-600 rounded">{error}</div>}

        <button
          onClick={handleBuy}
          disabled={buying}
          className="w-full py-3 gradient-green text-white rounded-lg font-semibold hover:opacity-95 disabled:opacity-50"
        >
          {buying ? 'Redirection vers le paiement...' : 'Acheter maintenant'}
        </button>

        <p className="mt-4 text-sm text-green-900/50 text-center">
          Paiement sécurisé par mobile money (Wave, Orange Money, MTN MoMo)
        </p>
      </div>
    </main>
  );
}
