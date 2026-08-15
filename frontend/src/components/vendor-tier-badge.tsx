import { TrophyIcon } from '@/components/icons';
import { VENDOR_TIERS } from '@/lib/constants';

export function VendorTierBadge({ tier }: { tier?: string | null }) {
  if (!tier || !VENDOR_TIERS[tier]) return null;
  const t = VENDOR_TIERS[tier];
  return (
    <span
      className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-bold"
      style={{ background: t.bg, color: t.color }}
    >
      <TrophyIcon size={13} />
      {t.label}
    </span>
  );
}
