'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { formatPrice } from '@/lib/constants';
import { CheckIcon } from '@/components/icons';

// Fallback affiché pendant le chargement / si l'API échoue — mêmes valeurs
// que les paliers seedés par la migration 036_summit_sponsors.sql, pour que
// la première impression reste correcte même sans réseau.
const FALLBACK_TIERS = [
  {
    id: 'fallback-bronze',
    name: 'Bronze',
    price_cfa: 100000,
    highlight: false,
    perks: ['Logo affiché dans la section partenaires de diarra.app/summit', 'Mention dans la page de remerciements post-événement'],
  },
  {
    id: 'fallback-argent',
    name: 'Argent',
    price_cfa: 250000,
    highlight: false,
    perks: [
      'Tout le palier Bronze',
      'Logo dans les emails de confirmation envoyés à chaque inscrit',
      'Stand dédié le jour de l’événement (UCAD)',
    ],
  },
  {
    id: 'fallback-or',
    name: 'Or',
    price_cfa: 500000,
    highlight: true,
    perks: [
      'Tout le palier Argent',
      'Logo en tête d’affiche sur la page de l’événement',
      'Mention orale à l’ouverture du Summit',
      'Publication dédiée sur les réseaux DIARRA / ABMCY',
    ],
  },
];

const FALLBACK_PARTNERS = [
  { name: 'DIARRA', logo: '/brand/diarra-icon.png', logoBg: '#0E6B46', role: 'Paiement mobile & marketplace' },
  { name: 'ABMCY', logo: '/partners/abmcy.png', logoBg: '#000000', role: "Réseau d'affaires & accompagnement" },
  { name: 'MAHU', logo: '/partners/mahu.png', logoBg: '#000000', role: 'Cartes connectées & identité numérique' },
  { name: 'Yes.abmcy', logo: '/partners/yes-abmcy.svg', logoBg: '#00a884', role: 'Messagerie acheteurs / vendeurs' },
];

interface Tier {
  id: string;
  name: string;
  price_cfa: number;
  perks: string[];
  highlight: boolean;
}

interface Sponsor {
  id: string;
  name: string;
  logo_key?: string | null;
  website_url?: string | null;
}

export function SummitSponsorTiers() {
  const [tiers, setTiers] = useState<Tier[] | null>(null);
  const [sponsors, setSponsors] = useState<Sponsor[]>([]);

  useEffect(() => {
    api
      .getSummitSponsors()
      .then((res) => {
        setTiers(res.tiers && res.tiers.length > 0 ? res.tiers : null);
        setSponsors(res.sponsors || []);
      })
      .catch(() => setTiers(null));
  }, []);

  const displayTiers = tiers ?? FALLBACK_TIERS;

  return (
    <>
      <section className="py-16 bg-green-50/40">
        <div className="max-w-6xl mx-auto px-4">
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2 text-center">// formules</p>
          <h2 className="font-display text-3xl font-bold text-green-950 text-center">Trois façons de s&rsquo;associer</h2>
          <p className="mt-3 text-green-900/70 text-center max-w-xl mx-auto">
            Chaque palier est cumulatif : tout ce qui est inclus dans un palier l&rsquo;est aussi dans les paliers
            supérieurs.
          </p>
          <div className="grid md:grid-cols-3 gap-6 mt-10">
            {displayTiers.map((tier) => (
              <div
                key={tier.id}
                className={`relative rounded-2xl border bg-white p-6 flex flex-col ${
                  tier.highlight ? 'border-lime-dark shadow-lift ring-2 ring-lime/30' : 'border-green-900/10 shadow-card'
                }`}
              >
                {tier.highlight && (
                  <span className="self-start mb-3 px-2.5 py-1 rounded-full bg-lime/20 text-green-800 text-[11px] font-semibold uppercase tracking-wide">
                    Le plus choisi
                  </span>
                )}
                <p className="font-mono text-xs font-semibold uppercase tracking-wide text-green-700/70">{tier.name}</p>
                <p className="mt-2 text-3xl font-bold text-forest font-display">{formatPrice(tier.price_cfa)}</p>
                <ul className="mt-5 space-y-2.5 flex-1">
                  {tier.perks.map((perk) => (
                    <li key={perk} className="flex items-start gap-2 text-sm text-green-900/75">
                      <CheckIcon size={15} className="shrink-0 mt-0.5 text-forest" />
                      <span>{perk}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="py-16 max-w-4xl mx-auto px-4">
        <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2 text-center">// écosystème</p>
        <h2 className="font-display text-2xl font-bold text-green-950 text-center mb-10">Déjà à nos côtés</h2>
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {FALLBACK_PARTNERS.map((p) => (
            <div key={p.name} className="rounded-2xl border border-green-900/10 bg-white shadow-card p-6 text-center">
              <div
                className="mx-auto w-14 h-14 rounded-2xl overflow-hidden flex items-center justify-center mb-3"
                style={{ backgroundColor: p.logoBg }}
              >
                <img src={p.logo} alt={p.name} className="w-full h-full object-cover" />
              </div>
              <p className="font-display font-bold text-green-950 text-sm">{p.name}</p>
              <p className="text-xs text-green-900/60 mt-1">{p.role}</p>
            </div>
          ))}
          {sponsors.map((s) => (
            <div key={s.id} className="rounded-2xl border border-green-900/10 bg-white shadow-card p-6 text-center">
              <div className="mx-auto w-14 h-14 rounded-2xl overflow-hidden flex items-center justify-center mb-3 bg-mist">
                {s.logo_key ? (
                  <img src={`/api/summit/sponsors/${s.id}/logo`} alt={s.name} className="w-full h-full object-cover" />
                ) : (
                  <span className="font-display font-bold text-forest text-lg">{s.name.charAt(0)}</span>
                )}
              </div>
              {s.website_url ? (
                <a
                  href={s.website_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-display font-bold text-green-950 text-sm hover:underline"
                >
                  {s.name}
                </a>
              ) : (
                <p className="font-display font-bold text-green-950 text-sm">{s.name}</p>
              )}
            </div>
          ))}
        </div>
      </section>
    </>
  );
}
