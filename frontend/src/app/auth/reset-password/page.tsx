'use client';

import { Suspense, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { AuthShell } from '@/components/auth-shell';

function ResetPasswordContent() {
  const params = useSearchParams();
  const token = params.get('token') || '';
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState('');
  const [done, setDone] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!token) {
      setError('Lien de réinitialisation invalide ou incomplet.');
      return;
    }
    if (password.length < 8) {
      setError('Le mot de passe doit contenir au moins 8 caractères.');
      return;
    }
    if (password !== confirm) {
      setError('Les deux mots de passe ne correspondent pas.');
      return;
    }

    setLoading(true);
    try {
      await api.resetPassword(token, password);
      setDone(true);
    } catch (err: any) {
      setError(err.message || 'Échec de la réinitialisation');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthShell>
      <Card className="w-full shadow-lift border-green-900/5">
        <CardHeader>
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest text-center">
            // nouveau mot de passe
          </p>
          <CardTitle className="font-display text-3xl font-bold text-center text-green-950">
            Choisir un nouveau mot de passe
          </CardTitle>
          <CardDescription className="text-center text-green-900/60">
            Votre mot de passe doit contenir au moins 8 caractères
          </CardDescription>
        </CardHeader>

        <CardContent>
          {done ? (
            <div className="text-center">
              <div className="p-4 bg-green-50 text-green-800 rounded-lg text-sm">
                Votre mot de passe a été réinitialisé. Vos anciennes sessions ont été révoquées.
              </div>
              <Button className="mt-6 w-full h-10 font-semibold" render={<Link href="/auth/login" />}>
                Se connecter
              </Button>
            </div>
          ) : (
            <>
              {error && (
                <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded-lg text-sm" role="alert">
                  {error}
                </div>
              )}
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="password">Nouveau mot de passe</Label>
                  <Input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    minLength={8}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="confirm">Confirmer le mot de passe</Label>
                  <Input
                    id="confirm"
                    type="password"
                    value={confirm}
                    onChange={(e) => setConfirm(e.target.value)}
                    required
                    minLength={8}
                  />
                </div>
                <Button type="submit" disabled={loading} className="w-full h-10 font-semibold">
                  {loading ? 'Enregistrement...' : 'Réinitialiser le mot de passe'}
                </Button>
              </form>
            </>
          )}

          <p className="mt-4 text-center text-sm text-green-900/70">
            <Link href="/auth/login" className="text-green-600 font-semibold hover:text-green-500">
              ← Retour à la connexion
            </Link>
          </p>
        </CardContent>
      </Card>
    </AuthShell>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense
      fallback={
        <main className="min-h-screen flex items-center justify-center text-green-900/50">
          Chargement...
        </main>
      }
    >
      <ResetPasswordContent />
    </Suspense>
  );
}
