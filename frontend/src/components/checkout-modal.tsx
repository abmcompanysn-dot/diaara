'use client';

import { useEffect, useRef, useState } from 'react';
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
import { XIcon, CheckIcon, AlertTriangleIcon } from '@/components/icons';

interface CheckoutModalProps {
  product: { id: string; title: string; price_cfa: number } | null;
  open: boolean;
  onClose: () => void;
}

type Step = 'form' | 'pending' | 'done' | 'error';

const formatPrice = (price: number) => `${price.toLocaleString()} FCFA`;

export default function CheckoutModal({ product, open, onClose }: CheckoutModalProps) {
  const [country, setCountry] = useState('SEN');
  const [operator, setOperator] = useState('');
  const [phone, setPhone] = useState('');
  const [email, setEmail] = useState('');
  const [step, setStep] = useState<Step>('form');
  const [orderToken, setOrderToken] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const modalRef = useRef<HTMLDivElement>(null);

  const countryConfig = CHECKOUT_COUNTRIES.find((c) => c.code === country) || CHECKOUT_COUNTRIES[0];
  const operators = countryConfig.operators;
  const guest = !isLoggedIn();

  // Réinitialise le formulaire à chaque ouverture.
  useEffect(() => {
    if (open) {
      setStep('form');
      setError('');
      setSubmitting(false);
      setPhone('');
      setEmail('');
      setOperator(operators[0]?.label || '');
      if (pollRef.current) {
        clearInterval(pollRef.current);
        pollRef.current = null;
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  // Ferme la modal avec Échap et bloque le scroll
  useEffect(() => {
    if (!open) return;
    document.body.style.overflow = 'hidden';
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.body.style.overflow = '';
      document.removeEventListener('keydown', handleEscape);
    };
  }, [open, onClose]);

  const handleCountryChange = (code: string | null) => {
    const c = code || 'SEN';
    setCountry(c);
    const ops = CHECKOUT_COUNTRIES.find((x) => x.code === c)?.operators || [];
    setOperator(ops[0]?.label || '');
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
      setOrderToken(token);
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

  if (!open || !product) return null;

  return (
    <div
      ref={modalRef}
      role="dialog"
      aria-modal="true"
      aria-labelledby="checkout-title"
      className="fixed inset-0 z-[100] flex items-center justify-center p-4"
    >
      <div className="absolute inset-0 bg-green-950/60 backdrop-blur-sm" onClick={onClose} aria-hidden />
      <div className="relative w-full max-w-md rounded-2xl bg-white shadow-lift border border-green-900/10 overflow-hidden">
        <div className="gradient-green text-white px-6 py-5 relative overflow-hidden">
          <div className="wax-pattern absolute inset-0" aria-hidden />
          <div className="relative flex justify-between items-start">
            <div>
              <p className="font-mono text-[11px] text-green-300 uppercase tracking-widest">// paiement sécurisé</p>
              <h2 id="checkout-title" className="font-display text-xl font-bold mt-1">{product.title}</h2>
              <p className="mt-1 font-mono text-lime font-semibold">{formatPrice(product.price_cfa)}</p>
            </div>
            <button
              onClick={onClose}
              className="text-white/70 hover:text-white transition-colors"
              aria-label="Fermer"
            >
              <XIcon size={24} />
            </button>
          </div>
        </div>

        <div className="p-6">
          {step === 'form' && (
            <div className="space-y-4">
              <p className="text-sm text-green-900/60">
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
              <h3 className="font-display text-lg font-bold mt-4 text-green-950">
                Confirmez sur votre téléphone
              </h3>
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
              <h3 className="font-display text-lg font-bold mt-4 text-green-950">
                Paiement confirmé
              </h3>
              <p className="mt-2 text-sm text-green-900/60 max-w-xs mx-auto">
                Merci ! Votre commande est en cours de traitement. La livraison vous sera envoyée
                par email.
              </p>
              <Button
                onClick={onClose}
                className="mt-6 bg-lime text-green-950 font-semibold hover:bg-green-300"
              >
                Continuer
              </Button>
            </div>
          )}

          {step === 'error' && (
            <div className="text-center py-8">
              <div className="mx-auto w-14 h-14 rounded-full bg-red-50 flex items-center justify-center">
                <AlertTriangleIcon size={32} className="text-red-600" />
              </div>
              <h3 className="font-display text-lg font-bold mt-4 text-green-950">
                Paiement non abouti
              </h3>
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
      </div>
    </div>
  );
}
