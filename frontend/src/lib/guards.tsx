'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from './auth';

function Loading() {
  return (
    <main className="min-h-screen flex items-center justify-center text-green-900/50">
      Chargement...
    </main>
  );
}

// RequireAuth : redirige vers /auth/login si non connecté.
export function RequireAuth({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (!loading && !user) router.replace('/auth/login');
  }, [loading, user, router]);

  if (loading) return <Loading />;
  if (!user) return null;
  return <>{children}</>;
}

// RequireRole : exige au moins un des rôles donnés (cumulables).
export function RequireRole({ roles, children }: { roles: string[]; children: React.ReactNode }) {
  const router = useRouter();
  const { user, loading, hasRole } = useAuth();

  const allowed = roles.some((r) => hasRole(r));

  useEffect(() => {
    if (!loading) {
      if (!user) router.replace('/auth/login');
      else if (!allowed) router.replace('/dashboard');
    }
  }, [loading, user, allowed, router]);

  if (loading) return <Loading />;
  if (!user || !allowed) return null;
  return <>{children}</>;
}

// RequireAdmin : redirige vers /dashboard si pas admin.
export function RequireAdmin({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { user, loading, isAdmin } = useAuth();

  useEffect(() => {
    if (!loading) {
      if (!user) router.replace('/auth/login');
      else if (!isAdmin) router.replace('/dashboard');
    }
  }, [loading, user, isAdmin, router]);

  if (loading) return <Loading />;
  if (!user || !isAdmin) return null;
  return <>{children}</>;
}
