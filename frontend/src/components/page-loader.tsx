import { cn } from '@/lib/utils';

/**
 * État de chargement de page : squelette discret centré, remplace
 * les simples textes « Chargement... ».
 */
export function PageLoader({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        'min-h-[40vh] flex flex-col items-center justify-center gap-4 text-green-900/50',
        className
      )}
      role="status"
      aria-live="polite"
    >
      <span
        className="w-9 h-9 rounded-full border-[3px] border-green-900/10 border-t-green-600 animate-spin"
        aria-hidden
      />
      <p className="font-mono text-sm">chargement…</p>
    </div>
  );
}
