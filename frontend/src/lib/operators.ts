// Pays de la zone XOF (franc CFA) où le paiement DIARRA est actif. Les prix du
// catalogue sont en XOF, donc ce choix reste limité à cette zone monétaire
// (PawaPay gère lui-même le choix de l'opérateur mobile money sur sa page).
export interface CheckoutCountry {
  code: string; // ISO 3166-1 alpha-3
  name: string;
}

export const CHECKOUT_COUNTRIES: CheckoutCountry[] = [
  { code: 'SEN', name: 'Sénégal' },
  { code: 'CIV', name: "Côte d'Ivoire" },
  { code: 'BEN', name: 'Bénin' },
  { code: 'BFA', name: 'Burkina Faso' },
];

// Opérateurs mobile money par pays — utilisé pour le moyen de versement
// vendeur (PawaPay n'a pas de page hébergée pour les versements, contrairement
// au checkout : l'opérateur doit être choisi explicitement ici). Miroir exact
// de payment.XOFOperators côté backend.
export interface PayoutOperator {
  label: string;
  provider: string;
}

export interface PayoutCountry {
  code: string;
  name: string;
  operators: PayoutOperator[];
}

export const PAYOUT_COUNTRIES: PayoutCountry[] = [
  {
    code: 'SEN',
    name: 'Sénégal',
    operators: [
      { label: 'Orange Money', provider: 'ORANGE_SEN' },
      { label: 'Wave', provider: 'WAVE_SEN' },
      { label: 'Free Money', provider: 'FREE_SEN' },
    ],
  },
  {
    code: 'CIV',
    name: "Côte d'Ivoire",
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_MOMO_CIV' },
      { label: 'Orange Money', provider: 'ORANGE_CIV' },
      { label: 'Wave', provider: 'WAVE_CIV' },
    ],
  },
  {
    code: 'BEN',
    name: 'Bénin',
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_MOMO_BEN' },
      { label: 'Moov Money', provider: 'MOOV_BEN' },
    ],
  },
  {
    code: 'BFA',
    name: 'Burkina Faso',
    operators: [
      { label: 'Moov Money', provider: 'MOOV_BFA' },
      { label: 'Orange Money', provider: 'ORANGE_BFA' },
    ],
  },
];

export function isLoggedIn(): boolean {
  if (typeof window === 'undefined') return false;
  return Boolean(localStorage.getItem('access_token'));
}
