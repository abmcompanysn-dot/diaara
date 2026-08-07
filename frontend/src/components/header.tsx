'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';
import { useAuth } from '@/lib/auth';
import { Button } from '@/components/ui/button';
import { headerNavItems } from '@/lib/navigation';
import { MenuIcon, XIcon } from '@/components/icons';
import { cn } from '@/lib/utils';

export function Header() {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);

  const links = headerNavItems(user);

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
          className="flex items-center gap-2 group"
          onClick={() => setMenuOpen(false)}
        >
          <span className="font-display text-2xl font-bold text-white tracking-tight">
            DIARRA
          </span>
          <span
            className="w-2 h-2 rounded-full bg-green-400 inline-block group-hover:scale-150 transition-transform"
            aria-hidden
          />
        </Link>

        <nav className="hidden md:flex items-center gap-6" aria-label="Navigation principale">
          {links.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              aria-current={isActive(link.href) ? 'page' : undefined}
              className={cn(
                'text-sm text-white/80 hover:text-green-300 transition-colors',
                isActive(link.href) && 'text-green-300 font-medium'
              )}
            >
              {link.label}
            </Link>
          ))}
        </nav>

        <div className="flex items-center gap-3">
          {user ? (
            <>
              <span className="hidden sm:inline text-xs text-white/50 truncate max-w-40">
                {user.email}
              </span>
              <Button
                onClick={handleLogout}
                variant="outline"
                size="sm"
                className="border-white/25 bg-transparent text-white hover:bg-white/10 hover:text-lime hover:border-lime"
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
            className="absolute right-0 top-0 h-full w-72 max-w-[85vw] bg-green-950 text-white shadow-lift flex flex-col"
          >
            <div className="flex items-center justify-between px-5 h-16 border-b border-white/10">
              <span className="font-display text-xl font-bold tracking-tight">DIARRA</span>
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
                <p className="text-xs text-white/50 truncate">{user.email}</p>
              </div>
            )}

            <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-1" aria-label="Navigation mobile">
              {links.map((link) => (
                <Link
                  key={link.href}
                  href={link.href}
                  onClick={() => setMenuOpen(false)}
                  aria-current={isActive(link.href) ? 'page' : undefined}
                  className={cn(
                    'flex items-center justify-between px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                    isActive(link.href)
                      ? 'bg-white/10 text-lime'
                      : 'text-white/80 hover:bg-white/5 hover:text-white'
                  )}
                >
                  {link.label}
                </Link>
              ))}
            </nav>

            <div className="px-5 py-5 border-t border-white/10 space-y-3">
              {user ? (
                <Button
                  onClick={handleLogout}
                  variant="outline"
                  className="w-full border-white/25 bg-transparent text-white hover:bg-white/10 hover:text-lime hover:border-lime"
                >
                  Déconnexion
                </Button>
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
