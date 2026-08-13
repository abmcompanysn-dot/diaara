'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { AuthShell } from '@/components/auth-shell';
import { cn } from '@/lib/utils';
import { friendlyError } from '@/lib/error-messages';

const ROLE_OPTIONS = [
  {
    value: 'vendeur',
    title: 'Vendeur',
    desc: 'Vendre vos biens numériques',
  },
  {
    value: 'closer',
    title: 'Affilié (closer)',
    desc: 'Gagner des commissions en partageant des liens',
  },
];

export default function RegisterPage() {
  const router = useRouter();
  const { login } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [phone, setPhone] = useState('');
  const [selected, setSelected] = useState<string[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const toggleRole = (value: string) => {
    setSelected((prev) =>
      prev.includes(value) ? prev.filter((r) => r !== value) : [...prev, value]
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const result = await api.register({
        email,
        password,
        phone: phone || undefined,
        roles: selected,
      });
      await login(result.access_token);
      router.push('/auth/verify-email');
    } catch (err: any) {
      if (err.status === 409) {
        setError(
          'Un compte existe déjà avec cet email. Connectez-vous avec votre mot de passe.'
        );
      } else {
        setError(friendlyError(err));
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthShell>
      <Card className="w-full shadow-lift border-green-900/5">
        <CardHeader>
          <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest text-center">
            // rejoindre diarra
          </p>
          <CardTitle className="font-display text-3xl font-bold text-center text-green-950">
            Créer un compte
          </CardTitle>
          <CardDescription className="text-center text-green-900/60">
            Achetez, vendez ou gagnez de l&apos;argent en partageant des liens
          </CardDescription>
        </CardHeader>

        <CardContent>
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

            <div className="space-y-2">
              <Label htmlFor="phone">Téléphone (optionnel)</Label>
              <Input
                id="phone"
                type="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="+221 ..."
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Mot de passe</Label>
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
              <Label>
                Type de compte
                <span className="text-green-900/50 font-normal"> (optionnel)</span>
              </Label>
              <div className="space-y-2">
                {ROLE_OPTIONS.map((opt) => {
                  const active = selected.includes(opt.value);
                  return (
                    <Button
                      key={opt.value}
                      type="button"
                      variant="ghost"
                      onClick={() => toggleRole(opt.value)}
                      className={cn(
                        'w-full h-auto flex items-start justify-start gap-3 p-3 border rounded-lg text-left transition-all',
                        active
                          ? 'border-ring bg-secondary'
                          : 'border-border hover:border-ring/60'
                      )}
                    >
                      <span
                        className={cn(
                          'mt-0.5 w-5 h-5 rounded-full border flex items-center justify-center shrink-0',
                          active ? 'border-primary bg-primary text-white' : 'border-border'
                        )}
                        aria-hidden
                      >
                        {active && (
                          <svg
                            width="12"
                            height="12"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="3"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          >
                            <path d="m4.5 12.5 5 5 10-11" />
                          </svg>
                        )}
                      </span>
                      <span>
                        <span className="block font-semibold text-foreground text-sm">
                          {opt.title}
                        </span>
                        <span className="block text-xs text-muted-foreground">{opt.desc}</span>
                      </span>
                    </Button>
                  );
                })}
              </div>
              <p className="text-xs text-muted-foreground">
                Inscrivez-vous comme vendeur ou affilié pour commencer à gagner de l&apos;argent.
              </p>
            </div>


            <Button type="submit" disabled={loading} className="w-full h-10 font-semibold">
              {loading ? "Inscription..." : "S'inscrire"}
            </Button>
          </form>
        </CardContent>

        <CardFooter className="justify-center text-sm text-green-900/70">
          Déjà un compte ?{' '}
          <Link href="/auth/login" className="text-green-600 font-semibold hover:text-green-500 ml-1">
            Se connecter
          </Link>
        </CardFooter>
      </Card>
    </AuthShell>
  );
}
