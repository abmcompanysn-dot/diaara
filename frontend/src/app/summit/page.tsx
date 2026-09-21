import Link from 'next/link';
import { ZapIcon, UserIcon, CheckIcon, BriefcaseIcon } from '@/components/icons';
import { SummitRegistrationForm } from '@/components/summit-registration-form';
import { SummitTickets } from '@/components/summit-tickets';
import { SummitSponsorForm } from '@/components/summit-sponsor-form';

export const metadata = {
  title: 'DIARRA Summit — Growth Business with AI',
  description:
    "DIARRA Summit, le 26 octobre 2026 à l'Université Cheikh Anta Diop de Dakar (UCAD) et en ligne : business, intelligence artificielle et produits numériques en Afrique. Billets et inscription.",
};

const TOPICS = [
  {
    icon: ZapIcon,
    title: 'IA & digitalisation',
    desc: "Comment l'intelligence artificielle transforme déjà la vente de produits numériques en Afrique francophone.",
  },
  {
    icon: BriefcaseIcon,
    title: 'Business & croissance',
    desc: 'Stratégies concrètes pour lancer et faire grandir une activité de produits digitaux, rentable et durable.',
  },
  {
    icon: UserIcon,
    title: 'Retours de terrain',
    desc: "Des vendeurs et entrepreneurs de la plateforme DIARRA partagent ce qui a marché, et ce qui n'a pas marché.",
  },
];

const PARTNERS = [
  {
    name: 'DIARRA',
    logo: '/brand/diarra-icon.png',
    logoBg: '#0E6B46',
    role: 'Infrastructure de paiement mobile sécurisée et hébergement de sous-domaines applicatifs (.diarra.app).',
  },
  {
    name: 'ABMCY',
    logo: '/partners/abmcy.png',
    logoBg: '#000000',
    role: "Réseau d'affaires, accompagnement entrepreneurial et hébergement de vitrines professionnelles (.abmcy.com).",
  },
  {
    name: 'MAHU',
    logo: '/partners/mahu.png',
    logoBg: '#000000',
    role: "Support matériel, cartes connectées et technologies d'identification numérique.",
  },
  {
    name: 'Yes.abmcy',
    logo: '/partners/yes-abmcy.svg',
    logoBg: '#00a884',
    role: "Messagerie instantanée intégrée pour échanger directement entre acheteurs et vendeurs (yes.abmcy.com).",
  },
];

