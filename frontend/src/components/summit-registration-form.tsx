'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';

type Profile = 'vendeur' | 'acheteur' | 'entrepreneur' | 'curieux';

const PROFILES: { value: Profile; label: string }[] = [
  { value: 'vendeur', label: 'Vendeur' },
  { value: 'acheteur', label: 'Acheteur' },
  { value: 'entrepreneur', label: 'Entrepreneur' },
  { value: 'curieux', label: 'Curieux' },
];

// ID du produit "DIARRA Summit — Tier Essentiel" (catalogue, 5000 FCFA) —
// voir cmd/seed_summit_tickets. L'inscription au Summit est devenue payante
// le 2026-10-05 : ce formulaire ne fait plus qu'une pré-saisie (nom, email,
// profil) avant de renvoyer vers le checkout du billet, qui gère lui-même
// le paiement mobile money. SummitRegistrationForm ne crée plus rien en
// base directement (voir SaleHandler.Create côté backend, qui crée
// l'inscription "pending" liée à la vente une fois la commande passée).
const SUMMIT_TIER_ESSENTIEL_PRODUCT_ID = 'ce3919cc-f2bc-423d-b552-3aa6195eae93';

export function SummitRegistrationForm() {
  const router = useRouter();
  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [profile, setProfile] = useState<Profile>('curieux');
  const [error, setError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!fullName.trim() || !email.trim()) {
      setError('Merci de remplir au moins le nom et l’email.');
      return;
    }
    const params = new URLSearchParams({
      product: SUMMIT_TIER_ESSENTIEL_PRODUCT_ID,
      summit_profile: profile,
      name: fullName.trim(),
      email: email.trim(),
    });
    router.push(`/checkout?${params.toString()}`);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <input
        type="text"
        placeholder="Nom complet"
        value={fullName}
        onChange={(e) => setFullName(e.target.value)}
        className="w-full h-11 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      <input
        type="email"
        placeholder="Email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        className="w-full h-11 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      <div>
        <p className="text-xs text-green-900/60 mb-1.5">Vous êtes plutôt :</p>
        <div className="grid grid-cols-2 gap-2">
          {PROFILES.map((p) => (
            <button
              key={p.value}
              type="button"
              onClick={() => setProfile(p.value)}
              className={`h-9 rounded-md text-sm font-medium border transition-colors ${
                profile === p.value
                  ? 'bg-[#0E6B46] text-white border-[#0E6B46]'
                  : 'bg-white text-green-900/70 border-green-900/15 hover:border-green-900/30'
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>
      </div>
      {error && <p className="text-xs text-red-600">{error}</p>}
      <button
        type="submit"
        className="w-full h-11 rounded-md bg-[#0E6B46] text-white text-sm font-semibold hover:bg-[#0c5a3c] transition-colors"
      >
        Continuer vers le paiement (5 000 FCFA)
      </button>
    </form>
  );
}
