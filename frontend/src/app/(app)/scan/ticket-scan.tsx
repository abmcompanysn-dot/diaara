'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { api } from '@/lib/api';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { CheckIcon, AlertTriangleIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

// Pas de scanner caméra embarqué : le QR du billet encode directement cette
// URL (?token=...), donc n'importe quel appareil photo natif (déjà présent
// sur tout smartphone) l'ouvre directement ici en scannant — inutile de
// réimplémenter un décodeur QR côté web.
interface ScanResult {
  buyer_name: string;
  event_title: string;
  offer_title: string;
  already_used: boolean;
  checked_in_at: string | null;
}

export default function TicketScan() {
  const searchParams = useSearchParams();
  const token = searchParams.get('token') || '';

  const [loading, setLoading] = useState(true);
  const [confirming, setConfirming] = useState(false);
  const [result, setResult] = useState<ScanResult | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!token) {
      setError('Aucun billet à vérifier (lien invalide).');
      setLoading(false);
      return;
    }
    api
      .getTicketScanStatus(token)
      .then(setResult)
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, [token]);

  const handleConfirm = async () => {
    setConfirming(true);
    setError('');
    try {
      const res = await api.confirmTicketScan(token);
      setResult(res);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setConfirming(false);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// vérification billet" title="Scan du billet" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader eyebrow="// vérification billet" title="Scan du billet" />

      <section className="max-w-md mx-auto px-4 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {result && (
          <div className="rounded-2xl border border-green-900/10 bg-white shadow-lift p-6 text-center">
            {result.already_used ? (
              <>
                <span className="mx-auto w-14 h-14 rounded-full bg-amber-100 text-amber-700 flex items-center justify-center mb-4">
                  <AlertTriangleIcon size={26} />
                </span>
                <p className="font-display text-lg font-bold text-green-950">Déjà scanné</p>
                {result.checked_in_at && (
                  <p className="text-xs text-green-900/50 mt-1">
                    Entrée enregistrée le {new Date(result.checked_in_at).toLocaleString('fr-FR')}
                  </p>
                )}
              </>
            ) : (
              <>
                <span className="mx-auto w-14 h-14 rounded-full bg-green-100 text-green-700 flex items-center justify-center mb-4">
                  <CheckIcon size={26} />
                </span>
                <p className="font-display text-lg font-bold text-green-950">Billet valide</p>
              </>
            )}

            <div className="mt-4 text-left space-y-1 border-t border-green-900/10 pt-4">
              <p className="text-sm"><span className="text-green-900/50">Événement :</span> {result.event_title}</p>
              <p className="text-sm"><span className="text-green-900/50">Offre :</span> {result.offer_title}</p>
              <p className="text-sm"><span className="text-green-900/50">Acheteur :</span> {result.buyer_name}</p>
            </div>

            {!result.already_used && (
              <button
                type="button"
                onClick={handleConfirm}
                disabled={confirming}
                className="w-full h-11 mt-6 rounded-md bg-[#0E6B46] text-white text-sm font-semibold hover:bg-[#0c5a3c] transition-colors disabled:opacity-60"
              >
                {confirming ? 'Confirmation...' : "Confirmer l'entrée"}
              </button>
            )}
          </div>
        )}
      </section>
    </main>
  );
}
