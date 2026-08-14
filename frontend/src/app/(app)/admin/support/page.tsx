'use client';

import { useEffect, useState, useCallback } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { useWebSocket } from '@/lib/use-websocket';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { friendlyError } from '@/lib/error-messages';

interface Ticket {
  id: string;
  user_id: string;
  subject: string;
  status: string;
  assigned_admin_id?: string | null;
  claimed_at?: string | null;
  created_at: string;
}

interface AdminSummary {
  id: string;
  email: string;
}

const STATUS_LABELS: Record<string, string> = {
  open: 'Ouvert',
  answered: 'Répondu',
  closed: 'Fermé',
};

export default function AdminSupportPage() {
  const { user } = useAuth();
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [admins, setAdmins] = useState<AdminSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [claimingId, setClaimingId] = useState<string | null>(null);
  const [assigningId, setAssigningId] = useState<string | null>(null);

  const loadTickets = useCallback(async () => {
    try {
      const result = await api.getTickets();
      setTickets(result.tickets);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTickets();
    api.getTicketAssignees().then((r) => setAdmins(r.admins)).catch(() => {});
  }, [loadTickets]);

  // Le canal support_channel relaie chaque nouveau ticket, nouveau message et
  // prise en charge : on recharge la liste dès qu'un événement arrive, pour
  // que tous les agents connectés voient l'état à jour sans recharger.
  useWebSocket('/ws/support', () => {
    loadTickets();
  });

  const handleClaim = async (ticket: Ticket) => {
    setClaimingId(ticket.id);
    setError('');
    try {
      await api.claimTicket(ticket.id);
      await loadTickets();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setClaimingId(null);
    }
  };

  const handleAssign = async (ticketId: string, adminId: string) => {
    setAssigningId(ticketId);
    setError('');
    try {
      await api.assignTicket(ticketId, adminId);
      await loadTickets();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setAssigningId(null);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Support" description="Tickets clients" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Support"
        description={`${tickets.length} ticket(s)`}
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            ← Dashboard
          </Button>
        }
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-10 space-y-3">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {tickets.length === 0 ? (
          <EmptyState title="Aucun ticket" description="Les demandes de support apparaîtront ici en direct." />
        ) : (
          tickets.map((ticket) => {
            const claimedByMe = ticket.assigned_admin_id === user?.id;
            const claimedByOther = Boolean(ticket.assigned_admin_id) && !claimedByMe;
            return (
              <Card key={ticket.id} className="border-green-900/5">
                <CardContent className="p-4 flex items-center justify-between gap-4 flex-wrap">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <Link href={`/support/ticket?id=${ticket.id}`} className="font-medium text-green-950 hover:text-green-700 truncate">
                        {ticket.subject}
                      </Link>
                      <Badge variant="outline">{STATUS_LABELS[ticket.status] || ticket.status}</Badge>
                      {claimedByMe && (
                        <Badge className="bg-green-100 text-green-700 hover:bg-green-100">Pris en charge par vous</Badge>
                      )}
                      {claimedByOther && (
                        <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100">Déjà pris en charge</Badge>
                      )}
                    </div>
                    <p className="text-xs text-green-900/50 mt-1">
                      {new Date(ticket.created_at).toLocaleString('fr-FR')}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    {!ticket.assigned_admin_id && (
                      <Button size="sm" disabled={claimingId === ticket.id} onClick={() => handleClaim(ticket)}>
                        {claimingId === ticket.id ? 'Prise en charge…' : 'Prendre en charge'}
                      </Button>
                    )}
                    <Select
                      value=""
                      onValueChange={(adminId) => adminId && handleAssign(ticket.id, adminId)}
                      disabled={assigningId === ticket.id}
                    >
                      <SelectTrigger className="w-40 bg-white" size="sm">
                        <SelectValue placeholder="Rediriger vers…" />
                      </SelectTrigger>
                      <SelectContent>
                        {admins
                          .filter((a) => a.id !== ticket.assigned_admin_id)
                          .map((a) => (
                            <SelectItem key={a.id} value={a.id}>
                              {a.email}
                            </SelectItem>
                          ))}
                      </SelectContent>
                    </Select>
                  </div>
                </CardContent>
              </Card>
            );
          })
        )}
      </section>
    </main>
  );
}
