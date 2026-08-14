'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useRef } from 'react';
import { useAuth } from '@/lib/auth';
import { useMobileMenu } from '@/lib/mobile-menu-context';
import { Button } from '@/components/ui/button';
import { CartIcon, HomeIcon, StoreIcon, WalletIcon, ShieldIcon, HeadsetIcon, ZapIcon, MegaphoneIcon, UserIcon } from '@/components/icons';
import { MenuIcon, XIcon, ChevronRightIcon } from '@/components/icons';
import { NotificationBell } from '@/components/notification-bell';
import { cn } from '@/lib/utils';

interface DrawerLink {
  href: string;
  label: string;
  Icon: typeof HomeIcon;
}

interface DrawerSection {
  id: string;
  label: string;
  links: DrawerLink[];
}

export function Header() {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const { open: menuOpen, setOpen: setMenuOpen } = useMobileMenu();
  const panelRef = useRef<HTMLDivElement>(null);

  // Liens publics affichés dans la nav desktop (toujours visibles).
  const publicLinks: DrawerLink[] = [
    { href: '/catalog', label: 'Catalogue', Icon: CartIcon },
    { href: '/how-it-works', label: 'Comment ça marche', Icon: ZapIcon },
  ];

  // Sections du tiroir mobile, regroupées par rôle (spec : Acheteur / Vendeur / Admin / Assistance).
  const sections: DrawerSection[] = [
    {
      id: 'buyer',
      label: 'Navigation',
      links: user
        ? [
            { href: '/dashboard', label: 'Mon espace', Icon: HomeIcon },
            { href: '/catalog', label: 'Catalogue des produits', Icon: CartIcon },
            { href: '/how-it-works', label: 'Comment ça marche', Icon: ZapIcon },
            { href: '/orders', label: 'Mes achats / commandes', Icon: WalletIcon },
          ]
        : publicLinks,
    },
    {
      id: 'vendeur',
      label: 'Espace vendeur',
      links: user?.roles?.includes('vendeur')
        ? [
            { href: '/vendor/products', label: 'Mes produits', Icon: StoreIcon },
            { href: '/vendor/earnings', label: 'Mes revenus / versements', Icon: WalletIcon },
          ]
        : [],
    },
    {
      id: 'closer',
      label: 'Affiliation',
      links: user?.roles?.includes('closer')
        ? [{ href: '/closer/dashboard', label: 'Mes liens & commissions', Icon: MegaphoneIcon }]
        : [],
    },
    {
      id: 'admin',
      label: 'Administration',
      links: user?.is_admin
        ? [
            { href: '/admin', label: 'Tableau de bord admin', Icon: ShieldIcon },
            { href: '/admin/users', label: 'Gestion utilisateurs', Icon: UserIcon },
            { href: '/admin/products', label: 'Modération produits', Icon: CartIcon },
            { href: '/admin/sales', label: 'Audit des ventes', Icon: WalletIcon },
            { href: '/admin/permissions', label: 'Droits admin', Icon: ShieldIcon },
          ]
        : [],
    },
    {
      id: 'support',
      label: 'Assistance',
      links: [{ href: '/support', label: 'Support & contact', Icon: HeadsetIcon }],
    },
  ];

  const visibleSections = sections.filter((s) => s.links.length > 0);

  const handleLogout = async () => {
    setMenuOpen(false);
    await logout();
    router.push('/');
  };

  // Bloque le scroll + fermeture par Échap quand le drawer est ouvert.
  useEffect(() => {
    if (!menuOpen) return;
    document.body.style.overflow = 'hidden';
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setMenuOpen(false);
    };
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.body.style.overflow = '';
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [menuOpen]);

  const isActive = (href: string) =>
    href === '/' ? pathname === '/' : pathname === href || pathname.startsWith(href + '/');

  return (
    <header className="sticky top-0 z-50 bg-green-950/95 backdrop-blur border-b border-green-400/20">
      <div className="max-w-6xl mx-auto px-4 h-16 flex items-center justify-between gap-4">
        <Link
          href="/"
          className="flex items-center group"
          onClick={() => setMenuOpen(false)}
        >
          <img src="/brand/diarra-logo-white-text.svg" alt="DIARRA" className="h-7 w-auto" />
        </Link>

        <nav className="hidden md:flex items-center gap-6" aria-label="Navigation principale">
          {publicLinks.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              aria-current={isActive(link.href) ? 'page' : undefined}
              className={cn(
                'flex items-center gap-1.5 text-sm text-white/80 hover:text-green-300 transition-colors',
                isActive(link.href) && 'text-green-300 font-medium'
              )}
            >
              <link.Icon size={16} className="shrink-0" />
              {link.label}
            </Link>
          ))}
          {user && (
            <Link
              href="/dashboard"
              aria-current={isActive('/dashboard') ? 'page' : undefined}
              className={cn(
                'flex items-center gap-1.5 text-sm text-white/80 hover:text-green-300 transition-colors',
                isActive('/dashboard') && 'text-green-300 font-medium'
              )}
            >
              <HomeIcon size={16} className="shrink-0" />
              Mon espace
            </Link>
          )}
        </nav>

        {/* E-mail + déconnexion : uniquement desktop, le mobile les retrouve dans le tiroir */}
        <div className="flex items-center gap-3">
          {user && <NotificationBell />}
          {user ? (
            <>
              <span className="hidden md:inline text-xs text-white/50 truncate max-w-40">
                {user.email}
              </span>
              <Button
                onClick={handleLogout}
                variant="outline"
                size="sm"
                className="hidden md:inline-flex border-white/25 bg-transparent text-white hover:bg-white/10 hover:text-lime hover:border-lime"
              >
                Déconnexion
              </Button>
            </>
          ) : (
            <>
              <Button
                variant="ghost"
                size="sm"
                render={<Link href="/auth/login" />}
                className="hidden sm:inline-flex text-white/80 hover:text-green-300 hover:bg-white/10"
              >
                Connexion
              </Button>
              <Button
                size="sm"
                render={<Link href="/auth/register" />}
                className="bg-green-400 text-green-950 font-semibold hover:bg-green-300"
              >
                Créer un compte
              </Button>
            </>
          )}

          {/* Bouton menu mobile */}
          <button
            type="button"
            onClick={() => setMenuOpen(true)}
            aria-label="Ouvrir le menu"
            aria-expanded={menuOpen}
            className="md:hidden inline-flex items-center justify-center w-10 h-10 rounded-lg text-white/80 hover:text-white hover:bg-white/10 transition-colors"
          >
            <MenuIcon size={22} />
          </button>
        </div>
      </div>

      {/* Drawer mobile */}
      {menuOpen && (
        <div className="md:hidden fixed inset-0 z-[60]" role="dialog" aria-modal="true" aria-label="Menu">
          <div
            className="absolute inset-0 bg-green-950/60 backdrop-blur-sm"
            onClick={() => setMenuOpen(false)}
            aria-hidden
          />
          <div
            ref={panelRef}
            className="absolute right-0 top-0 h-full w-72 max-w-[85vw] bg-green-950 text-white shadow-lift flex flex-col animate-in slide-in-from-right duration-200"
          >
            <div className="flex items-center justify-between px-5 h-16 border-b border-white/10">
              <img src="/brand/diarra-logo-white-text.svg" alt="DIARRA" className="h-6 w-auto" />
              <button
                type="button"
                onClick={() => setMenuOpen(false)}
                aria-label="Fermer le menu"
                className="inline-flex items-center justify-center w-10 h-10 rounded-lg text-white/80 hover:text-white hover:bg-white/10 transition-colors"
              >
                <XIcon size={22} />
              </button>
            </div>

            {user && (
              <div className="px-5 py-4 border-b border-white/10">
                <div className="flex items-center gap-3">
                  <span
                    className="w-10 h-10 rounded-full bg-lime/15 text-lime flex items-center justify-center font-display font-bold shrink-0"
                    aria-hidden
                  >
                    {(user.email || '?').charAt(0).toUpperCase()}
                  </span>
                  <p className="text-sm text-white/90 truncate min-w-0">{user.email}</p>
                </div>
                {(user.is_admin || (user.roles && user.roles.length > 0)) && (
                  <div className="flex flex-wrap gap-1.5 mt-3">
                    {user.is_admin && (
                      <span className="px-2 py-0.5 rounded-full bg-lime/15 text-lime text-[11px] font-medium">Admin</span>
                    )}
                    {user.roles?.includes('vendeur') && (
                      <span className="px-2 py-0.5 rounded-full bg-white/10 text-white/70 text-[11px] font-medium">Vendeur</span>
                    )}
                    {user.roles?.includes('closer') && (
                      <span className="px-2 py-0.5 rounded-full bg-white/10 text-white/70 text-[11px] font-medium">Affilié</span>
                    )}
                  </div>
                )}
              </div>
            )}

            <nav className="flex-1 overflow-y-auto py-3 px-3 space-y-4" aria-label="Navigation mobile">
              {visibleSections.map((section) => (
                <div key={section.id}>
                  <p className="px-3 pb-1.5 text-[11px] uppercase tracking-widest text-white/35 font-mono">
                    {section.label}
                  </p>
                  <div className="space-y-1">
                    {section.links.map((link) => {
                      const active = isActive(link.href);
                      return (
                        <Link
                          key={link.href}
                          href={link.href}
                          onClick={() => setMenuOpen(false)}
                          aria-current={active ? 'page' : undefined}
                          className={cn(
                            'flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-colors',
                            active
                              ? 'bg-white/10 text-lime'
                              : 'text-white/80 hover:bg-white/5 hover:text-white'
                          )}
                        >
                          <span
                            className={cn(
                              'inline-flex items-center justify-center w-9 h-9 rounded-lg shrink-0 transition-colors',
                              active ? 'bg-lime/15 text-lime' : 'bg-white/10 text-white/70'
                            )}
                            aria-hidden
                          >
                            <link.Icon size={18} />
                          </span>
                          <span className="flex-1">{link.label}</span>
                          <ChevronRightIcon size={16} className="text-white/30 shrink-0" />
                        </Link>
                      );
                    })}
                  </div>
                </div>
              ))}
            </nav>

            <div className="px-5 py-5 border-t border-white/10 space-y-3">
              {user ? (
                <>
                  <div className="flex items-center justify-between text-xs text-white/40">
                    <span>Devise</span>
                    <span className="font-mono px-2 py-0.5 rounded bg-white/10 text-white/70">FCFA</span>
                  </div>
                  <Button
                    onClick={handleLogout}
                    className="w-full bg-red-500/15 text-red-300 hover:bg-red-500/25 border border-red-500/20"
                  >
                    Déconnexion
                  </Button>
                </>
              ) : (
                <>
                  <Button
                    size="lg"
                    render={<Link href="/auth/register" />}
                    className="w-full bg-green-400 text-green-950 font-semibold hover:bg-green-300"
                  >
                    Créer un compte
                  </Button>
                  <Button
                    variant="outline"
                    size="lg"
                    render={<Link href="/auth/login" />}
                    className="w-full border-white/25 bg-transparent text-white hover:bg-white/10 hover:text-lime hover:border-lime"
                  >
                    Connexion
                  </Button>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </header>
  );
}
