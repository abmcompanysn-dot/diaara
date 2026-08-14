import { cn } from '@/lib/utils';

interface SegmentedTabsProps<T extends string> {
  options: { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
}

/** Onglets pilules horizontaux (défilables) — filtre de statut espace vendeur. */
export function SegmentedTabs<T extends string>({ options, value, onChange }: SegmentedTabsProps<T>) {
  return (
    <div className="flex gap-2 overflow-x-auto pb-1 -mx-4 px-4 sm:mx-0 sm:px-0" role="tablist">
      {options.map((opt) => {
        const active = opt.value === value;
        return (
          <button
            key={opt.value}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onChange(opt.value)}
            className={cn(
              'shrink-0 text-xs font-semibold px-3.5 py-2 rounded-full border transition-colors whitespace-nowrap',
              active
                ? 'bg-[#0E6B46] text-white border-[#0E6B46]'
                : 'bg-white text-green-900/60 border-green-900/10 hover:border-green-900/20'
            )}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}
