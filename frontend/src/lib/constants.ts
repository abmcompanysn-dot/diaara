/** Libellés uniques des catégories produits (partagés catalogue / fiche / formulaire). */
export const CATEGORY_LABELS: Record<string, string> = {
  subscription: "Clés d'abonnement",
  account: 'Comptes',
  ebook: 'Ebooks',
  pdf: 'PDF',
  software: 'Logiciels',
  service: 'Services',
  other: 'Autres',
};

/** Statuts de commande / vente (français). */
export const ORDER_STATUS_LABELS: Record<string, string> = {
  pending: 'En attente',
  paid: 'Payé',
  delivered: 'Livré',
  failed: 'Échec',
  refund_pending: 'Remboursement en cours',
  refunded: 'Remboursé',
};

/** Styles de badge par statut de vente. */
export const SALE_STATUS_BADGE: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-100',
  paid: 'bg-green-100 text-green-700 hover:bg-green-100',
  delivered: 'bg-green-100 text-green-700 hover:bg-green-100',
  failed: 'bg-red-100 text-red-700 hover:bg-red-100',
  refund_pending: 'bg-blue-100 text-blue-700 hover:bg-blue-100',
  refunded: 'bg-gray-100 text-gray-600 hover:bg-gray-100',
};

/** Statuts de modération des produits. */
export const PRODUCT_STATUS_LABELS: Record<string, string> = {
  pending: 'En attente',
  approved: 'Approuvé',
  rejected: 'Refusé',
};

export const PRODUCT_STATUS_BADGE: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-100',
  approved: 'bg-green-100 text-green-700 hover:bg-green-100',
  rejected: 'bg-red-100 text-red-700 hover:bg-red-100',
};

/** Statuts de versement. */
// Libellés « vérité » (vue admin). Côté vendeur, la page revenus remappe tout
// sauf 'paid' vers « En cours de traitement » (voir vendorPayoutLabel).
export const PAYOUT_STATUS_LABELS: Record<string, string> = {
  requested: 'En attente',
  processing: 'En traitement',
  paid: 'Payé',
  failed: 'Échec',
};

export const PAYOUT_STATUS_BADGE: Record<string, string> = {
  paid: 'bg-green-100 text-green-700 hover:bg-green-100',
  processing: 'bg-blue-100 text-blue-700 hover:bg-blue-100',
  requested: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-100',
  failed: 'bg-red-100 text-red-700 hover:bg-red-100',
};

/** Statuts des tickets support. */
export const TICKET_STATUS_LABELS: Record<string, string> = {
  open: 'Ouvert',
  answered: 'Répondu',
  closed: 'Fermé',
};

/** Moyens de paiement affichés (badges de marque). */
export const PAYMENTS = [
  { name: 'Wave', color: 'bg-green-400', text: 'text-green-950' },
  { name: 'Orange Money', color: 'bg-money-om', text: 'text-white' },
  { name: 'MTN MoMo', color: 'bg-money-mtn', text: 'text-green-950' },
  { name: 'Free', color: 'bg-white', text: 'text-green-950' },
];

/** Logos de tous les opérateurs mobile money couverts par le site (fichiers
 * dans /public/payments, mêmes assets que PAYOUT_COUNTRIES). */
export const PAYMENT_LOGOS = [
  { name: 'Wave', logo: 'wave.png' },
  { name: 'Orange Money', logo: 'orange-money.png' },
  { name: 'MTN MoMo', logo: 'mtn-momo.png' },
  { name: 'Moov Money', logo: 'moov-money.png' },
  { name: 'Airtel Money', logo: 'at-money.png' },
  { name: 'Vodacom M-Pesa', logo: 'vodacom.png' },
  { name: 'M-Pesa', logo: 'mpesa.png' },
  { name: 'Safaricom M-Pesa', logo: 'safaricom-mpesa.png' },
  { name: 'HaloPesa', logo: 'halopesa.png' },
  { name: 'TNM Mpamba', logo: 'tnm.png' },
  { name: 'Zamtel Money', logo: 'zamtel.png' },
  { name: 'Movitel', logo: 'movitel.png' },
  { name: 'Mixx by Yas', logo: 'mixx-yas.png' },
  { name: 'Telecel Cash', logo: 'telecel-cash.png' },
];

/** Libellés des rôles utilisateurs (admin). */
export const ROLE_LABELS: Record<string, string> = {
  vendeur: 'Vendeur',
  closer: 'Affilié',
};

export function formatPrice(price: number): string {
  return `${price.toLocaleString('fr-FR')} FCFA`;
}

// Trophées vendeur (voir backend/internal/model/vendor_tier.go) — même clés
// que model.VendorTier ("bronze" | "or" | "diamant" | "").
export const VENDOR_TIERS: Record<string, { label: string; color: string; bg: string; next?: string; threshold: number }> = {
  bronze: { label: 'Bronze', color: '#8C5A2B', bg: '#F5E3D3', next: 'Or', threshold: 500_000 },
  or: { label: 'Or', color: '#8A6300', bg: '#FFF3D6', next: 'Diamant', threshold: 5_000_000 },
  diamant: { label: 'Diamant', color: '#0E6B8C', bg: '#E0F7FA', threshold: 5_000_000 },
};
