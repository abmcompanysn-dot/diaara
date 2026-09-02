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
import { formatPrice, PAYOUT_STATUS_BADGE, PAYOUT_STATUS_LABELS } from '@/lib/constants';
import { findPayoutOperator, maskPhone } from '@/lib/operators';
import { friendlyError } from '@/lib/error-messages';
import { openPayoutReceipt } from '@/lib/payout-receipt';
import { SearchIcon, FileIcon } from '@/components/icons';

interface Payout {
  id: string;
  user_id: string;
  user_email: string;
  amount_cfa: number;
  fee_cfa?: number;
  status: string;
  provider: string;
  phone_number: string;
  operator: string;
  is_manual?: boolean;
  manual_note?: string | null;
  provider_reference?: string | null;
  failure_reason?: string | null;
  requested_at: string;
  paid_at?: string | null;
  // Moyen de versement enregistré par le vendeur (repli quand le versement
  // lui-même n'a pas d'opérateur/numéro — cas d'un versement manuel créé de
  // toutes pièces) — voir PayoutRepo.ListAllAdmin côté backend.
  vendor_payout_phone?: string | null;
  vendor_payout_operator?: string | null;
  vendor_payout_country?: string | null;
}

// paymentMethodOf — le moyen de paiement à afficher pour un versement : celui
// du versement lui-même s'il en a un, sinon celui enregistré par le vendeur
// (versement manuel créé sans numéro saisi), sinon "non renseigné".
function paymentMethodOf(p: Payout): { label: string; phone: string } | null {
  const operator = p.operator || p.vendor_payout_operator || '';
  const phone = p.phone_number || p.vendor_payout_phone || '';
  if (!operator && !phone) return null;
  const op = findPayoutOperator(operator);
  return {
    label: op ? `${op.label} (${op.countryName})` : operator || '—',
    phone: maskPhone(phone, op?.dialCode) || phone || '—',
  };
}

