'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { formatPrice, SALE_STATUS_BADGE, ORDER_STATUS_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';
import { openSaleReceipt } from '@/lib/sale-receipt';
import { FileIcon } from '@/components/icons';

interface Sale {
  id: string;
  product_id: string;
  buyer_id: string;
  buyer_name?: string;
  referral_link_id?: string;
  amount_cfa: number;
  platform_fee_cfa: number;
  closer_commission_cfa: number;
  vendor_amount_cfa: number;
  status: string;
  payment_reference: string;
  created_at: string;
}

export default function AdminSalesPage() {
  const [sales, setSales] = useState<Sale[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refundingId, setRefundingId] = useState<string | null>(null);

  useEffect(() => {
    loadSales();
  }, []);

  const loadSales = async () => {
    setLoading(true);
    try {
      const result = await api.getSales();
      setSales(result.sales);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleRefund = async (sale: Sale) => {
    if (
      !confirm(
        `Rembourser ${formatPrice(sale.amount_cfa)} à l'acheteur ? Cette action envoie réellement l'argent via PawaPay et ne peut pas être annulée.`
      )
    ) {
      return;
    }
    setRefundingId(sale.id);
    setError('');
    try {
      await api.refundSale(sale.id);
      await loadSales();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setRefundingId(null);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Ventes" description="Historique des transactions" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Ventes"
        description={`${sales.length} vente(s) enregistrée(s)`}
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            ← Dashboard
          </Button>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {sales.length === 0 ? (
          <EmptyState
            title="Aucune vente pour le moment"
            description="Les ventes apparaîtront ici dès qu'un acheteur finalisera un paiement."
          />
        ) : (
          <div className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Date</TableHead>
                  <TableHead>Montant</TableHead>
                  <TableHead>Frais plateforme</TableHead>
                  <TableHead>Commission affilié</TableHead>
                  <TableHead>Vendeur</TableHead>
                  <TableHead>Statut</TableHead>
                  <TableHead></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sales.map((sale) => (
                  <TableRow key={sale.id}>
                    <TableCell className="text-sm">
                      {new Date(sale.created_at).toLocaleString('fr-FR')}
                    </TableCell>
                    <TableCell className="font-mono">{formatPrice(sale.amount_cfa)}</TableCell>
                    <TableCell className="font-mono">{formatPrice(sale.platform_fee_cfa)}</TableCell>
                    <TableCell>
                      {sale.referral_link_id ? (
                        <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100">
                          {formatPrice(sale.closer_commission_cfa)}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground/40">—</span>
                      )}
                    </TableCell>
                    <TableCell className="font-mono">{formatPrice(sale.vendor_amount_cfa)}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className={SALE_STATUS_BADGE[sale.status]}>
                        {ORDER_STATUS_LABELS[sale.status] || sale.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {sale.status === 'paid' && (
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={refundingId === sale.id}
                            onClick={() => handleRefund(sale)}
                          >
                            {refundingId === sale.id ? 'Remboursement…' : 'Rembourser'}
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => openSaleReceipt(sale)}
                          className="gap-1.5"
                        >
                          <FileIcon size={14} />
                          Reçu
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </section>
    </main>
  );
}
