'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ArrowLeftIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

interface PendingEvent {
  id: string;
  title: string;
  slug: string;
  vendor_id: string;
  created_at: string;
}

export default function AdminEventsPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [events, setEvents] = useState<PendingEvent[]>([]);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = () => {
    setLoading(true);
    api
      .adminListPendingEvents()
      .then((res) => setEvents(res.events || []))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const handleModerate = async (id: string, status: 'approved' | 'rejected') => {
    setBusyId(id);
    try {
      await api.adminModerateEvent(id, { status });
      setEvents((prev) => prev.filter((e) => e.id !== id));
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Événements en attente" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Événements en attente"
        description="Modération des événements créés par les vendeurs"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Tableau de bord
          </Button>
        }
      />

      <section className="max-w-2xl mx-auto px-4 sm:px-6 py-10 space-y-4">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {events.length === 0 ? (
          <p className="text-sm text-muted-foreground italic text-center py-10">
            Aucun événement en attente de modération.
          </p>
        ) : (
          events.map((event) => (
            <Card key={event.id} className="shadow-card border-green-900/5">
              <CardHeader>
                <CardTitle className="text-base">
                  <Link href={`/event?id=${event.slug}`} className="hover:underline">
                    {event.title}
                  </Link>
                </CardTitle>
                <CardDescription>
                  Créé le {new Date(event.created_at).toLocaleDateString('fr-FR')}. Les offres payantes de cet
                  événement sont modérées séparément dans Produits.
                </CardDescription>
              </CardHeader>
              <CardContent className="flex gap-2">
                <Button
                  size="sm"
                  disabled={busyId === event.id}
                  onClick={() => handleModerate(event.id, 'approved')}
                >
                  Approuver
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={busyId === event.id}
                  className="text-destructive hover:text-destructive"
                  onClick={() => handleModerate(event.id, 'rejected')}
                >
                  Refuser
                </Button>
              </CardContent>
            </Card>
          ))
        )}
      </section>
    </main>
  );
}
