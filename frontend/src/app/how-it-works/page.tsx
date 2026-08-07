import Link from 'next/link';
import { CheckIcon } from '@/components/icons';

const BUYER_STEPS = [
  {
    num: '01',
    title: 'Créez votre compte',
    desc: 'Inscription gratuite en moins d\'une minute, avec votre email ou votre téléphone.',
  },
  {
    num: '02',
    title: 'Parcourez le catalogue',
    desc: 'Filtrez par catégorie : clés d\'abonnement, comptes, ebooks, PDF, logiciels.',
  },
  {
    num: '03',
    title: 'Payez par mobile money',
    desc: 'Wave, Orange Money ou MTN MoMo. Le vendeur n\'est payé qu\'après confirmation.',
  },
  {
    num: '04',
    title: 'Téléchargez votre fichier',
    desc: 'Lien de livraison sécurisé valable 5 minutes, 3 téléchargements maximum.',
  },
];

const SELLER_STEPS = [
  {
    num: '01',
    title: 'Créez un compte vendeur',
    desc: 'Ajoutez votre rôle vendeur lors de l\'inscription ou depuis votre profil.',
  },
  {
    num: '02',
    title: 'Déposez votre produit',
    desc: 'Titre, prix en FCFA, fichier (max 50 Mo). Le produit passe en modération.',
  },
  {
    num: '03',
    title: 'Recevez vos ventes',
    desc: 'Chaque vente confirmée crédite votre solde après commission de 15 %.',
  },
  {
    num: '04',
    title: 'Demandez un versement',
    desc: 'Vos gains sont versés sur votre mobile money sous 48h.',
  },
];

export default function HowItWorksPage() {
  return (
    <>
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-4xl mx-auto px-4 py-20 text-center">
          <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-4">
            // guide complet
          </p>
          <h1 className="font-display text-4xl sm:text-5xl font-bold tracking-tight">
            Comment ça marche ?
          </h1>
          <p className="mt-5 text-white/75 max-w-xl mx-auto">
            Que vous soyez acheteur, vendeur ou affilié, le fonctionnement de DIARRA tient
            en quelques étapes simples.
          </p>
        </div>
      </section>

      <section className="py-16 max-w-6xl mx-auto px-4">
        <div className="grid lg:grid-cols-2 gap-12">
          <div>
            <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-6">
              // côté acheteur
            </p>
            <div className="space-y-6">
              {BUYER_STEPS.map((step) => (
                <div key={step.num} className="flex gap-5">
                  <div className="shrink-0 w-12 h-12 rounded-xl gradient-green text-white font-mono font-bold flex items-center justify-center">
                    {step.num}
                  </div>
                  <div>
                    <h3 className="font-display font-bold text-lg text-green-950">{step.title}</h3>
                    <p className="text-green-900/70 text-sm mt-1">{step.desc}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div>
            <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-6">
              // côté vendeur
            </p>
            <div className="space-y-6">
              {SELLER_STEPS.map((step) => (
                <div key={step.num} className="flex gap-5">
                  <div className="shrink-0 w-12 h-12 rounded-xl bg-green-100 text-green-700 font-mono font-bold flex items-center justify-center">
                    {step.num}
                  </div>
                  <div>
                    <h3 className="font-display font-bold text-lg text-green-950">{step.title}</h3>
                    <p className="text-green-900/70 text-sm mt-1">{step.desc}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* Sécurité */}
      <section className="py-16 bg-green-50/60">
        <div className="max-w-6xl mx-auto px-4">
          <div className="text-center mb-12">
            <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-3">
              // sécurité & confiance
            </p>
            <h2 className="font-display text-3xl sm:text-4xl font-bold tracking-tight text-green-950">
              Un paiement protégé, une livraison maîtrisée
            </h2>
          </div>
          <div className="grid gap-6 md:grid-cols-3">
            {[
              {
                title: 'Paiement sécurisé',
                desc: 'Le mobile money est validé par notre partenaire PawaPay avant que le vendeur ne soit payé.',
              },
              {
                title: 'Livraison contrôlée',
                desc: 'Les fichiers sont stockés sur un stockage sécurisé, jamais exposés publiquement.',
              },
              {
                title: 'Support réactif',
                desc: 'Un ticket de support vous permet de signaler tout problème avec une commande.',
              },
            ].map((item) => (
              <div key={item.title} className="bg-white rounded-xl p-6 shadow-card border border-green-900/5">
                <span className="w-10 h-10 rounded-lg bg-green-500 text-white flex items-center justify-center mb-4" aria-hidden>
                  <CheckIcon size={20} />
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
          <h2 className="font-display text-3xl font-bold text-green-950 mb-4">
            Prêt à faire votre premier pas ?
          </h2>
          <div className="flex flex-wrap justify-center gap-4">
            <Link
              href="/auth/register"
              className="px-6 py-3 rounded-full bg-green-700 text-white font-semibold hover:bg-green-600 transition-colors"
            >
              Créer un compte
            </Link>
            <Link
              href="/catalog"
              className="px-6 py-3 rounded-full border-2 border-green-700 text-green-700 font-semibold hover:bg-green-50 transition-colors"
            >
              Voir le catalogue
            </Link>
          </div>
        </div>
      </section>
    </>
  );
}
