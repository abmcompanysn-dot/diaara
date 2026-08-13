'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ProductImage } from '@/components/product-image';
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
  category: string;
  cover_image_key?: string;
}

const formatPrice = (price: number) => `${price.toLocaleString()} FCFA`;

export default function CheckoutView() {
  const searchParams = useSearchParams();
  const productId = searchParams.get('product') || '';
  const { user } = useAuth();

  const [product, setProduct] = useState<Product | null>(null);
  const [loadingProduct, setLoadingProduct] = useState(true);
  const [productError, setProductError] = useState('');

  const [name, setName] = useState('');
  const [country, setCountry] = useState('SEN');
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const guest = !isLoggedIn();
  const selectedCountry = CHECKOUT_COUNTRIES.find((c) => c.code === country);

  useEffect(() => {
    if (user?.email && guest === false) setEmail(user.email);
  }, [user, guest]);

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
      {/* Bandeau réduit : focus direct sur le formulaire, retour rapide vers le produit */}
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-2xl mx-auto px-4 py-4 flex items-center justify-between gap-3">
          <Link
            href={`/product?id=${product.id}`}
            className="inline-flex items-center gap-2 font-mono text-sm text-white/70 hover:text-lime transition-colors shrink-0"
          >
            <ArrowLeftIcon size={16} />
            Retour
          </Link>
          <span className="inline-flex items-center gap-1.5 text-xs text-lime font-medium shrink-0">
            <LockIcon size={13} />
            Sécurisé
          </span>
        </div>
      </section>

      <section className="max-w-2xl mx-auto px-4 py-6 sm:py-10">
        {/* Résumé de commande : miniature + titre + total, rappel constant de l'achat */}
        <div className="flex items-center gap-3 bg-white rounded-xl shadow-card border border-green-900/5 p-3 mb-4">
          <div className="w-14 h-14 rounded-lg overflow-hidden shrink-0">
            <ProductImage product={product} className="h-14" />
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium text-green-950 truncate">{product.title}</p>
            <p className="text-xs text-green-900/50 mt-0.5">
              Total à payer :{' '}
              <span className="font-mono font-semibold text-green-700">
                {formatPrice(product.price_cfa)}
              </span>
            </p>
          </div>
        </div>

        <div className="bg-white rounded-xl shadow-card border border-green-900/5 p-6 sm:p-8">
          <div className="space-y-4">
            <p className="font-display font-bold text-green-950 flex items-center gap-2">
              👤 Vos informations de livraison
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
              <Label htmlFor="cc-email">
                Adresse e-mail (pour recevoir votre produit) {guest && '*'}
              </Label>
              <Input
                id="cc-email"
                type="email"
                placeholder="vous@exemple.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={!guest}
                className="bg-white"
              />
              <p className="text-xs text-green-900/50">
                {guest
                  ? 'Pas besoin de compte : vous recevrez votre commande sur cet email.'
                  : 'Votre lien de téléchargement sera envoyé à cette adresse.'}
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="cc-country">Pays de facturation</Label>
              <Select value={country} onValueChange={(v) => setCountry(v || 'SEN')}>
                <SelectTrigger id="cc-country" className="bg-white w-full">
                  <SelectValue placeholder="Choisir le pays">
                    {selectedCountry && (
                      <span className="flex items-center gap-2">
                        <span>{selectedCountry.flag}</span>
                        <span>
                          {selectedCountry.name} ({selectedCountry.currency})
                        </span>
                      </span>
                    )}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {CHECKOUT_COUNTRIES.map((c) => (
                    <SelectItem key={c.code} value={c.code}>
                      <span className="flex items-center gap-2">
                        <span>{c.flag}</span>
                        <span>
                          {c.name} ({c.currency})
                        </span>
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-green-900/50">
                Le montant est converti automatiquement dans la devise de votre pays.
              </p>
            </div>

            {error && (
              <div className="p-3 bg-red-50 text-red-700 rounded text-sm" role="alert">
                {error}
              </div>
            )}

            <Button
              onClick={handleSubmit}
              disabled={submitting || !name || (guest && !email)}
              className="w-full h-12 font-semibold bg-green-950 text-white hover:bg-green-900 text-base gap-2"
            >
              <LockIcon size={16} />
              {submitting ? 'Redirection vers PawaPay...' : `Payer ${formatPrice(product.price_cfa)}`}
            </Button>

            <div className="flex flex-col items-center gap-2.5 pt-1">
              <p className="text-[11px] text-center text-green-900/40">
                ✅ Redirection sécurisée vers PawaPay · Orange Money, Wave, MTN MoMo, Moov Money…
              </p>
              <div className="flex flex-wrap justify-center gap-2">
                <img src="/payments/orange-money.png" alt="Orange Money" className="h-6 object-contain" />
                <img src="/payments/mtn-momo.png" alt="MTN MoMo" className="h-6 object-contain" />
                <img src="/payments/moov-money.png" alt="Moov Money" className="h-6 object-contain" />
                <span className="h-6 px-2.5 inline-flex items-center rounded bg-green-400 text-green-950 text-xs font-bold">
                  Wave
                </span>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
