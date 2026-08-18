export const metadata = {
  title: 'Questions fréquentes',
  description:
    'Livraison, paiement mobile money, remboursement, commission vendeur — les réponses aux questions les plus posées sur DIARRA.',
};

const FAQ_ITEMS = [
  {
    q: 'Combien de temps après le paiement je reçois mon fichier ?',
    a: 'Instantanément. Dès que le paiement mobile money est confirmé, un lien de téléchargement sécurisé est généré — valable 5 minutes et limité à 3 téléchargements.',
  },
  {
    q: 'Quels moyens de paiement sont acceptés ?',
    a: 'Le mobile money : Wave, Orange Money, MTN MoMo, Moov Money, M-Pesa et les principaux opérateurs du continent. Le paiement est validé par notre partenaire PawaPay avant que le vendeur ne soit crédité.',
  },
  {
    q: 'Et si le fichier ne correspond pas ou que je ne le reçois pas ?',
    a: "Ouvrez un ticket depuis votre espace \"Commandes\" ou le support. Un remboursement peut être initié par l'administration DIARRA vers votre moyen de paiement d'origine.",
  },
  {
    q: 'Comment sont vérifiés les produits vendus sur DIARRA ?',
    a: 'Chaque produit déposé passe en modération avant d\'être visible dans le catalogue. Toute modification (titre, description, prix, catégorie) le repasse automatiquement en attente de validation.',
  },
  {
    q: 'Combien coûte la vente sur DIARRA pour un vendeur ?',
    a: 'Une commission de 15 % est prélevée sur chaque vente confirmée. Le reste est crédité sur votre solde, disponible pour un versement mobile money sous 48h.',
  },
  {
    q: 'Mes données de paiement sont-elles stockées par DIARRA ?',
    a: "Non. Les paiements transitent par notre partenaire PawaPay — DIARRA ne stocke aucune information de carte ou de compte mobile money. Voir la page Confidentialité pour le détail.",
  },
];

const faqJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'FAQPage',
  mainEntity: FAQ_ITEMS.map((item) => ({
    '@type': 'Question',
    name: item.q,
    acceptedAnswer: { '@type': 'Answer', text: item.a },
  })),
};

export default function FaqPage() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(faqJsonLd) }}
      />
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-3xl mx-auto px-4 py-20 text-center">
          <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-4">
            // questions fréquentes
          </p>
          <h1 className="font-display text-4xl sm:text-5xl font-bold tracking-tight">
            Une question avant d&apos;acheter ou de vendre ?
          </h1>
          <p className="mt-5 text-white/75 max-w-xl mx-auto">
            Les réponses aux questions qu&apos;on nous pose le plus souvent. Pour le reste,
            le support répond directement.
          </p>
        </div>
      </section>

      <section className="py-16 max-w-3xl mx-auto px-4">
        <div className="divide-y divide-green-900/10">
          {FAQ_ITEMS.map((item) => (
            <div key={item.q} className="py-6">
              <h2 className="font-display font-bold text-lg text-green-950">{item.q}</h2>
              <p className="mt-2 text-green-900/70 text-sm leading-relaxed">{item.a}</p>
            </div>
          ))}
        </div>

        <div className="mt-12 text-center">
          <p className="text-green-900/70">
            Votre question n&apos;est pas dans la liste ? Écrivez-nous via la bulle de contact
            en bas de l&apos;écran.
          </p>
        </div>
      </section>
    </>
  );
}
