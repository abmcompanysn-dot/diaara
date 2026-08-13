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
//
// phoneLength = nombre de chiffres attendu du numéro local (sans le 0, sans
// l'indicatif) — indicatif à titre de guide pour l'UI, la validation finale
// reste côté PawaPay. logo = fichier dans /public/payments ; sinon badge texte
// (badgeColor/badgeText, mêmes conventions que la page d'accueil).
export interface PayoutOperator {
  label: string;
  provider: string;
  logo?: string;
  badgeColor?: string;
  badgeText?: string;
}

export interface PayoutCountry {
  code: string;
  name: string;
  dialCode: string;
  phoneLength: number;
  operators: PayoutOperator[];
}

export const PAYOUT_COUNTRIES: PayoutCountry[] = [
  {
    code: 'SEN',
    name: 'Sénégal',
    dialCode: '221',
    phoneLength: 9,
    operators: [
      { label: 'Orange Money', provider: 'ORANGE_SEN', logo: 'orange-money.png' },
      { label: 'Wave', provider: 'WAVE_SEN', badgeColor: 'bg-green-400', badgeText: 'text-green-950' },
      { label: 'Free Money', provider: 'FREE_SEN', badgeColor: 'bg-white border border-green-900/15', badgeText: 'text-green-950' },
    ],
  },
  {
    code: 'CIV',
    name: "Côte d'Ivoire",
    dialCode: '225',
    phoneLength: 10,
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_MOMO_CIV', logo: 'mtn-momo.png' },
      { label: 'Orange Money', provider: 'ORANGE_CIV', logo: 'orange-money.png' },
      { label: 'Wave', provider: 'WAVE_CIV', badgeColor: 'bg-green-400', badgeText: 'text-green-950' },
    ],
  },
  {
    code: 'BEN',
    name: 'Bénin',
    dialCode: '229',
    phoneLength: 8,
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_MOMO_BEN', logo: 'mtn-momo.png' },
      { label: 'Moov Money', provider: 'MOOV_BEN', logo: 'moov-money.png' },
    ],
  },
  {
    code: 'BFA',
    name: 'Burkina Faso',
    dialCode: '226',
    phoneLength: 8,
    operators: [
      { label: 'Moov Money', provider: 'MOOV_BFA', logo: 'moov-money.png' },
      { label: 'Orange Money', provider: 'ORANGE_BFA', logo: 'orange-money.png' },
    ],
  },
];

export function isLoggedIn(): boolean {
  if (typeof window === 'undefined') return false;
  return Boolean(localStorage.getItem('access_token'));
}
