import { formatPrice } from '@/lib/constants';
import { escapeHtml } from '@/lib/html-escape';

export interface ReceiptSale {
  id: string;
  amount_cfa: number;
  status: string;
  buyer_name?: string;
  payment_reference: string;
  created_at: string;
}

export interface ReceiptProduct {
  title: string;
}

export function saleReference(sale: ReceiptSale): string {
  return `CMD-${sale.id.replace(/-/g, '').slice(0, 8).toUpperCase()}`;
}

// Ouvre une fenêtre imprimable (l'utilisateur peut "Enregistrer en PDF" depuis
// la boîte de dialogue d'impression du navigateur) — même approche que
// openPayoutReceipt, aucune génération PDF côté serveur nécessaire.
export function openSaleReceipt(sale: ReceiptSale, product?: ReceiptProduct | null, shopName?: string | null) {
  const win = window.open('', '_blank', 'width=480,height=700');
  if (!win) return;

  const reference = saleReference(sale);
  const date = new Date(sale.created_at).toLocaleString('fr-FR');

  win.document.write(`<!doctype html>
<html lang="fr">
<head>
<meta charset="utf-8" />
<title>Reçu ${reference}</title>
<style>
  * { box-sizing: border-box; }
  body { font-family: -apple-system, Segoe UI, Arial, sans-serif; color: #0a1f16; padding: 32px; max-width: 420px; margin: 0 auto; }
  .logo { height: 28px; margin-bottom: 16px; }
  h1 { font-size: 18px; margin: 0 0 2px; }
  .sub { color: #4b6358; font-size: 12px; margin-bottom: 24px; }
  .row { display: flex; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid #e3ece7; font-size: 13px; }
  .row span:first-child { color: #4b6358; }
  .row span:last-child { font-weight: 600; text-align: right; }
  .total { font-size: 16px; margin-top: 8px; }
  .footer { margin-top: 28px; font-size: 11px; color: #7c8f85; text-align: center; }
  @media print { body { padding: 0; } }
</style>
</head>
<body>
  <img class="logo" src="${window.location.origin}/brand/diarra-logo-dark-text.png" alt="DIARRA" />
  <h1>Reçu d'achat</h1>
  <p class="sub">Référence ${reference}</p>

  <div class="row"><span>Client</span><span>${escapeHtml(sale.buyer_name) || '—'}</span></div>
  <div class="row"><span>Produit</span><span>${escapeHtml(product?.title) || '—'}</span></div>
  ${shopName ? `<div class="row"><span>Boutique</span><span>${escapeHtml(shopName)}</span></div>` : ''}
  <div class="row"><span>Date</span><span>${escapeHtml(date)}</span></div>
  <div class="row"><span>Référence paiement</span><span>${escapeHtml(sale.payment_reference.slice(0, 12))}</span></div>
  <div class="row total"><span>Montant payé</span><span>${escapeHtml(formatPrice(sale.amount_cfa))}</span></div>

  <p class="footer">Document généré automatiquement — DIARRA</p>
</body>
</html>`);
  win.document.close();
  win.focus();
  win.print();
}
