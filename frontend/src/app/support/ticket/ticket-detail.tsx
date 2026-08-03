'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { api } from '@/lib/api';

interface Message {
  id: string;
  author_id: string;
  body: string;
  created_at: string;
}

interface Ticket {
  id: string;
  subject: string;
  status: string;
  user_id: string;
}

export default function TicketDetailPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const id = searchParams.get('id') || '';

  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [body, setBody] = useState('');
  const [sending, setSending] = useState(false);

  useEffect(() => {
    if (id) load();
  }, [id]);

  const load = async () => {
    setLoading(true);
    try {
      const result = await api.getTicketMessages(id);
      setTicket(result.ticket);
      setMessages(result.messages);
    } catch (err: any) {
      setError(err.message || 'Ticket introuvable');
    } finally {
      setLoading(false);
    }
  };

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!body.trim()) return;
    setSending(true);
    try {
      await api.addTicketMessage(id, { body });
      setBody('');
      await load();
    } catch (err: any) {
      setError(err.message || 'Envoi impossible');
    } finally {
      setSending(false);
    }
  };

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-3xl mx-auto">
      <header className="mb-6">
        <button onClick={() => router.push('/support')} className="text-sm text-green-600 mb-2">
          ← Retour au support
        </button>
        <h1 className="text-2xl font-bold">{ticket?.subject}</h1>
        <p className="text-green-900/50 text-sm">Statut : {ticket?.status}</p>
      </header>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-600 rounded">{error}</div>}

      <div className="space-y-3 mb-6">
        {messages.map((msg) => (
          <div key={msg.id} className="p-3 border rounded-lg bg-green-50">
            <div className="flex items-center justify-between mb-1">
              <span className="text-xs font-medium text-green-900/50">
                {msg.author_id === ticket?.user_id ? 'Vous' : 'Support'}
              </span>
              <span className="text-xs text-green-900/40">
                {new Date(msg.created_at).toLocaleString('fr-FR')}
              </span>
            </div>
            <p className="whitespace-pre-wrap">{msg.body}</p>
          </div>
        ))}
        {messages.length === 0 && (
          <p className="text-center text-green-900/40 py-6">Aucun message.</p>
        )}
      </div>

      {ticket?.status !== 'closed' && (
        <form onSubmit={handleSend} className="flex gap-2">
          <textarea
            placeholder="Votre réponse..."
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={3}
            className="flex-1 p-2 border rounded"
          />
          <button
            type="submit"
            disabled={sending}
            className="px-4 py-2 rounded gradient-green text-white disabled:opacity-50"
          >
            {sending ? 'Envoi...' : 'Envoyer'}
          </button>
        </form>
      )}
    </main>
  );
}
