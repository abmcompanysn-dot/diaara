'use client';

import { useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';

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
      <div className="max-w-md w-full p-8 bg-white rounded-2xl shadow-card border border-green-900/5">
        <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-2 text-center">
          // mot de passe oublié
        </p>
        <h1 className="font-display text-3xl font-bold text-center text-green-950">
          Réinitialiser le mot de passe
        </h1>

        {sent ? (
          <div className="mt-6 p-4 bg-green-50 text-green-800 rounded-lg text-sm">
            Si un compte existe avec cet email, un lien de réinitialisation vient d&apos;être envoyé.
          </div>
        ) : (
          <>
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
              <button
                type="submit"
                disabled={loading}
                className="w-full py-3 gradient-green text-white font-semibold rounded-lg hover:opacity-95 transition-opacity disabled:opacity-50"
              >
                {loading ? 'Envoi...' : 'Recevoir un lien de réinitialisation'}
              </button>
            </form>
          </>
        )}

        <p className="mt-4 text-center text-sm text-green-900/70">
          <Link href="/auth/login" className="text-green-600 font-semibold hover:text-green-500">
            ← Retour à la connexion
          </Link>
        </p>
      </div>
    </main>
  );
}
