'use client';

import { useEffect, useMemo, useState } from 'react';
import { api } from '@/lib/api';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { SegmentedTabs } from '@/components/segmented-tabs';
import { SearchIcon } from '@/components/icons';
import { formatPrice, SALE_STATUS_BADGE, ORDER_STATUS_LABELS } from '@/lib/constants';
import { CHECKOUT_COUNTRIES } from '@/lib/operators';
import { friendlyError } from '@/lib/error-messages';

const STATUS_FILTERS: { value: string; label: string }[] = [
  { value: 'all', label: 'Tous' },
  { value: 'paid', label: 'Payé' },
  { value: 'pending', label: 'En attente' },
  { value: 'delivered', label: 'Livré' },
  { value: 'failed', label: 'Échec' },
  { value: 'refund_pending', label: 'Remb. en cours' },
  { value: 'refunded', label: 'Remboursé' },
];

interface VendorSale {
  id: string;
  product_title: string;
  buyer_name: string;
  buyer_email: string;
  buyer_phone?: string | null;
  country?: string | null;
  amount_cfa: number;
  status: string;
  created_at: string;
  reminded_at?: string | null;
  reminder_count?: number;
}

function countryName(code?: string | null): string {
  if (!code) return '—';
  const c = CHECKOUT_COUNTRIES.find((c) => c.code === code);
  return c ? `${c.flag} ${c.name}` : code;
}

