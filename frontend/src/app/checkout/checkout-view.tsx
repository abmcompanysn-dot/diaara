'use client';

import { useEffect, useRef, useState } from 'react';
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
import { ArrowLeftIcon, CheckIcon, AlertTriangleIcon, LockIcon } from '@/components/icons';

interface Product {
  id: string;
  title: string;
  price_cfa: number;
}

type Step = 'form' | 'pending' | 'done' | 'error';

const formatPrice = (price: number) => `${price.toLocaleString()} FCFA`;

export default function CheckoutView() {
  const searchParams = useSearchParams();
  const productId = searchParams.get('product') || '';

  const [product, setProduct] = useState<Product | null>(null);
  const [loadingProduct, setLoadingProduct] = useState(true);
  const [productError, setProductError] = useState('');

  const [country, setCountry] = useState('SEN');
  const [operator, setOperator] = useState('');
  const [phone, setPhone] = useState('');
  const [email, setEmail] = useState('');
  const [step, setStep] = useState<Step>('form');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const countryConfig = CHECKOUT_COUNTRIES.find((c) => c.code === country) || CHECKOUT_COUNTRIES[0];
  const operators = countryConfig.operators;
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

  useEffect(() => {
    setOperator(operators[0]?.label || '');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [country]);

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const handleCountryChange = (code: string | null) => {
    setCountry(code || 'SEN');
  };

  const handleSubmit = async () => {
    if (!product) return;
    setSubmitting(true);
    setError('');
    try {
      const result = await api.createOrder({
        product_id: product.id,
        ...(guest ? { buyer_email: email } : {}),
        phone,
        operator,
        country,
      });
      const token = result.order?.checkout_token;
      if (!token) throw new Error('checkout_token_missing');
      setStep('pending');
      startPolling(token);
    } catch (err: any) {
      setError(err.message || "Échec de l'achat");
      setStep('error');
      setSubmitting(false);
    }
  };

  const startPolling = (token: string) => {
    const check = async () => {
      try {
        const res = await api.getCheckoutStatus(token);
        const status = res.order?.status;
        if (status === 'paid') {
          stopPolling();
          setStep('done');
        } else if (status === 'failed') {
          stopPolling();
          setError('Le paiement a échoué. Vérifiez votre solde et réessayez.');
          setStep('error');
        }
      } catch {
        // Le polling continue tant que la commande n'a pas de statut final.
      }
    };
    pollRef.current = setInterval(check, 3000);
    check();
  };

  const stopPolling = () => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
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
          {step === 'form' && (
            <div className="space-y-4">
              <p className="text-sm text-green-900/60 flex items-start gap-2">
                <LockIcon size={16} className="mt-0.5 shrink-0 text-green-600" />
                Payez par mobile money. Un message vous sera envoyé sur votre téléphone pour
                confirmer le paiement.
              </p>

              <div className="space-y-2">
                <Label htmlFor="cc-country">Pays</Label>
                <Select value={country} onValueChange={handleCountryChange}>
                  <SelectTrigger className="bg-white">
                    <SelectValue placeholder="Choisir le pays" />
                  </SelectTrigger>
                  <SelectContent>
                    {CHECKOUT_COUNTRIES.map((c) => (
                      <SelectItem key={c.code} value={c.code}>
                        {c.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="cc-operator">Opérateur mobile money</Label>
                <Select value={operator} onValueChange={(v) => setOperator(v || '')}>
                  <SelectTrigger className="bg-white">
                    <SelectValue placeholder="Choisir l'opérateur" />
                  </SelectTrigger>
                  <SelectContent>
                    {operators.map((op) => (
                      <SelectItem key={op.provider} value={op.label}>
                        {op.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="cc-phone">Numéro de téléphone</Label>
                <Input
                  id="cc-phone"
                  type="tel"
                  inputMode="numeric"
                  placeholder="Ex : 77 123 45 67"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  className="bg-white"
                />
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
                disabled={submitting || !phone || !operator || (guest && !email)}
                className="w-full h-11 font-semibold bg-lime text-green-950 hover:bg-green-300"
              >
                {submitting ? 'Initialisation du paiement...' : `Payer ${formatPrice(product.price_cfa)}`}
              </Button>
              <p className="text-[11px] text-center text-green-900/40">
                Paiement traité par PawaPay · Orange Money, Wave, MTN MoMo, Free
              </p>
            </div>
          )}

          {step === 'pending' && (
            <div className="text-center py-8">
              <div className="mx-auto w-12 h-12 rounded-full border-4 border-green-100 border-t-lime animate-spin" />
              <h2 className="font-display text-lg font-bold mt-4 text-green-950">
                Confirmez sur votre téléphone
              </h2>
              <p className="mt-2 text-sm text-green-900/60 max-w-xs mx-auto">
                Un message vous a été envoyé pour autoriser le paiement. Entrez votre code PIN
                lorsque vous le recevez.
              </p>
              <p className="mt-4 font-mono text-xs text-green-900/40">
                en attente de confirmation...
              </p>
            </div>
          )}

          {step === 'done' && (
            <div className="text-center py-8">
              <div className="mx-auto w-14 h-14 rounded-full bg-lime/20 flex items-center justify-center">
                <CheckIcon size={32} className="text-green-700" />
              </div>
              <h2 className="font-display text-lg font-bold mt-4 text-green-950">
                Paiement confirmé
              </h2>
              <p className="mt-2 text-sm text-green-900/60 max-w-xs mx-auto">
                Merci ! Votre commande est en cours de traitement. La livraison vous sera envoyée
                par email.
              </p>
              <div className="mt-6 flex flex-wrap justify-center gap-3">
                <Button className="bg-lime text-green-950 font-semibold hover:bg-green-300" render={<Link href="/catalog" />}>
                  Continuer mes achats
                </Button>
                <Button variant="outline" render={<Link href="/orders" />}>
                  Voir mes commandes
                </Button>
              </div>
            </div>
          )}

          {step === 'error' && (
            <div className="text-center py-8">
              <div className="mx-auto w-14 h-14 rounded-full bg-red-50 flex items-center justify-center">
                <AlertTriangleIcon size={32} className="text-red-600" />
              </div>
              <h2 className="font-display text-lg font-bold mt-4 text-green-950">
                Paiement non abouti
              </h2>
              <p className="mt-2 text-sm text-green-900/60 max-w-xs mx-auto">{error}</p>
              <Button
                onClick={() => setStep('form')}
                className="mt-6 bg-lime text-green-950 font-semibold hover:bg-green-300"
              >
                Réessayer
              </Button>
            </div>
          )}
        </div>
      </section>
    </main>
  );
}
