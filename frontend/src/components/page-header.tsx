import type { ReactNode } from 'react';
import Link from 'next/link';
import { ArrowLeftIcon } from '@/components/icons';

interface PageHeaderProps {
  eyebrow: string;
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
  /** Lien de retour affiché en rond en haut à gauche à la place de l'eyebrow (écrans secondaires de l'espace vendeur). */
  back?: string;
}

/**
 * En-tête de page de l'espace connecté : bandeau identité DIARRA
 * (dégradé vert + pattern wax + eyebrow mono + titre display).
 */
export function PageHeader({ eyebrow, title, description, actions, back }: PageHeaderProps) {
  return (
    <section className="gradient-green text-white relative overflow-hidden">
      <div className="wax-pattern absolute inset-0" aria-hidden />
      <div className="relative max-w-6xl mx-auto px-4 sm:px-6 py-10 lg:py-12">
        <div className="flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
          <div>
            {back ? (
              <Link
                href={back}
                aria-label="Retour"
                className="inline-flex w-9 h-9 rounded-xl bg-white/15 border border-white/20 items-center justify-center hover:bg-white/25 transition-colors"
              >
                <ArrowLeftIcon size={18} className="text-white" />
              </Link>
            ) : (
              <p className="font-mono text-sm text-green-300 uppercase tracking-widest">
                {eyebrow}
              </p>
            )}
            <h1 className="font-display text-3xl sm:text-4xl font-bold tracking-tight mt-2">
              {title}
            </h1>
            {description && (
              <div className="mt-2 text-white/75 max-w-lg text-sm sm:text-base">{description}</div>
            )}
          </div>
          {actions && <div className="shrink-0 flex flex-wrap items-center gap-3">{actions}</div>}
        </div>
      </div>
    </section>
  );
}
