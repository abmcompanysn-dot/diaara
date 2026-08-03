'use client';

import Link from 'next/link';
import {
  CartIcon,
  PackageIcon,
  StoreIcon,
  WalletIcon,
  HeadsetIcon,
  HomeIcon,
  MegaphoneIcon,
  ShieldIcon,
} from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
import { useAuth } from '@/lib/auth';
import { RequireAuth } from '@/lib/guards';

export default function DashboardPage() {
  const { user, isAdmin, hasRole } = useAuth();

  const sections = [
    { href: '/catalog', title: 'Catalogue', desc: 'Parcourir et acheter des produits', Icon: CartIcon, show: true },
    { href: '/orders', title: 'Mes commandes', desc: 'Suivre mes achats et téléchargements', Icon: PackageIcon, show: true },
    { href: '/vendor/products', title: 'Espace vendeur', desc: 'Gérer mes produits et mes ventes', Icon: StoreIcon, show: hasRole('vendeur') },
    { href: '/vendor/earnings', title: 'Mes revenus', desc: 'Solde et demandes de versement', Icon: WalletIcon, show: hasRole('vendeur') },
    { href: '/closer/dashboard', title: 'Affiliation', desc: 'Mes liens et mes commissions', Icon: MegaphoneIcon, show: hasRole('closer') },
    { href: '/admin', title: 'Administration', desc: 'Modération et gestion de la plateforme', Icon: ShieldIcon, show: isAdmin },
    { href: '/support', title: 'Support', desc: 'Ouvrir un ticket, suivre mes demandes', Icon: HeadsetIcon, show: true },
  ].filter((s) => s.show);

  return (
    <RequireAuth>
      <main className="min-h-screen bg-paper">
        <section className="gradient-green text-white relative overflow-hidden">
          <div className="wax-pattern absolute inset-0" aria-hidden />
          <div className="relative max-w-6xl mx-auto px-4 py-16">
            <p className="font-mono text-sm text-green-300 mb-3 uppercase tracking-widest">
              // mon espace
            </p>
            <h1 className="font-display text-4xl sm:text-5xl font-bold tracking-tight">
              Bonjour {user?.email}
            </h1>
            <p className="mt-3 text-white/75 max-w-lg">
              Tout est au même endroit : vos achats, votre boutique, vos revenus, votre support.
            </p>
          </div>
        </section>

        <section className="max-w-6xl mx-auto px-4 py-12">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {sections.map((s) => (
              <Link key={s.href} href={s.href} className="group focus:outline-none">
                <Card className="h-full shadow-card border-green-900/5 hover:shadow-lift transition-all hover:-translate-y-0.5 group-hover:border-ring/50">
                  <CardContent className="p-6">
                    <span
                      className="w-12 h-12 rounded-lg gradient-green-soft flex items-center justify-center mb-3"
                      aria-hidden
                    >
                      <s.Icon size={24} className="text-green-700" />
                    </span>
                    <h2 className="font-display font-bold text-lg text-green-950 group-hover:text-green-600 transition-colors">
                      {s.title}
                    </h2>
                    <p className="text-sm text-green-900/60 mt-1">{s.desc}</p>
                  </CardContent>
                </Card>
              </Link>
            ))}
            <Link href="/" className="group focus:outline-none">
              <Card className="h-full bg-green-950 border-green-800 shadow-card hover:shadow-lift transition-all hover:-translate-y-0.5">
                <CardContent className="p-6">
                  <span className="w-12 h-12 rounded-lg bg-white/10 flex items-center justify-center mb-3" aria-hidden>
                    <HomeIcon size={24} className="text-lime" />
                  </span>
                  <h2 className="font-display font-bold text-lg text-white group-hover:text-lime transition-colors">
                    Retour à l&apos;accueil
                  </h2>
                  <p className="text-sm text-white/60 mt-1">La page de présentation du marché</p>
                </CardContent>
              </Card>
            </Link>
          </div>
        </section>
      </main>
    </RequireAuth>
  );
}
