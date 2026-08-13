import Link from 'next/link';
import { KeyIcon, UserIcon, BookIcon, FileIcon, MonitorIcon, ZapIcon, CheckIcon, DownloadIcon } from '@/components/icons';

const CATEGORIES = [
  { name: 'Clés d\'abonnement', Icon: KeyIcon, desc: 'Spotify, Netflix, Canva Pro, Microsoft 365', href: '/catalog' },
  { name: 'Comptes', Icon: UserIcon, desc: 'Comptes premium et accès partagés', href: '/catalog' },
  { name: 'Ebooks', Icon: BookIcon, desc: 'Livres numériques et guides pratiques', href: '/catalog' },
  { name: 'PDF & cours', Icon: FileIcon, desc: 'Documents, formations et templates', href: '/catalog' },
  { name: 'Logiciels', Icon: MonitorIcon, desc: 'Licences et outils numériques', href: '/catalog' },
  { name: 'Services', Icon: ZapIcon, desc: 'Prestations et accompagnements', href: '/catalog' },
];

const STEPS = [
  {
    num: '01',
    title: 'Choisissez votre produit',
    desc: 'Parcourez le catalogue et sélectionnez le bien numérique qui vous intéresse.',
  },
  {
    num: '02',
    title: 'Payez en mobile money',
    desc: 'Wave, Orange Money ou MTN MoMo. Le paiement est sécurisé et vérifié instantanément.',
  },
  {
    num: '03',
    title: 'Recevez votre fichier',
    desc: 'Livraison immédiate via un lien de téléchargement sécurisé, valable 5 minutes.',
  },
];

const PAYMENTS = [
  { name: 'Wave', color: 'bg-green-400', text: 'text-green-950' },
  { name: 'Orange Money', color: 'bg-money-om', text: 'text-white' },
  { name: 'MTN MoMo', color: 'bg-money-mtn', text: 'text-green-950' },
];

const COUNTRIES = [
  { name: 'Bénin', flag: '🇧🇯' },
  { name: 'Burkina Faso', flag: '🇧🇫' },
  { name: 'Cameroun', flag: '🇨🇲' },
  { name: 'Congo-Brazzaville', flag: '🇨🇬' },
  { name: 'RD Congo', flag: '🇨🇩' },
  { name: 'Gabon', flag: '🇬🇦' },
  { name: 'Ghana', flag: '🇬🇭' },
  { name: "Côte d'Ivoire", flag: '🇨🇮' },
  { name: 'Kenya', flag: '🇰🇪' },
  { name: 'Lesotho', flag: '🇱🇸' },
  { name: 'Malawi', flag: '🇲🇼' },
  { name: 'Mozambique', flag: '🇲🇿' },
  { name: 'Nigeria', flag: '🇳🇬' },
  { name: 'Rwanda', flag: '🇷🇼' },
  { name: 'Sénégal', flag: '🇸🇳' },
  { name: 'Sierra Leone', flag: '🇸🇱' },
  { name: 'Tanzanie', flag: '🇹🇿' },
  { name: 'Ouganda', flag: '🇺🇬' },
  { name: 'Zambie', flag: '🇿🇲' },
];

// Réseaux mobile money présents sur le continent (couverture réseau PawaPay).
// Le paiement actif sur DIARRA reste limité aux pays listés dans lib/operators.ts.
const PAYMENT_LOGOS = [
  { name: 'Orange Money', file: 'orange-money.png' },
  { name: 'MTN MoMo', file: 'mtn-momo.png' },
  { name: 'Moov Money', file: 'moov-money.png' },
  { name: 'M-Pesa', file: 'mpesa.png' },
  { name: 'Safaricom M-Pesa', file: 'safaricom-mpesa.png' },
  { name: 'Vodacom', file: 'vodacom.png' },
  { name: 'AT Money', file: 'at-money.png' },
  { name: 'Mixx by Yas', file: 'mixx-yas.png' },
  { name: 'HaloPesa', file: 'halopesa.png' },
  { name: 'Movitel', file: 'movitel.png' },
  { name: 'TNM', file: 'tnm.png' },
  { name: 'Telecel Cash', file: 'telecel-cash.png' },
  { name: 'Zamtel', file: 'zamtel.png' },
];

