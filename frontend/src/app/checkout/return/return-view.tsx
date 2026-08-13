'use client';

import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { CheckIcon, AlertTriangleIcon } from '@/components/icons';

type Step = 'pending' | 'done' | 'error';

export default function CheckoutReturnView() {
  const searchParams = useSearchParams();
  const token = searchParams.get('token') || '';

  const [step, setStep] = useState<Step>('pending');
  const [error, setError] = useState('');
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (!token) {
      setError('Commande introuvable.');
      setStep('error');
      return;
    }

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

    return () => stopPolling();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const stopPolling = () => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  };

  return (
    <main className="bg-paper min-h-[calc(100vh-4rem)] flex items-center justify-center px-4">
      <div className="w-full max-w-md bg-white rounded-xl shadow-card border border-green-900/5 p-6 sm:p-8">
        {step === 'pending' && (
          <div className="text-center py-8">
            <div className="mx-auto w-12 h-12 rounded-full border-4 border-green-100 border-t-lime animate-spin" />
            <h1 className="font-display text-lg font-bold mt-4 text-green-950">
              Vérification du paiement
            </h1>
            <p className="mt-2 text-sm text-green-900/60 max-w-xs mx-auto">
              Merci de patienter pendant que nous confirmons votre paiement auprès
              de PawaPay.
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
            <h1 className="font-display text-lg font-bold mt-4 text-green-950">
              Paiement confirmé
            </h1>
            <p className="mt-2 text-sm text-green-900/60 max-w-xs mx-auto">
              Merci ! Votre commande est en cours de traitement. La livraison vous
              sera envoyée par email.
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
            <h1 className="font-display text-lg font-bold mt-4 text-green-950">
              Paiement non abouti
            </h1>
            <p className="mt-2 text-sm text-green-900/60 max-w-xs mx-auto">{error}</p>
            <Button
              className="mt-6 bg-lime text-green-950 font-semibold hover:bg-green-300"
              render={<Link href="/catalog" />}
            >
              Retour au catalogue
            </Button>
          </div>
        )}
      </div>
    </main>
  );
}
