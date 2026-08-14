'use client';

import Link from 'next/link';
import { HomeIcon } from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
import { useAuth } from '@/lib/auth';
import { RequireAuth } from '@/lib/guards';
import { PageHeader } from '@/components/page-header';
import { userRoleLabels, visibleNavItems } from '@/lib/navigation';

export default function DashboardPage() {
  const { user } = useAuth();
  const roles = userRoleLabels(user);

  // Les cartes reproduisent la navigation de la sidebar (cohérence totale).
  const sections = visibleNavItems(user)
    .filter((l) => l.href !== '/dashboard')
    .map((l) => ({ href: l.href, title: l.label, desc: '', Icon: l.Icon }));

  const descriptions: Record<string, string> = {
    '/catalog': 'Parcourir et acheter des produits',
    '/orders': 'Suivre mes achats et téléchargements',
    '/vendor': 'Gérer ma boutique, mes produits et mes ventes',
    '/vendor/earnings': 'Solde et demandes de versement',
    '/closer/dashboard': 'Mes liens et mes commissions',
    '/admin': 'Modération et gestion de la plateforme',
    '/support': 'Ouvrir un ticket, suivre mes demandes',
    '/account': 'Profil, boutique et moyen de retrait',
  };

  return (
    <RequireAuth>
      <main className="min-h-screen bg-paper">
        <PageHeader
          eyebrow="// mon espace"
          title={`Bonjour ${user?.email}`}
          description="Tout est au même endroit : vos achats, votre boutique, vos revenus, votre support."
        />

        <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
          <div className="mb-6">
            <p className="text-[11px] uppercase tracking-widest text-green-900/40 font-mono mb-2">
              votre statut
            </p>
            <div className="flex flex-wrap gap-1.5">
              {roles.map((role) => (
                <span
                  key={role}
                  className="px-2.5 py-1 rounded-full bg-green-900/5 border border-green-900/10 text-xs font-medium text-green-900/80"
                >
                  {role}
                </span>
              ))}
            </div>
          </div>

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
                    <p className="text-sm text-green-900/60 mt-1">{descriptions[s.href]}</p>
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