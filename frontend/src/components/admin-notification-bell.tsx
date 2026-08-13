'use client';

import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';

interface Counts {
  pending_moderation: number;
  open_tickets: number;
  failed_sales_24h: number;
  total: number;
}

const EMPTY: Counts = { pending_moderation: 0, open_tickets: 0, failed_sales_24h: 0, total: 0 };

// Cloche de notifications admin : compteurs (modération, tickets, échecs de
// paiement) rafraîchis périodiquement, avec menu déroulant de raccourcis.
export function AdminNotificationBell() {
  const [counts, setCounts] = useState<Counts>(EMPTY);
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const load = () => {
      api
        .getAdminNotifications()
        .then(setCounts)
        .catch(() => {});
    };
    load();
    const interval = setInterval(load, 60000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, []);

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label="Notifications admin"
        className="relative inline-flex items-center justify-center w-10 h-10 rounded-lg text-white/80 hover:text-white hover:bg-white/10 transition-colors"
      >
        <svg width={22} height={22} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden>
          <path d="M6 8a6 6 0 0 1 12 0c0 5 2 6 2 6H4s2-1 2-6" />
          <path d="M10.5 20a1.5 1.5 0 0 0 3 0" />
        </svg>
        {counts.total > 0 && (
          <span className="absolute top-1 right-1 min-w-[16px] h-4 px-1 rounded-full bg-lime text-green-950 text-[10px] font-bold flex items-center justify-center">
            {counts.total > 99 ? '99+' : counts.total}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 mt-2 w-72 bg-white rounded-xl shadow-lift border border-green-900/10 py-2 z-50 text-green-950">
          <p className="px-4 py-1.5 text-[11px] uppercase tracking-widest text-green-900/40 font-mono">
            Notifications
          </p>
          <Link
            href="/admin/products"
            onClick={() => setOpen(false)}
            className="flex items-center justify-between px-4 py-2.5 hover:bg-green-900/5 text-sm"
          >
            <span>Produits en attente de modération</span>
            <span className="font-mono font-bold">{counts.pending_moderation}</span>
          </Link>
          <Link
            href="/support"
            onClick={() => setOpen(false)}
            className="flex items-center justify-between px-4 py-2.5 hover:bg-green-900/5 text-sm"
          >
            <span>Tickets support ouverts</span>
            <span className="font-mono font-bold">{counts.open_tickets}</span>
          </Link>
          <Link
            href="/admin/sales"
            onClick={() => setOpen(false)}
            className="flex items-center justify-between px-4 py-2.5 hover:bg-green-900/5 text-sm"
          >
            <span>Paiements échoués (24h)</span>
            <span className="font-mono font-bold">{counts.failed_sales_24h}</span>
          </Link>
          {counts.total === 0 && (
            <p className="px-4 py-3 text-sm text-green-900/50">Rien à signaler pour le moment.</p>
          )}
        </div>
      )}
    </div>
  );
}
