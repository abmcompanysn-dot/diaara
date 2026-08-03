'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface Sale {
  id: string;
  product_id: string;
  buyer_id: string;
  referral_link_id?: string;
  amount_cfa: number;
  platform_fee_cfa: number;
  closer_commission_cfa: number;
  vendor_amount_cfa: number;
  status: string;
  created_at: string;
}

const STATUS_STYLE: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-100',
  paid: 'bg-green-100 text-green-700 hover:bg-green-100',
  delivered: 'bg-green-100 text-green-700 hover:bg-green-100',
  failed: 'bg-red-100 text-red-700 hover:bg-red-100',
  refunded: 'bg-gray-100 text-gray-600 hover:bg-gray-100',
};

export default function AdminSalesPage() {
  const [sales, setSales] = useState<Sale[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadSales();
  }, []);

  const loadSales = async () => {
    setLoading(true);
    try {
      const result = await api.getSales();
      setSales(result.sales);
    } catch (err: any) {
      setError(err.message || 'Impossible de charger les ventes');
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-6xl mx-auto">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Ventes</h1>
          <p className="text-green-700">{sales.length} vente(s)</p>
        </div>
        <Button variant="outline" size="sm" render={<Link href="/admin" />}>
          ← Dashboard
        </Button>
      </header>

      {error && <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded">{error}</div>}

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Date</TableHead>
              <TableHead>Montant</TableHead>
              <TableHead>Frais plateforme</TableHead>
              <TableHead>Commission affilié</TableHead>
              <TableHead>Vendeur</TableHead>
              <TableHead>Statut</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sales.map((sale) => (
              <TableRow key={sale.id}>
                <TableCell className="text-sm">
                  {new Date(sale.created_at).toLocaleString('fr-FR')}
                </TableCell>
                <TableCell>{sale.amount_cfa.toLocaleString('fr-FR')} FCFA</TableCell>
                <TableCell>{sale.platform_fee_cfa.toLocaleString('fr-FR')} FCFA</TableCell>
                <TableCell>
                  {sale.referral_link_id ? (
                    <Badge className="bg-amber-100 text-amber-700 hover:bg-amber-100">
                      {sale.closer_commission_cfa.toLocaleString('fr-FR')} FCFA
                    </Badge>
                  ) : (
                    <span className="text-muted-foreground/40">—</span>
                  )}
                </TableCell>
                <TableCell>{sale.vendor_amount_cfa.toLocaleString('fr-FR')} FCFA</TableCell>
                <TableCell>
                  <Badge variant="outline" className={STATUS_STYLE[sale.status]}>
                    {sale.status}
                  </Badge>
                </TableCell>
              </TableRow>
            ))}
            {sales.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="p-8 text-center text-muted-foreground">
                  Aucune vente pour le moment.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </main>
  );
}
