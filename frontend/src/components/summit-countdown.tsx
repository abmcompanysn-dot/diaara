'use client';

import { useEffect, useState } from 'react';

const SUMMIT_DATE = new Date('2026-10-26T15:00:00+00:00').getTime();

function getRemaining() {
  const diff = Math.max(0, SUMMIT_DATE - Date.now());
  return {
    days: Math.floor(diff / 86400000),
    hours: Math.floor((diff % 86400000) / 3600000),
    minutes: Math.floor((diff % 3600000) / 60000),
    seconds: Math.floor((diff % 60000) / 1000),
    done: diff <= 0,
  };
}

const UNITS: { key: keyof ReturnType<typeof getRemaining>; label: string }[] = [
  { key: 'days', label: 'jours' },
  { key: 'hours', label: 'heures' },
  { key: 'minutes', label: 'min' },
  { key: 'seconds', label: 'sec' },
];

export function SummitCountdown() {
  const [remaining, setRemaining] = useState<ReturnType<typeof getRemaining> | null>(null);

  useEffect(() => {
    setRemaining(getRemaining());
    const interval = setInterval(() => setRemaining(getRemaining()), 1000);
    return () => clearInterval(interval);
  }, []);

  if (!remaining || remaining.done) return null;

  return (
    <div className="flex items-center justify-center gap-3 sm:gap-4" role="timer" aria-label="Temps restant avant le DIARRA Summit">
      {UNITS.map((u) => (
        <div key={u.key} className="flex flex-col items-center">
          <span className="font-display text-2xl sm:text-3xl font-bold text-lime tabular-nums">
            {String(remaining[u.key]).padStart(2, '0')}
          </span>
          <span className="text-[11px] uppercase tracking-wide text-white/60">{u.label}</span>
        </div>
      ))}
    </div>
  );
}
