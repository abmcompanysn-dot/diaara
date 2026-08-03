'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
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
        <p className="text-destructive">{error}</p>
        <Button variant="link" render={<Link href="/catalog" />}>
          Retour au catalogue
        </Button>
      </main>
    );
  }

  const formatPrice = (price: number) => `${price.toLocaleString()} FCFA`;

  return (
    <main className="min-h-screen p-8 max-w-3xl mx-auto">
      <nav className="mb-6">
        <Button variant="outline" size="sm" render={<Link href="/catalog" />}>
          ← Catalogue
        </Button>
      </nav>

      <div className="border rounded-lg p-8">
        <div className="flex justify-between items-start mb-4">
          <h1 className="text-3xl font-bold">{product.title}</h1>
          <span className="text-2xl text-primary font-bold">{formatPrice(product.price_cfa)}</span>
        </div>

        <Badge className="mb-4">{product.category}</Badge>

        <p className="text-muted-foreground mb-8 whitespace-pre-line">
          {product.description || 'Pas de description pour ce produit.'}
        </p>

        {error && <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded">{error}</div>}

        <Button
          onClick={handleBuy}
          disabled={buying}
          className="w-full h-11 font-semibold"
        >
          {buying ? 'Redirection vers le paiement...' : 'Acheter maintenant'}
        </Button>

        <p className="mt-4 text-sm text-green-900/50 text-center">
          Paiement sécurisé par mobile money (Wave, Orange Money, MTN MoMo)
        </p>
      </div>
    </main>
  );
}
