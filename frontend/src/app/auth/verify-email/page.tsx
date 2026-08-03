'use client';

import { Suspense, useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';

function VerifyEmailContent() {
  const params = useSearchParams();
  const token = params.get('token') || '';
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');

  useEffect(() => {
    if (!token) {
      setStatus('error');
      return;
    }
    api
      .verifyEmail(token)
      .then(() => setStatus('success'))
      .catch(() => setStatus('error'));
  }, [token]);

  return (
    <main className="min-h-screen flex items-center justify-center bg-paper px-4 py-16">
      <Card className="w-full max-w-md shadow-lift border-green-900/5 text-center">
        <CardHeader>
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2">
            // vérification email
          </p>
          <CardTitle className="font-display text-3xl font-bold text-green-950">
            {status === 'loading' && 'Vérification...'}
            {status === 'success' && 'Email vérifié'}
            {status === 'error' && 'Lien invalide'}
          </CardTitle>
          <CardDescription className="text-green-900/60">
            {status === 'loading' && 'Vérification en cours...'}
          </CardDescription>
        </CardHeader>

        <CardContent>
          {status === 'success' && (
            <>
              <p className="text-muted-foreground text-sm mb-6">
                Votre adresse email est maintenant vérifiée. Vous pouvez vous connecter.
              </p>
              <Button className="w-full h-10 font-semibold" render={<a href="/auth/login" />}>
                Se connecter
              </Button>
            </>
          )}

          {status === 'error' && (
            <>
              <p className="text-destructive text-sm">
                Ce lien de vérification est invalide ou a expiré. Vérifiez l&apos;adresse reçue par
                email.
              </p>
              <Button
                variant="link"
                className="mt-4 text-primary font-semibold"
                render={<a href="/auth/login" />}
              >
                ← Retour à la connexion
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </main>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense
      fallback={
        <main className="min-h-screen flex items-center justify-center text-green-900/50">
          Chargement...
        </main>
      }
    >
      <VerifyEmailContent />
    </Suspense>
  );
}
