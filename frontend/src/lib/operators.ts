// Opérateurs mobile money supportés par PawaPay (zone CFA) — miroir du backend.
export interface Operator {
  label: string;
  provider: string;
  country: string;
}

export interface CheckoutCountry {
  code: string; // ISO 3166-1 alpha-3
  name: string;
  dialCode: string;
  operators: Operator[];
}

export const CHECKOUT_COUNTRIES: CheckoutCountry[] = [
  {
    code: 'SEN',
    name: 'Sénégal',
    dialCode: '221',
    operators: [
      { label: 'Orange Money', provider: 'ORANGE_SEN', country: 'SEN' },
      { label: 'Wave', provider: 'WAVE_SEN', country: 'SEN' },
      { label: 'Free Money', provider: 'FREE_SEN', country: 'SEN' },
    ],
  },
  {
    code: 'CIV',
    name: "Côte d'Ivoire",
    dialCode: '225',
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_MOMO_CIV', country: 'CIV' },
      { label: 'Orange Money', provider: 'ORANGE_CIV', country: 'CIV' },
      { label: 'Wave', provider: 'WAVE_CIV', country: 'CIV' },
    ],
  },
  {
    code: 'BEN',
    name: 'Bénin',
    dialCode: '229',
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_MOMO_BEN', country: 'BEN' },
      { label: 'Moov Money', provider: 'MOOV_BEN', country: 'BEN' },
    ],
  },
  {
    code: 'BFA',
    name: 'Burkina Faso',
    dialCode: '226',
    operators: [
      { label: 'Moov Money', provider: 'MOOV_BFA', country: 'BFA' },
      { label: 'Orange Money', provider: 'ORANGE_BFA', country: 'BFA' },
    ],
  },
];

export function isLoggedIn(): boolean {
  if (typeof window === 'undefined') return false;
  return Boolean(localStorage.getItem('access_token'));
}
