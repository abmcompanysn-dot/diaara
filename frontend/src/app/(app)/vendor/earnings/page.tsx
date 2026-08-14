'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetFooter } from '@/components/ui/sheet';
import { Toaster } from '@/components/ui/toast';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { PayoutMethodForm } from '@/components/payout-method-form';
import { ArrowLeftIcon, CheckIcon, DownloadIcon, WalletIcon, ChevronDownIcon } from '@/components/icons';
import { formatPrice, PAYOUT_STATUS_BADGE, PAYOUT_STATUS_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';
import { PAYOUT_COUNTRIES, findPayoutOperator, maskPhone } from '@/lib/operators';
import { openPayoutReceipt, payoutReference } from '@/lib/payout-receipt';
import { useToast } from '@/hooks/use-toast';
import { cn } from '@/lib/utils';

function operatorLogo(country: string | null, operator: string | null): string | undefined {
  if (!country || !operator) return undefined;
  return PAYOUT_COUNTRIES.find((c) => c.code === country)?.operators.find((o) => o.provider === operator)?.logo;
}

interface Payout {
  id: string;
  amount_cfa: number;
  status: string;
  phone_number: string;
  operator: string;
  requested_at: string;
  paid_at?: string | null;
  failure_reason?: string | null;
}

interface PayoutMethod {
  phone: string | null;
  operator: string | null;
  operator_label: string;
  country: string | null;
}

const STATUS_DOT: Record<string, string> = {
  paid: 'bg-green-500',
  processing: 'bg-blue-500',
  requested: 'bg-yellow-500',
  failed: 'bg-red-500',
};

export default function VendorEarningsPage() {
  const [totalEarned, setTotalEarned] = useState(0);
  const [available, setAvailable] = useState(0);
  const [history, setHistory] = useState<Payout[]>([]);
  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [sheetOpen, setSheetOpen] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const [payoutMethod, setPayoutMethod] = useState<PayoutMethod | null>(null);
  const [editingMethod, setEditingMethod] = useState(false);
  const [methodSubmitting, setMethodSubmitting] = useState(false);
  const [methodError, setMethodError] = useState('');
  const [payoutLimits, setPayoutLimits] = useState<Record<string, { min: number; max: number }>>({});

  const { toasts, toast, dismiss } = useToast();
  const { user } = useAuth();

  const hasPayoutMethod = Boolean(payoutMethod?.operator);
  const phoneVerified = Boolean(user?.phone_verified_at);
  const operatorLimit = payoutMethod?.operator ? payoutLimits[payoutMethod.operator] : undefined;

  useEffect(() => {
    loadEarnings();
    loadPayoutMethod();
    loadPayoutLimits();
  }, []);

  const loadEarnings = async () => {
    setLoading(true);
    try {
      const result = await api.getVendorEarnings();
      setTotalEarned(result.total_earned);
      setAvailable(result.available);
      setHistory(result.history);
    } catch (err: any) {
      toast({ variant: 'error', title: 'Chargement impossible', description: friendlyError(err) });
    } finally {
      setLoading(false);
    }
  };

  const loadPayoutMethod = async () => {
    try {
      const result = await api.getPayoutMethod();
      setPayoutMethod(result.payout_method);
    } catch {
      // Pas grave si l'appel échoue au chargement initial : le formulaire sera vide.
    }
  };

  const loadPayoutLimits = async () => {
    try {
      const result = await api.getPayoutLimits();
      setPayoutLimits(result.limits);
    } catch {
      // Pas grave : sans les limites, on retombe sur la validation solde/téléphone uniquement.
    }
  };

  const openEditMethod = () => {
    setMethodError('');
    setEditingMethod(true);
  };

  const handleSaveMethod = async (data: { phone: string; operator: string; country: string }) => {
    setMethodError('');
    setMethodSubmitting(true);
    try {
      const result = await api.setPayoutMethod(data);
      setPayoutMethod(result.payout_method);
      setEditingMethod(false);
      toast({ variant: 'success', title: 'Moyen de versement enregistré', description: `${result.payout_method.operator_label} — vos prochains versements y seront envoyés.` });
    } catch (err: any) {
      setMethodError(friendlyError(err));
    } finally {
      setMethodSubmitting(false);
    }
  };

  // Calculées à partir de l'historique réel plutôt que du champ "pending" de
  // l'API (qui agrège aussi les versements déjà payés) pour distinguer
  // précisément ce qui est en cours de ce qui est déjà versé.
  const totalPaidOut = useMemo(() => history.filter((p) => p.status === 'paid').reduce((sum, p) => sum + p.amount_cfa, 0), [history]);
  const inProgress = useMemo(
    () => history.filter((p) => p.status === 'requested' || p.status === 'processing').reduce((sum, p) => sum + p.amount_cfa, 0),
    [history]
  );

  const amountNum = parseInt(amount, 10);
  const hasValidAmount = !isNaN(amountNum) && amountNum > 0;
  const netAmount = hasValidAmount ? amountNum : 0; // Aucuns frais appliqués sur les versements PawaPay.

  const blockReason = !hasPayoutMethod
    ? "Ajoutez d'abord un moyen de versement pour demander un retrait."
    : !phoneVerified
      ? "Votre numéro de téléphone doit être vérifié avant de pouvoir retirer des fonds."
      : available <= 0
        ? 'Aucun solde disponible pour le moment.'
        : operatorLimit && available < operatorLimit.min
          ? `Solde insuffisant : ${payoutMethod?.operator_label} accepte un minimum de ${formatPrice(operatorLimit.min)}.`
          : hasValidAmount && amountNum > available
            ? 'Le montant dépasse votre solde disponible.'
            : hasValidAmount && operatorLimit && amountNum < operatorLimit.min
              ? `Montant minimum pour ${payoutMethod?.operator_label} : ${formatPrice(operatorLimit.min)}.`
              : '';

  const canSubmit =
    hasPayoutMethod &&
    phoneVerified &&
    hasValidAmount &&
    amountNum <= available &&
    (!operatorLimit || amountNum >= operatorLimit.min);

  const handlePayout = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;

    setSubmitting(true);
    try {
      await api.requestPayout(amountNum);
      toast({ variant: 'success', title: 'Demande envoyée', description: `Votre versement de ${formatPrice(amountNum)} est en cours de traitement.` });
      setAmount('');
      setSheetOpen(false);
      loadEarnings();
    } catch (err: any) {
      toast({ variant: 'error', title: 'Versement impossible', description: friendlyError(err) });
    } finally {
      setSubmitting(false);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// espace vendeur" title="Mes revenus & Versements" description="Vos gains de vente et versements" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// espace vendeur"
        title="Mes revenus & Versements"
        description="Suivez vos gains, votre moyen de versement et l'historique de vos retraits"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/vendor/products" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Mes produits
          </Button>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-8 sm:py-10 space-y-6">
        {/* Carte principale : solde disponible + retrait en un tap */}
        <div className="rounded-2xl gradient-green text-white p-6 sm:p-8 shadow-lift relative overflow-hidden">
          <div className="wax-pattern absolute inset-0 opacity-70" aria-hidden />
          <div className="relative">
            <p className="font-mono text-xs uppercase tracking-widest text-green-300">💰 Solde disponible</p>
            <p className="font-display text-4xl sm:text-5xl font-bold mt-2 font-mono">{formatPrice(available)}</p>

            <Button
              onClick={() => setSheetOpen(true)}
              disabled={!hasPayoutMethod || !phoneVerified || available <= 0}
              className="mt-5 w-full sm:w-auto h-12 px-8 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300 text-base"
            >
              💸 Retirer mes fonds
            </Button>
            {!hasPayoutMethod && (
              <p className="mt-2 text-xs text-white/60">Ajoutez un moyen de versement ci-dessous pour activer les retraits.</p>
            )}
            {hasPayoutMethod && !phoneVerified && (
              <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-white/80 bg-white/10 rounded-lg px-3 py-2.5">
                <span>Numéro de téléphone non vérifié — requis pour les retraits.</span>
                <Button
                  size="sm"
                  variant="outline"
                  className="h-7 px-3 text-xs bg-transparent border-white/30 text-white hover:bg-white/10 hover:text-white"
                  render={<Link href={user?.phone ? '/auth/verify-phone' : '/account'} />}
                >
                  Vérifier maintenant
                </Button>
              </div>
            )}

            <div className="mt-6 flex flex-wrap gap-x-8 gap-y-2 text-sm border-t border-white/15 pt-4">
              <p className="text-white/70">
                En attente <span className="block sm:inline font-mono font-semibold text-white">{formatPrice(inProgress)}</span>
              </p>
              <p className="text-white/70">
                Versements effectués <span className="block sm:inline font-mono font-semibold text-white">{formatPrice(totalPaidOut)}</span>
              </p>
              <p className="text-white/70">
                Total généré <span className="block sm:inline font-mono font-semibold text-white">{formatPrice(totalEarned)}</span>
              </p>
            </div>
          </div>
        </div>

        {/* Moyen de versement */}
        <Card className="border-green-900/5">
          <CardHeader className="pb-3 flex flex-row items-center justify-between">
            <div>
              <CardTitle className="text-lg">Moyen de versement</CardTitle>
              <CardDescription>Le compte mobile money qui recevra vos versements</CardDescription>
            </div>
            {!editingMethod && (
              <Button variant="outline" size="sm" onClick={openEditMethod}>
                {hasPayoutMethod ? 'Modifier' : 'Ajouter'}
              </Button>
            )}
          </CardHeader>
          <CardContent>
            {!editingMethod ? (
              hasPayoutMethod ? (
                <div className="flex items-center gap-3 flex-wrap">
                  {operatorLogo(payoutMethod!.country, payoutMethod!.operator) ? (
                    <img
                      src={`/payments/${operatorLogo(payoutMethod!.country, payoutMethod!.operator)}`}
                      alt={payoutMethod!.operator_label}
                      className="h-8 max-w-[110px] object-contain"
                    />
                  ) : (
                    <Badge className="bg-green-100 text-green-700 hover:bg-green-100">
                      {payoutMethod!.operator_label}
                    </Badge>
                  )}
                  <span className="font-mono text-sm text-green-900/70">
                    {maskPhone(payoutMethod!.phone, findPayoutOperator(payoutMethod!.operator)?.dialCode) || payoutMethod!.phone}
                  </span>
                  <Badge className="bg-green-100 text-green-700 hover:bg-green-100 gap-1">
                    <CheckIcon size={11} />
                    Vérifié
                  </Badge>
                  {operatorLimit && (
                    <span className="text-xs text-green-900/50">
                      Retrait minimum : {formatPrice(operatorLimit.min)}
                    </span>
                  )}
                </div>
              ) : (
                <div className="flex flex-col items-start gap-3 py-2">
                  <div className="w-10 h-10 rounded-full bg-yellow-100 flex items-center justify-center">
                    <WalletIcon size={18} className="text-yellow-700" />
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Aucun moyen de versement enregistré. Ajoutez-en un pour pouvoir demander un versement.
                  </p>
                  <Button size="sm" onClick={openEditMethod}>
                    Ajouter un moyen
                  </Button>
                </div>
              )
            ) : (
              <PayoutMethodForm
                initialCountry={payoutMethod?.country}
                initialOperator={payoutMethod?.operator}
                initialPhone={payoutMethod?.phone}
                onSave={handleSaveMethod}
                onCancel={() => setEditingMethod(false)}
                saving={methodSubmitting}
                error={methodError}
              />
            )}
          </CardContent>
        </Card>

        {/* Historique */}
        <section>
          <h2 className="font-semibold text-lg mb-4 text-green-950">Historique des versements</h2>
          {history.length === 0 ? (
            <p className="text-muted-foreground text-sm">Aucun versement pour le moment.</p>
          ) : (
            <>
              {/* Vue mobile : liste verticale avec détails dépliables */}
              <div className="sm:hidden space-y-2">
                {history.map((payout) => {
                  const op = findPayoutOperator(payout.operator);
                  const expanded = expandedId === payout.id;
                  return (
                    <div key={payout.id} className="bg-white rounded-xl border border-green-900/10 shadow-card overflow-hidden">
                      <button
                        type="button"
                        onClick={() => setExpandedId(expanded ? null : payout.id)}
                        className="w-full flex items-center gap-3 p-3.5 text-left"
                        aria-expanded={expanded}
                      >
                        <span className="w-9 h-9 rounded-full bg-green-50 flex items-center justify-center shrink-0 text-green-700">
                          <WalletIcon size={16} />
                        </span>
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-medium text-green-950">
                            {new Date(payout.requested_at).toLocaleDateString('fr-FR', { day: '2-digit', month: 'short' })}
                          </p>
                          <p className="text-xs text-green-900/50 font-mono truncate">#{payoutReference(payout)}</p>
                        </div>
                        <div className="text-right shrink-0">
                          <p className="font-mono font-bold text-sm text-green-950">{formatPrice(payout.amount_cfa)}</p>
                          <span className={cn('inline-flex items-center gap-1 text-[11px] font-medium', {
                            'text-green-700': payout.status === 'paid',
                            'text-blue-700': payout.status === 'processing',
                            'text-yellow-700': payout.status === 'requested',
                            'text-red-700': payout.status === 'failed',
                          })}>
                            <span className={cn('w-1.5 h-1.5 rounded-full', STATUS_DOT[payout.status] || 'bg-gray-400')} />
                            {PAYOUT_STATUS_LABELS[payout.status] || payout.status}
                          </span>
                        </div>
                        <ChevronDownIcon
                          size={16}
                          className={cn('text-green-900/30 shrink-0 transition-transform', expanded && 'rotate-180')}
                        />
                      </button>
                      {expanded && (
                        <div className="px-3.5 pb-3.5 pt-0 border-t border-green-900/5 space-y-2">
                          <div className="flex items-center justify-between text-xs text-green-900/60 pt-3">
                            <span>Opérateur</span>
                            {op?.logo ? (
                              <img src={`/payments/${op.logo}`} alt={op.label} className="h-4 object-contain" />
                            ) : (
                              <span>{op?.label || payout.operator}</span>
                            )}
                          </div>
                          {payout.status === 'failed' && payout.failure_reason && (
                            <p className="text-xs text-red-600">{payout.failure_reason}</p>
                          )}
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => openPayoutReceipt(payout)}
                            className="w-full gap-1.5"
                          >
                            <DownloadIcon size={14} />
                            Télécharger le reçu
                          </Button>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* Vue desktop : tableau */}
              <div className="hidden sm:block rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Date</TableHead>
                      <TableHead>Référence</TableHead>
                      <TableHead>Opérateur</TableHead>
                      <TableHead>Montant net</TableHead>
                      <TableHead>Statut</TableHead>
                      <TableHead className="text-right">Reçu</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {history.map((payout) => {
                      const op = findPayoutOperator(payout.operator);
                      return (
                        <TableRow key={payout.id}>
                          <TableCell className="whitespace-nowrap text-sm">
                            {new Date(payout.requested_at).toLocaleString('fr-FR', { dateStyle: 'short', timeStyle: 'short' })}
                          </TableCell>
                          <TableCell className="font-mono text-xs text-green-900/70">#{payoutReference(payout)}</TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              {op?.logo ? (
                                <img src={`/payments/${op.logo}`} alt={op.label} className="h-5 max-w-[70px] object-contain" />
                              ) : (
                                <span className="text-sm">{op?.label || payout.operator}</span>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="font-mono">{formatPrice(payout.amount_cfa)}</TableCell>
                          <TableCell>
                            <div>
                              <Badge className={PAYOUT_STATUS_BADGE[payout.status]}>
                                {PAYOUT_STATUS_LABELS[payout.status] || payout.status}
                              </Badge>
                              {payout.status === 'failed' && payout.failure_reason && (
                                <p className="text-xs text-red-600 mt-1 max-w-[180px]">{payout.failure_reason}</p>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="text-right">
                            <Button variant="ghost" size="sm" onClick={() => openPayoutReceipt(payout)}>
                              <DownloadIcon size={14} className="mr-1.5" />
                              Reçu
                            </Button>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </div>
            </>
          )}
        </section>
      </section>

      {/* Tiroir de retrait (bottom sheet) : formulaire tactile déclenché depuis la
          carte de solde, clavier numérique natif via inputMode="numeric". */}
      <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
        <SheetContent side="bottom" className="rounded-t-2xl max-h-[85vh] overflow-y-auto">
          <SheetHeader>
            <SheetTitle>Retirer mes fonds</SheetTitle>
            <SheetDescription>
              Disponible : <span className="font-semibold text-green-900">{formatPrice(available)}</span>
            </SheetDescription>
          </SheetHeader>
          {/* Un seul <form> englobe le corps et le pied du tiroir, pour que le bouton
              de confirmation (dans SheetFooter) déclenche bien la soumission. */}
          <form onSubmit={handlePayout} className="flex flex-col flex-1 min-h-0">
            <div className="px-4 space-y-4">
              <div className="flex gap-2">
                <Input
                  type="number"
                  inputMode="numeric"
                  autoFocus
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  placeholder={operatorLimit ? `Montant en FCFA (min. ${operatorLimit.min.toLocaleString('fr-FR')})` : 'Montant en FCFA'}
                  className="flex-1 h-12 text-lg"
                  min={operatorLimit?.min || 1}
                  max={available}
                  disabled={!hasPayoutMethod}
                />
                <Button
                  type="button"
                  variant="outline"
                  className="h-12"
                  disabled={!hasPayoutMethod || available <= 0}
                  onClick={() => setAmount(String(available))}
                >
                  Max
                </Button>
              </div>

              <div className="rounded-lg bg-green-50/60 border border-green-900/5 p-3 space-y-1.5 text-sm">
                <div className="flex justify-between text-green-900/70">
                  <span>Frais de transfert</span>
                  <span className="font-mono">0 FCFA</span>
                </div>
                <div className="flex justify-between font-semibold text-green-950">
                  <span>Montant net reçu</span>
                  <span className="font-mono">{formatPrice(netAmount)}</span>
                </div>
              </div>

              <p className={`text-xs ${blockReason ? 'text-yellow-700' : 'text-muted-foreground'}`}>
                {blockReason || 'Versements traités sous 48h.'}
              </p>
            </div>
            <SheetFooter>
              <Button
                type="submit"
                disabled={submitting || !canSubmit}
                className="w-full h-12 text-base font-semibold"
              >
                {submitting ? 'Envoi…' : 'Confirmer le retrait'}
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>

      <Toaster toasts={toasts} onDismiss={dismiss} />
    </main>
  );
}
