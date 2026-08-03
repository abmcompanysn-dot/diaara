'use client';

import { useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await api.forgotPassword(email);
      setSent(true);
    } catch (err: any) {
      setError(err.message || 'Échec de la demande');
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="min-h-screen flex items-center justify-center bg-paper px-4 py-16">
      <Card className="w-full max-w-md shadow-lift border-green-900/5">
        <CardHeader>
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest text-center">
            // mot de passe oublié
          </p>
          <CardTitle className="font-display text-3xl font-bold text-center text-green-950">
            Réinitialiser le mot de passe
          </CardTitle>
          <CardDescription className="text-center text-green-900/60">
            Entrez votre email pour recevoir un lien de réinitialisation
          </CardDescription>
        </CardHeader>

        <CardContent>
          {sent ? (
            <div className="p-4 bg-green-50 text-green-800 rounded-lg text-sm">
              Si un compte existe avec cet email, un lien de réinitialisation vient d&apos;être
              envoyé.
            </div>
          ) : (
            <>
              {error && (
                <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded-lg text-sm">
                  {error}
                </div>
              )}
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="vous@exemple.com"
                    required
                  />
                </div>
                <Button type="submit" disabled={loading} className="w-full h-10 font-semibold">
                  {loading ? 'Envoi...' : 'Recevoir un lien de réinitialisation'}
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
    </main>
  );
}