export default function AdminPayoutsPage() {
  const [payouts, setPayouts] = useState<Payout[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [msg, setMsg] = useState('');
  const [busyId, setBusyId] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [retryTarget, setRetryTarget] = useState<Payout | null>(null);
  const [autoTarget, setAutoTarget] = useState<Payout | null>(null);

  // Règlement manuel d'un versement existant
  const [settleTarget, setSettleTarget] = useState<Payout | null>(null);
  const [settleNote, setSettleNote] = useState('');
  const [settleFee, setSettleFee] = useState('0');

  // Création d'un versement manuel de toutes pièces
  const [showCreate, setShowCreate] = useState(false);
  const [newUserId, setNewUserId] = useState('');
  const [newAmount, setNewAmount] = useState('');
  const [newFee, setNewFee] = useState('0');
  const [newPhone, setNewPhone] = useState('');
  const [newNote, setNewNote] = useState('');

  useEffect(() => {
    load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const r = await api.getAdminPayouts();
      setPayouts(r.payouts);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleRetry(p: Payout) {
    setRetryTarget(null);
    setBusyId(p.id);
    setMsg('');
    setError('');
    try {
      await api.retryPayout(p.id);
      setMsg('Relance du versement envoyée au prestataire.');
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  async function handleSettleAuto(p: Payout) {
    setAutoTarget(null);
    setBusyId(p.id);
    setMsg('');
    setError('');
    try {
      await api.settlePayoutAuto(p.id);
      setMsg(`Versement de ${p.user_email} envoyé à ${p.provider}. En cours de traitement.`);
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  async function handleCheckProvider(p: Payout) {
    setBusyId(p.id);
    setMsg('');
    setError('');
    try {
      const r = await api.checkPayoutProvider(p.id);
      if (r.payout_status === 'paid') {
        setMsg(`${p.provider} confirme : versement payé. Le vendeur est notifié.`);
      } else if (r.payout_status === 'failed') {
        setMsg(`${p.provider} indique un échec (${r.provider_status}). Vous pouvez relancer ou régler à la main.`);
      } else {
        setMsg(`Statut ${p.provider} : ${r.provider_status}. Aucun changement.`);
      }
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  async function handleSettle() {
    if (!settleTarget) return;
    const fee = Number(settleFee) || 0;
    setBusyId(settleTarget.id);
    setMsg('');
    setError('');
    try {
      await api.settlePayoutManual(settleTarget.id, { note: settleNote.trim(), fee_cfa: fee });
      setMsg(`Versement marqué "payé à la main" pour ${settleTarget.user_email}.`);
      setSettleTarget(null);
      setSettleNote('');
      setSettleFee('0');
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setMsg('');
    setError('');
    const amount = Number(newAmount) || 0;
    const fee = Number(newFee) || 0;
    if (!newUserId.trim() || amount <= 0) {
      setError('Identifiant vendeur et montant (> 0) obligatoires.');
      return;
    }
    setBusyId('create');
    try {
      await api.createManualPayout({
        user_id: newUserId.trim(),
        amount,
        fee_cfa: fee,
        phone: newPhone.trim(),
        note: newNote.trim(),
      });
      setMsg('Versement manuel enregistré.');
      setShowCreate(false);
      setNewUserId('');
      setNewAmount('');
      setNewFee('0');
      setNewPhone('');
      setNewNote('');
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  const filtered = useMemo(() => {
    let list = payouts;
    if (statusFilter !== 'all') list = list.filter((p) => p.status === statusFilter);
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      list = list.filter(
        (p) => p.user_email.toLowerCase().includes(q) || (p.phone_number || '').toLowerCase().includes(q)
      );
    }
    return list;
  }, [payouts, search, statusFilter]);

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Versements" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Versements"
        description={`${payouts.length} demande(s) — chaque demande vendeur est à régler ici, automatiquement (PawaPay/KPay) ou à la main`}
        actions={
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => setShowCreate((v) => !v)}>
              {showCreate ? 'Fermer' : '+ Versement manuel'}
            </Button>
            <Button variant="outline" size="sm" render={<Link href="/admin" />}>
              ← Dashboard
            </Button>
          </div>
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

        {showCreate && (
          <form
            onSubmit={handleCreate}
            className="mb-6 p-5 rounded-xl border border-green-900/10 bg-white shadow-card space-y-3"
          >
            <h2 className="font-display font-bold text-green-950">Enregistrer un versement fait à la main</h2>
            <p className="text-xs text-green-900/60">
              Pour tracer un paiement déjà envoyé au vendeur hors PawaPay/KPay (Wave perso, espèces, virement).
              L'identifiant vendeur se copie depuis la page « Utilisateurs ».
            </p>
            <div className="grid gap-3 sm:grid-cols-2">
              <Input
                placeholder="Identifiant vendeur (UUID)"
                value={newUserId}
                onChange={(e) => setNewUserId(e.target.value)}
              />
              <Input
                placeholder="Téléphone destinataire (optionnel)"
                value={newPhone}
                onChange={(e) => setNewPhone(e.target.value)}
              />
              <Input
                type="number"
                placeholder="Montant versé (FCFA)"
                value={newAmount}
                onChange={(e) => setNewAmount(e.target.value)}
              />
              <Input
                type="number"
                placeholder="Frais / taxe retenus (FCFA)"
                value={newFee}
                onChange={(e) => setNewFee(e.target.value)}
              />
            </div>
            <Input
              placeholder="Note / référence (ex. Wave TX ABC123, payé le 02/09)"
              value={newNote}
              onChange={(e) => setNewNote(e.target.value)}
            />
            <Button type="submit" disabled={busyId === 'create'}>
              {busyId === 'create' ? 'Enregistrement…' : 'Enregistrer le versement'}
            </Button>
          </form>
        )}

        {payouts.length > 0 && (
          <div className="flex flex-wrap gap-3 mb-5">
            <div className="relative flex-1 min-w-55">
              <SearchIcon size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-green-900/40" />
              <Input
                placeholder="Rechercher par email ou téléphone..."
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
                <SelectItem value="requested">En attente (à traiter)</SelectItem>
                <SelectItem value="processing">En traitement</SelectItem>
                <SelectItem value="paid">Payé</SelectItem>
                <SelectItem value="failed">Échec</SelectItem>
              </SelectContent>
            </Select>
          </div>
        )}

        {payouts.length === 0 ? (
          <EmptyState title="Aucun versement" description="Les demandes de versement des vendeurs apparaîtront ici." />
        ) : filtered.length === 0 ? (
          <EmptyState title="Aucun résultat" description="Aucun versement ne correspond à cette recherche." />
        ) : (
          <div className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Date</TableHead>
                  <TableHead>Vendeur</TableHead>
                  <TableHead>Moyen de paiement</TableHead>
                  <TableHead>Montant</TableHead>
                  <TableHead>Frais</TableHead>
                  <TableHead>Voie</TableHead>
                  <TableHead>Statut</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.map((p) => {
                  const method = paymentMethodOf(p);
                  return (
                  <TableRow key={p.id}>
                    <TableCell className="text-sm whitespace-nowrap">
                      {new Date(p.requested_at).toLocaleDateString('fr-FR')}
                    </TableCell>
                    <TableCell className="text-sm">
                      {p.user_email}
                      {p.manual_note && (
                        <span className="block text-[11px] text-green-900/40">{p.manual_note}</span>
                      )}
                    </TableCell>
                    <TableCell className="text-sm text-green-900/70 whitespace-nowrap">
                      {method ? (
                        <>
                          {method.label}
                          <span className="block text-[11px] text-green-900/40">{method.phone}</span>
                        </>
                      ) : (
                        <>
                          <span className="text-amber-700">— non renseigné</span>
                          <Link href="/admin/users" className="block text-[11px] text-primary underline">
                            Voir la fiche vendeur
                          </Link>
                        </>
                      )}
                    </TableCell>
                    <TableCell className="font-mono whitespace-nowrap">{formatPrice(p.amount_cfa)}</TableCell>
                    <TableCell className="font-mono text-sm text-green-900/60">
                      {p.fee_cfa ? formatPrice(p.fee_cfa) : '—'}
                    </TableCell>
                    <TableCell className="text-xs">
                      {p.is_manual || p.provider === 'manual' ? (
                        <Badge className="bg-slate-100 text-slate-700 hover:bg-slate-100">Manuel</Badge>
                      ) : (
                        <span className="text-green-900/60">{p.provider}</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge className={PAYOUT_STATUS_BADGE[p.status]}>
                        {PAYOUT_STATUS_LABELS[p.status] || p.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap items-center gap-2">
                        {p.status === 'requested' && p.provider !== 'manual' && p.provider !== 'off' && p.provider !== '' && (
                          <Button
                            variant="default"
                            size="sm"
                            disabled={busyId === p.id}
                            onClick={() => setAutoTarget(p)}
                          >
                            {busyId === p.id ? '…' : `Régler via ${p.provider}`}
                          </Button>
                        )}
                        {p.status === 'processing' && p.provider !== 'manual' && (
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={busyId === p.id}
                            onClick={() => handleCheckProvider(p)}
                            title={`Interroge ${p.provider} sur le vrai statut de ce versement.`}
                          >
                            {busyId === p.id ? '…' : `Vérifier chez ${p.provider}`}
                          </Button>
                        )}
                        {p.status === 'failed' && p.provider !== 'manual' && (
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={busyId === p.id}
                            onClick={() => setRetryTarget(p)}
                          >
                            Relancer
                          </Button>
                        )}
                        {p.status !== 'paid' && (
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={busyId === p.id}
                            onClick={() => {
                              setSettleTarget(p);
                              setSettleNote('');
                              setSettleFee('0');
                            }}
                          >
                            Marquer payé (manuel)
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          className="gap-1.5"
                          onClick={() => openPayoutReceipt(p)}
                        >
                          <FileIcon size={14} />
                          Reçu
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </section>

      <ConfirmDialog
        open={!!autoTarget}
        title="Régler ce versement automatiquement ?"
        description={
          autoTarget
            ? `${formatPrice(autoTarget.amount_cfa)} seront envoyés à ${autoTarget.user_email} via ${autoTarget.provider} (${autoTarget.operator} · ${autoTarget.phone_number}). Le versement passera « en traitement » puis « payé » à la confirmation du prestataire.`
            : undefined
        }
        confirmLabel="Envoyer le versement"
        cancelLabel="Annuler"
        onConfirm={() => autoTarget && handleSettleAuto(autoTarget)}
        onCancel={() => setAutoTarget(null)}
      />

      <ConfirmDialog
        open={!!retryTarget}
        title="Relancer ce versement ?"
        description={
          retryTarget
            ? `Une nouvelle tentative d'envoi de ${formatPrice(retryTarget.amount_cfa)} sera faite via ${retryTarget.provider}.`
            : undefined
        }
        confirmLabel="Relancer"
        cancelLabel="Annuler"
        onConfirm={() => retryTarget && handleRetry(retryTarget)}
        onCancel={() => setRetryTarget(null)}
      />

      {settleTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-lift space-y-4">
            <h2 className="font-display font-bold text-lg text-green-950">Marquer ce versement payé à la main</h2>
            <p className="text-sm text-green-900/70">
              {settleTarget.user_email} — {formatPrice(settleTarget.amount_cfa)}. Confirme que l'argent a bien été
              envoyé au vendeur par un autre moyen. Aucun appel PawaPay/KPay ne sera fait.
            </p>
            {(() => {
              const method = paymentMethodOf(settleTarget);
              return method ? (
                <div className="p-3 rounded-lg bg-green-900/5 text-sm">
                  <p className="font-medium text-green-950">Où envoyer l'argent</p>
                  <p className="text-green-900/70">
                    {method.label} · {method.phone}
                  </p>
                </div>
              ) : (
                <div className="p-3 rounded-lg bg-amber-50 text-amber-800 text-sm">
                  Ce vendeur n'a enregistré aucun moyen de versement. Vérifiez avec lui avant d'envoyer l'argent —{' '}
                  <Link href="/admin/users" className="underline">
                    voir sa fiche
                  </Link>
                  .
                </div>
              );
            })()}
            <Input
              placeholder="Note / référence (ex. Wave TX ABC123)"
              value={settleNote}
              onChange={(e) => setSettleNote(e.target.value)}
            />
            <Input
              type="number"
              placeholder="Frais / taxe retenus (FCFA, 0 si aucun)"
              value={settleFee}
              onChange={(e) => setSettleFee(e.target.value)}
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setSettleTarget(null)}>
                Annuler
              </Button>
              <Button onClick={handleSettle} disabled={busyId === settleTarget.id}>
                {busyId === settleTarget.id ? 'Enregistrement…' : 'Confirmer'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
