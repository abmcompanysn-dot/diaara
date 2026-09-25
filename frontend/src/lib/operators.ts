// Les 20 pays couverts par PawaPay (docs.pawapay.io/v2/docs/providers). Le
// catalogue DIARRA est tarifé en FCFA (XOF) ; pour les pays hors zone
// XOF/XAF, le backend convertit automatiquement le montant vers la devise
// locale avant la redirection PawaPay (voir payment.ConvertFromXOF).
// PawaPay gère lui-même le choix de l'opérateur mobile money sur sa page.
export interface CheckoutCountry {
  code: string; // ISO 3166-1 alpha-3
  name: string;
  currency: string;
  flag: string; // Emoji drapeau, pour le sélecteur pays du checkout.
}

// Retiré temporairement : BFA, ETH, GHA, LSO, MWI, MOZ, NGA, TZA — non
// activés pour les dépôts sur notre compte PawaPay (DEPOSITS_NOT_ALLOWED en
// production, incident 2026-08-27/28). À réintégrer dès l'activation
// confirmée côté PawaPay (vérifier via GET /v2/active-conf).
export const CHECKOUT_COUNTRIES: CheckoutCountry[] = [
  { code: 'SEN', name: 'Sénégal', currency: 'XOF', flag: '🇸🇳' },
  { code: 'CIV', name: "Côte d'Ivoire", currency: 'XOF', flag: '🇨🇮' },
  { code: 'BEN', name: 'Bénin', currency: 'XOF', flag: '🇧🇯' },
  { code: 'CMR', name: 'Cameroun', currency: 'XAF', flag: '🇨🇲' },
  { code: 'GAB', name: 'Gabon', currency: 'XAF', flag: '🇬🇦' },
  { code: 'COG', name: 'Congo-Brazzaville', currency: 'XAF', flag: '🇨🇬' },
  { code: 'COD', name: 'RD Congo', currency: 'CDF', flag: '🇨🇩' },
  { code: 'KEN', name: 'Kenya', currency: 'KES', flag: '🇰🇪' },
  { code: 'RWA', name: 'Rwanda', currency: 'RWF', flag: '🇷🇼' },
  { code: 'UGA', name: 'Ouganda', currency: 'UGX', flag: '🇺🇬' },
  { code: 'ZMB', name: 'Zambie', currency: 'ZMW', flag: '🇿🇲' },
  { code: 'SLE', name: 'Sierra Leone', currency: 'SLE', flag: '🇸🇱' },
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
  // pawaPayCode — PAYDUNYA_COUNTRIES uniquement : code PawaPay équivalent
  // (voir backend payment.PayDunyaOperator.PawaPayCode), pour fusionner les
  // deux listes en opérateurs logiques (voir LOGICAL_OPERATORS).
  pawaPayCode?: string;
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
      { label: 'Wave', provider: 'WAVE_SEN', logo: 'wave.png' },
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
      { label: 'Wave', provider: 'WAVE_CIV', logo: 'wave.png' },
    ],
  },
  {
    code: 'BEN',
    name: 'Bénin',
    dialCode: '229',
    // Depuis la réforme de numérotation 2021, le "01" initial fait partie
    // intégrante du numéro béninois (pas un préfixe à retirer) : 10 chiffres
    // au total, ex "01 90 01 02 03" -> +229 01 90 01 02 03.
    phoneLength: 10,
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
  {
    code: 'CMR',
    name: 'Cameroun',
    dialCode: '237',
    phoneLength: 9,
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_MOMO_CMR', logo: 'mtn-momo.png' },
      { label: 'Orange Money', provider: 'ORANGE_CMR', logo: 'orange-money.png' },
    ],
  },
  {
    code: 'GAB',
    name: 'Gabon',
    dialCode: '241',
    phoneLength: 8,
    operators: [
      { label: 'Airtel Money', provider: 'AIRTEL_GAB', logo: 'at-money.png' },
    ],
  },
  {
    code: 'COG',
    name: 'Congo-Brazzaville',
    dialCode: '242',
    phoneLength: 9,
    operators: [
      { label: 'Airtel Money', provider: 'AIRTEL_COG', logo: 'at-money.png' },
      { label: 'MTN MoMo', provider: 'MTN_MOMO_COG', logo: 'mtn-momo.png' },
    ],
  },
  {
    code: 'COD',
    name: 'RD Congo',
    dialCode: '243',
    phoneLength: 9,
    operators: [
      { label: 'Vodacom M-Pesa', provider: 'VODACOM_MPESA_COD', logo: 'vodacom.png' },
      { label: 'Airtel Money', provider: 'AIRTEL_COD', logo: 'at-money.png' },
      { label: 'Orange Money', provider: 'ORANGE_COD', logo: 'orange-money.png' },
    ],
  },
  {
    code: 'GHA',
    name: 'Ghana',
    dialCode: '233',
    phoneLength: 9,
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_MOMO_GHA', logo: 'mtn-momo.png' },
      { label: 'AirtelTigo Money', provider: 'AIRTELTIGO_GHA', badgeColor: 'bg-blue-100', badgeText: 'text-blue-900' },
      { label: 'Vodafone Cash', provider: 'VODAFONE_GHA', badgeColor: 'bg-red-100', badgeText: 'text-red-900' },
    ],
  },
  {
    code: 'NGA',
    name: 'Nigeria',
    dialCode: '234',
    phoneLength: 10,
    operators: [
      { label: 'Airtel Money', provider: 'AIRTEL_NGA', logo: 'at-money.png' },
      { label: 'MTN MoMo', provider: 'MTN_MOMO_NGA', logo: 'mtn-momo.png' },
    ],
  },
  {
    code: 'KEN',
    name: 'Kenya',
    dialCode: '254',
    phoneLength: 9,
    operators: [
      { label: 'M-Pesa', provider: 'MPESA_KEN', logo: 'safaricom-mpesa.png' },
    ],
  },
  {
    code: 'RWA',
    name: 'Rwanda',
    dialCode: '250',
    phoneLength: 9,
    operators: [
      { label: 'Airtel Money', provider: 'AIRTEL_RWA', logo: 'at-money.png' },
      { label: 'MTN MoMo', provider: 'MTN_MOMO_RWA', logo: 'mtn-momo.png' },
    ],
  },
  {
    code: 'UGA',
    name: 'Ouganda',
    dialCode: '256',
    phoneLength: 9,
    operators: [
      { label: 'Airtel Money', provider: 'AIRTEL_OAPI_UGA', logo: 'at-money.png' },
      { label: 'MTN MoMo', provider: 'MTN_MOMO_UGA', logo: 'mtn-momo.png' },
    ],
  },
  {
    code: 'TZA',
    name: 'Tanzanie',
    dialCode: '255',
    phoneLength: 9,
    operators: [
      { label: 'Airtel Money', provider: 'AIRTEL_TZA', logo: 'at-money.png' },
      { label: 'Vodacom M-Pesa', provider: 'VODACOM_TZA', logo: 'vodacom.png' },
      { label: 'Tigo Pesa', provider: 'TIGO_TZA', badgeColor: 'bg-blue-100', badgeText: 'text-blue-900' },
      { label: 'HaloPesa', provider: 'HALOTEL_TZA', logo: 'halopesa.png' },
    ],
  },
  {
    code: 'ZMB',
    name: 'Zambie',
    dialCode: '260',
    phoneLength: 9,
    operators: [
      { label: 'Airtel Money', provider: 'AIRTEL_OAPI_ZMB', logo: 'at-money.png' },
      { label: 'MTN MoMo', provider: 'MTN_MOMO_ZMB', logo: 'mtn-momo.png' },
      { label: 'Zamtel Money', provider: 'ZAMTEL_ZMB', logo: 'zamtel.png' },
    ],
  },
  {
    code: 'MWI',
    name: 'Malawi',
    dialCode: '265',
    phoneLength: 9,
    operators: [
      { label: 'Airtel Money', provider: 'AIRTEL_MWI', logo: 'at-money.png' },
      { label: 'TNM Mpamba', provider: 'TNM_MWI', logo: 'tnm.png' },
    ],
  },
  {
    code: 'MOZ',
    name: 'Mozambique',
    dialCode: '258',
    phoneLength: 9,
    operators: [
      { label: 'Movitel', provider: 'MOVITEL_MOZ', logo: 'movitel.png' },
      { label: 'Vodacom M-Pesa', provider: 'VODACOM_MOZ', logo: 'vodacom.png' },
    ],
  },
  {
    code: 'LSO',
    name: 'Lesotho',
    dialCode: '266',
    phoneLength: 8,
    operators: [
      { label: 'M-Pesa', provider: 'MPESA_LSO', logo: 'mpesa.png' },
    ],
  },
  {
    code: 'SLE',
    name: 'Sierra Leone',
    dialCode: '232',
    phoneLength: 8,
    operators: [
      { label: 'Orange Money', provider: 'ORANGE_SLE', logo: 'orange-money.png' },
    ],
  },
  {
    code: 'ETH',
    name: 'Éthiopie',
    dialCode: '251',
    phoneLength: 9,
    operators: [
      { label: 'Safaricom M-Pesa', provider: 'MPESA_ETH', logo: 'safaricom-mpesa.png' },
    ],
  },
];

