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

export function isLoggedIn(): boolean {
  if (typeof window === 'undefined') return false;
  return Boolean(localStorage.getItem('access_token'));
}
