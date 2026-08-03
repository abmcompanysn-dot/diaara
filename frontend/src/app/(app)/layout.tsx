'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import { useAuth } from '@/lib/auth';
import { RequireAuth } from '@/lib/guards';
import {
  HomeIcon,
  CartIcon,
  PackageIcon,
  StoreIcon,
  WalletIcon,
  MegaphoneIcon,
  ShieldIcon,
  HeadsetIcon,
} from '@/components/icons';

const BASE_ICON = 'w-5 h-5 shrink-0';

export default function ClientLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { user, isAdmin, hasRole } = useAuth();

  const links = [
    { href: '/dashboard', label: 'Mon espace', Icon: HomeIcon, show: true },
    { href: '/catalog', label: 'Catalogue', Icon: CartIcon, show: true },
    { href: '/orders', label: 'Mes commandes', Icon: PackageIcon, show: true },
    { href: '/vendor/products', label: 'Espace vendeur', Icon: StoreIcon, show: hasRole('vendeur') },
    { href: '/vendor/earnings', label: 'Mes revenus', Icon: WalletIcon, show: hasRole('vendeur') },
    { href: '/closer/dashboard', label: 'Affiliation', Icon: MegaphoneIcon, show: hasRole('closer') },
    { href: '/admin', label: 'Administration', Icon: ShieldIcon, show: isAdmin },
    { href: '/support', label: 'Support', Icon: HeadsetIcon, show: true },
  ].filter((l) => l.show);

  const isActive = (href: string) =>
    href === '/dashboard' ? pathname === href : pathname === href || pathname.startsWith(href + '/');

  return (
    <RequireAuth>
      <div className="min-h-screen bg-paper">
        <div className="max-w-7xl mx-auto flex flex-col md:flex-row">
          {/* Sidebar (desktop) */}
          <aside className="hidden md:block w-64 shrink-0 border-r border-green-900/10 py-8 pr-6">
            <nav className="space-y-1 sticky top-24" aria-label="Navigation de l'espace">
              {links.map((l) => (
                <Link
                  key={l.href}
                  href={l.href}
                  aria-current={isActive(l.href) ? 'page' : undefined}
                  className={cn(
                    'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                    isActive(l.href)
                      ? 'bg-green-950 text-white'
                      : 'text-green-900/70 hover:bg-green-900/5 hover:text-green-950'
                  )}
                >
                  <l.Icon size={20} className={BASE_ICON} />
                  {l.label}
                </Link>
              ))}
            </nav>
            <div className="sticky top-24 mt-8 border-t border-green-900/10 pt-4 space-y-3">
              <Link
                href="/"
                className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium text-green-900/70 hover:bg-green-900/5 hover:text-green-950 transition-colors"
              >
                <HomeIcon size={20} className={BASE_ICON} />
                Retour à l&apos;accueil
              </Link>
              {user && (
                <p className="px-3 text-xs text-green-900/50 truncate" title={user.email}>
                  {user.email}
                </p>
              )}
            </div>
          </aside>

          {/* Contenu */}
          <main className="flex-1 min-w-0">
            {/* Barre de navigation mobile */}
            <div className="md:hidden overflow-x-auto border-b border-green-900/10 py-2 px-4">
              <div className="flex gap-2">
                {links.map((l) => (
                  <Link
                    key={l.href}
                    href={l.href}
                    className={cn(
                      'flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-colors',
                      isActive(l.href)
                        ? 'bg-green-950 text-white'
                        : 'bg-white text-green-900/70 border border-green-900/10'
                    )}
                  >
                    <l.Icon size={14} />
                    {l.label}
                  </Link>
                ))}
              </div>
            </div>
            {children}
          </main>
        </div>
      </div>
    </RequireAuth>
  );
}
