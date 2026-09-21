'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { ZapIcon, EditIcon, UserIcon, TrashIcon } from '@/components/icons';
import { PRODUCT_STATUS_BADGE, PRODUCT_STATUS_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface EventItem {
  id: string;
  title: string;
  slug: string;
  moderation_status: string;
  moderation_note?: string | null;
  event_date?: string | null;
  created_at: string;
}

export default function VendorEventsPage() {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toDelete, setToDelete] = useState<EventItem | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = () => {
    api
      .getVendorEvents()
      .then((res) => setEvents(res.events || []))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const handleDelete = async () => {
    if (!toDelete) return;
    setDeleting(true);
    try {
      await api.deleteEvent(toDelete.id);
      setToDelete(null);
      load();
    } catch (err: any) {
      setError(friendlyError(err));
      setToDelete(null);
    } finally {
      setDeleting(false);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader back="/vendor" eyebrow="// espace vendeur" title="Mes événements" />
        <PageLoader />
      </main>
    );

  return (
    <main className="relative min-h-screen pb-20">
      <PageHeader
        back="/vendor"
        eyebrow="// espace vendeur"
        title="Mes événements"
        description={`${events.length} événement(s)`}
        actions={
          <Button size="sm" render={<Link href="/vendor/events/new" />}>
            + Nouvel événement
          </Button>
        }
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-6">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {events.length === 0 ? (
          <EmptyState
            icon={ZapIcon}
            title="Aucun événement pour le moment"
            description="Créez votre premier événement (webinaire, atelier, lancement...) avec des offres payantes ou gratuites."
            action={<Button render={<Link href="/vendor/events/new" />}>Créer mon premier événement</Button>}
          />
        ) : (
          <div className="space-y-2">
            {events.map((event) => (
              <div key={event.id} className="bg-white rounded-xl border border-green-900/10 shadow-card p-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <Link href={`/event?id=${event.slug}`} className="font-medium text-primary hover:underline line-clamp-2">
                      {event.title}
                    </Link>
                    {event.event_date && (
                      <p className="text-xs text-green-900/50 mt-0.5">
                        {new Date(event.event_date).toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })}
                      </p>
                    )}
                    <div className="mt-2">
                      <Badge className={PRODUCT_STATUS_BADGE[event.moderation_status]}>
                        {PRODUCT_STATUS_LABELS[event.moderation_status] || event.moderation_status}
                      </Badge>
                      {event.moderation_status === 'rejected' && event.moderation_note && (
                        <p className="text-[11px] text-red-600 mt-1">{event.moderation_note}</p>
                      )}
                    </div>
                  </div>
                  <div className="flex gap-1 shrink-0">
                    <Button
                      variant="ghost"
                      size="sm"
                      render={<Link href={`/vendor/events/registrations?id=${event.id}`} />}
                    >
                      <UserIcon size={16} className="mr-1.5" />
                      Inscrits
                    </Button>
                    <Button variant="ghost" size="sm" render={<Link href={`/vendor/events/edit?id=${event.id}`} />}>
                      <EditIcon size={16} className="mr-1.5" />
                      Modifier
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={() => setToDelete(event)}
                    >
                      <TrashIcon size={16} className="mr-1.5" />
                      Supprimer
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <ConfirmDialog
        open={!!toDelete}
        title="Supprimer cet événement ?"
        description={`« ${toDelete?.title} » sera définitivement supprimé, ainsi que ses offres et ses inscriptions.`}
        confirmLabel={deleting ? 'Suppression...' : 'Supprimer'}
        cancelLabel="Annuler"
        danger
        onConfirm={handleDelete}
        onCancel={() => setToDelete(null)}
      />
    </main>
  );
}
