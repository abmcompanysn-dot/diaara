'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';

export default function RegisterPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [phone, setPhone] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const result = await api.register({ email, password, phone });
      localStorage.setItem('access_token', result.access_token);
      localStorage.setItem('refresh_token', result.refresh_token);
      router.push('/dashboard');
    } catch (err: any) {
      setError(err.message || "Échec de l'inscription");
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="min-h-screen flex items-center justify-center bg-paper px-4 py-16">
      <div className="max-w-md w-full p-8 bg-white rounded-2xl shadow-card border border-green-900/5">
        <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2 text-center">
          // rejoindre diarra
        </p>
        <h1 className="font-display text-3xl font-bold text-center text-green-950">
          Créer un compte
        </h1>

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
            <label className="block text-sm font-medium mb-1 text-green-900/80">
              Téléphone (optionnel)
            </label>
            <input
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              className="w-full p-3 border border-green-900/10 rounded-lg focus:border-green-500 focus:shadow-glow transition-all"
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
              minLength={8}
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 gradient-green text-white font-semibold rounded-lg hover:opacity-95 transition-opacity disabled:opacity-50"
          >
            {loading ? "Inscription..." : "S'inscrire"}
          </button>
        </form>

        <p className="mt-4 text-center text-sm text-green-900/70">
          Déjà un compte ?{' '}
          <Link href="/auth/login" className="text-green-600 font-semibold hover:text-green-500">
            Se connecter
          </Link>
        </p>
      </div>
    </main>
  );
}
