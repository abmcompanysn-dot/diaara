'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

export function Footer() {
  const pathname = usePathname();

  // Même raisonnement que le Header : l'espace vendeur se comporte comme une
  // app autonome, sans le pied de page marketing du reste du site.
  if (pathname.startsWith('/vendor')) return null;

  return (
    <footer className="bg-green-950 text-white/70">
      <div className="wax-pattern h-4" aria-hidden />
      <div className="max-w-6xl mx-auto px-4 py-12">
        <div className="grid gap-8 md:grid-cols-4">
          <div>
            <img src="/brand/diarra-logo-white-text.svg" alt="DIARRA" className="h-7 w-auto mb-3" />
            <p className="text-sm">
              Le marché des biens numériques en Afrique. Payez en mobile money, recevez
              instantanément.
            </p>
          </div>
          <div>
            <p className="font-semibold text-green-300 mb-3 text-sm">Acheter</p>
            <ul className="space-y-2 text-sm">
              <li><Link href="/catalog" className="hover:text-green-300">Catalogue</Link></li>
              <li><Link href="/how-it-works" className="hover:text-green-300">Comment ça marche</Link></li>
              <li><Link href="/faq" className="hover:text-green-300">Questions fréquentes</Link></li>
            </ul>
          </div>
          <div>
            <p className="font-semibold text-green-300 mb-3 text-sm">Vendre</p>
            <ul className="space-y-2 text-sm">
              <li><Link href="/sell" className="hover:text-green-300">Vendre sur DIARRA</Link></li>
              <li><Link href="/closer" className="hover:text-green-300">Affiliation</Link></li>
            </ul>
          </div>
          <div>
            <p className="font-semibold text-green-300 mb-3 text-sm">Paiements acceptés</p>
            <div className="flex flex-wrap items-center gap-2">
              {[
                { logo: 'orange-money.png', label: 'Orange Money' },
                { logo: 'mtn-momo.png', label: 'MTN MoMo' },
                { logo: 'moov-money.png', label: 'Moov Money' },
                { logo: 'wave.png', label: 'Wave' },
                { logo: 'safaricom-mpesa.png', label: 'M-Pesa' },
                { logo: 'vodacom.png', label: 'Vodacom' },
                { logo: 'at-money.png', label: 'Airtel Money' },
                { logo: 'tnm.png', label: 'TNM Mpamba' },
                { logo: 'zamtel.png', label: 'Zamtel Money' },
                { logo: 'movitel.png', label: 'Movitel' },
                { logo: 'halopesa.png', label: 'HaloPesa' },
              ].map((op) => (
                <span key={op.logo} className="h-6 px-1.5 rounded bg-white/90 flex items-center" title={op.label}>
                  <img src={`/payments/${op.logo}`} alt={op.label} className="h-4 max-w-[56px] object-contain" />
                </span>
              ))}
            </div>
          </div>
        </div>
        <div className="mt-10 pt-6 border-t border-white/10 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 text-xs">
          <div className="flex flex-col sm:flex-row gap-1 sm:gap-4">
            <p>© {new Date().getFullYear()} DIARRA. Tous droits réservés.</p>
            <div className="flex gap-3">
              <Link href="/mentions-legales" className="hover:text-lime">Mentions légales</Link>
              <Link href="/confidentialite" className="hover:text-lime">Confidentialité</Link>
              <Link href="/cgu" className="hover:text-lime">CGU</Link>
            </div>
          </div>
          <div className="flex items-center gap-4">
            {process.env.NEXT_PUBLIC_TIKTOK_URL && (
              <a
                href={process.env.NEXT_PUBLIC_TIKTOK_URL}
                target="_blank"
                rel="noopener noreferrer"
                aria-label="DIARRA sur TikTok"
                className="text-white/60 hover:text-lime transition-colors"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
                  <path d="M16.6 5.82s.51.5 0 0A4.278 4.278 0 0 1 15.54 3h-3.09v12.4a2.592 2.592 0 0 1-2.59 2.5c-1.42 0-2.6-1.16-2.6-2.6 0-1.72 1.66-3.01 3.37-2.48V9.66c-3.45-.46-6.47 2.22-6.47 5.64 0 3.33 2.76 5.7 5.69 5.7 3.14 0 5.69-2.55 5.69-5.7V9.01a7.35 7.35 0 0 0 4.3 1.38V7.3s-1.88.09-3.24-1.48z" />
                </svg>
              </a>
            )}
            <p className="font-mono text-white/40">paiement sécurisé · livraison instantanée</p>
          </div>
        </div>
      </div>
    </footer>
  );
}
