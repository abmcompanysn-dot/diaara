'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { PayoutMethodForm } from '@/components/payout-method-form';
import { ArrowLeftIcon } from '@/components/icons';
import { formatPrice, PAYOUT_STATUS_BADGE, PAYOUT_STATUS_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';
import { PAYOUT_COUNTRIES } from '@/lib/operators';

function operatorLogo(country: string | null, operator: string | null): string | undefined {
  if (!country || !operator) return undefined;
  return PAYOUT_COUNTRIES.find((c) => c.code === country)?.operators.find((o) => o.provider === operator)?.logo;
}

interface Payout {
  id: string;
  amount_cfa: number;
  status: string;
  requested_at: string;
}

interface PayoutMethod {
  phone: string | null;
  operator: string | null;
  operator_label: string;
  country: string | null;
}

export default function VendorEarningsPage() {
  const [totalEarned, setTotalEarned] = useState(0);
  const [available, setAvailable] = useState(0);
  const [history, setHistory] = useState<Payout[]>([]);
  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');

  const [payoutMethod, setPayoutMethod] = useState<PayoutMethod | null>(null);
  const [editingMethod, setEditingMethod] = useState(false);
  const [methodSubmitting, setMethodSubmitting] = useState(false);
  const [methodError, setMethodError] = useState('');

  const hasPayoutMethod = Boolean(payoutMethod?.operator);

  useEffect(() => {
    loadEarnings();
    loadPayoutMethod();
  }, []);

  const loadEarnings = async () => {
    setLoading(true);
    try {
      const result = await api.getVendorEarnings();
      setTotalEarned(result.total_earned);
      setAvailable(result.available);
      setHistory(result.history);
    } catch (err: any) {
      setError(friendlyError(err));
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
    } catch (err: any) {
      setMethodError(friendlyError(err));
    } finally {
      setMethodSubmitting(false);
    }
  };

  const handlePayout = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setMessage('');

    const amountNum = parseInt(amount, 10);
    if (isNaN(amountNum) || amountNum <= 0) {
      setError('Montant invalide');
      return;
    }

    setSubmitting(true);
    try {
      await api.requestPayout(amountNum);
      setMessage('Demande de versement envoyée !');
      setAmount('');
      loadEarnings();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSubmitting(false);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// espace vendeur" title="Mes revenus" description="Vos gains de vente et versements" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// espace vendeur"
        title="Mes revenus"
        description="Vos gains de vente et versements"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/vendor/products" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Mes produits
          </Button>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10 space-y-8">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}
        {message && <div className="p-3 bg-green-50 text-green-700 rounded text-sm" role="status">{message}</div>}

        <div className="grid gap-4 md:grid-cols-3">
          <Card className="bg-green-50/60 border-green-900/5">
            <CardContent className="p-6">
              <p className="text-sm text-green-700">Total gagné</p>
              <p className="text-2xl font-bold font-mono mt-1">{formatPrice(totalEarned)}</p>
            </CardContent>
          </Card>
          <Card className="bg-green-50/60 border-green-900/5">
            <CardContent className="p-6">
              <p className="text-sm text-green-700">Disponible</p>
              <p className="text-2xl font-bold font-mono mt-1">{formatPrice(available)}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-6">
              <p className="text-sm text-green-700">Versements effectués</p>
              <p className="text-2xl font-bold font-mono mt-1">{history.length}</p>
            </CardContent>
          </Card>
        </div>

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
                <div className="flex items-center gap-3">
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
                  <span className="font-mono text-sm text-green-900/70">{payoutMethod!.phone}</span>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Aucun moyen de versement enregistré. Ajoutez-en un pour pouvoir demander un versement.
                </p>
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

        <Card className="border-green-900/5">
          <CardHeader className="pb-3">
            <CardTitle className="text-lg">Demander un versement</CardTitle>
            <CardDescription>Commission plateforme : 15 % sur chaque vente</CardDescription>
          </CardHeader>
          <CardContent>
            {!hasPayoutMethod && (
              <p className="mb-3 text-sm text-yellow-700 bg-yellow-50 rounded p-3">
                Enregistrez d&apos;abord votre moyen de versement ci-dessus.
              </p>
            )}
            <form onSubmit={handlePayout} className="flex flex-col sm:flex-row gap-3 sm:gap-4">
              <Input
                type="number"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                placeholder="Montant en FCFA"
                className="flex-1"
                min={1}
                max={available}
                disabled={!hasPayoutMethod}
              />
              <Button type="submit" disabled={submitting || available <= 0 || !hasPayoutMethod} className="sm:self-start">
                {submitting ? 'Envoi…' : 'Demander'}
              </Button>
            </form>
            <p className="mt-2 text-xs text-muted-foreground">Versements traités sous 48h</p>
          </CardContent>
        </Card>

        <section>
          <h2 className="font-semibold text-lg mb-4 text-green-950">Historique des versements</h2>
          {history.length === 0 ? (
            <p className="text-muted-foreground text-sm">Aucun versement pour le moment.</p>
          ) : (
            <div className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Montant</TableHead>
                    <TableHead>Date</TableHead>
                    <TableHead>Statut</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {history.map((payout) => (
                    <TableRow key={payout.id}>
                      <TableCell className="font-mono">{formatPrice(payout.amount_cfa)}</TableCell>
                      <TableCell>{new Date(payout.requested_at).toLocaleDateString('fr-FR')}</TableCell>
                      <TableCell>
                        <Badge className={PAYOUT_STATUS_BADGE[payout.status]}>
                          {PAYOUT_STATUS_LABELS[payout.status] || payout.status}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </section>
      </section>
    </main>
  );
}
