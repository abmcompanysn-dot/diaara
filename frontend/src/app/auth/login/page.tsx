'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const result = await api.login({ email, password });
      localStorage.setItem('access_token', result.access_token);
      localStorage.setItem('refresh_token', result.refresh_token);
      router.push('/dashboard');
    } catch (err: any) {
      setError(err.message || 'Échec de la connexion');
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="min-h-screen flex items-center justify-center bg-paper px-4 py-16">
      <div className="max-w-md w-full p-8 bg-white rounded-2xl shadow-card border border-green-900/5">
        <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2 text-center">
          // espace client
        </p>
        <h1 className="font-display text-3xl font-bold text-center text-green-950">Connexion</h1>

        {error && (
          <div className="mt-6 mb-2 p-3 bg-red-50 text-red-600 rounded-lg text-sm">{error}</div>
        )}

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1 text-green-900/80">Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full p-3 border border-green-900/10 rounded-lg focus:border-green-500 focus:shadow-glow transition-all"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1 text-green-900/80">Mot de passe</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full p-3 border border-green-900/10 rounded-lg focus:border-green-500 focus:shadow-glow transition-all"
              required
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 gradient-green text-white font-semibold rounded-lg hover:opacity-95 transition-opacity disabled:opacity-50"
          >
            {loading ? 'Connexion...' : 'Se connecter'}
          </button>
        </form>

        <p className="mt-4 text-center text-sm text-green-900/70">
          Pas de compte ?{' '}
          <Link href="/auth/register" className="text-green-600 font-semibold hover:text-green-500">
            S&apos;inscrire
          </Link>
        </p>
        <p className="mt-2 text-center text-sm">
          <Link href="/auth/forgot-password" className="text-green-900/50 hover:text-green-600">
            Mot de passe oublié ?
          </Link>
        </p>
      </div>
    </main>
  );
}
