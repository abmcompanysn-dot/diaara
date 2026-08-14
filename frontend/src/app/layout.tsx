import type { Metadata } from 'next';
import { AuthProvider } from '@/lib/auth';
import { MobileMenuProvider } from '@/lib/mobile-menu-context';
import { Header } from '@/components/header';
import { Footer } from '@/components/footer';
import './globals.css';

export const metadata: Metadata = {
  title: 'DIARRA - Le marché des biens numériques',
  description:
    'Achetez et vendez des produits numériques en Afrique : clés d\'abonnement, comptes, ebooks, PDF. Paiement sécurisé par mobile money.',
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
