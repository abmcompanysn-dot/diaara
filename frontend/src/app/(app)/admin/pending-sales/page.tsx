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
import { SearchIcon } from '@/components/icons';

interface PendingSale {
  id: string;
  product_title: string;
  buyer_name: string;
  buyer_email: string;
  buyer_phone?: string | null;
  vendor_email: string;
  country?: string | null;
  amount_cfa: number;
  status: string;
  payment_provider: string;
  created_at: string;
  reminded_at?: string | null;
  reminder_count?: number;
}

export default function AdminPendingSalesPage() {
  const [sales, setSales] = useState<PendingSale[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [msg, setMsg] = useState('');
  const [busyId, setBusyId] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [confirmTarget, setConfirmTarget] = useState<PendingSale | null>(null);

  useEffect(() => {
    load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const r = await api.getPendingSales();
      setSales(r.sales);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleRemind(sale: PendingSale) {
    setBusyId(sale.id);
    setMsg('');
    setError('');
    try {
      await api.remindSale(sale.id);
      setSales((prev) =>
        prev.map((s) =>
          s.id === sale.id
            ? { ...s, reminded_at: new Date().toISOString(), reminder_count: (s.reminder_count || 0) + 1 }
            : s
        )
      );
      setMsg(`Relance envoyée à ${sale.buyer_email}.`);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  async function handleMarkPaid(sale: PendingSale) {
    setConfirmTarget(null);
    setBusyId(sale.id);
    setMsg('');
    setError('');
    try {
      await api.markSalePaid(sale.id);
      setMsg(`Vente confirmée : ${sale.buyer_email} va recevoir son fichier, le vendeur est crédité.`);
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  // Traductions courtes des statuts renvoyés par PawaPay.
  const PROVIDER_STATUS_LABELS: Record<string, string> = {
    COMPLETED: 'payé (COMPLETED)',
    FAILED: 'échoué (FAILED)',
    PROCESSING: 'en cours (PROCESSING)',
    ACCEPTED: 'accepté, en attente (ACCEPTED)',
    IN_RECONCILIATION: 'en réconciliation',
    NOT_FOUND: 'introuvable — acheteur jamais allé au bout',
  };

  async function handleCheckProvider(sale: PendingSale) {
    setBusyId(sale.id);
    setMsg('');
    setError('');
    try {
      const r = await api.checkSaleProvider(sale.id);
      const label = PROVIDER_STATUS_LABELS[r.provider_status] || r.provider_status;
      if (r.provider_status === 'COMPLETED') {
        setMsg(
          `PawaPay confirme le paiement (${label}). Vente confirmée : ${sale.buyer_email} reçoit son fichier, le vendeur est crédité.`
        );
      } else if (r.provider_status === 'FAILED') {
        setMsg(`PawaPay indique un paiement ${label}. La vente est passée en « échec ».`);
      } else {
        setMsg(`Statut PawaPay pour cette commande : ${label}. Aucun changement.`);
      }
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  const filtered = useMemo(() => {
    let list = sales;
    if (statusFilter !== 'all') list = list.filter((s) => s.status === statusFilter);
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      list = list.filter(
        (s) =>
          s.product_title.toLowerCase().includes(q) ||
          s.buyer_name.toLowerCase().includes(q) ||
          s.buyer_email.toLowerCase().includes(q) ||
          (s.buyer_phone || '').toLowerCase().includes(q)
      );
    }
    return list;
  }, [sales, search, statusFilter]);

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Paiements en attente & échoués" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Paiements en attente & échoués"
        description={`${sales.length} commande(s) non aboutie(s) — contactez ou relancez les acheteurs`}
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
        {msg && (
          <div className="mb-4 p-3 bg-green-900/5 text-green-900 rounded text-sm" role="status">
            {msg}
          </div>
        )}

        {sales.length > 0 && (
          <div className="flex flex-wrap gap-3 mb-5">
            <div className="relative flex-1 min-w-55">
              <SearchIcon size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-green-900/40" />
              <Input
                placeholder="Rechercher par produit, acheteur, email ou téléphone..."
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
                <SelectItem value="all">Tous</SelectItem>
                <SelectItem value="pending">En attente</SelectItem>
                <SelectItem value="failed">Échec</SelectItem>
              </SelectContent>
            </Select>
          </div>
        )}

        {sales.length === 0 ? (
          <EmptyState
            title="Rien à relancer"
            description="Toutes les commandes ont abouti. Les paiements non finalisés apparaîtront ici."
          />
        ) : filtered.length === 0 ? (
          <EmptyState title="Aucun résultat" description="Aucune commande ne correspond à cette recherche." />
        ) : (
          <div className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Date</TableHead>
                  <TableHead>Produit</TableHead>
                  <TableHead>Acheteur</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Téléphone</TableHead>
                  <TableHead>Montant</TableHead>
                  <TableHead>Statut</TableHead>
                  <TableHead>Relances</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.map((sale) => (
                  <TableRow key={sale.id}>
                    <TableCell className="text-sm whitespace-nowrap">
                      {new Date(sale.created_at).toLocaleString('fr-FR')}
                    </TableCell>
                    <TableCell className="max-w-[160px] truncate">{sale.product_title}</TableCell>
                    <TableCell className="whitespace-nowrap">{sale.buyer_name}</TableCell>
                    <TableCell className="text-sm text-green-900/70">{sale.buyer_email}</TableCell>
                    <TableCell className="text-sm text-green-900/70 whitespace-nowrap">
                      {sale.buyer_phone || '—'}
                    </TableCell>
                    <TableCell className="font-mono whitespace-nowrap">{formatPrice(sale.amount_cfa)}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className={SALE_STATUS_BADGE[sale.status]}>
                        {ORDER_STATUS_LABELS[sale.status] || sale.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-green-900/60">
                      {sale.reminder_count || 0}
                      {sale.reminded_at && (
                        <span className="block text-[11px] text-green-900/40">
                          {new Date(sale.reminded_at).toLocaleDateString('fr-FR')}
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {sale.status === 'pending' && (
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={busyId === sale.id}
                            onClick={() => handleRemind(sale)}
                          >
                            {busyId === sale.id ? '…' : 'Relancer'}
                          </Button>
                        )}
                        {sale.payment_provider !== 'kpay' && (
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={busyId === sale.id}
                            onClick={() => handleCheckProvider(sale)}
                            title="Interroge PawaPay sur le vrai statut du paiement, et confirme la vente si PawaPay répond « payé »."
                          >
                            {busyId === sale.id ? '…' : 'Vérifier chez PawaPay'}
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={busyId === sale.id}
                          onClick={() => setConfirmTarget(sale)}
                        >
                          Confirmer le paiement
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
        open={!!confirmTarget}
        title="Confirmer ce paiement à la main ?"
        description={
          confirmTarget
            ? `La commande de ${confirmTarget.buyer_name} (${formatPrice(
                confirmTarget.amount_cfa
              )}) passera en « payée ». L'acheteur recevra son fichier par email et le vendeur sera crédité. À n'utiliser que si vous avez la preuve que l'argent a bien été reçu.`
            : undefined
        }
        confirmLabel="Oui, marquer payée"
        cancelLabel="Annuler"
        onConfirm={() => confirmTarget && handleMarkPaid(confirmTarget)}
        onCancel={() => setConfirmTarget(null)}
      />
    </main>
  );
}
