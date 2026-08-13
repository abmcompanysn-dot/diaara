// Les 20 pays couverts par PawaPay (docs.pawapay.io/v2/docs/providers). Le
// catalogue DIARRA est tarifé en FCFA (XOF) ; pour les pays hors zone
// XOF/XAF, le backend convertit automatiquement le montant vers la devise
// locale avant la redirection PawaPay (voir payment.ConvertFromXOF).
// PawaPay gère lui-même le choix de l'opérateur mobile money sur sa page.
export interface CheckoutCountry {
  code: string; // ISO 3166-1 alpha-3
  name: string;
  currency: string;
}

export const CHECKOUT_COUNTRIES: CheckoutCountry[] = [
  { code: 'SEN', name: 'Sénégal', currency: 'XOF' },
  { code: 'CIV', name: "Côte d'Ivoire", currency: 'XOF' },
  { code: 'BEN', name: 'Bénin', currency: 'XOF' },
  { code: 'BFA', name: 'Burkina Faso', currency: 'XOF' },
  { code: 'CMR', name: 'Cameroun', currency: 'XAF' },
  { code: 'GAB', name: 'Gabon', currency: 'XAF' },
  { code: 'COG', name: 'Congo-Brazzaville', currency: 'XAF' },
  { code: 'COD', name: 'RD Congo', currency: 'CDF' },
  { code: 'GHA', name: 'Ghana', currency: 'GHS' },
  { code: 'NGA', name: 'Nigeria', currency: 'NGN' },
  { code: 'KEN', name: 'Kenya', currency: 'KES' },
  { code: 'RWA', name: 'Rwanda', currency: 'RWF' },
  { code: 'UGA', name: 'Ouganda', currency: 'UGX' },
  { code: 'TZA', name: 'Tanzanie', currency: 'TZS' },
  { code: 'ZMB', name: 'Zambie', currency: 'ZMW' },
  { code: 'MWI', name: 'Malawi', currency: 'MWK' },
  { code: 'MOZ', name: 'Mozambique', currency: 'MZN' },
  { code: 'LSO', name: 'Lesotho', currency: 'LSL' },
  { code: 'SLE', name: 'Sierra Leone', currency: 'SLE' },
  { code: 'ETH', name: 'Éthiopie', currency: 'ETB' },
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
