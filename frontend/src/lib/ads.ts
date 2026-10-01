// Sponsorisation de produits sur Facebook/Instagram — types, libellés et
// calculs partagés par les pages vendeur et admin (voir backend
// handler/ad_handler.go, handler/ad_meta_connect.go et model/ad_campaign.go).
//
// Le vendeur connecte SON compte Facebook, pub sur SA page avec SON compte
// publicitaire, et paie Meta directement : DIARRA ne prélève rien.

export interface AdCampaign {
  id: string;
  product_id: string;
  product_title?: string;
  vendor_email?: string;
  platform: string;
  status: string;
  budget_cfa: number;
  ad_account_id: string;
  page_id: string;
  currency: string;
  duration_days: number;
  countries: string[];
  message: string;
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

// Connexion Facebook du vendeur (le jeton n'est jamais renvoyé au navigateur).
export interface MetaConnection {
  fb_user_id: string;
  fb_user_name: string;
  token_expires_at?: string;
  page_id?: string;
  page_name?: string;
  ad_account_id?: string;
  ad_account_name?: string;
  currency?: string;
  needs_reconnect: boolean;
  created_at: string;
  updated_at: string;
}

export interface MetaStatus {
  configured: boolean;
  enabled: boolean;
  connected: boolean;
  ready: boolean;
  connection: MetaConnection | null;
  supported_currencies: string[];
}

export interface MetaPage {
  id: string;
  name: string;
}

export interface MetaAdAccount {
  id: string;
  name: string;
  currency: string;
  account_status: number;
  active: boolean;
  currency_supported: boolean;
  usable: boolean;
}

export const AD_STATUS_LABELS: Record<string, string> = {
  launching: 'Création en cours',
  in_review: 'En vérification chez Meta',
  active: 'En diffusion',
  completed: 'Terminée',
  stopped: 'Arrêtée',
  rejected: 'Refusée par Meta',
  failed: 'Non lancée',
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

// Statut Meta d'un compte publicitaire (account_status), pour expliquer
// pourquoi un compte n'est pas utilisable.
export const AD_ACCOUNT_STATUS_LABELS: Record<number, string> = {
  1: 'Actif',
  2: 'Désactivé',
  3: 'Paiement en attente',
  7: 'En vérification',
  8: 'En attente de règlement',
  9: 'Période de grâce',
  100: 'Fermeture en cours',
  101: 'Fermé',
};

// Miroir de backend minAdBudget : budget total minimum = minimum par jour ×
// nombre de jours.
export function minBudgetFor(days: number, minDaily: number): number {
  return days * minDaily;
}

// Arrondit au millier supérieur, pour des montants proposés lisibles.
export function roundUpThousand(n: number): number {
  return Math.ceil(n / 1000) * 1000;
}

// Parité fixe du franc CFA : 1 EUR = 655,957 FCFA.
const XOF_PER_EUR = 655.957;

// Montant approximatif dans la devise du compte publicitaire du vendeur
// (affichage seulement ; la conversion qui fait foi est faite par le backend).
// USD : taux indicatif, aligné sur payment.USDRates côté backend.
const XOF_PER_USD = 568.76;

export function formatInAccountCurrency(amountCfa: number, currency?: string): string | null {
  switch ((currency || '').toUpperCase()) {
    case 'EUR':
      return `${(Math.floor((amountCfa / XOF_PER_EUR) * 100) / 100).toLocaleString('fr-FR', { minimumFractionDigits: 2 })} €`;
    case 'USD':
      return `${(Math.floor((amountCfa / XOF_PER_USD) * 100) / 100).toLocaleString('fr-FR', { minimumFractionDigits: 2 })} $`;
    default:
      return null; // XOF/XAF : même montant, rien à ajouter
  }
}

export function ctr(clicks: number, impressions: number): string {
  if (!impressions) return '—';
  return `${((clicks / impressions) * 100).toLocaleString('fr-FR', { maximumFractionDigits: 1 })} %`;
}

// Messages de retour de Facebook Login (?meta=error&reason=...).
export const META_CONNECT_ERRORS: Record<string, string> = {
  denied: 'Connexion Facebook annulée. Vous pouvez réessayer quand vous voulez.',
  permissions:
    'DIARRA a besoin de l’autorisation « Gérer vos publicités » pour créer vos pubs. Reconnectez-vous et laissez toutes les autorisations cochées.',
  expired: 'La demande de connexion a expiré. Recommencez.',
  state: 'La connexion n’a pas pu être vérifiée. Recommencez depuis cette page, dans le même navigateur.',
  exchange: 'Facebook n’a pas validé la connexion. Réessayez dans quelques instants.',
  unavailable: 'La sponsorisation n’est pas encore disponible.',
  internal: 'Une erreur est survenue pendant la connexion. Réessayez.',
};
