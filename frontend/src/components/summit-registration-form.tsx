'use client';

import { useState } from 'react';
import { api } from '@/lib/api';
import { CheckIcon } from '@/components/icons';

type Profile = 'vendeur' | 'acheteur' | 'entrepreneur' | 'curieux';

const PROFILES: { value: Profile; label: string }[] = [
  { value: 'vendeur', label: 'Vendeur' },
  { value: 'acheteur', label: 'Acheteur' },
  { value: 'entrepreneur', label: 'Entrepreneur' },
  { value: 'curieux', label: 'Curieux' },
];

// Formulaire d'inscription publique au DIARRA Summit (26 novembre 2026, en
// ligne). Même esprit que SupportContactWidget : pas de compte requis,
// soumis à la limite de débit globale côté backend.
export function SummitRegistrationForm() {
  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [profile, setProfile] = useState<Profile>('curieux');
  const [submitting, setSubmitting] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!fullName.trim() || !email.trim() || !phone.trim()) {
      setError('Merci de remplir tous les champs.');
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      await api.submitSummitRegistration({
        full_name: fullName.trim(),
        email: email.trim(),
        phone: phone.trim(),
        profile,
      });
      setSent(true);
    } catch (err) {
      const message =
        err instanceof Error && err.message.includes('already_registered')
          ? 'Cet email est déjà inscrit — à bientôt au Summit !'
          : "Échec de l'inscription. Réessayez dans un instant.";
      setError(message);
    } finally {
      setSubmitting(false);
    }
  };

  if (sent) {
    return (
      <div className="flex flex-col items-center text-center gap-2 py-6">
        <span className="w-11 h-11 rounded-full bg-green-100 text-green-700 flex items-center justify-center">
          <CheckIcon size={20} />
        </span>
        <p className="text-sm font-semibold text-green-950">Inscription confirmée !</p>
        <p className="text-xs text-green-900/60">
          Un email de confirmation vient de vous être envoyé.
        </p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <input
        type="text"
        placeholder="Nom complet"
        value={fullName}
        onChange={(e) => setFullName(e.target.value)}
        className="w-full h-11 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      <input
        type="email"
        placeholder="Email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        className="w-full h-11 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      <input
        type="tel"
        placeholder="Ex: +221 77 123 45 67"
        value={phone}
        onChange={(e) => setPhone(e.target.value)}
        className="w-full h-11 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      <div>
        <p className="text-xs text-green-900/60 mb-1.5">Vous êtes plutôt :</p>
        <div className="grid grid-cols-2 gap-2">
          {PROFILES.map((p) => (
            <button
              key={p.value}
              type="button"
              onClick={() => setProfile(p.value)}
              className={`h-9 rounded-md text-sm font-medium border transition-colors ${
                profile === p.value
                  ? 'bg-[#0E6B46] text-white border-[#0E6B46]'
                  : 'bg-white text-green-900/70 border-green-900/15 hover:border-green-900/30'
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>
      </div>
      {error && <p className="text-xs text-red-600">{error}</p>}
      <button
        type="submit"
        disabled={submitting}
        className="w-full h-11 rounded-md bg-[#0E6B46] text-white text-sm font-semibold hover:bg-[#0c5a3c] transition-colors disabled:opacity-60"
      >
        {submitting ? 'Inscription...' : "Je m'inscris gratuitement"}
      </button>
    </form>
  );
}