// Opérateurs mobile money couverts par PayDunya — miroir exact de
// backend/internal/payment/paydunya_operators.go PayDunyaOperators. Codes
// distincts de PAYOUT_COUNTRIES (PawaPay) : ex "ORANGE_SN" ici vs
// "ORANGE_SEN" côté PawaPay — ne jamais les confondre, PayDunya n'accepte
// que ses propres codes.
export const PAYDUNYA_COUNTRIES: PayoutCountry[] = [
  {
    code: 'SEN',
    name: 'Sénégal',
    dialCode: '221',
    phoneLength: 9,
    operators: [
      { label: 'Orange Money', provider: 'ORANGE_SN', logo: 'orange-money.png', pawaPayCode: 'ORANGE_SEN' },
      { label: 'Wave', provider: 'WAVE_SN', logo: 'wave.png', pawaPayCode: 'WAVE_SEN' },
      { label: 'Free Money', provider: 'FREE_SN', badgeColor: 'bg-white border border-green-900/15', badgeText: 'text-green-950', pawaPayCode: 'FREE_SEN' },
      { label: 'Expresso', provider: 'EXPRESSO_SN', badgeColor: 'bg-red-100', badgeText: 'text-red-900' },
      { label: 'Djamo', provider: 'DJAMO_SN', badgeColor: 'bg-purple-100', badgeText: 'text-purple-900' },
    ],
  },
  {
    code: 'BEN',
    name: 'Bénin',
    dialCode: '229',
    phoneLength: 10,
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_BJ', logo: 'mtn-momo.png', pawaPayCode: 'MTN_MOMO_BEN' },
      { label: 'Moov Money', provider: 'MOOV_BJ', logo: 'moov-money.png', pawaPayCode: 'MOOV_BEN' },
      { label: 'Celtiis Cash', provider: 'CELTIIS_BJ', badgeColor: 'bg-blue-100', badgeText: 'text-blue-900' },
    ],
  },
  {
    code: 'CIV',
    name: "Côte d'Ivoire",
    dialCode: '225',
    phoneLength: 10,
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_CI', logo: 'mtn-momo.png', pawaPayCode: 'MTN_MOMO_CIV' },
      { label: 'Moov Money', provider: 'MOOV_CI', logo: 'moov-money.png' },
      { label: 'Wave', provider: 'WAVE_CI', logo: 'wave.png', pawaPayCode: 'WAVE_CIV' },
      { label: 'Djamo', provider: 'DJAMO_CI', badgeColor: 'bg-purple-100', badgeText: 'text-purple-900' },
    ],
  },
  {
    code: 'TGO',
    name: 'Togo',
    dialCode: '228',
    phoneLength: 8,
    operators: [
      { label: 'T-Money', provider: 'TMONEY_TG', badgeColor: 'bg-yellow-100', badgeText: 'text-yellow-900' },
      { label: 'Moov Money', provider: 'MOOV_TG', logo: 'moov-money.png' },
    ],
  },
  {
    code: 'MLI',
    name: 'Mali',
    dialCode: '223',
    phoneLength: 8,
    operators: [
      // Moov Mali retiré : pas activé sur notre compte marchand PayDunya
      // (relevé 2026-09-25), voir backend/internal/payment/paydunya_operators.go.
      { label: 'Orange Money', provider: 'ORANGE_ML', logo: 'orange-money.png' },
    ],
  },
  {
    code: 'BFA',
    name: 'Burkina Faso',
    dialCode: '226',
    phoneLength: 8,
    operators: [
      { label: 'Moov Money', provider: 'MOOV_BF', logo: 'moov-money.png', pawaPayCode: 'MOOV_BFA' },
    ],
  },
  {
    code: 'CMR',
    name: 'Cameroun',
    dialCode: '237',
    phoneLength: 9,
    operators: [
      { label: 'MTN MoMo', provider: 'MTN_CM', logo: 'mtn-momo.png', pawaPayCode: 'MTN_MOMO_CMR' },
    ],
  },
];

