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
  refunded: 'Remboursé',
};

/** Styles de badge par statut de vente. */
export const SALE_STATUS_BADGE: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-100',
  paid: 'bg-green-100 text-green-700 hover:bg-green-100',
  delivered: 'bg-green-100 text-green-700 hover:bg-green-100',
  failed: 'bg-red-100 text-red-700 hover:bg-red-100',
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
export const PAYOUT_STATUS_LABELS: Record<string, string> = {
  requested: 'Demandé',
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

/** Libellés des rôles utilisateurs (admin). */
export const ROLE_LABELS: Record<string, string> = {
  vendeur: 'Vendeur',
  closer: 'Affilié',
};

export function formatPrice(price: number): string {
  return `${price.toLocaleString('fr-FR')} FCFA`;
}

/** Montant minimum qu'un vendeur peut demander en versement (miroir de model.MinPayoutAmountCFA côté backend). */
export const MIN_PAYOUT_AMOUNT_CFA = 1000;
