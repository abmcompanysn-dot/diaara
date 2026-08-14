'use client';

import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { BellIcon } from '@/components/icons';
import { cn } from '@/lib/utils';

interface Notification {
  id: string;
  type: string;
  title: string;
  body?: string;
  link?: string;
  read_at: string | null;
  created_at: string;
}

const POLL_INTERVAL_MS = 30_000;

export function NotificationBell() {
  const [open, setOpen] = useState(false);
  const [count, setCount] = useState(0);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loaded, setLoaded] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    loadCount();
    const interval = setInterval(loadCount, POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onClickOutside = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onClickOutside);
    return () => document.removeEventListener('mousedown', onClickOutside);
  }, [open]);

  const loadCount = async () => {
    try {
      const result = await api.getUnreadNotificationCount();
      setCount(result.count);
    } catch {
      // Silencieux : la clochette reste juste sans badge si l'appel échoue.
    }
  };

  const toggleOpen = async () => {
    const next = !open;
    setOpen(next);
    if (next && !loaded) {
      try {
        const result = await api.getNotifications();
        setNotifications(result.notifications);
        setLoaded(true);
      } catch {
        // Silencieux : le panneau reste vide si l'appel échoue.
      }
    }
  };

  const handleMarkAllRead = async () => {
    try {
      await api.markAllNotificationsRead();
      setNotifications((list) => list.map((n) => ({ ...n, read_at: n.read_at || new Date().toISOString() })));
      setCount(0);
    } catch {
      // Pas grave, l'utilisateur peut réessayer.
    }
  };

  const handleClickNotification = async (n: Notification) => {
    if (!n.read_at) {
      try {
        await api.markNotificationRead(n.id);
        setNotifications((list) => list.map((x) => (x.id === n.id ? { ...x, read_at: new Date().toISOString() } : x)));
        setCount((c) => Math.max(0, c - 1));
      } catch {
        // Pas grave.
      }
    }
    setOpen(false);
  };

  return (
    <div className="relative" ref={panelRef}>
      <button
        type="button"
        onClick={toggleOpen}
        aria-label="Notifications"
        aria-expanded={open}
        className="relative inline-flex items-center justify-center w-10 h-10 rounded-lg text-white/80 hover:text-white hover:bg-white/10 transition-colors"
      >
        <BellIcon size={20} />
        {count > 0 && (
          <span className="absolute top-1 right-1 min-w-[16px] h-4 px-1 rounded-full bg-lime text-green-950 text-[10px] font-bold flex items-center justify-center">
            {count > 9 ? '9+' : count}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 mt-2 w-80 max-w-[90vw] bg-white rounded-xl shadow-lift border border-green-900/10 overflow-hidden z-50">
          <div className="flex items-center justify-between px-4 py-3 border-b border-green-900/5">
            <p className="font-semibold text-sm text-green-950">Notifications</p>
            {count > 0 && (
              <button type="button" onClick={handleMarkAllRead} className="text-xs text-green-700 hover:underline">
                Tout marquer comme lu
              </button>
            )}
          </div>
          <div className="max-h-96 overflow-y-auto">
            {notifications.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-8 px-4">Aucune notification pour le moment.</p>
            ) : (
              notifications.map((n) => {
                const content = (
                  <div
                    className={cn(
                      'px-4 py-3 border-b border-green-900/5 last:border-0 hover:bg-green-50/60 transition-colors',
                      !n.read_at && 'bg-lime/5'
                    )}
                  >
                    <div className="flex items-start gap-2">
                      {!n.read_at && <span className="w-2 h-2 rounded-full bg-lime shrink-0 mt-1.5" aria-hidden />}
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-medium text-green-950">{n.title}</p>
                        {n.body && <p className="text-xs text-green-900/60 mt-0.5">{n.body}</p>}
                        <p className="text-[11px] text-green-900/40 mt-1">
                          {new Date(n.created_at).toLocaleString('fr-FR', { dateStyle: 'short', timeStyle: 'short' })}
                        </p>
                      </div>
                    </div>
                  </div>
                );
                return n.link ? (
                  <Link key={n.id} href={n.link} onClick={() => handleClickNotification(n)} className="block">
                    {content}
                  </Link>
                ) : (
                  <button key={n.id} type="button" onClick={() => handleClickNotification(n)} className="block w-full text-left">
                    {content}
                  </button>
                );
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
}
