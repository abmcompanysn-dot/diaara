'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { formatPrice } from '@/lib/constants';
import { CheckIcon } from '@/components/icons';

// Copie de secours (fallback), affichée tant que les 3 produits billets
// n'ont pas encore été créés/approuvés côté backend (voir
// backend/cmd/seed_summit_tickets). Une fois approuvés, ils sont récupérés
// dynamiquement via l'API (catégorie "event") et ce fallback disparaît —
// évite d'avoir deux endroits à maintenir en désynchronisation sur les prix.
const FALLBACK_TIERS = [
  {
    title: 'Tier Essentiel',
    priceCFA: 5000,
    highlight: false,
    inclusions: [
      'Accès à l’événement DIARRA Summit en ligne',
      'Identité numérique / carte connectée MAHU',
      'Référencement au répertoire des partenaires ABMCY & DIARRA',
      'Support technique initial',
    ],
  },
  {
    title: 'Tier Business 2026',
    priceCFA: 15000,
    highlight: true,
    inclusions: [
      'Tout le Tier Essentiel',
      'Carte intelligente et augmentée MAHU',
      'Kit de présentation professionnel prêt à l’emploi',
      'Visibilité renforcée au sein du réseau ABMCY',
    ],
  },
  {
    title: 'Tier Premium / Site Vitrine',
    priceCFA: 25000,
    highlight: false,
    inclusions: [
      'Tout le Tier Business 2026',
      'Site web vitrine personnel clé en main',
      'Sous-domaine utilisateur.abmcy.com ou utilisateur.diarra.app',
      'Accès à l’espace de coworking / innovation ABMCY',
      'Carte MAHU Premium + badge de certification ABMCY',
    ],
  },
];

type LiveProduct = { id: string; slug?: string; title: string; price_cfa: number };

export function SummitTickets() {
  const [liveProducts, setLiveProducts] = useState<LiveProduct[] | null>(null);

  useEffect(() => {
    api
      .getProducts({ category: 'event' })
      .then((res) => setLiveProducts(res.products || []))
      .catch(() => setLiveProducts([]));
  }, []);

  return (
    <div className="grid md:grid-cols-3 gap-6">
      {FALLBACK_TIERS.map((tier) => {
        // Associe le produit réel (catalogue) au tier par prix, s'il existe déjà.
        const live = liveProducts?.find((p) => p.price_cfa === tier.priceCFA);
        return (
          <div
            key={tier.title}
            className={`rounded-2xl border p-6 flex flex-col ${
              tier.highlight
                ? 'border-forest bg-white shadow-lift ring-2 ring-forest/20'
                : 'border-green-900/10 bg-white shadow-lift'
            }`}
          >
            {tier.highlight && (
              <span className="self-start mb-3 px-2.5 py-1 rounded-full bg-lime/20 text-green-800 text-[11px] font-semibold uppercase tracking-wide">
                Le plus choisi
              </span>
            )}
            <h3 className="font-display text-lg font-bold text-green-950">{tier.title}</h3>
            <p className="mt-1 text-2xl font-bold text-forest">{formatPrice(tier.priceCFA)}</p>
            <ul className="mt-4 space-y-2 flex-1">
              {tier.inclusions.map((item) => (
                <li key={item} className="flex items-start gap-2 text-sm text-green-900/75">
                  <CheckIcon size={15} className="shrink-0 mt-0.5 text-forest" />
                  <span>{item}</span>
                </li>
              ))}
            </ul>
            {live ? (
              <Link
                href={`/product?id=${live.slug || live.id}`}
                className="mt-6 h-11 rounded-md bg-[#0E6B46] text-white text-sm font-semibold flex items-center justify-center hover:bg-[#0c5a3c] transition-colors"
              >
                Choisir ce billet
              </Link>
            ) : (
              <a
                href="#inscription"
                className="mt-6 h-11 rounded-md border border-green-900/15 text-green-900/70 text-sm font-semibold flex items-center justify-center hover:border-green-900/30 transition-colors"
              >
                Bientôt disponible à l&rsquo;achat
              </a>
            )}
          </div>
        );
      })}
    </div>
  );
}
