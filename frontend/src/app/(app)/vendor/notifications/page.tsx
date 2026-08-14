'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { BellIcon, CheckIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

interface Notification {
  id: string;
  type: string;
  title: string;
  body?: string;
  link?: string;
  read_at: string | null;
  created_at: string;
}

export default function VendorNotificationsPage() {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api
      .getNotifications()
      .then((r) => setNotifications(r.notifications))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, []);

  const unreadCount = notifications.filter((n) => !n.read_at).length;

  const markAllRead = async () => {
    try {
      await api.markAllNotificationsRead();
      setNotifications((list) => list.map((n) => ({ ...n, read_at: n.read_at || new Date().toISOString() })));
    } catch (err: any) {
      setError(friendlyError(err));
    }
  };

  const markRead = async (n: Notification) => {
    if (n.read_at) return;
    try {
      await api.markNotificationRead(n.id);
      setNotifications((list) => list.map((x) => (x.id === n.id ? { ...x, read_at: new Date().toISOString() } : x)));
    } catch {
      // Pas grave : la notification reste juste marquée non-lue.
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader back="/vendor" eyebrow="// espace vendeur" title="Notifications" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        back="/vendor"
        eyebrow="// espace vendeur"
        title="Notifications"
        description="Ventes, versements et alertes en direct"
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-6">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {unreadCount > 0 && (
          <div className="flex justify-end mb-3">
            <button type="button" onClick={markAllRead} className="text-xs font-semibold text-[#0E6B46] hover:underline">
              Tout marquer comme lu
            </button>
          </div>
        )}

        {notifications.length === 0 ? (
          <EmptyState icon={BellIcon} title="Aucune notification" description="Les alertes sur vos ventes, versements et produits apparaîtront ici." />
        ) : (
          <div className="space-y-2">
            {notifications.map((n) => {
              const row = (
                <div
                  className={`bg-white rounded-2xl border p-3.5 flex items-start gap-3 ${
                    n.read_at ? 'border-green-900/10' : 'border-[#C9F22E]'
                  }`}
                >
                  <span className="w-10 h-10 rounded-xl bg-[#EDF8F2] flex items-center justify-center shrink-0 text-[#0A4F35]">
                    <CheckIcon size={17} />
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="text-[13.5px] font-semibold text-[#0B2318]">{n.title}</p>
                    {n.body && <p className="text-xs text-[#4C6459] mt-0.5">{n.body}</p>}
                    <p className="text-[11px] text-[#4C6459]/80 mt-1">
                      {new Date(n.created_at).toLocaleString('fr-FR', { dateStyle: 'short', timeStyle: 'short' })}
                    </p>
                  </div>
                  {!n.read_at && <span className="w-2 h-2 rounded-full bg-[#C9F22E] shrink-0 mt-1.5" aria-hidden />}
                </div>
              );
              return n.link ? (
                <Link key={n.id} href={n.link} onClick={() => markRead(n)} className="block">
                  {row}
                </Link>
              ) : (
                <button key={n.id} type="button" onClick={() => markRead(n)} className="block w-full text-left">
                  {row}
                </button>
              );
            })}
          </div>
        )}
      </section>
    </main>
  );
}
