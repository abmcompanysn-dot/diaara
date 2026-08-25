'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth';
import { PhoneVerifyForm } from '@/components/phone-verify-form';
import { AuthShell } from '@/components/auth-shell';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

// Vérification du téléphone par code OTP (SMS) — requise avant de pouvoir
// demander un versement ; optionnelle pour les autres actions.
export default function VerifyPhonePage() {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (loading) return;
    if (!user) {
      router.replace('/auth/login');
      return;
    }
    if (!user.phone) {
      router.replace('/dashboard');
      return;
    }
    if (user.phone_verified_at) {
      router.replace('/dashboard');
    }
  }, [loading, user, router]);

  if (loading || !user || !user.phone || user.phone_verified_at) {
    return (
      <main className="min-h-screen flex items-center justify-center text-green-900/50">
        Chargement...
      </main>
    );
  }

  return (
    <AuthShell>
      <Card className="w-full shadow-lift border-green-900/5">
        <CardHeader>
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest text-center">
            // vérification téléphone
          </p>
          <CardTitle className="font-display text-3xl font-bold text-center text-green-950">
            Vérifiez votre téléphone
          </CardTitle>
          <CardDescription className="text-center text-green-900/60">
            Nécessaire pour demander un versement sur votre compte mobile money.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <PhoneVerifyForm
            phone={user.phone}
            onVerified={() => router.replace('/dashboard')}
            onSkip={() => router.replace('/dashboard')}
            skipLabel="Plus tard"
          />
        </CardContent>
      </Card>
    </AuthShell>
  );
}
