import Link from 'next/link';
import { KeyIcon, BookIcon, BriefcaseIcon, CheckIcon } from '@/components/icons';

const FEATURES = [
  {
    title: 'Commission unique de 15 %',
    desc: 'Pas de frais d\'inscription, pas d\'abonnement. Vous ne payez que 15 % quand vous vendez.',
  },
  {
    title: 'Paiement mobile money intégré',
    desc: 'Wave, Orange Money, MTN MoMo. Nous encaissons pour vous, vous êtes payé sous 48h.',
  },
  {
    title: 'Livraison automatisée',
    desc: 'Votre fichier est livré via un lien sécurisé à 3 téléchargements, sans action de votre part.',
  },
  {
    title: 'Tableau de bord vendeur',
    desc: 'Suivez vos ventes, vos revenus et vos versements en temps réel.',
  },
  {
    title: 'Support pour vos clients',
    desc: 'Nous gérons les tickets et litiges pour que vous vous concentriez sur vos produits.',
  },
  {
    title: 'Votre prix, vos règles',
    desc: 'Vous fixez le prix en FCFA. Une commission d\'affiliation peut être ajoutée si vous voulez.',
  },
];

const COMMISSION_TABLE = [
  { label: 'Prix de vente', value: '100 000 FCFA', highlight: false },
  { label: 'Votre part (85 %)', value: '85 000 FCFA', highlight: false },
  { label: 'Commission DIARRA (15 %)', value: '15 000 FCFA', highlight: false },
  { label: 'Total encaissé / mois (ex. 10 ventes)', value: '850 000 FCFA', highlight: true },
];

export default function SellPage() {
  return (
    <>
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="absolute top-0 right-0 w-96 h-96 bg-green-400/20 rounded-full blur-3xl" aria-hidden />
        <div className="relative max-w-6xl mx-auto px-4 py-20 lg:py-24 grid lg:grid-cols-2 gap-12 items-center">
          <div>
            <p className="font-mono text-sm text-green-300 mb-6 uppercase tracking-widest">
              // espace vendeur
            </p>
            <h1 className="font-display text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight leading-[1.05]">
              Vendez vos biens numériques,
              <span className="underline-accent text-lime"> sans créer de site</span>.
            </h1>
            <p className="mt-6 text-white/75 max-w-md text-lg">
              Un compte vendeur DIARRA, votre fichier, votre prix. Nous gérons le paiement,
              la livraison et le support. Vous encaissez.
            </p>
            <div className="mt-8 flex flex-wrap gap-4">
              <Link
                href="/auth/register"
                className="px-6 py-3 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300 transition-colors"
              >
                Commencer à vendre
              </Link>
              <Link
                href="/how-it-works"
                className="px-6 py-3 rounded-full border border-white/30 text-white hover:border-lime hover:text-lime transition-colors"
              >
                Comment ça marche
              </Link>
            </div>
          </div>

          <div className="bg-white text-green-950 rounded-2xl shadow-lift p-8 max-w-md mx-auto w-full">
            <p className="font-mono text-sm text-green-700/60 mb-6">// simulateur simple</p>
            <div className="space-y-3 font-mono text-sm">
              {COMMISSION_TABLE.map((row) => (
                <div
                  key={row.label}
                  className={`flex justify-between py-3 border-b border-green-900/10 ${
                    row.highlight ? 'bg-green-50 px-3 rounded-lg border-0' : ''
                  }`}
                >
                  <span className="text-green-900/60">{row.label}</span>
                  <span className={`font-bold ${row.highlight ? 'text-green-600' : 'text-green-950'}`}>
                    {row.value}
                  </span>
                </div>
              ))}
            </div>
            <p className="mt-6 text-center text-xs text-green-900/50">
              Vos versements sont envoyés directement sur votre mobile money
            </p>
          </div>
        </div>
      </section>

      <section className="py-20 max-w-6xl mx-auto px-4">
        <div className="text-center mb-14">
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-3">
            // pourquoi DIARRA
          </p>
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight text-green-950">
            Tout pour vendre, <span className="underline-green text-green-600">rien pour vous freiner</span>
          </h2>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((f) => (
            <div
              key={f.title}
              className="bg-white rounded-xl p-6 shadow-card hover:shadow-lift transition-all hover:-translate-y-0.5 border border-green-900/5"
            >
              <span className="w-10 h-10 rounded-lg gradient-green-soft flex items-center justify-center mb-4 text-green-700" aria-hidden>
                <CheckIcon size={20} />
              </span>
              <h3 className="font-display font-bold text-lg text-green-950">{f.title}</h3>
              <p className="text-sm text-green-900/70 mt-1">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Types de produits */}
      <section className="py-16 bg-green-50/60">
        <div className="max-w-6xl mx-auto px-4">
          <div className="text-center mb-12">
            <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight text-green-950">
              Que pouvez-vous vendre ?
            </h2>
          </div>
          <div className="grid gap-6 md:grid-cols-3">
            {[
              { Icon: KeyIcon, title: 'Clés & abonnements', desc: 'Spotify, Netflix, Canva Pro, Microsoft 365…' },
              { Icon: BookIcon, title: 'Ebooks & guides', desc: 'Votre savoir transformé en produit à revenus récurrents.' },
              { Icon: BriefcaseIcon, title: 'Formations & templates', desc: 'Cours, modèles, outils prêts à l\'emploi.' },
            ].map((item) => (
              <div key={item.title} className="bg-white rounded-xl p-8 text-center shadow-card border border-green-900/5">
                <span className="w-12 h-12 rounded-lg gradient-green-soft flex items-center justify-center mx-auto mb-4 text-green-700" aria-hidden>
                  <item.Icon size={24} />
                </span>
                <h3 className="font-display font-bold text-lg text-green-950">{item.title}</h3>
                <p className="text-sm text-green-900/70 mt-1">{item.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="py-16 px-4">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="font-display text-3xl font-bold text-green-950 mb-3">
            Votre premier produit est à un clic
          </h2>
          <p className="text-green-900/70 mb-6">
            Fichier max 50 Mo, prix libre en FCFA, modération sous 24h.
          </p>
          <Link
            href="/auth/register"
            className="inline-block px-8 py-3 rounded-full bg-green-700 text-white font-semibold hover:bg-green-600 transition-colors"
          >
            Ouvrir une boutique
          </Link>
        </div>
      </section>
    </>
  );
}
