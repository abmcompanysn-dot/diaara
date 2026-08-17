'use client';

import { useState } from 'react';
import { api } from '@/lib/api';
import { SegmentedTabs } from '@/components/segmented-tabs';
import { MessageIcon, XIcon, CheckIcon } from '@/components/icons';

type ContactMethod = 'email' | 'whatsapp';

// Bouton flottant visible sur tout le site (visiteur non authentifié inclus) :
// permet d'envoyer un message au support sans créer de compte. Distinct du
// système de tickets (authentifié) — ici, juste un formulaire + notification
// email aux agents support enregistrés dans l'admin.
export function SupportContactWidget() {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [method, setMethod] = useState<ContactMethod>('whatsapp');
  const [contactValue, setContactValue] = useState('');
  const [message, setMessage] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');

  const resetAndClose = () => {
    setOpen(false);
    setTimeout(() => {
      setName('');
      setContactValue('');
      setMessage('');
      setSent(false);
      setError('');
    }, 200);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !contactValue.trim() || !message.trim()) {
      setError('Merci de remplir tous les champs.');
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      await api.submitSupportContact({
        name: name.trim(),
        contact_method: method,
        contact_value: contactValue.trim(),
        message: message.trim(),
      });
      setSent(true);
    } catch {
      setError("Échec de l'envoi. Réessayez dans un instant.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed bottom-5 left-5 z-40">
      {open && (
        <div className="mb-3 w-[calc(100vw-2.5rem)] max-w-sm rounded-2xl border border-green-900/10 bg-white shadow-lift p-4 animate-in fade-in slide-in-from-bottom-2">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-bold text-green-950">Contacter le support</h3>
            <button type="button" onClick={resetAndClose} aria-label="Fermer" className="text-green-900/50 hover:text-green-900">
              <XIcon size={16} />
            </button>
          </div>

          {sent ? (
            <div className="flex flex-col items-center text-center gap-2 py-4">
              <span className="w-9 h-9 rounded-full bg-green-100 text-green-700 flex items-center justify-center">
                <CheckIcon size={18} />
              </span>
              <p className="text-sm text-green-950 font-medium">Message envoyé !</p>
              <p className="text-xs text-green-900/60">Le support vous répondra bientôt.</p>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-3">
              <input
                type="text"
                placeholder="Votre nom"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full h-10 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
              />
              <SegmentedTabs
                options={[
                  { value: 'whatsapp', label: 'WhatsApp' },
                  { value: 'email', label: 'Email' },
                ]}
                value={method}
                onChange={(v) => {
                  setMethod(v);
                  setContactValue('');
                }}
              />
              <input
                type={method === 'email' ? 'email' : 'tel'}
                placeholder={method === 'email' ? 'Votre email' : 'Ex: +221 77 123 45 67'}
                value={contactValue}
                onChange={(e) => setContactValue(e.target.value)}
                className="w-full h-10 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
              />
              <textarea
                placeholder="Votre message"
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                rows={3}
                className="w-full px-3 py-2 rounded-md border border-green-900/15 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-green-600/30"
              />
              {error && <p className="text-xs text-red-600">{error}</p>}
              <button
                type="submit"
                disabled={submitting}
                className="w-full h-10 rounded-md bg-[#0E6B46] text-white text-sm font-semibold hover:bg-[#0c5a3c] transition-colors disabled:opacity-60"
              >
                {submitting ? 'Envoi...' : 'Envoyer'}
              </button>
            </form>
          )}
        </div>
      )}

      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-label="Contacter le support"
        className="w-14 h-14 rounded-full bg-[#0E6B46] text-white shadow-lift flex items-center justify-center hover:bg-[#0c5a3c] transition-colors"
      >
        {open ? <XIcon size={20} /> : <MessageIcon size={22} />}
      </button>
    </div>
  );
}
