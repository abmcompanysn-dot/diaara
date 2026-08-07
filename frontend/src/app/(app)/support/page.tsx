'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { HeadsetIcon } from '@/components/icons';
import { TICKET_STATUS_LABELS } from '@/lib/constants';

interface Ticket {
  id: string;
  subject: string;
  status: string;
  created_at: string;
}

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

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// support" title="Support" description="Besoin d'aide ? Ouvrez un ticket" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// support"
        title="Support"
        description="Besoin d'aide ? Ouvrez un ticket"
        actions={
          <Button onClick={() => setShowForm(!showForm)}>
            {showForm ? 'Annuler' : 'Nouveau ticket'}
          </Button>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {showForm && (
          <form onSubmit={handleCreate} className="mb-6 p-6 border rounded-xl bg-white shadow-card border-green-900/10 space-y-4">
            <div className="space-y-2">
              <Label htmlFor="subject">Sujet du ticket</Label>
              <Input
                id="subject"
                type="text"
                placeholder="Ex : Problème de téléchargement"
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="message">Description</Label>
              <Textarea
                id="message"
                placeholder="Décrivez votre problème en détail..."
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                required
                rows={4}
              />
            </div>
            <Button type="submit" disabled={submitting}>
              {submitting ? 'Envoi...' : 'Envoyer'}
            </Button>
          </form>
        )}

        {tickets.length === 0 ? (
          <EmptyState
            icon={HeadsetIcon}
            title="Aucun ticket pour le moment"
            description="Cliquez sur « Nouveau ticket » pour démarrer une conversation avec le support."
          />
        ) : (
          <div className="space-y-2">
            {tickets.map((ticket) => (
              <Link
                key={ticket.id}
                href={`/support/ticket?id=${ticket.id}`}
                className="block p-4 border rounded-xl bg-white shadow-card border-green-900/10 hover:border-green-600/30 transition-all group"
              >
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <p className="font-medium text-green-950 group-hover:text-green-600 transition-colors">
                      {ticket.subject}
                    </p>
                    <p className="text-sm text-muted-foreground mt-1">
                      {new Date(ticket.created_at).toLocaleDateString('fr-FR')}
                    </p>
                  </div>
                  <Badge variant="outline">{TICKET_STATUS_LABELS[ticket.status] || ticket.status}</Badge>
                </div>
              </Link>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
