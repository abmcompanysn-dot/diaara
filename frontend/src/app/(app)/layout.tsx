'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import { useAuth } from '@/lib/auth';
import { RequireAuth } from '@/lib/guards';
import { NAV_SECTIONS, userRoleLabels, visibleNavItems } from '@/lib/navigation';
import { useMobileMenu } from '@/lib/mobile-menu-context';
import { HomeIcon, CartIcon, PackageIcon, StoreIcon, ShieldIcon, MenuIcon } from '@/components/icons';

const BASE_ICON = 'w-5 h-5 shrink-0';

export default function ClientLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { user } = useAuth();
  const { setOpen: setMenuOpen } = useMobileMenu();

  const links = visibleNavItems(user);
  const roles = userRoleLabels(user);

  const isActive = (href: string) =>
    href === '/dashboard' ? pathname === href : pathname === href || pathname.startsWith(href + '/');

  // Barre du bas mobile : 3 raccourcis universels + 1 raccourci contextuel selon le rôle
  // le plus pertinent + le menu (qui ouvre le tiroir complet du header, toutes les sections).
  // Limité à 5 icônes pour éviter la troncature de texte sur petit écran (voir header.tsx).
  const contextualItem = user?.roles?.includes('vendeur')
    ? { href: '/vendor/products', label: 'Vendre', Icon: StoreIcon }
    : user?.is_admin
      ? { href: '/admin', label: 'Admin', Icon: ShieldIcon }
      : null;

  const mobileNavItems = [
    { href: '/dashboard', label: 'Accueil', Icon: HomeIcon },
    { href: '/catalog', label: 'Catalogue', Icon: CartIcon },
    { href: '/orders', label: 'Commandes', Icon: PackageIcon },
    ...(contextualItem ? [contextualItem] : []),
  ];

  return (
    <RequireAuth>
      <div className="min-h-screen bg-paper">
        <div className="max-w-6xl mx-auto flex flex-col md:flex-row">
          {/* Sidebar (desktop) */}
          <aside className="hidden md:block w-64 shrink-0 border-r border-green-900/10 py-8 pr-6">
            <div className="sticky top-24">
              {roles.length > 0 && (
                <div className="mb-6 px-3">
                  <p className="text-[11px] uppercase tracking-widest text-green-900/40 font-mono mb-2">
                    votre statut
                  </p>
                  <div className="flex flex-wrap gap-1.5">
                    {roles.map((role) => (
                      <span
                        key={role}
                        className="px-2 py-0.5 rounded-full bg-green-900/5 border border-green-900/10 text-xs font-medium text-green-900/80"
                      >
                        {role}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              <nav className="space-y-4" aria-label="Navigation de l'espace">
                {NAV_SECTIONS.map((section) => {
                  const items = links.filter((l) => l.section === section.id);
                  if (items.length === 0) return null;
                  return (
                    <div key={section.id}>
                      {section.label && (
                        <p className="px-3 pb-1.5 text-[11px] uppercase tracking-widest text-green-900/40 font-mono">
                          {section.label}
                        </p>
                      )}
                      <div className="space-y-1">
                        {items.map((l) => (
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
                      </div>
                    </div>
                  );
                })}
              </nav>

              <div className="mt-8 border-t border-green-900/10 pt-4 space-y-3">
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
            </div>
          </aside>

          {/* Contenu : pb-24 (96px) pour ne jamais laisser la barre du bas fixe recouvrir le
              contenu en fin de défilement (voir espace vendeur / admin, ergonomie mobile). */}
          <main className="flex-1 min-w-0 pb-24 md:pb-0">

            {/* Bannière de vérification en attente */}
            {user && !user.email_verified_at && (
              <Link
                href="/auth/verify-email"
                className="block bg-amber-50 border-b border-amber-200 px-4 py-2.5 text-sm text-amber-800 hover:bg-amber-100 transition-colors"
              >
                <span className="font-semibold">Email non vérifié —</span>{' '}
                entrez votre code pour activer votre compte →
              </Link>
            )}

            {children}
          </main>
        </div>
      </div>

      {/* Barre de navigation mobile (bas d'écran, façon application) : 4-5 raccourcis
          universels maximum + "Menu" qui ouvre le tiroir complet (header.tsx) pour les
          liens spécialisés (Administration, Support, Affiliation...). Évite la troncature
          de texte et le débordement horizontal observés avec les 7 liens précédents. */}
      <nav
        className="md:hidden fixed bottom-0 inset-x-0 z-50 bg-white border-t border-green-900/10 pb-[env(safe-area-inset-bottom)]"
        aria-label="Navigation mobile"
      >
        <div className="flex">
          {mobileNavItems.map((l) => {
            const active = isActive(l.href);
            return (
              <Link
                key={l.href}
                href={l.href}
                aria-current={active ? 'page' : undefined}
                className={cn(
                  'flex-1 flex flex-col items-center justify-center gap-1 py-2.5 transition-colors',
                  active ? 'text-green-700' : 'text-green-900/45'
                )}
              >
                <l.Icon size={21} className={active ? 'text-green-700' : 'text-green-900/45'} />
                <span className="text-[10.5px] font-medium leading-none truncate max-w-16 px-1">
                  {l.label}
                </span>
              </Link>
            );
          })}
          <button
            type="button"
            onClick={() => setMenuOpen(true)}
            className="flex-1 flex flex-col items-center justify-center gap-1 py-2.5 text-green-900/45 transition-colors"
          >
            <MenuIcon size={21} />
            <span className="text-[10.5px] font-medium leading-none truncate max-w-16 px-1">Menu</span>
          </button>
        </div>
      </nav>
    </RequireAuth>
  );
}
