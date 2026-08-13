'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { CHECKOUT_COUNTRIES, isLoggedIn } from '@/lib/operators';
import { friendlyError } from '@/lib/error-messages';
import { ArrowLeftIcon, LockIcon } from '@/components/icons';

interface Product {
  id: string;
  title: string;
  price_cfa: number;
}

const formatPrice = (price: number) => `${price.toLocaleString()} FCFA`;

export default function CheckoutView() {
  const searchParams = useSearchParams();
  const productId = searchParams.get('product') || '';

  const [product, setProduct] = useState<Product | null>(null);
  const [loadingProduct, setLoadingProduct] = useState(true);
  const [productError, setProductError] = useState('');

  const [name, setName] = useState('');
  const [country, setCountry] = useState('SEN');
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const guest = !isLoggedIn();

  useEffect(() => {
    if (!productId) {
      setProductError('Produit introuvable');
      setLoadingProduct(false);
      return;
    }
    api
      .getProduct(productId)
      .then((res) => {
        setProduct(res.product);
        setLoadingProduct(false);
      })
      .catch(() => {
        setProductError('Produit introuvable');
        setLoadingProduct(false);
      });
  }, [productId]);

  const handleSubmit = async () => {
    if (!product) return;
    setSubmitting(true);
    setError('');
    try {
      const result = await api.createOrder({
        product_id: product.id,
        buyer_name: name,
        country,
        ...(guest ? { buyer_email: email } : {}),
      });
      const redirectUrl = result.checkout?.redirect_url;
      if (!redirectUrl) throw new Error('redirect_url_missing');
      window.location.href = redirectUrl;
    } catch (err: any) {
      setError(friendlyError(err));
      setSubmitting(false);
    }
  };

  if (loadingProduct) {
    return (
      <main className="min-h-[calc(100vh-4rem)] flex items-center justify-center">
        <p className="font-mono text-green-900/50">chargement…</p>
      </main>
    );
  }

  if (!product) {
    return (
      <main className="min-h-[calc(100vh-4rem)] flex flex-col items-center justify-center gap-4">
        <p className="text-destructive font-semibold">{productError}</p>
        <Button variant="outline" size="sm" render={<Link href="/catalog" />}>
          <ArrowLeftIcon size={16} className="mr-2" />
          Retour au catalogue
        </Button>
      </main>
    );
  }

  return (
    <main className="bg-paper min-h-[calc(100vh-4rem)]">
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-2xl mx-auto px-4 py-10">
          <Link
            href={`/product?id=${product.id}`}
            className="inline-flex items-center gap-2 font-mono text-sm text-white/60 hover:text-lime transition-colors"
          >
            <ArrowLeftIcon size={16} />
            {product.title}
          </Link>
          <p className="font-mono text-sm text-green-300 mt-6 uppercase tracking-widest">
            // paiement sécurisé
          </p>
          <h1 className="font-display text-2xl sm:text-3xl font-bold tracking-tight mt-2">
            {product.title}
          </h1>
          <p className="mt-2 font-mono text-lime font-semibold text-lg">{formatPrice(product.price_cfa)}</p>
        </div>
      </section>

      <section className="max-w-2xl mx-auto px-4 py-10">
        <div className="bg-white rounded-xl shadow-card border border-green-900/5 p-6 sm:p-8">
          <div className="space-y-4">
            <p className="text-sm text-green-900/60 flex items-start gap-2">
              <LockIcon size={16} className="mt-0.5 shrink-0 text-green-600" />
              Vous serez redirigé vers PawaPay pour choisir votre opérateur mobile
              money (Orange Money, Wave, MTN MoMo, Moov Money…) et confirmer le
              paiement.
            </p>

            <div className="space-y-2">
              <Label htmlFor="cc-name">Nom complet</Label>
              <Input
                id="cc-name"
                type="text"
                placeholder="Ex : Awa Diarra"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="bg-white"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="cc-country">Pays</Label>
              <Select value={country} onValueChange={(v) => setCountry(v || 'SEN')}>
                <SelectTrigger className="bg-white">
                  <SelectValue placeholder="Choisir le pays" />
                </SelectTrigger>
                <SelectContent>
                  {CHECKOUT_COUNTRIES.map((c) => (
                    <SelectItem key={c.code} value={c.code}>
                      {c.name} ({c.currency})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-green-900/50">
                Le montant est converti automatiquement dans la devise de votre pays.
              </p>
            </div>

            {guest && (
              <div className="space-y-2">
                <Label htmlFor="cc-email">Votre email (livraison)</Label>
                <Input
                  id="cc-email"
                  type="email"
                  placeholder="vous@exemple.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="bg-white"
                />
                <p className="text-xs text-green-900/50">
                  Pas besoin de compte : vous recevrez votre commande sur cet email.
                </p>
              </div>
            )}

            {error && (
              <div className="p-3 bg-red-50 text-red-700 rounded text-sm" role="alert">
                {error}
              </div>
            )}

            <Button
              onClick={handleSubmit}
              disabled={submitting || !name || (guest && !email)}
              className="w-full h-11 font-semibold bg-lime text-green-950 hover:bg-green-300"
            >
              {submitting ? 'Redirection vers PawaPay...' : `Payer ${formatPrice(product.price_cfa)}`}
            </Button>
            <p className="text-[11px] text-center text-green-900/40">
              Paiement traité par PawaPay · Orange Money, Wave, MTN MoMo, Moov Money…
            </p>
          </div>
        </div>
      </section>
    </main>
  );
}
