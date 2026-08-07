import type { ReactNode } from 'react';

interface PageHeaderProps {
  eyebrow: string;
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
}

/**
 * En-tête de page de l'espace connecté : bandeau identité DIARRA
 * (dégradé vert + pattern wax + eyebrow mono + titre display).
 */
export function PageHeader({ eyebrow, title, description, actions }: PageHeaderProps) {
  return (
    <section className="gradient-green text-white relative overflow-hidden">
      <div className="wax-pattern absolute inset-0" aria-hidden />
      <div className="relative max-w-6xl mx-auto px-4 sm:px-6 py-10 lg:py-12">
        <div className="flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="font-mono text-sm text-green-300 uppercase tracking-widest">
              {eyebrow}
            </p>
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