export default function VendorSalesPage() {
  const [sales, setSales] = useState<VendorSale[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [remindingId, setRemindingId] = useState<string | null>(null);
  const [remindMsg, setRemindMsg] = useState('');

  useEffect(() => {
    api
      .getVendorSales()
      .then((r) => setSales(r.sales))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, []);

  async function handleRemind(saleId: string) {
    setRemindingId(saleId);
    setRemindMsg('');
    try {
      await api.remindVendorSale(saleId);
      setSales((prev) =>
        prev.map((s) =>
          s.id === saleId
            ? { ...s, reminded_at: new Date().toISOString(), reminder_count: (s.reminder_count || 0) + 1 }
            : s
        )
      );
      setRemindMsg('Relance envoyée par email à l’acheteur.');
    } catch (err: any) {
      setRemindMsg(friendlyError(err));
    } finally {
      setRemindingId(null);
    }
  }

  const filtered = useMemo(() => {
    let list = sales;
    if (statusFilter !== 'all') {
      list = list.filter((s) => s.status === statusFilter);
    }
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      list = list.filter(
        (s) =>
          s.product_title.toLowerCase().includes(q) ||
          s.buyer_name.toLowerCase().includes(q) ||
          s.buyer_email.toLowerCase().includes(q)
      );
    }
    return list;
  }, [sales, search, statusFilter]);

  if (loading)
    return (
      <main>
        <PageHeader back="/vendor" eyebrow="// espace vendeur" title="Mes clients" description="Ventes et coordonnées des acheteurs" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        back="/vendor"
        eyebrow="// espace vendeur"
        title="Mes clients"
        description={`${sales.length} vente(s)`}
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-6">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {sales.length > 0 && (
          <div className="space-y-3 mb-5">
            <div className="relative">
              <SearchIcon size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-green-900/40" />
              <Input
                placeholder="Rechercher par produit ou client..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9 bg-white"
              />
            </div>
            <SegmentedTabs options={STATUS_FILTERS} value={statusFilter} onChange={setStatusFilter} />
          </div>
        )}

        {remindMsg && (
          <div className="mb-4 p-3 bg-green-900/5 text-green-900 rounded text-sm" role="status">
            {remindMsg}
          </div>
        )}

        {sales.length === 0 ? (
          <EmptyState title="Aucune vente pour le moment" description="Les ventes de vos produits apparaîtront ici avec les coordonnées de vos clients." />
        ) : filtered.length === 0 ? (
          <EmptyState title="Aucun résultat" description="Aucune vente ne correspond à cette recherche." />
        ) : (
          <>
            {/* Vue mobile : cartes empilées, pas de scroll horizontal */}
            <div className="sm:hidden space-y-2">
              {filtered.map((sale) => (
                <div key={sale.id} className="bg-white rounded-xl border border-green-900/10 shadow-card p-3.5">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-green-950 truncate">{sale.product_title}</p>
                      <p className="text-xs text-green-900/60 truncate">{sale.buyer_name} · {sale.buyer_email}</p>
                      {sale.buyer_phone && (
                        <p className="text-xs text-green-900/60 truncate">📞 {sale.buyer_phone}</p>
                      )}
                    </div>
                    <span className="font-mono text-sm font-bold text-green-950 shrink-0">
                      {formatPrice(sale.amount_cfa)}
                    </span>
                  </div>
                  <div className="flex items-center justify-between gap-2 mt-3 pt-3 border-t border-green-900/5">
                    <span className="text-xs text-green-900/50">
                      {countryName(sale.country)} · {new Date(sale.created_at).toLocaleDateString('fr-FR')}
                    </span>
                    <Badge variant="outline" className={SALE_STATUS_BADGE[sale.status]}>
                      {ORDER_STATUS_LABELS[sale.status] || sale.status}
                    </Badge>
                  </div>
                  {sale.status === 'pending' && (
                    <button
                      type="button"
                      onClick={() => handleRemind(sale.id)}
                      disabled={remindingId === sale.id}
                      className="mt-3 w-full rounded-lg border border-green-900/20 bg-green-900/5 px-3 py-2 text-xs font-medium text-green-900 disabled:opacity-50"
                    >
                      {remindingId === sale.id
                        ? 'Envoi…'
                        : sale.reminder_count
                          ? `Relancer à nouveau (${sale.reminder_count} déjà envoyée${sale.reminder_count > 1 ? 's' : ''})`
                          : 'Relancer par email'}
                    </button>
                  )}
                </div>
              ))}
            </div>

            {/* Vue desktop : tableau */}
            <div className="hidden sm:block rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Date</TableHead>
                    <TableHead>Produit</TableHead>
                    <TableHead>Client</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Téléphone</TableHead>
                    <TableHead>Pays</TableHead>
                    <TableHead>Montant</TableHead>
                    <TableHead>Statut</TableHead>
                    <TableHead>Action</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filtered.map((sale) => (
                    <TableRow key={sale.id}>
                      <TableCell className="text-sm whitespace-nowrap">
                        {new Date(sale.created_at).toLocaleString('fr-FR')}
                      </TableCell>
                      <TableCell className="max-w-[160px] truncate">{sale.product_title}</TableCell>
                      <TableCell>{sale.buyer_name}</TableCell>
                      <TableCell className="text-sm text-green-900/70">{sale.buyer_email}</TableCell>
                      <TableCell className="text-sm text-green-900/70 whitespace-nowrap">{sale.buyer_phone || '—'}</TableCell>
                      <TableCell className="whitespace-nowrap">{countryName(sale.country)}</TableCell>
                      <TableCell className="font-mono">{formatPrice(sale.amount_cfa)}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className={SALE_STATUS_BADGE[sale.status]}>
                          {ORDER_STATUS_LABELS[sale.status] || sale.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {sale.status === 'pending' ? (
                          <button
                            type="button"
                            onClick={() => handleRemind(sale.id)}
                            disabled={remindingId === sale.id}
                            className="rounded-lg border border-green-900/20 bg-green-900/5 px-2.5 py-1.5 text-xs font-medium text-green-900 disabled:opacity-50 whitespace-nowrap"
                          >
                            {remindingId === sale.id
                              ? 'Envoi…'
                              : sale.reminder_count
                                ? `Relancer (${sale.reminder_count})`
                                : 'Relancer'}
                          </button>
                        ) : (
                          <span className="text-xs text-green-900/30">—</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </>
        )}
      </section>
    </main>
  );
}
