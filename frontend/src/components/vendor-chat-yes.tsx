'use client';

import { useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { friendlyError } from '@/lib/error-messages';

interface VendorChatYesProps {
  productId: string;
  priceCfa: number;
  country: string;
  referralLinkId?: string;
}

// Achat conversationnel via YES.abmcy Business (bêta, réservé aux vendeurs
// activés — voir Product.yes_chat_enabled). Remplace VendorChat (chat
// Firebase gratuit) pour ces vendeurs : un micro-ticket payant ouvre la
// discussion, le solde se règle plus tard une fois l'acheteur convaincu.
export function VendorChatYes({ productId, priceCfa, country, referralLinkId }: VendorChatYesProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const microTicketLabel = '600 FCFA'; // valeur par défaut affichée ; le montant réel exact vient du backend au moment du paiement

  const handleOpen = async () => {
    setError('');
    setLoading(true);
    try {
      const result = await api.openVendorConversation({
        product_id: productId,
        referral_link_id: referralLinkId,
        country,
      });
      window.location.href = result.payment_redirect_url;
    } catch (err: any) {
      setError(friendlyError(err));
      setLoading(false);
    }
  };

  return (
    <div className="rounded-xl border border-green-900/10 bg-white p-4">
      <p className="text-sm text-green-900/70">
        Posez vos questions au vendeur avant d&apos;acheter « {priceCfa.toLocaleString('fr-FR')} FCFA ».
        Un ticket d&apos;entrée de <strong>{microTicketLabel}</strong> ouvre la discussion ; vous ne payez
        le solde que si vous êtes convaincu.
      </p>
      {error && (
        <p className="text-xs text-destructive mt-2" role="alert">
          {error}
        </p>
      )}
      <Button onClick={handleOpen} disabled={loading} className="w-full mt-3">
        {loading ? 'Ouverture…' : 'Discuter avec le vendeur'}
      </Button>
    </div>
  );
}

// Variante affichée à un visiteur non connecté : invite à se connecter avant
// d'ouvrir la conversation (même paiement en jeu, il faut un compte DIARRA).
export function VendorChatYesLoggedOut() {
  return (
    <p className="text-sm text-green-900/60">
      <Link href="/auth/login" className="text-green-700 font-medium hover:underline">
        Connectez-vous
      </Link>{' '}
      pour discuter avec le vendeur.
    </p>
  );
}
