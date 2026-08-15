'use client';

import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { friendlyError } from '@/lib/error-messages';
import { CheckIcon, AlertTriangleIcon, DownloadIcon } from '@/components/icons';
import { ProductImage } from '@/components/product-image';

type Step = 'pending' | 'done' | 'error';

interface PurchasedProduct {
  id: string;
  title: string;
  category: string;
  cover_image_key?: string;
}

const POLL_INTERVAL_MS = 3000;
const MAX_AUTO_POLLS = 20; // ~60s avant de proposer une vérification manuelle

export default function CheckoutReturnView() {
  const searchParams = useSearchParams();
  const token = searchParams.get('token') || '';

  const [step, setStep] = useState<Step>('pending');
  const [error, setError] = useState('');
  const [deliveryUrl, setDeliveryUrl] = useState('');
  const [delivering, setDelivering] = useState(false);
  const [productId, setProductId] = useState('');
  const [product, setProduct] = useState<PurchasedProduct | null>(null);
  const [timedOut, setTimedOut] = useState(false);
  const [checkingNow, setCheckingNow] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollCountRef = useRef(0);

  const handleDownload = async () => {
    setDelivering(true);
    try {
      const result = await api.getDeliveryByToken(token);
      setDeliveryUrl(result.signed_url);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setDelivering(false);
    }
  };

  useEffect(() => {
    if (!token) {
      setError('Commande introuvable.');
      setStep('error');
      return;
    }

    const check = async (manual?: boolean) => {
      try {
        const res = await api.getCheckoutStatus(token);
        const status = res.order?.status;
        if (res.order?.product_id) setProductId(res.order.product_id);
        if (status === 'paid') {
          stopPolling();
          setStep('done');
        } else if (status === 'failed') {
          stopPolling();
          setError('Le paiement a échoué. Vérifiez votre solde et réessayez.');
          setStep('error');
        } else if (!manual) {
          pollCountRef.current += 1;
          if (pollCountRef.current >= MAX_AUTO_POLLS) {
            stopPolling();
            setTimedOut(true);
          }
        }
      } catch {
        // Le polling continue tant que la commande n'a pas de statut final.
      }
    };
    pollRef.current = setInterval(check, POLL_INTERVAL_MS);
    check();

    return () => stopPolling();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  // Dès que la commande révèle son produit (paiement confirmé ou échoué),
  // on récupère sa fiche pour afficher son image sur cet écran.
  useEffect(() => {
    if (!productId) return;
    api
      .getProduct(productId)
      .then((res) => setProduct(res.product))
      .catch(() => {
        // Pas bloquant : l'écran reste utilisable sans l'image.
      });
  }, [productId]);

  const stopPolling = () => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  };

  const handleManualCheck = async () => {
    setCheckingNow(true);
    try {
      const res = await api.getCheckoutStatus(token);
      const status = res.order?.status;
      if (res.order?.product_id) setProductId(res.order.product_id);
      if (status === 'paid') {
        setStep('done');
      } else if (status === 'failed') {
        setError('Le paiement a échoué. Vérifiez votre solde et réessayez.');
        setStep('error');
      }
    } catch {
      // Rien de nouveau, l'utilisateur reste sur l'écran d'attente.
    } finally {
      setCheckingNow(false);
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

            {timedOut && (
              <div className="mt-6 pt-6 border-t border-green-900/10">
                <p className="text-sm text-green-900/70">
                  Ça prend plus de temps que prévu. Votre paiement peut quand même
                  se confirmer — vous recevrez un email dès que ce sera fait.
                </p>
                <div className="mt-4 flex flex-col gap-2">
                  <Button className="h-10" variant="outline" onClick={handleManualCheck} disabled={checkingNow}>
                    {checkingNow ? 'Vérification...' : 'Vérifier maintenant'}
                  </Button>
                  <div className="flex justify-center gap-4 text-sm">
                    <Link href="/orders" className="text-green-700 underline">
                      Mes commandes
                    </Link>
                    <Link href="/support" className="text-green-700 underline">
                      Contacter le support
                    </Link>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {step === 'done' && (
          <div className="text-center py-8">
            {product ? (
              <div className="mx-auto w-20 h-20 rounded-xl overflow-hidden shadow-card">
                <ProductImage product={product} className="h-20" />
              </div>
            ) : (
              <div className="mx-auto w-14 h-14 rounded-full bg-lime/20 flex items-center justify-center">
                <CheckIcon size={32} className="text-green-700" />
              </div>
            )}
            <h1 className="font-display text-lg font-bold mt-4 text-green-950">
              Paiement confirmé
            </h1>
            {product && (
              <p className="mt-1 text-sm font-medium text-green-900/80 max-w-xs mx-auto line-clamp-2">
                {product.title}
              </p>
            )}
            <p className="mt-2 text-sm text-green-900/60 max-w-xs mx-auto">
              Merci ! Un email de confirmation vous a été envoyé. Vous pouvez
              aussi télécharger votre fichier directement ci-dessous.
            </p>

            {error && (
              <div className="mt-4 p-3 bg-red-50 text-red-700 rounded text-sm text-left" role="alert">
                {error}
              </div>
            )}

            <div className="mt-6">
              {deliveryUrl ? (
                <Button
                  render={
                    <a href={deliveryUrl} download className="block w-full">
                      <DownloadIcon size={18} className="mr-2" />
                      Télécharger mon fichier
                    </a>
                  }
                  className="w-full h-11 font-semibold bg-lime text-green-950 hover:bg-green-300"
                />
              ) : (
                <Button
                  onClick={handleDownload}
                  disabled={delivering}
                  className="w-full h-11 font-semibold bg-lime text-green-950 hover:bg-green-300"
                >
                  <DownloadIcon size={18} className="mr-2" />
                  {delivering ? 'Préparation du téléchargement…' : 'Télécharger mon fichier'}
                </Button>
              )}
              <p className="mt-2 text-xs text-green-900/40 text-center">
                Lien valable 5 minutes · 3 téléchargements maximum
              </p>
            </div>

            <div className="mt-4 flex flex-wrap justify-center gap-3">
              <Button className="h-10" variant="outline" render={<Link href="/catalog" />}>
                Continuer mes achats
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
            <div className="mt-6 flex flex-col items-center gap-2">
              {productId && (
                <Button
                  className="h-10 w-full max-w-[220px] bg-lime text-green-950 font-semibold hover:bg-green-300"
                  render={<Link href={`/product?id=${productId}`} />}
                >
                  Réessayer le paiement
                </Button>
              )}
              <Button variant="outline" className="h-10 w-full max-w-[220px]" render={<Link href="/catalog" />}>
                Retour au catalogue
              </Button>
              <Link href="/support" className="text-sm text-green-700 underline mt-1">
                Contacter le support
              </Link>
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
