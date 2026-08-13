'use client';

import { CheckIcon, AlertTriangleIcon, XIcon } from '@/components/icons';
import type { ToastItem } from '@/hooks/use-toast';

const VARIANT_STYLES: Record<ToastItem['variant'], string> = {
  success: 'bg-green-50 border-green-200 text-green-900',
  error: 'bg-red-50 border-red-200 text-red-900',
  info: 'bg-white border-green-900/10 text-green-950',
};

const VARIANT_ICON: Record<ToastItem['variant'], React.ReactNode> = {
  success: <CheckIcon size={16} className="text-green-600 shrink-0 mt-0.5" />,
  error: <AlertTriangleIcon size={16} className="text-red-600 shrink-0 mt-0.5" />,
  info: <AlertTriangleIcon size={16} className="text-green-700 shrink-0 mt-0.5" />,
};

export function Toaster({ toasts, onDismiss }: { toasts: ToastItem[]; onDismiss: (id: number) => void }) {
  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 left-4 sm:left-auto z-50 flex flex-col gap-2 sm:w-96" role="region" aria-live="polite">
      {toasts.map((t) => (
        <div
          key={t.id}
          role={t.variant === 'error' ? 'alert' : 'status'}
          className={`flex items-start gap-2.5 rounded-xl border p-3.5 shadow-lift animate-in fade-in slide-in-from-bottom-2 ${VARIANT_STYLES[t.variant]}`}
        >
          {VARIANT_ICON[t.variant]}
          <div className="flex-1 min-w-0">
            <p className="text-sm font-semibold">{t.title}</p>
            {t.description && <p className="text-sm mt-0.5 opacity-80">{t.description}</p>}
          </div>
          <button
            type="button"
            onClick={() => onDismiss(t.id)}
            aria-label="Fermer"
            className="shrink-0 opacity-50 hover:opacity-100 transition-opacity"
          >
            <XIcon size={14} />
          </button>
        </div>
      ))}
    </div>
  );
}
