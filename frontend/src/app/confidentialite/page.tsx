import Link from 'next/link';
import { ArrowLeftIcon } from '@/components/icons';

export default function ConfidentialitePage() {
  return (
    <main className="bg-paper min-h-[calc(100vh-4rem)]">
      <section className="gradient-green text-white py-10">
        <div className="max-w-3xl mx-auto px-4">
          <Link href="/" className="inline-flex items-center gap-2 text-sm text-white/70 hover:text-lime transition-colors mb-4">
            <ArrowLeftIcon size={16} />
            Retour à l&apos;accueil
          </Link>
          <p className="font-mono text-xs text-lime uppercase tracking-widest mb-2">// vos données</p>
          <h1 className="font-display text-3xl sm:text-4xl font-bold">Politique de confidentialité</h1>
        </div>
      </section>

      <section className="max-w-3xl mx-auto px-4 py-10 space-y-8 text-green-900/80 leading-relaxed">
        <p className="text-sm text-green-900/50">Dernière mise à jour : 14 août 2026</p>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">Qui traite vos données</h2>
          <p>
            Les données collectées sur DIARRA sont traitées par <strong>ATEKOSSI BRUNEL MAHUZONSOU</strong>
            (NINEA 012834182), éditeur de la plateforme, dont le siège est à Dakar Médina, Rue 13 x 12, Sénégal.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">Données collectées</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>Informations de compte : email, numéro de téléphone, nom, pays.</li>
            <li>Informations de boutique (vendeurs) : nom de boutique, produits, moyen de versement mobile money.</li>
            <li>Informations de commande : nom de l&apos;acheteur, produit acheté, montant, référence de paiement.</li>
            <li>Données techniques : adresse IP, journaux de connexion, à des fins de sécurité uniquement.</li>
          </ul>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">Pourquoi nous les utilisons</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>Créer et sécuriser votre compte, vous permettre de vous reconnecter.</li>
            <li>Traiter vos achats, versements et remboursements via notre prestataire de paiement PawaPay.</li>
            <li>Vous envoyer les emails liés à votre activité (confirmation de commande, versement, support).</li>
            <li>Vous mettre en relation avec un vendeur ou un client via la messagerie de la plateforme.</li>
            <li>Prévenir la fraude et respecter nos obligations légales.</li>
          </ul>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">Partage des données</h2>
          <p>
            Vos données ne sont jamais vendues. Elles sont partagées uniquement avec les prestataires
            nécessaires au fonctionnement du service : PawaPay (paiement mobile money), notre prestataire
            d&apos;envoi d&apos;emails, et Firebase/Google lorsque vous utilisez la connexion Google ou la
            messagerie en direct.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">Vos droits</h2>
          <p>
            Vous pouvez demander l&apos;accès, la correction ou la suppression de vos données personnelles
            à tout moment en écrivant à <a href="mailto:support@abmcy.com" className="text-green-700 underline">support@abmcy.com</a>.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">Conservation</h2>
          <p>
            Les données de compte et de transaction sont conservées le temps nécessaire à la gestion de la
            relation commerciale et au respect de nos obligations comptables et fiscales.
          </p>
        </div>

        <p className="text-sm text-green-900/50 pt-4 border-t border-green-900/10">
          Voir aussi : <Link href="/mentions-legales" className="text-green-700 underline">Mentions légales</Link>
          {' · '}
          <Link href="/cgu" className="text-green-700 underline">Conditions générales d&apos;utilisation</Link>
        </p>
      </section>
    </main>
  );
}
