'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';

interface Ticket {
  id: string;
  subject: string;
  status: string;
  created_at: string;
}

const STATUS_LABELS: Record<string, string> = {
  open: 'Ouvert',
  answered: 'Répondu',
  closed: 'Fermé',
};

export default function SupportPage() {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [subject, setSubject] = useState('');
  const [message, setMessage] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    loadTickets();
  }, []);

  const loadTickets = async () => {
    setLoading(true);
    try {
      const result = await api.getTickets();
      setTickets(result.tickets);
    } catch (err: any) {
      setError(err.message || 'Impossible de charger les tickets');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await api.createTicket({ subject, message });
      setSubject('');
      setMessage('');
      setShowForm(false);
      await loadTickets();
    } catch (err: any) {
      setError(err.message || 'Création impossible');
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-4xl mx-auto">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Support</h1>
          <p className="text-green-700">Besoin d&apos;aide ? Ouvrez un ticket</p>
        </div>
        <Button onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Annuler' : 'Nouveau ticket'}
        </Button>
      </header>

      {error && <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded">{error}</div>}

      {showForm && (
        <form onSubmit={handleCreate} className="mb-6 p-4 border rounded-lg space-y-3">
          <Input
            type="text"
            placeholder="Sujet du ticket"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            required
          />
          <Textarea
            placeholder="Décrivez votre problème..."
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            required
            rows={4}
          />
          <Button type="submit" disabled={submitting}>
            {submitting ? 'Envoi...' : 'Envoyer'}
          </Button>
        </form>
      )}

      {tickets.length === 0 ? (
        <div className="text-center py-12 border rounded-lg">
          <p className="text-green-900/50 mb-4">Aucun ticket pour le moment.</p>
          <p className="text-sm text-green-900/40">Cliquez sur « Nouveau ticket » pour démarrer.</p>
        </div>
      ) : (
        <ul className="divide-y">
          {tickets.map((ticket) => (
            <li key={ticket.id} className="py-3 flex items-center justify-between">
              <Link href={`/support/ticket?id=${ticket.id}`} className="text-primary font-medium">
                {ticket.subject}
              </Link>
              <div className="flex items-center gap-3">
                <span className="text-sm text-muted-foreground">
                  {new Date(ticket.created_at).toLocaleDateString('fr-FR')}
                </span>
                <Badge>{STATUS_LABELS[ticket.status] || ticket.status}</Badge>
              </div>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