export default function Home() {
  return (
    <>
      {/* Hero */}
      <section className="relative gradient-green text-white overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="absolute top-0 right-0 w-96 h-96 bg-green-400/20 rounded-full blur-3xl" aria-hidden />
        <div className="absolute bottom-0 left-1/3 w-80 h-80 bg-green-500/20 rounded-full blur-3xl" aria-hidden />
        <div className="relative max-w-6xl mx-auto px-4 pt-20 pb-24 lg:pt-28 lg:pb-32 grid lg:grid-cols-2 gap-12 items-center">
          <div>
            <p className="font-mono text-sm text-green-300 mb-6 uppercase tracking-widest">
              // le marché des biens numériques
            </p>
            <h1 className="font-display text-4xl sm:text-5xl lg:text-6xl font-bold leading-[1.05] tracking-tight">
              Achetez et vendez le numérique,
              <span className="underline-accent text-lime"> payé en mobile money</span>.
            </h1>
            <p className="mt-6 text-lg text-white/75 max-w-md">
              Clés d&apos;abonnement, comptes premium, ebooks, PDF… Tout s&apos;achète en
              quelques secondes, se reçoit immédiatement, et se paie avec votre moyen préféré.
            </p>
            <div className="mt-8 flex flex-wrap gap-4">
              <Link
                href="/catalog"
                className="px-6 py-3 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300 hover:text-green-950 transition-colors"
              >
                Explorer le catalogue
              </Link>
              <Link
                href="/sell"
                className="px-6 py-3 rounded-full border border-white/30 text-white hover:border-lime hover:text-lime transition-colors"
              >
                Vendre sur DIARRA
              </Link>
            </div>
            <div className="mt-10 flex items-center gap-3">
              <span className="text-sm text-white/60">Paiements sécurisés :</span>
              <div className="flex gap-2">
                {PAYMENTS.map((p) => (
                  <span
                    key={p.name}
                    className={`px-3 py-1 rounded ${p.color} ${p.text} text-xs font-bold`}
                  >
                    {p.name}
                  </span>
                ))}
              </div>
            </div>
          </div>

          {/* Ticket de caisse animé */}
          <div className="relative">
            <div className="rounded-xl bg-white text-green-950 shadow-lift p-6 max-w-sm mx-auto rotate-1">
              <div className="border-b-2 border-dashed border-green-900/20 pb-4 flex justify-between items-start">
                <div>
                  <p className="font-mono text-xs text-green-900/50">DIARRA · TICKET</p>
                  <p className="font-display font-bold text-lg mt-1">Canva Pro 1 an</p>
                </div>
                <span className="font-mono text-sm font-bold">12 500 FCFA</span>
              </div>

              <div className="py-4 space-y-2 font-mono text-sm">
                <p className="receipt-cursor flex justify-between">
                  <span>Paiement Wave…</span>
                  <span className="text-green-500 flex items-center"><CheckIcon size={16} /></span>
                </p>
                <p className="flex justify-between text-green-900/60">
                  <span>Statut</span>
                  <span className="text-green-600 font-bold">CONFIRMÉ</span>
                </p>
                <p className="flex justify-between text-green-900/60">
                  <span>Livraison</span>
                  <span>Lien sécurisé</span>
                </p>
              </div>

              <div className="border-t-2 border-dashed border-green-900/20 pt-4">
                <p className="text-center text-xs text-green-900/50 mb-2">--- votre fichier est prêt ---</p>
                <div className="gradient-green text-lime rounded-lg py-3 text-center font-mono text-sm font-bold">
                  <DownloadIcon size={16} className="inline mr-1" />Télécharger · 3 min restantes
                </div>
              </div>
            </div>
            <div className="absolute -bottom-4 -left-4 w-full h-full bg-lime rounded-xl -z-10" aria-hidden />
          </div>
        </div>
      </section>

      {/* Ticker */}
      <div className="bg-lime text-green-950 overflow-hidden py-3 border-y border-green-900/10">
        <div className="ticker-track whitespace-nowrap font-mono text-sm font-bold">
          <span className="px-8">
            Clés d&apos;abonnement ✦ Comptes premium ✦ Ebooks ✦ PDF &amp; cours ✦ Logiciels ✦
            Livraison instantanée ✦ Paiement mobile money ✦ 15% de commission plateforme ✦
          </span>
          <span className="px-8">
            Clés d&apos;abonnement ✦ Comptes premium ✦ Ebooks ✦ PDF &amp; cours ✦ Logiciels ✦
            Livraison instantanée ✦ Paiement mobile money ✦ 15% de commission plateforme ✦
          </span>
        </div>
      </div>

      {/* Statistiques */}
      <section className="py-16 max-w-6xl mx-auto px-4">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-6 text-center">
          {[
            { value: '< 1 min', label: 'Temps moyen pour acheter' },
            { value: '3', label: 'Moyens de paiement mobile money' },
            { value: '15 %', label: 'Commission plateforme unique' },
            { value: '5 min', label: 'Validité du lien de livraison' },
          ].map((stat) => (
            <div key={stat.label}>
              <p className="font-display text-3xl font-bold text-green-700">{stat.value}</p>
              <p className="text-sm text-green-900/60 mt-1">{stat.label}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Couverture pays */}
      <section className="py-20 bg-green-950 text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-6xl mx-auto px-4">
          <div className="text-center mb-12">
            <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-3">
              // couverture réseau
            </p>
            <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
              Disponible dans <span className="underline-accent text-lime">toute l&apos;Afrique</span>
            </h2>
            <p className="mt-4 text-white/70 max-w-2xl mx-auto">
              DIARRA s&apos;appuie sur le réseau mobile money PawaPay, présent dans {COUNTRIES.length} pays.
              Le paiement est déjà actif au Sénégal, en Côte d&apos;Ivoire, au Bénin et au Burkina Faso,
              et s&apos;étend progressivement au reste du continent.
            </p>
          </div>
          <div className="flex flex-wrap justify-center gap-3">
            {COUNTRIES.map((c) => (
              <span
                key={c.name}
                className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-white/5 border border-white/10 text-sm text-white/85"
              >
                <span aria-hidden>{c.flag}</span>
                {c.name}
              </span>
            ))}
          </div>
        </div>
      </section>

      {/* Moyens de paiement mobile money */}
      <section className="py-20 max-w-6xl mx-auto px-4">
        <div className="text-center mb-12">
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-3">
            // moyens de paiement
          </p>
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight text-green-950">
            Tous vos <span className="underline-green">réseaux mobile money</span>
          </h2>
          <p className="mt-4 text-green-900/60 max-w-2xl mx-auto">
            Payez et soyez payé avec l&apos;opérateur mobile money de votre pays.
          </p>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
          <div className="flex items-center justify-center bg-green-400 rounded-xl p-4 h-20 shadow-card">
            <span className="font-display font-bold text-green-950">Wave</span>
          </div>
          {PAYMENT_LOGOS.map((p) => (
            <div
              key={p.name}
              className="flex items-center justify-center bg-white rounded-xl p-4 h-20 shadow-card border border-green-900/5"
            >
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={`/payments/${p.file}`} alt={p.name} className="max-h-10 max-w-full object-contain" />
            </div>
          ))}
        </div>
      </section>

      {/* Catégories */}
      <section className="py-20 max-w-6xl mx-auto px-4">
        <div className="text-center mb-12">
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-3">
            // le catalogue
          </p>
          <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight text-green-950">
            Tout le numérique, <span className="underline-green">une seule place</span>
          </h2>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {CATEGORIES.map((cat) => (
            <Link
              key={cat.name}
              href={cat.href}
              className="group bg-white rounded-xl p-6 shadow-card hover:shadow-lift transition-all hover:-translate-y-0.5 border border-green-900/5"
            >
              <span className="w-12 h-12 rounded-lg gradient-green-soft flex items-center justify-center mb-3" aria-hidden>
                <cat.Icon size={24} className="text-green-700" />
              </span>
              <h3 className="font-display font-bold text-lg text-green-950 group-hover:text-green-600 transition-colors">
                {cat.name}
              </h3>
              <p className="text-sm text-green-900/60 mt-1">{cat.desc}</p>
            </Link>
          ))}
        </div>
      </section>

      {/* Comment ça marche */}
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-6xl mx-auto px-4 py-20">
          <div className="text-center mb-14">
            <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-3">
              // 3 étapes
            </p>
            <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
              De la sélection à la livraison
            </h2>
          </div>
          <div className="grid gap-8 md:grid-cols-3">
            {STEPS.map((step) => (
              <div key={step.num} className="relative bg-white/5 backdrop-blur rounded-xl p-6 border border-white/10">
                <p className="font-mono text-4xl font-bold text-green-300 mb-4">{step.num}</p>
                <h3 className="font-display font-bold text-xl mb-2">{step.title}</h3>
                <p className="text-white/70 leading-relaxed">{step.desc}</p>
              </div>
            ))}
          </div>
          <div className="text-center mt-12">
            <Link
              href="/how-it-works"
              className="inline-block px-6 py-3 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300 transition-colors"
            >
              Voir les détails
            </Link>
          </div>
        </div>
      </section>

      {/* Vendre */}
      <section className="py-20 max-w-6xl mx-auto px-4">
        <div className="grid lg:grid-cols-2 gap-12 items-center">
          <div>
            <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-3">
              // pour les vendeurs
            </p>
            <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight leading-tight text-green-950">
              Vendez vos fichiers,
              <br />
              <span className="underline-green text-green-600">encaissez en FCFA</span>.
            </h2>
            <p className="mt-6 text-green-900/70 leading-relaxed max-w-md">
              Ebook, clé d&apos;abonnement, formation… Déposez votre produit, nous nous
              occupons du paiement, de la livraison sécurisée et du support.
            </p>
            <ul className="mt-6 space-y-3">
              {[
                'Commission plateforme unique : 15 %',
                'Paiements Wave, Orange Money, MTN MoMo',
                'Versements sous 48h sur votre mobile money',
                'Livraison sécurisée : lien limité à 3 téléchargements',
              ].map((item) => (
                <li key={item} className="flex items-start gap-3 text-sm text-green-900/80">
                  <span className="mt-0.5 w-5 h-5 rounded-full bg-green-500 text-white flex items-center justify-center" aria-hidden>
                    <CheckIcon size={14} />
                  </span>
                  <span>{item}</span>
                </li>
              ))}
            </ul>
            <Link
              href="/sell"
              className="mt-8 inline-block px-6 py-3 rounded-full bg-green-700 text-white font-semibold hover:bg-green-600 transition-colors"
            >
              Devenir vendeur
            </Link>
          </div>
          <div className="bg-white rounded-xl p-8 shadow-card border border-green-900/5">
            <p className="font-mono text-sm text-green-700/60 mb-6">// exemple de vente</p>
            <div className="space-y-3 font-mono text-sm">
              <div className="flex justify-between py-3 border-b border-green-900/10">
                <span className="text-green-900/60">Prix produit</span>
                <span className="font-bold text-green-950">10 000 FCFA</span>
              </div>
              <div className="flex justify-between py-3 border-b border-green-900/10">
                <span className="text-green-900/60">Commission DIARRA (15 %)</span>
                <span className="text-money-om font-bold">- 1 500 FCFA</span>
              </div>
              <div className="flex justify-between py-3">
                <span className="text-green-900/60">Votre gain</span>
                <span className="font-bold text-green-600">8 500 FCFA</span>
              </div>
            </div>
            <p className="mt-6 text-center text-xs text-green-900/50">
              Encaissé directement sur votre mobile money
            </p>
          </div>
        </div>
      </section>

      {/* Affiliation teaser */}
      <section className="py-20 px-4">
        <div className="max-w-6xl mx-auto grid lg:grid-cols-2 gap-12 items-center bg-green-950 text-white rounded-2xl overflow-hidden relative">
          <div className="wax-pattern absolute inset-0" aria-hidden />
          <div className="relative p-10 sm:p-14">
            <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-3">
              // pour les affiliés
            </p>
            <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight leading-tight">
              Partagez un lien,
              <br />
              <span className="underline-accent text-lime">gagnez une commission</span>.
            </h2>
            <p className="mt-4 text-white/70 max-w-md">
              Recommandez les produits de la marketplace avec votre lien d&apos;affiliation
              et touchez une commission sur chaque vente.
            </p>
            <Link
              href="/closer"
              className="mt-8 inline-block px-6 py-3 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300 transition-colors"
            >
              Devenir affilié
            </Link>
          </div>
          <div className="relative p-10 sm:p-14">
            <div className="bg-white/5 backdrop-blur rounded-xl p-6 border border-white/10 font-mono text-sm space-y-3">
              <div className="flex justify-between py-3 border-b border-white/10">
                <span className="text-white/60">Produit vendu</span>
                <span className="font-bold text-white">10 000 FCFA</span>
              </div>
              <div className="flex justify-between py-3 border-b border-white/10">
                <span className="text-white/60">Votre commission (30 %)</span>
                <span className="font-bold text-lime">3 000 FCFA</span>
              </div>
              <div className="flex justify-between py-3">
                <span className="text-white/60">Votre lien</span>
                <span className="text-green-300">votre-lien-daffiliation</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* CTA final */}
      <section className="pb-20 px-4">
        <div className="max-w-4xl mx-auto gradient-green text-white rounded-2xl p-10 sm:p-14 text-center relative overflow-hidden">
          <div className="wax-pattern absolute inset-0" aria-hidden />
          <div className="absolute top-0 left-1/4 w-72 h-72 bg-lime/20 rounded-full blur-3xl" aria-hidden />
          <div className="relative">
            <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">
              Prêt à rejoindre le marché ?
            </h2>
            <p className="mt-4 text-white/70 max-w-md mx-auto">
              Créez votre compte gratuitement. Achetez en une minute, vendez sans limite.
            </p>
            <div className="mt-8 flex flex-wrap justify-center gap-4">
              <Link
                href="/auth/register"
                className="px-6 py-3 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300 transition-colors"
              >
                Créer mon compte
              </Link>
              <Link
                href="/catalog"
                className="px-6 py-3 rounded-full border border-white/30 text-white hover:border-lime hover:text-lime transition-colors"
              >
                Voir le catalogue
              </Link>
            </div>
          </div>
        </div>
      </section>
    </>
  );
}
