'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth';
import { Button } from '@/components/ui/button';

export function Header() {
  const router = useRouter();
  const { user, isAdmin, hasRole, logout } = useAuth();

  const navLinks = [
    { href: '/catalog', label: 'Catalogue' },
    { href: '/how-it-works', label: 'Comment ça marche' },
    { href: '/dashboard', label: 'Mon espace' },
  ];
  if (hasRole('vendeur')) navLinks.push({ href: '/vendor/products', label: 'Espace vendeur' });
  if (hasRole('closer')) navLinks.push({ href: '/closer/dashboard', label: 'Affiliation' });
  if (isAdmin) navLinks.push({ href: '/admin', label: 'Admin' });

  const handleLogout = async () => {
    await logout();
    router.push('/');
  };

  return (
    <header className="sticky top-0 z-50 bg-green-950/95 backdrop-blur border-b border-green-400/20">
      <div className="max-w-6xl mx-auto px-4 h-16 flex items-center justify-between">
        <Link href="/" className="flex items-center gap-2 group">
          <span className="font-display text-2xl font-bold text-white tracking-tight">
            DIARRA
          </span>
          <span className="w-2 h-2 rounded-full bg-green-400 inline-block group-hover:scale-150 transition-transform" aria-hidden />
        </Link>

        <nav className="hidden md:flex items-center gap-6" aria-label="Navigation principale">
          {navLinks.map((link) => (
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
          {user ? (
            <>
              <span className="hidden sm:inline text-xs text-white/50">
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
        </div>
      </div>
    </header>
  );
}
