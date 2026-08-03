import Link from 'next/link';
import { CheckIcon } from '@/components/icons';

const AFFILIATE_STEPS = [
  {
    num: '01',
    title: 'Créez votre compte',
    desc: 'Inscription gratuite, sélectionnez le rôle affilié (closer) dans votre profil.',
  },
  {
    num: '02',
    title: 'Générez vos liens',
    desc: 'Choisissez des produits du catalogue et créez votre lien d\'affiliation personnalisé.',
  },
  {
    num: '03',
    title: 'Partagez-les',
    desc: 'WhatsApp, réseaux sociaux, blogs… chaque clic et chaque vente sont suivis.',
  },
  {
    num: '04',
    title: 'Encaissez vos commissions',
    desc: 'Une commission sur chaque vente réalisée via votre lien, versée sur mobile money.',
  },
];

const PERKS = [
  'Suivi des clics et conversions en temps réel',
  'Liens personnalisés et faciles à mémoriser',
  'Versements regroupés sous 48h',
  'Large catalogue de produits à promouvoir',
];

export default function CloserPage() {
  return (
    <>
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-4xl mx-auto px-4 py-20 text-center">
          <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-4">
            // programme d&apos;affiliation
          </p>
          <h1 className="font-display text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight leading-[1.05]">
            Recommandez, <span className="underline-accent text-lime">gagnez à chaque vente</span>.
          </h1>
          <p className="mt-6 text-white/75 max-w-xl mx-auto text-lg">
            Pas de stock, pas de fichier à gérer. Vous partagez des liens, vous touchez une
            commission sur chaque vente que vous générez.
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-4">
            <Link
              href="/auth/register"
              className="px-6 py-3 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300 transition-colors"
            >
              Devenir affilié
            </Link>
            <Link
              href="/catalog"
              className="px-6 py-3 rounded-full border border-white/30 text-white hover:border-lime hover:text-lime transition-colors"
            >
              Voir les produits
            </Link>
          </div>
        </div>
      </section>

      <section className="py-20 max-w-6xl mx-auto px-4">
        <div className="text-center mb-14">
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-3">
            // comment ça marche
          </p>
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight text-green-950">
            Votre lien, votre revenu
          </h2>
        </div>
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          {AFFILIATE_STEPS.map((step) => (
            <div key={step.num} className="bg-white rounded-xl p-6 shadow-card border border-green-900/5">
              <p className="font-mono text-4xl font-bold text-green-300 mb-4">{step.num}</p>
              <h3 className="font-display font-bold text-lg text-green-950">{step.title}</h3>
              <p className="text-sm text-green-900/70 mt-1">{step.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Exemple de commission */}
      <section className="py-16 bg-green-50/60">
        <div className="max-w-6xl mx-auto px-4 grid lg:grid-cols-2 gap-12 items-center">
          <div>
            <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-3">
              // un exemple
            </p>
            <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight text-green-950">
              Vos commissions, <span className="underline-green text-green-600">simulées</span>
            </h2>
            <p className="mt-4 text-green-900/70 leading-relaxed">
              Votre commission est un pourcentage du prix du produit, défini par le vendeur.
              Plus vous vendez, plus vous gagnez. Tout est suivi dans votre espace affilié.
            </p>
            <ul className="mt-6 space-y-3">
              {PERKS.map((item) => (
                <li key={item} className="flex items-start gap-3 text-sm text-green-900/80">
                  <span className="mt-0.5 w-5 h-5 rounded-full bg-green-500 text-white flex items-center justify-center" aria-hidden>
                    <CheckIcon size={14} />
                  </span>
                  <span>{item}</span>
                </li>
              ))}
            </ul>
          </div>
          <div className="bg-white rounded-xl p-8 shadow-card border border-green-900/5">
            <p className="font-mono text-sm text-green-700/60 mb-6">// simulateur de revenus</p>
            <div className="space-y-3 font-mono text-sm">
              <div className="flex justify-between py-3 border-b border-green-900/10">
                <span className="text-green-900/60">Prix d&apos;un produit</span>
                <span className="font-bold text-green-950">10 000 FCFA</span>
              </div>
              <div className="flex justify-between py-3 border-b border-green-900/10">
                <span className="text-green-900/60">Commission par vente (30 %)</span>
                <span className="font-bold text-green-600">3 000 FCFA</span>
              </div>
              <div className="flex justify-between py-3 border-b border-green-900/10">
                <span className="text-green-900/60">10 ventes / mois</span>
                <span className="font-bold text-green-950">30 000 FCFA</span>
              </div>
              <div className="flex justify-between py-3">
                <span className="text-green-900/60">30 ventes / mois</span>
                <span className="font-bold text-lime-dark text-green-700">90 000 FCFA</span>
              </div>
            </div>
            <p className="mt-6 text-center text-xs text-green-900/50">
              Commissions créditées puis versées sur votre mobile money
            </p>
          </div>
        </div>
      </section>

      <section className="py-16 px-4">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="font-display text-3xl font-bold text-green-950 mb-3">
            Commencez à recommander dès aujourd&apos;hui
          </h2>
          <Link
            href="/auth/register"
            className="inline-block mt-4 px-8 py-3 rounded-full bg-green-700 text-white font-semibold hover:bg-green-600 transition-colors"
          >
            Créer mon compte affilié
          </Link>
        </div>
      </section>
    </>
  );
}
