'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface Payout {
  id: string;
  amount_cfa: number;
  status: string;
  requested_at: string;
}

const PAYOUT_BADGE: Record<string, string> = {
  paid: 'bg-green-100 text-green-700 hover:bg-green-100',
  failed: 'bg-red-100 text-red-700 hover:bg-red-100',
  pending: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-100',
};

export default function VendorEarningsPage() {
  const [totalEarned, setTotalEarned] = useState(0);
  const [available, setAvailable] = useState(0);
  const [history, setHistory] = useState<Payout[]>([]);
  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(true);
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
      setError(err.message || 'Impossible de charger les revenus');
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

    try {
      await api.requestPayout(amountNum);
      setMessage('Demande de versement envoyée !');
      setAmount('');
      loadEarnings();
    } catch (err: any) {
      setError(err.message || 'Demande de versement échouée');
    }
  };

  const formatPrice = (price: number) => `${price.toLocaleString()} FCFA`;

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-4xl mx-auto">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Mes revenus</h1>
          <p className="text-green-700">Vos gains de vente et versements</p>
        </div>
        <Button variant="outline" size="sm" render={<Link href="/vendor/products" />}>
          Mes produits
        </Button>
      </header>

      {error && <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded">{error}</div>}
      {message && <div className="mb-4 p-3 bg-green-50 text-green-700 rounded">{message}</div>}

      <div className="grid gap-4 md:grid-cols-3 mb-8">
        <Card className="bg-green-50/60 border-green-900/5">
          <CardContent className="p-6">
            <p className="text-sm text-green-700">Total gagné</p>
            <p className="text-2xl font-bold">{formatPrice(totalEarned)}</p>
          </CardContent>
        </Card>
        <Card className="bg-green-50/60 border-green-900/5">
          <CardContent className="p-6">
            <p className="text-sm text-green-700">Disponible</p>
            <p className="text-2xl font-bold">{formatPrice(available)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-sm text-green-700">Versements effectués</p>
            <p className="text-2xl font-bold">{history.length}</p>
          </CardContent>
        </Card>
      </div>

      <Card className="mb-8 border-green-900/5">
        <CardHeader className="pb-3">
          <CardTitle className="text-lg">Demander un versement</CardTitle>
          <CardDescription>Commission plateforme : 15 % sur chaque vente</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handlePayout} className="flex gap-4">
            <Input
              type="number"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="Montant en FCFA"
              className="flex-1"
              min={1}
              max={available}
            />
            <Button type="submit" disabled={available <= 0}>
              Demander
            </Button>
          </form>
          <p className="mt-2 text-xs text-muted-foreground">Versements traités sous 48h</p>
        </CardContent>
      </Card>

      <section>
        <h2 className="font-semibold text-lg mb-4">Historique des versements</h2>
        {history.length === 0 ? (
          <p className="text-muted-foreground">Aucun versement pour le moment.</p>
        ) : (
          <div className="rounded-lg border">
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
                    <TableCell>{formatPrice(payout.amount_cfa)}</TableCell>
                    <TableCell>
                      {new Date(payout.requested_at).toLocaleDateString('fr-FR')}
                    </TableCell>
                    <TableCell>
                      <Badge className={PAYOUT_BADGE[payout.status]}>{payout.status}</Badge>
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
