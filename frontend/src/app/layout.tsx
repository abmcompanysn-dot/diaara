import type { Metadata } from 'next';
import { AuthProvider } from '@/lib/auth';
import { MobileMenuProvider } from '@/lib/mobile-menu-context';
import { Header } from '@/components/header';
import { Footer } from '@/components/footer';
import './globals.css';

const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL || 'https://diarra.abmcy.com';
const SITE_DESCRIPTION =
  'Achetez et vendez des produits numériques en Afrique : clés d\'abonnement, comptes, ebooks, PDF. Paiement sécurisé par mobile money.';

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: {
    default: 'DIARRA - Le marché des biens numériques',
    template: '%s | DIARRA',
  },
  description: SITE_DESCRIPTION,
  keywords: ['Diarra', 'DIARRA', 'marketplace numérique Afrique', 'vendre produits numériques', 'mobile money'],
  openGraph: {
    type: 'website',
    locale: 'fr_FR',
    url: SITE_URL,
    siteName: 'DIARRA',
    title: 'DIARRA - Le marché des biens numériques',
    description: SITE_DESCRIPTION,
  },
  alternates: {
    canonical: SITE_URL,
  },
};

// Aide Google à reconnaître "Diarra" comme nom de marque/site (utile pour
// l'affichage enrichi quand quelqu'un recherche "diarra").
const organizationJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'Organization',
  name: 'DIARRA',
  alternateName: 'Diarra',
  url: SITE_URL,
  description: SITE_DESCRIPTION,
};

const websiteJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'WebSite',
  name: 'DIARRA',
  url: SITE_URL,
};

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
          href="https://fonts.googleapis.com/css2?family=Manrope:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;600&display=swap"
          rel="stylesheet"
        />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(organizationJsonLd) }}
        />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(websiteJsonLd) }}
        />
      </head>
      <body className="min-h-screen flex flex-col">
        <AuthProvider>
          <MobileMenuProvider>
            <Header />
            <div className="flex-1">{children}</div>
            <Footer />
          </MobileMenuProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
