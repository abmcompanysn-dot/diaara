import Link from 'next/link';
import { ArrowLeftIcon } from '@/components/icons';

export default function MentionsLegalesPage() {
  return (
    <main className="bg-paper min-h-[calc(100vh-4rem)]">
      <section className="gradient-green text-white py-10">
        <div className="max-w-3xl mx-auto px-4">
          <Link href="/" className="inline-flex items-center gap-2 text-sm text-white/70 hover:text-lime transition-colors mb-4">
            <ArrowLeftIcon size={16} />
            Retour à l&apos;accueil
          </Link>
          <p className="font-mono text-xs text-lime uppercase tracking-widest mb-2">// informations légales</p>
          <h1 className="font-display text-3xl sm:text-4xl font-bold">Mentions légales</h1>
        </div>
      </section>

      <section className="max-w-3xl mx-auto px-4 py-10 space-y-8 text-green-900/80 leading-relaxed">
        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">Éditeur du site</h2>
          <p>
            Le site DIARRA (diarra.abmcy.com) est édité par <strong>ATEKOSSI BRUNEL MAHUZONSOU</strong>,
            entreprise individuelle immatriculée au Sénégal.
          </p>
          <ul className="mt-3 space-y-1">
            <li><strong>NINEA :</strong> 012834182</li>
            <li><strong>RCCM :</strong> SN DKR 2026 A 6465</li>
            <li><strong>Date d&apos;immatriculation :</strong> 13/02/2026</li>
            <li><strong>Siège :</strong> Dakar Médina, Rue 13 x 12, Dakar, Sénégal</li>
            <li><strong>Téléphone :</strong> +221 77 758 79 99</li>
            <li><strong>Contact :</strong> <a href="mailto:support@abmcy.com" className="text-green-700 underline">support@abmcy.com</a></li>
            <li><strong>Activités déclarées :</strong> Commerce général, import-export, prestations de services, développement logiciel et solutions numériques, organisation de voyages, événementiel, e-commerce.</li>
            <li><strong>Gérant :</strong> Brunel Mahuzonsou Atekossi</li>
          </ul>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">Hébergement</h2>
          <p>Le site est hébergé sur une infrastructure de serveurs privés (VPS) située en Europe.</p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">Activité de la plateforme</h2>
          <p>
            DIARRA est une place de marché en ligne permettant l&apos;achat et la vente de biens numériques
            (fichiers, licences, abonnements) entre particuliers et professionnels, avec paiement par mobile
            money via le prestataire PawaPay.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">Propriété intellectuelle</h2>
          <p>
            La marque DIARRA, son logo et l&apos;ensemble des éléments graphiques du site sont la propriété
            de l&apos;éditeur. Les produits vendus sur la plateforme restent la propriété intellectuelle de
            leurs vendeurs respectifs, sauf mention contraire.
          </p>
        </div>

        <p className="text-sm text-green-900/50 pt-4 border-t border-green-900/10">
          Voir aussi : <Link href="/confidentialite" className="text-green-700 underline">Politique de confidentialité</Link>
          {' · '}
          <Link href="/cgu" className="text-green-700 underline">Conditions générales d&apos;utilisation</Link>
        </p>
      </section>
    </main>
  );
}
