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
import { ArrowLeftIcon } from '@/components/icons';
import { formatPrice, PAYOUT_STATUS_BADGE, PAYOUT_STATUS_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface Payout {
  id: string;
  amount_cfa: number;
  status: string;
  requested_at: string;
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

  useEffect(() => {
    loadEarnings();
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
          <CardHeader className="pb-3">
            <CardTitle className="text-lg">Demander un versement</CardTitle>
            <CardDescription>Commission plateforme : 15 % sur chaque vente</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handlePayout} className="flex flex-col sm:flex-row gap-3 sm:gap-4">
              <Input
                type="number"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                placeholder="Montant en FCFA"
                className="flex-1"
                min={1}
                max={available}
              />
              <Button type="submit" disabled={submitting || available <= 0} className="sm:self-start">
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