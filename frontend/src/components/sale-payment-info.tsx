// Affichage des détails de paiement d'une vente (numéro réellement utilisé,
// opérateur, raison d'échec — voir migration 047 côté backend), partagé par
// les pages ventes vendeur / admin / paiements en attente, pour pouvoir
// relancer un client (appel ou WhatsApp) en sachant ce qui a bloqué.
import { LOGICAL_OPERATORS, CHECKOUT_COUNTRIES } from '@/lib/operators';

export interface SalePaymentFields {
  payer_phone?: string | null;
  buyer_phone?: string | null;
  payment_operator?: string | null;
  payment_provider?: string;
  failure_reason?: string | null;
  status: string;
}

// Codes d'échec PawaPay (failureCode) + codes posés par DIARRA
// (not_validated, card_declined, payment_init_failed...).
const FAILURE_LABELS: Record<string, string> = {
  not_validated: "Le client n'a pas validé le paiement sur son téléphone",
  PAYMENT_NOT_APPROVED: "Le client n'a pas validé le paiement sur son téléphone",
  PAYER_NOT_FOUND: "Numéro sans compte mobile money chez cet opérateur",
  INVALID_PHONE_NUMBER: 'Numéro de téléphone invalide',
  INSUFFICIENT_BALANCE: 'Solde insuffisant',
  PAYER_LIMIT_REACHED: 'Plafond du compte mobile money atteint',
  WALLET_LIMIT_REACHED: 'Plafond du compte mobile money atteint',
  PAYMENT_IN_PROGRESS: 'Un autre paiement était déjà en cours sur ce numéro',
  MANUALLY_CANCELLED: 'Paiement annulé',
  cancelled: 'Paiement annulé par le client',
  card_declined: 'Carte refusée',
  UNSPECIFIED_FAILURE: "Échec côté opérateur (raison non précisée)",
  UNKNOWN_ERROR: "Échec côté opérateur (raison non précisée)",
  FAILED: "Échec côté opérateur (raison non précisée)",
  failed: "Échec côté opérateur (raison non précisée)",
};

export function failureLabel(reason?: string | null): string {
  if (!reason) return '';
  if (reason.startsWith('payment_init_failed')) {
    return "Le paiement n'a pas pu démarrer (numéro ou opérateur refusé)";
  }
  return FAILURE_LABELS[reason] || reason;
}

export function operatorLabel(sale: SalePaymentFields): string {
  if (sale.payment_provider === 'paypal') return 'Carte / PayPal';
  if (!sale.payment_operator) return '';
  const op = LOGICAL_OPERATORS.find(
    (o) => o.provider === sale.payment_operator || o.payDunyaProvider === sale.payment_operator
  );
  if (!op) return sale.payment_operator;
  const flag = CHECKOUT_COUNTRIES.find((c) => c.code === op.country)?.flag;
  return flag ? `${op.label} ${flag}` : op.label;
}

// Numéro à contacter : celui saisi au paiement en priorité (le plus fiable,
// c'est le compte mobile money), sinon celui du compte DIARRA.
export function contactPhone(sale: SalePaymentFields): string {
  return (sale.payer_phone || sale.buyer_phone || '').trim();
}

function digitsOnly(phone: string): string {
  return phone.replace(/\D/g, '');
}

// "2250546968556" -> "+225 05 46 96 85 56" (indicatif reconnu via les
// opérateurs connus), sinon affiché tel quel.
export function formatPhone(phone: string): string {
  const d = digitsOnly(phone);
  const dial = LOGICAL_OPERATORS.map((o) => o.dialCode).find((c) => d.startsWith(c) && d.length > c.length + 6);
  if (!dial) return phone;
  const local = d.slice(dial.length).replace(/(\d{2})(?=\d)/g, '$1 ');
  return `+${dial} ${local}`;
}

export function PhoneContact({ phone }: { phone: string }) {
  if (!phone) return <span className="text-green-900/30">—</span>;
  const d = digitsOnly(phone);
  return (
    <span className="inline-flex flex-wrap items-center gap-x-2 gap-y-0.5 whitespace-nowrap">
      <a href={`tel:+${d}`} className="text-green-950 hover:underline">
        {formatPhone(phone)}
      </a>
      <a
        href={`https://wa.me/${d}`}
        target="_blank"
        rel="noopener noreferrer"
        className="text-[11px] font-medium text-green-700 hover:underline"
      >
        WhatsApp
      </a>
    </span>
  );
}

// Opérateur utilisé + raison d'échec (si échec). showRaw (admin uniquement) :
// code/message brut du prestataire en infobulle — pas pour le vendeur, il
// peut contenir des détails techniques internes.
export function PaymentDetails({ sale, showRaw = false }: { sale: SalePaymentFields; showRaw?: boolean }) {
  const op = operatorLabel(sale);
  const reason = sale.status === 'failed' ? failureLabel(sale.failure_reason) : '';
  if (!op && !reason) return <span className="text-green-900/30">—</span>;
  return (
    <span className="block">
      {op && <span className="block text-sm text-green-950 whitespace-nowrap">{op}</span>}
      {reason && (
        <span className="block text-xs text-red-600" title={showRaw ? sale.failure_reason || undefined : undefined}>
          {reason}
        </span>
      )}
    </span>
  );
}