// LogicalOperator — opérateur mobile money PHYSIQUE, indépendant du
// prestataire qui le traite (fusion de PAYOUT_COUNTRIES et
// PAYDUNYA_COUNTRIES) — miroir de backend payment.LogicalOperator. Le code
// est stable (code PawaPay quand il existe, sinon code PayDunya) et sert de
// clé pour operator_providers (voir api.getCheckoutConfig) et pour l'envoi
// au backend (CreateOrderInput.operator).
export interface LogicalOperator extends PayoutOperator {
  country: string;
  dialCode: string;
  pawaPayProvider?: string; // code à envoyer si le prestataire résolu est pawapay
  payDunyaProvider?: string; // code à envoyer si le prestataire résolu est paydunya
}

function buildLogicalOperators(): LogicalOperator[] {
  const out: LogicalOperator[] = [];
  for (const country of PAYOUT_COUNTRIES) {
    for (const op of country.operators) {
      out.push({
        ...op,
        provider: op.provider,
        country: country.code,
        dialCode: country.dialCode,
        pawaPayProvider: op.provider,
      });
    }
  }
  for (const country of PAYDUNYA_COUNTRIES) {
    for (const op of country.operators) {
      if (op.pawaPayCode) {
        const existing = out.find((o) => o.pawaPayProvider === op.pawaPayCode);
        if (existing) {
          existing.payDunyaProvider = op.provider;
          continue;
        }
      }
      out.push({
        ...op,
        country: country.code,
        dialCode: country.dialCode,
        payDunyaProvider: op.provider,
      });
    }
  }
  return out;
}

