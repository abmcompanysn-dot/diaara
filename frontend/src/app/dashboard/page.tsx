'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { CartIcon, PackageIcon, StoreIcon, WalletIcon, HeadsetIcon, HomeIcon } from '@/components/icons';

const SECTIONS = [
  { href: '/catalog', title: 'Catalogue', desc: 'Parcourir et acheter des produits', Icon: CartIcon },
  { href: '/orders', title: 'Mes commandes', desc: 'Suivre mes achats et téléchargements', Icon: PackageIcon },
  { href: '/vendor/products', title: 'Espace vendeur', desc: 'Gérer mes produits et mes ventes', Icon: StoreIcon },
  { href: '/vendor/earnings', title: 'Mes revenus', desc: 'Solde et demandes de versement', Icon: WalletIcon },
  { href: '/support', title: 'Support', desc: 'Ouvrir un ticket, suivre mes demandes', Icon: HeadsetIcon },
];

export default function DashboardPage() {
  const router = useRouter();
  const [authed, setAuthed] = useState<boolean | null>(null);

  useEffect(() => {
    const token = localStorage.getItem('access_token');
    if (!token) {
      router.replace('/auth/login');
      return;
    }
    setAuthed(true);
  }, [router]);

  if (authed === null) {
    return <main className="min-h-screen flex items-center justify-center text-green-900/50">Chargement...</main>;
  }

  return (
    <main className="min-h-screen bg-paper">
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-6xl mx-auto px-4 py-16">
          <p className="font-mono text-sm text-green-300 mb-3 uppercase tracking-widest">
            // mon espace
          </p>
          <h1 className="font-display text-4xl sm:text-5xl font-bold tracking-tight">
            Bonjour
          </h1>
          <p className="mt-3 text-white/75 max-w-lg">
            Tout est au même endroit : vos achats, votre boutique, vos revenus, votre support.
          </p>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-4 py-12">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {SECTIONS.map((s) => (
            <Link
              key={s.href}
              href={s.href}
              className="group bg-white rounded-xl p-6 shadow-card hover:shadow-lift transition-all hover:-translate-y-0.5 border border-green-900/5"
            >
              <span className="w-12 h-12 rounded-lg gradient-green-soft flex items-center justify-center mb-3" aria-hidden>
                <s.Icon size={24} className="text-green-700" />
              </span>
              <h2 className="font-display font-bold text-lg text-green-950 group-hover:text-green-600 transition-colors">
                {s.title}
              </h2>
              <p className="text-sm text-green-900/60 mt-1">{s.desc}</p>
            </Link>
          ))}
          <Link
            href="/"
            className="group bg-green-950 text-white rounded-xl p-6 shadow-card hover:shadow-lift transition-all hover:-translate-y-0.5"
          >
            <span className="w-12 h-12 rounded-lg bg-white/10 flex items-center justify-center mb-3" aria-hidden>
              <HomeIcon size={24} className="text-lime" />
            </span>
            <h2 className="font-display font-bold text-lg group-hover:text-lime transition-colors">
              Retour à l&apos;accueil
            </h2>
            <p className="text-sm text-white/60 mt-1">La page de présentation du marché</p>
          </Link>
        </div>
      </section>
    </main>
  );
}
