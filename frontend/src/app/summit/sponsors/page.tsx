import Link from 'next/link';
import { SummitSponsorTiers } from '@/components/summit-sponsor-tiers';

export const metadata = {
  title: 'Devenir sponsor — DIARRA Summit',
  description:
    "Proposition de partenariat pour le DIARRA Summit — Growth Business with AI, le 26 octobre 2026 à l'Université Cheikh Anta Diop de Dakar (UCAD) et en ligne. Paliers, avantages et contact.",
};

const WHY = [
  {
    num: '01',
    title: 'Public qualifié',
    desc: 'Entrepreneurs, vendeurs de produits digitaux, développeurs et décideurs déjà actifs sur l’écosystème DIARRA / ABMCY / MAHU.',
  },
  {
    num: '02',
    title: 'Portée panafricaine',
    desc: 'Diffusion en ligne en plus du présentiel à l’UCAD : votre marque touche l’audience africaine francophone bien au-delà de Dakar.',
  },
  {
    num: '03',
    title: 'Contenu à forte valeur',
    desc: 'Ateliers pratiques et cas d’usage réels autour de produits numériques propulsés par l’IA — un cadre crédible pour votre marque.',
  },
];

export default function SummitSponsorsPage() {
  return (
    <>
      <section className="relative overflow-hidden text-white">
        {/* Image de fond pleine section : l'image du hero couvre toute la
            section (pas une carte séparée), avec un voile sombre dégradé
            pour garder le texte lisible par-dessus — voir la demande de
            l'utilisateur ("l'image doit prendre toute la section hero"). */}
        <img
          src="/summit/hero.jpg"
          alt=""
          aria-hidden
          className="absolute inset-0 w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-linear-to-b from-forest-2/80 via-forest-2/75 to-forest-2/90" aria-hidden />
        <div className="wax-pattern absolute inset-0 opacity-20" aria-hidden />
        <div className="relative max-w-4xl mx-auto px-4 py-24 text-center">
          <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-4">// dossier partenaires</p>
          <h1 className="font-display text-4xl sm:text-5xl font-bold tracking-tight">DIARRA Summit</h1>
          <p className="mt-3 text-lg text-green-300 font-semibold">Faire croître son business avec l&rsquo;IA</p>
          <p className="mt-1 text-sm text-green-300/70 font-medium uppercase tracking-wide">Growth Business with AI</p>
          <p className="mt-5 text-white/85 max-w-xl mx-auto">
            Le rendez-vous business et intelligence artificielle qui réunit entrepreneurs, développeurs, agences et
            décideurs autour des leviers concrets de croissance pour l&rsquo;Afrique — porté par l&rsquo;écosystème
            DIARRA, ABMCY, MAHU et Yes.abmcy.
          </p>
          <div className="flex flex-wrap justify-center gap-2 mt-8">
            <span className="font-mono text-xs font-semibold bg-white/10 border border-white/20 text-white px-3.5 py-1.5 rounded-full">
              26 octobre 2026
            </span>
            <span className="font-mono text-xs font-semibold bg-white/10 border border-white/20 text-white px-3.5 py-1.5 rounded-full">
              UCAD, Dakar — Sénégal
            </span>
            <span className="font-mono text-xs font-semibold bg-white/10 border border-white/20 text-white px-3.5 py-1.5 rounded-full">
              Présentiel + en ligne
            </span>
          </div>
        </div>
      </section>

      <section className="py-16 max-w-6xl mx-auto px-4">
        <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2 text-center">// pourquoi s&rsquo;associer</p>
        <h2 className="font-display text-3xl font-bold text-green-950 text-center">Une audience qui construit l&rsquo;Afrique numérique</h2>
        <p className="mt-3 text-green-900/70 text-center max-w-xl mx-auto">
          Le Summit rassemble les personnes qui créent, vendent et financent les produits numériques du continent —
          l&rsquo;audience la plus directement pertinente pour une marque tech ou financière.
        </p>
        <div className="grid sm:grid-cols-3 gap-6 mt-10">
          {WHY.map((w) => (
            <div key={w.num} className="rounded-2xl border border-green-900/10 bg-white shadow-card p-6">
              <p className="font-mono text-xs font-semibold text-green-600">{w.num}</p>
              <h3 className="font-semibold text-green-950 mt-2.5">{w.title}</h3>
              <p className="mt-1.5 text-sm text-green-900/70">{w.desc}</p>
            </div>
          ))}
        </div>
      </section>

      <SummitSponsorTiers />

      <section className="pb-16 max-w-3xl mx-auto px-4">
        <div className="rounded-2xl gradient-green text-white p-8 sm:p-10 text-center">
          <h2 className="font-display text-2xl font-bold">Discutons de votre partenariat</h2>
          <p className="mt-2 text-white/80 max-w-md mx-auto text-sm">
            Choisissez une formule ou parlons d&rsquo;un accord sur mesure — l&rsquo;équipe DIARRA vous répond
            rapidement.
          </p>
          <div className="flex flex-wrap justify-center gap-3 mt-6">
            <Link
              href="/summit#sponsors"
              className="inline-block px-6 h-11 leading-11 rounded-full bg-lime text-green-950 font-semibold text-sm hover:brightness-95 transition"
            >
              Devenir sponsor
            </Link>
            <Link
              href="/summit"
              className="inline-block px-6 h-11 leading-11 rounded-full bg-white/10 border border-white/25 text-white font-semibold text-sm hover:bg-white/15 transition"
            >
              Voir l&rsquo;événement
            </Link>
          </div>
        </div>
      </section>
    </>
  );
}
