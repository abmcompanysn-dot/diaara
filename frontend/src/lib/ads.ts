// Sponsorisation de produits sur Facebook/Instagram — libellés et calculs
// partagés par les pages vendeur et admin (voir backend handler/ad_handler.go
// et model/ad_campaign.go).

export interface AdCampaign {
  id: string;
  product_id: string;
  product_title?: string;
  vendor_email?: string;
  platform: string;
  status: string;
  amount_cfa: number;
  commission_cfa: number;
  ad_budget_cfa: number;
  duration_days: number;
  countries: string[];
  message: string;
  refunded: boolean;
  starts_at?: string;
  ends_at?: string;
  failure_reason?: string;
  impressions: number;
  reach: number;
  clicks: number;
  spend_cfa: number;
  stats_updated_at?: string;
  created_at: string;
}

export const AD_STATUS_LABELS: Record<string, string> = {
  launching: 'Création en cours',
  in_review: 'En vérification chez Meta',
  active: 'En diffusion',
  completed: 'Terminée',
  stopped: 'Arrêtée',
  rejected: 'Refusée par Meta (remboursée)',
  failed: 'Non lancée (remboursée)',
};

export const AD_STATUS_BADGE: Record<string, string> = {
  launching: 'bg-amber-100 text-amber-800 hover:bg-amber-100',
  in_review: 'bg-amber-100 text-amber-800 hover:bg-amber-100',
  active: 'bg-green-100 text-green-800 hover:bg-green-100',
  completed: 'bg-green-900/5 text-green-900/70 hover:bg-green-900/5',
  stopped: 'bg-green-900/5 text-green-900/70 hover:bg-green-900/5',
  rejected: 'bg-red-100 text-red-800 hover:bg-red-100',
  failed: 'bg-red-100 text-red-800 hover:bg-red-100',
};

// Miroir de backend splitAmount / minAmountFor (commission arrondie au FCFA
// supérieur, minimum = budget pub journalier minimum × jours, avant commission).
export function splitAmount(amount: number, commissionPct: number): { commission: number; adBudget: number } {
  const commission = Math.ceil((amount * commissionPct) / 100);
  return { commission, adBudget: amount - commission };
}

export function minAmountFor(days: number, minDaily: number, commissionPct: number): number {
  return Math.ceil((days * minDaily) / (1 - commissionPct / 100));
}

// Arrondit au millier supérieur, pour des montants proposés lisibles.
export function roundUpThousand(n: number): number {
  return Math.ceil(n / 1000) * 1000;
}

export function ctr(clicks: number, impressions: number): string {
  if (!impressions) return '—';
  return `${((clicks / impressions) * 100).toLocaleString('fr-FR', { maximumFractionDigits: 1 })} %`;
}
