'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth';
import { OtpForm } from '@/components/otp-form';
import { AuthShell } from '@/components/auth-shell';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

// Vérification d'email par code OTP (6 chiffres) — étape obligatoire à
// l'inscription avant l'accès au dashboard.
export default function VerifyEmailPage() {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (loading) return;
    if (!user) {
      router.replace('/auth/login');
      return;
    }
    if (user.email_verified_at) {
      // Déjà vérifié : on enchaîne vers le téléphone (si non vérifié) ou le dashboard.
      if (user.phone && !user.phone_verified_at) router.replace('/auth/verify-phone');
      else router.replace('/dashboard');
    }
  }, [loading, user, router]);

  if (loading || !user || user.email_verified_at) {
    return (
      <main className="min-h-screen flex items-center justify-center text-green-900/50">
        Chargement...
      </main>
    );
  }

  const handleVerified = () => {
    if (user.phone && !user.phone_verified_at) router.replace('/auth/verify-phone');
    else router.replace('/dashboard');
  };

  return (
    <AuthShell>
      <Card className="w-full shadow-lift border-green-900/5">
        <CardHeader>
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest text-center">
            // vérification email
          </p>
          <CardTitle className="font-display text-3xl font-bold text-center text-green-950">
            Vérifiez votre email
          </CardTitle>
          <CardDescription className="text-center text-green-900/60">
            Un code à 6 chiffres a été envoyé à <span className="font-medium">{user.email}</span>.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <OtpForm
            channel="email"
            title="Vérifier l'email"
            description="Saisissez le code reçu par email pour activer votre compte."
            onVerified={handleVerified}
          />
        </CardContent>
      </Card>
    </AuthShell>
  );
}
