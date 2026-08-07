import type { ReactNode } from 'react';
import type { IconType } from '@/lib/navigation';
import { CartIcon } from '@/components/icons';

interface EmptyStateProps {
  icon?: IconType;
  title: string;
  description?: string;
  action?: ReactNode;
}

/**
 * État vide : message directionnel + action (invitation à agir).
 */
export function EmptyState({ icon: Icon = CartIcon, title, description, action }: EmptyStateProps) {
  return (
    <div className="text-center py-14 px-6 border border-dashed border-green-900/15 rounded-xl bg-white/60">
      <span
        className="mx-auto w-12 h-12 rounded-lg gradient-green-soft flex items-center justify-center mb-4"
        aria-hidden
      >
        <Icon size={24} className="text-green-700" />
      </span>
      <p className="font-display font-bold text-lg text-green-950">{title}</p>
      {description && <p className="text-sm text-green-900/60 mt-1">{description}</p>}
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}
