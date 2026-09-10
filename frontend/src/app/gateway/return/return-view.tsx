'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { CheckIcon } from '@/components/icons';

// Page de retour APRÈS un paiement passé par la passerelle de paiement
// (clients externes, ex. ABMCY Core) — pas le checkout DIARRA classique
// (voir /checkout/return). La transaction n'a ni produit ni livraison :
// le résultat réel est notifié au client externe par webhook signé. Ici on
// se contente de rassurer l'utilisateur et, si l'ouvreur a fourni une URL
// de retour, on l'y renvoie.
export default function GatewayReturnView() {
  const params = useSearchParams();
  const depositId = params.get('depositId') || '';
  const returnUrl = params.get('return_url') || params.get('returnUrl') || '';
  const [countdown, setCountdown] = useState(returnUrl ? 4 : 0);

  useEffect(() => {
    // 1) Si cette page a été ouverte dans une fenêtre par un widget/opener,
    //    on prévient l'ouvreur et on ferme.
    if (typeof window !== 'undefined' && window.opener && !window.opener.closed) {
      try {
        window.opener.postMessage(
          { abmcy_pay: true, gateway_return: true, deposit_id: depositId },
          '*',
        );
      } catch {
        /* origine différente : rien à faire */
      }
      const t = setTimeout(() => window.close(), 1500);
      return () => clearTimeout(t);
    }
    // 2) Sinon, si une URL de retour a été passée, on y redirige après un
    //    court décompte.
    if (returnUrl) {
      const iv = setInterval(() => setCountdown((c) => c - 1), 1000);
      const t = setTimeout(() => {
        window.location.href = returnUrl;
      }, 4000);
      return () => {
        clearInterval(iv);
        clearTimeout(t);
      };
    }
  }, [depositId, returnUrl]);

  return (
    <main className="bg-paper min-h-[calc(100vh-4rem)] flex items-center justify-center px-4">
      <div className="w-full max-w-md bg-white rounded-xl shadow-card border border-green-900/5 p-6 sm:p-8 text-center py-10">
        <div className="mx-auto w-14 h-14 rounded-full bg-lime/20 flex items-center justify-center">
          <CheckIcon size={32} className="text-green-700" />
        </div>
        <h1 className="font-display text-lg font-bold mt-4 text-green-950">
          Paiement pris en compte
        </h1>
        <p className="mt-2 text-sm text-green-900/60 max-w-xs mx-auto">
          Merci. Votre paiement est en cours de confirmation. Vous pouvez fermer
          cette fenêtre&nbsp;: le marchand sera notifié automatiquement.
        </p>

        {returnUrl && (
          <p className="mt-5 text-sm text-green-900/70">
            Retour vers le marchand
            {countdown > 0 ? ` dans ${countdown}s…` : '…'}
            <br />
            <a href={returnUrl} className="text-green-700 underline">
              Continuer maintenant
            </a>
          </p>
        )}

        {depositId && (
          <p className="mt-6 font-mono text-[11px] text-green-900/30 break-all">
            réf. {depositId}
          </p>
        )}
      </div>
    </main>
  );
}