export default function SummitPage() {
  return (
    <>
      <section className="relative overflow-hidden text-white">
        {/* Image de fond pleine section (même traitement que /summit/sponsors,
            à la demande de l'utilisateur) : l'image couvre toute la section
            hero, avec un voile dégradé pour garder le texte lisible. */}
        <img
          src="/summit/hero.jpg"
          alt=""
          aria-hidden
          className="absolute inset-0 w-full h-full object-cover"
        />
        <div className="absolute inset-0 bg-linear-to-b from-forest-2/80 via-forest-2/75 to-forest-2/90" aria-hidden />
        <div className="wax-pattern absolute inset-0 opacity-20" aria-hidden />
        <div className="relative max-w-4xl mx-auto px-4 py-24 text-center">
          <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-4">
            // UCAD, Dakar &amp; en ligne &middot; 26 octobre 2026
          </p>
          <h1 className="font-display text-4xl sm:text-5xl font-bold tracking-tight">
            DIARRA Summit
          </h1>
          <p className="mt-3 text-lg text-green-300 font-semibold">Faire croître son business avec l&rsquo;IA</p>
          <p className="mt-1 text-sm text-green-300/70 font-medium uppercase tracking-wide">Growth Business with AI</p>
          <p className="mt-5 text-white/85 max-w-xl mx-auto">
            Business, intelligence artificielle et produits numériques en Afrique.
            Une matinée à l&rsquo;Université Cheikh Anta Diop de Dakar (UCAD), Sénégal, et diffusée en direct en ligne,
            pour comprendre où va la digitalisation du continent — et comment l&rsquo;écosystème DIARRA, ABMCY, MAHU et
            Yes.abmcy y participe.
          </p>
          <a
            href="#billets"
            className="inline-block mt-8 px-7 h-12 leading-12 rounded-full bg-lime text-green-950 font-semibold text-sm hover:brightness-95 transition"
          >
            Voir les billets
          </a>
        </div>
      </section>

      <section className="py-16 max-w-6xl mx-auto px-4">
        <div className="grid lg:grid-cols-2 gap-12 items-start">
          <div>
            <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-6">
              // au programme
            </p>
            <div className="space-y-6">
              {TOPICS.map((t) => (
                <div key={t.title} className="flex gap-4">
                  <span className="shrink-0 w-11 h-11 rounded-xl bg-green-100 text-green-700 flex items-center justify-center">
                    <t.icon size={20} />
                  </span>
                  <div>
                    <h3 className="font-semibold text-green-950">{t.title}</h3>
                    <p className="mt-1 text-sm text-green-900/70">{t.desc}</p>
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-10 rounded-2xl border border-green-900/10 bg-green-50/50 p-5">
              <div className="flex items-center gap-2 text-green-800 font-semibold text-sm mb-2">
                <CheckIcon size={16} />
                Sur place à l&rsquo;UCAD ou en ligne
              </div>
              <p className="text-sm text-green-900/70">
                Rendez-vous à l&rsquo;Université Cheikh Anta Diop de Dakar (UCAD), Sénégal. Le lien de connexion pour
                suivre l&rsquo;événement en direct sera envoyé par email à tous les inscrits (quel que soit le billet
                choisi) à l&rsquo;approche de l&rsquo;événement, le <strong>26 octobre 2026</strong>.
              </p>
            </div>
          </div>

          <div id="inscription" className="rounded-2xl border border-green-900/10 bg-white shadow-lift p-6 sm:p-8">
            <h2 className="font-display text-xl font-bold text-green-950">Je m&rsquo;inscris</h2>
            <p className="mt-1 text-sm text-green-900/60">
              Quelques infos et c&rsquo;est fait — la confirmation arrive par email.
            </p>
            <div className="mt-6">
              <SummitRegistrationForm />
            </div>
          </div>
        </div>
      </section>

      <section id="billets" className="py-16 bg-green-50/40">
        <div className="max-w-6xl mx-auto px-4">
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2 text-center">
            // billets
          </p>
          <h2 className="font-display text-3xl font-bold text-green-950 text-center">
            Choisissez votre formule
          </h2>
          <p className="mt-3 text-green-900/70 text-center max-w-xl mx-auto">
            Paiement sécurisé par mobile money (Wave, Orange Money, MTN MoMo) via DIARRA.
            Une équipe DIARRA / ABMCY / MAHU / Yes.abmcy vous recontacte sous 48h pour la mise en place
            des éléments inclus dans votre formule.
          </p>
          <div className="mt-10">
            <SummitTickets />
          </div>
        </div>
      </section>

      <section className="py-16 max-w-4xl mx-auto px-4">
        <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2 text-center">
          // écosystème & partenaires
        </p>
        <h2 className="font-display text-2xl font-bold text-green-950 text-center mb-10">
          Un événement porté par quatre acteurs complémentaires
        </h2>
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {PARTNERS.map((p) => (
            <div key={p.name} className="rounded-2xl border border-green-900/10 bg-white shadow-lift p-6 text-center">
              <div
                className="mx-auto w-16 h-16 rounded-2xl overflow-hidden flex items-center justify-center mb-4"
                style={{ backgroundColor: p.logoBg }}
              >
                <img src={p.logo} alt={p.name} className="w-full h-full object-cover" />
              </div>
              <p className="font-display font-bold text-green-950 mb-1">{p.name}</p>
              <p className="text-sm text-green-900/70">{p.role}</p>
            </div>
          ))}
        </div>
      </section>

      <section id="sponsors" className="py-16 bg-green-50/40">
        <div className="max-w-md mx-auto px-4">
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2 text-center">
            // votre marque ici
          </p>
          <h2 className="font-display text-2xl font-bold text-green-950 text-center">Devenir sponsor</h2>
          <p className="mt-3 text-sm text-green-900/70 text-center">
            Votre entreprise souhaite être visible auprès des vendeurs et entrepreneurs du DIARRA Summit ?
            Dites-nous-en plus, l&rsquo;équipe DIARRA vous recontacte.
          </p>
          <p className="mt-2 text-center">
            <Link href="/summit/sponsors" className="text-sm font-semibold text-forest underline underline-offset-2">
              Voir les paliers de sponsoring &rarr;
            </Link>
          </p>
          <div className="mt-6 rounded-2xl border border-green-900/10 bg-white shadow-lift p-6 sm:p-8">
            <SummitSponsorForm />
          </div>
        </div>
      </section>
    </>
  );
}