export const LOGICAL_OPERATORS: LogicalOperator[] = buildLogicalOperators();

// Retrouve un opérateur de versement à partir de son provider (ex: "WAVE_SEN"),
// tous pays confondus — utile pour afficher l'historique des versements où
// seul le provider est stocké (pas le pays séparément).
export function findPayoutOperator(provider: string | null | undefined): (PayoutOperator & { countryName: string; dialCode: string }) | undefined {
  if (!provider) return undefined;
  for (const country of PAYOUT_COUNTRIES) {
    const op = country.operators.find((o) => o.provider === provider);
    if (op) return { ...op, countryName: country.name, dialCode: country.dialCode };
  }
  return undefined;
}

// Masque un numéro au format "+221 77 *** 89" : garde les 2 premiers et 2
// derniers chiffres locaux, masque le reste. `msisdn` est le numéro complet
// stocké (indicatif + numéro local, sans le "+").
export function maskPhone(msisdn: string | null | undefined, dialCode: string | undefined): string {
  if (!msisdn) return '';
  const local = dialCode && msisdn.startsWith(dialCode) ? msisdn.slice(dialCode.length) : msisdn;
  if (local.length < 4) return `+${dialCode ? `${dialCode} ` : ''}${local}`;
  const first = local.slice(0, 2);
  const last = local.slice(-2);
  return `+${dialCode || ''} ${first} *** ${last}`;
}

export function isLoggedIn(): boolean {
  if (typeof window === 'undefined') return false;
  return Boolean(localStorage.getItem('access_token'));
}
