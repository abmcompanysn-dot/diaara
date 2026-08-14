'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { formatPrice, SALE_STATUS_BADGE, ORDER_STATUS_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';
import { openSaleReceipt } from '@/lib/sale-receipt';
import { FileIcon, SearchIcon } from '@/components/icons';

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
  const [refundTarget, setRefundTarget] = useState<Sale | null>(null);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');

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
    setRefundTarget(null);
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

  const filtered = useMemo(() => {
    let list = sales;
    if (statusFilter !== 'all') {
      list = list.filter((s) => s.status === statusFilter);
    }
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      list = list.filter(
        (s) => (s.buyer_name || '').toLowerCase().includes(q) || s.payment_reference.toLowerCase().includes(q)
      );
    }
    return list;
  }, [sales, search, statusFilter]);

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

        {sales.length > 0 && (
          <div className="flex flex-wrap gap-3 mb-5">
            <div className="relative flex-1 min-w-55">
              <SearchIcon size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-green-900/40" />
              <Input
                placeholder="Rechercher par acheteur ou référence..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9 bg-white"
              />
            </div>
            <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v || 'all')}>
              <SelectTrigger className="w-44 bg-white">
                <SelectValue placeholder="Statut" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Tous les statuts</SelectItem>
                <SelectItem value="pending">En attente</SelectItem>
                <SelectItem value="paid">Payé</SelectItem>
                <SelectItem value="delivered">Livré</SelectItem>
                <SelectItem value="failed">Échec</SelectItem>
                <SelectItem value="refund_pending">Remboursement en cours</SelectItem>
                <SelectItem value="refunded">Remboursé</SelectItem>
              </SelectContent>
            </Select>
          </div>
        )}

        {sales.length === 0 ? (
          <EmptyState
            title="Aucune vente pour le moment"
            description="Les ventes apparaîtront ici dès qu'un acheteur finalisera un paiement."
          />
        ) : filtered.length === 0 ? (
          <EmptyState title="Aucun résultat" description="Aucune vente ne correspond à cette recherche." />
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
                {filtered.map((sale) => (
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
                            onClick={() => setRefundTarget(sale)}
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

      <ConfirmDialog
        open={!!refundTarget}
        title="Rembourser cette vente ?"
        description={
          refundTarget
            ? `${formatPrice(refundTarget.amount_cfa)} seront renvoyés à l'acheteur via PawaPay. Cette action envoie réellement l'argent et ne peut pas être annulée.`
            : undefined
        }
        confirmLabel="Rembourser"
        cancelLabel="Annuler"
        danger
        onConfirm={() => refundTarget && handleRefund(refundTarget)}
        onCancel={() => setRefundTarget(null)}
      />
    </main>
  );
}
