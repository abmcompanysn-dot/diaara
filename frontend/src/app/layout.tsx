import type { Metadata } from 'next';
import Link from 'next/link';
import './globals.css';

export const metadata: Metadata = {
  title: 'DIARRA - Le marché des biens numériques',
  description:
    'Achetez et vendez des produits numériques en Afrique : clés d\'abonnement, comptes, ebooks, PDF. Paiement sécurisé par mobile money.',
};

const NAV_LINKS = [
  { href: '/catalog', label: 'Catalogue' },
  { href: '/how-it-works', label: 'Comment ça marche' },
  { href: '/sell', label: 'Vendre' },
  { href: '/closer', label: 'Affiliation' },
];

function Header() {
  return (
    <header className="sticky top-0 z-50 bg-green-950/95 backdrop-blur border-b border-green-400/20">
      <div className="max-w-6xl mx-auto px-4 h-16 flex items-center justify-between">
        <Link href="/" className="flex items-center gap-2 group">
          <span className="font-display text-2xl font-bold text-white tracking-tight">
            DIARRA
          </span>
          <span className="w-2 h-2 rounded-full bg-green-400 inline-block group-hover:scale-150 transition-transform" aria-hidden />
        </Link>

        <nav className="hidden md:flex items-center gap-8" aria-label="Navigation principale">
          {NAV_LINKS.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="text-sm text-white/80 hover:text-green-300 transition-colors"
            >
              {link.label}
            </Link>
          ))}
        </nav>

        <div className="flex items-center gap-3">
          <Link
            href="/auth/login"
            className="hidden sm:inline text-sm text-white/80 hover:text-green-300 transition-colors"
          >
            Connexion
          </Link>
          <Link
            href="/auth/register"
            className="px-4 py-2 rounded-full bg-green-400 text-green-950 text-sm font-semibold hover:bg-green-300 transition-colors"
          >
            Créer un compte
          </Link>
        </div>
      </div>
    </header>
  );
}

function Footer() {
  return (
    <footer className="bg-green-950 text-white/70">
      <div className="wax-pattern h-4" aria-hidden />
      <div className="max-w-6xl mx-auto px-4 py-12">
        <div className="grid gap-8 md:grid-cols-4">
          <div>
            <p className="font-display text-2xl font-bold text-white mb-2">DIARRA</p>
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
            <div className="flex flex-wrap gap-2 text-xs">
              {['Wave', 'Orange Money', 'MTN MoMo'].map((p) => (
                <span key={p} className="px-2 py-1 rounded bg-green-800 text-white/80">
                  {p}
                </span>
              ))}
            </div>
          </div>
        </div>
        <div className="mt-10 pt-6 border-t border-white/10 flex flex-col sm:flex-row justify-between gap-2 text-xs">
          <p>© {new Date().getFullYear()} DIARRA. Tous droits réservés.</p>
          <p className="font-mono text-white/40">paiement sécurisé · livraison instantanée</p>
        </div>
      </div>
    </footer>
  );
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="fr">
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:opsz,wght@12..96,600;12..96,700;12..96,800&family=Work+Sans:wght@400;500;600;700&family=Space+Mono:wght@400;700&display=swap"
          rel="stylesheet"
        />
      </head>
      <body className="min-h-screen flex flex-col">
        <Header />
        <div className="flex-1">{children}</div>
        <Footer />
      </body>
    </html>
  );
}
