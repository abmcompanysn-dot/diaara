'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ArrowLeftIcon } from '@/components/icons';
import { TICKET_STATUS_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

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
      setError(friendlyError(err));
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
      setError(friendlyError(err));
    } finally {
      setSending(false);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// support" title="Ticket" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// support"
        title={ticket?.subject || 'Ticket'}
        description={
          <span className="inline-flex items-center gap-2">
            Statut : <Badge variant="outline">{ticket?.status ? TICKET_STATUS_LABELS[ticket.status] : '...'}</Badge>
          </span>
        }
        actions={
          <Button variant="outline" size="sm" render={<Link href="/support" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Retour au support
          </Button>
        }
      />

      <section className="max-w-3xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        <div className="space-y-3 mb-6">
          {messages.map((msg) => (
            <div key={msg.id} className="p-4 border rounded-xl bg-green-50/60 border-green-900/10">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-medium text-green-900/60">
                  {msg.author_id === ticket?.user_id ? 'Vous' : 'Support'}
                </span>
                <span className="text-xs text-green-900/40">
                  {new Date(msg.created_at).toLocaleString('fr-FR')}
                </span>
              </div>
              <p className="whitespace-pre-wrap text-sm leading-relaxed">{msg.body}</p>
            </div>
          ))}
          {messages.length === 0 && (
            <p className="text-center text-green-900/40 py-6">Aucun message.</p>
          )}
        </div>

        {ticket?.status !== 'closed' && (
          <form onSubmit={handleSend} className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="reply">Votre réponse</Label>
              <Textarea
                id="reply"
                placeholder="Écrivez votre message..."
                value={body}
                onChange={(e) => setBody(e.target.value)}
                rows={3}
              />
            </div>
            <Button type="submit" disabled={sending}>
              {sending ? 'Envoi...' : 'Envoyer'}
            </Button>
          </form>
        )}
      </section>
    </main>
  );
}
