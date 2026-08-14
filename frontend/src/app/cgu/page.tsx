import Link from 'next/link';
import { ArrowLeftIcon } from '@/components/icons';

export default function CGUPage() {
  return (
    <main className="bg-paper min-h-[calc(100vh-4rem)]">
      <section className="gradient-green text-white py-10">
        <div className="max-w-3xl mx-auto px-4">
          <Link href="/" className="inline-flex items-center gap-2 text-sm text-white/70 hover:text-lime transition-colors mb-4">
            <ArrowLeftIcon size={16} />
            Retour à l&apos;accueil
          </Link>
          <p className="font-mono text-xs text-lime uppercase tracking-widest mb-2">// règles d&apos;usage</p>
          <h1 className="font-display text-3xl sm:text-4xl font-bold">Conditions générales d&apos;utilisation</h1>
        </div>
      </section>

      <section className="max-w-3xl mx-auto px-4 py-10 space-y-8 text-green-900/80 leading-relaxed">
        <p className="text-sm text-green-900/50">Dernière mise à jour : 14 août 2026</p>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">1. Objet</h2>
          <p>
            DIARRA est une plateforme éditée par <strong>ATEKOSSI BRUNEL MAHUZONSOU</strong> (NINEA 012834182,
            RCCM SN DKR 2026 A 6465) permettant à des vendeurs de proposer des biens numériques à la vente et à
            des acheteurs de les acquérir, avec paiement par mobile money.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">2. Comptes</h2>
          <p>
            L&apos;inscription est gratuite. Un compte peut évoluer de client à vendeur (ou affilié) à tout
            moment. Chaque utilisateur est responsable de la confidentialité de son mot de passe et de
            l&apos;exactitude des informations fournies.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">3. Vente de produits</h2>
          <p>
            Les vendeurs sont seuls responsables du contenu, de la légalité et de la conformité des produits
            qu&apos;ils publient. Chaque produit est soumis à modération avant publication. DIARRA se réserve
            le droit de refuser ou retirer tout produit non conforme, illicite, ou trompeur.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">4. Paiement et commission</h2>
          <p>
            Les paiements sont traités par mobile money via le prestataire PawaPay. Une commission de
            plateforme est prélevée sur chaque vente avant reversement au vendeur. Les versements aux
            vendeurs sont effectués sur le moyen mobile money qu&apos;ils ont enregistré.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">5. Remboursements</h2>
          <p>
            En cas de litige, un acheteur peut ouvrir un ticket auprès du support. Un remboursement peut être
            initié par l&apos;administration de la plateforme vers le moyen de paiement d&apos;origine.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">6. Affiliation</h2>
          <p>
            Les utilisateurs disposant du rôle affilié (« closer ») peuvent générer des liens de
            recommandation et percevoir une commission sur les ventes réalisées via ces liens, dans la limite
            fixée par chaque vendeur.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">7. Responsabilité</h2>
          <p>
            DIARRA agit en tant qu&apos;intermédiaire technique entre acheteurs et vendeurs. La plateforme ne
            saurait être tenue responsable de la qualité, du contenu ou de la conformité des produits vendus
            par des tiers.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">8. Modification des présentes conditions</h2>
          <p>
            Ces conditions peuvent être mises à jour à tout moment. La version en vigueur est celle publiée
            sur cette page.
          </p>
        </div>

        <div>
          <h2 className="font-display font-bold text-xl text-green-950 mb-2">9. Contact</h2>
          <p>
            Pour toute question : <a href="mailto:support@abmcy.com" className="text-green-700 underline">support@abmcy.com</a>.
          </p>
        </div>

        <p className="text-sm text-green-900/50 pt-4 border-t border-green-900/10">
          Voir aussi : <Link href="/mentions-legales" className="text-green-700 underline">Mentions légales</Link>
          {' · '}
          <Link href="/confidentialite" className="text-green-700 underline">Politique de confidentialité</Link>
        </p>
      </section>
    </main>
  );
}
