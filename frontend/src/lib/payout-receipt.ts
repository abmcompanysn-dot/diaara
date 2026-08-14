import { formatPrice, PAYOUT_STATUS_LABELS } from '@/lib/constants';
import { findPayoutOperator, maskPhone } from '@/lib/operators';
import { escapeHtml } from '@/lib/html-escape';

export interface ReceiptPayout {
  id: string;
  amount_cfa: number;
  status: string;
  phone_number: string;
  operator: string;
  requested_at: string;
  paid_at?: string | null;
  failure_reason?: string | null;
}

export function payoutReference(payout: ReceiptPayout): string {
  return `PAY-${payout.id.replace(/-/g, '').slice(0, 8).toUpperCase()}`;
}

// Ouvre une fenêtre imprimable (l'utilisateur peut "Enregistrer en PDF" depuis
// la boîte de dialogue d'impression du navigateur) — aucune génération PDF
// côté serveur nécessaire.
export function openPayoutReceipt(payout: ReceiptPayout) {
  const op = findPayoutOperator(payout.operator);
  const win = window.open('', '_blank', 'width=480,height=700');
  if (!win) return;

  const reference = payoutReference(payout);
  const requestedDate = new Date(payout.requested_at).toLocaleString('fr-FR');
  const paidDate = payout.paid_at ? new Date(payout.paid_at).toLocaleString('fr-FR') : null;
  const statusLabel = PAYOUT_STATUS_LABELS[payout.status] || payout.status;
  const destination = op ? `${op.label} (${op.countryName})` : payout.operator;
  const maskedPhone = maskPhone(payout.phone_number, op?.dialCode) || payout.phone_number;

  win.document.write(`<!doctype html>
<html lang="fr">
<head>
<meta charset="utf-8" />
<title>Reçu ${reference}</title>
<style>
  * { box-sizing: border-box; }
  body { font-family: -apple-system, Segoe UI, Arial, sans-serif; color: #0a1f16; padding: 32px; max-width: 420px; margin: 0 auto; }
  h1 { font-size: 18px; margin: 0 0 2px; }
  .sub { color: #4b6358; font-size: 12px; margin-bottom: 24px; }
  .row { display: flex; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid #e3ece7; font-size: 13px; }
  .row span:first-child { color: #4b6358; }
  .row span:last-child { font-weight: 600; text-align: right; }
  .total { font-size: 16px; margin-top: 8px; }
  .status { display: inline-block; padding: 3px 10px; border-radius: 999px; background: #dcfce7; color: #15803d; font-size: 12px; font-weight: 600; }
  .footer { margin-top: 28px; font-size: 11px; color: #7c8f85; text-align: center; }
  @media print { body { padding: 0; } }
</style>
</head>
<body>
  <h1>DIARRA — Reçu de versement</h1>
  <p class="sub">Référence ${reference}</p>

  <div class="row"><span>Statut</span><span><span class="status">${escapeHtml(statusLabel)}</span></span></div>
  <div class="row"><span>Date de la demande</span><span>${escapeHtml(requestedDate)}</span></div>
  ${paidDate ? `<div class="row"><span>Date du versement</span><span>${escapeHtml(paidDate)}</span></div>` : ''}
  <div class="row"><span>Destination</span><span>${escapeHtml(destination)}</span></div>
  <div class="row"><span>Numéro</span><span>${escapeHtml(maskedPhone)}</span></div>
  <div class="row"><span>Montant brut</span><span>${escapeHtml(formatPrice(payout.amount_cfa))}</span></div>
  <div class="row"><span>Frais</span><span>0 FCFA</span></div>
  <div class="row total"><span>Montant net reçu</span><span>${escapeHtml(formatPrice(payout.amount_cfa))}</span></div>
  ${payout.failure_reason ? `<div class="row"><span>Motif</span><span>${escapeHtml(payout.failure_reason)}</span></div>` : ''}

  <p class="footer">Document généré automatiquement — DIARRA</p>
</body>
</html>`);
  win.document.close();
  win.focus();
  win.print();
}
