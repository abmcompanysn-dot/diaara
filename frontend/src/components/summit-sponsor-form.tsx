'use client';

import { useState } from 'react';
import { api } from '@/lib/api';
import { CheckIcon } from '@/components/icons';

// Demande "devenir sponsor" du DIARRA Summit. Réutilise le widget de
// contact support existant (POST /api/support/contact — les agents support
// reçoivent l'email) plutôt qu'un nouvel endpoint/table dédié : le volume
// attendu (demandes de sponsoring ponctuelles) ne justifie pas une table
// séparée avec sa propre modération/liste admin.
export function SummitSponsorForm() {
  const [company, setCompany] = useState('');
  const [contactValue, setContactValue] = useState('');
  const [message, setMessage] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!company.trim() || !contactValue.trim()) {
      setError('Merci de remplir au moins le nom de votre entreprise et un contact.');
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      await api.submitSupportContact({
        name: `[Sponsor DIARRA Summit] ${company.trim()}`,
        contact_method: contactValue.includes('@') ? 'email' : 'whatsapp',
        contact_value: contactValue.trim(),
        message: message.trim() || 'Souhaite devenir sponsor du DIARRA Summit — aucun détail supplémentaire fourni.',
      });
      setSent(true);
    } catch {
      setError("Échec de l'envoi. Réessayez dans un instant.");
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
        <p className="text-sm font-semibold text-green-950">Demande envoyée !</p>
        <p className="text-xs text-green-900/60">L&rsquo;équipe DIARRA vous recontacte rapidement.</p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <input
        type="text"
        placeholder="Nom de votre entreprise"
        value={company}
        onChange={(e) => setCompany(e.target.value)}
        className="w-full h-11 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      <input
        type="text"
        placeholder="Email ou WhatsApp"
        value={contactValue}
        onChange={(e) => setContactValue(e.target.value)}
        className="w-full h-11 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      <textarea
        placeholder="Votre message (optionnel)"
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        rows={3}
        className="w-full px-3 py-2 rounded-md border border-green-900/15 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      {error && <p className="text-xs text-red-600">{error}</p>}
      <button
        type="submit"
        disabled={submitting}
        className="w-full h-11 rounded-md bg-[#0E6B46] text-white text-sm font-semibold hover:bg-[#0c5a3c] transition-colors disabled:opacity-60"
      >
        {submitting ? 'Envoi...' : 'Devenir sponsor'}
      </button>
    </form>
  );
}
