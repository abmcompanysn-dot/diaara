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
import { ArrowLeftIcon } from '@/components/icons';
import { formatPrice, SALE_STATUS_BADGE, ORDER_STATUS_LABELS } from '@/lib/constants';
import { CHECKOUT_COUNTRIES } from '@/lib/operators';
import { friendlyError } from '@/lib/error-messages';

interface VendorSale {
  id: string;
  product_title: string;
  buyer_name: string;
  buyer_email: string;
  country?: string | null;
  amount_cfa: number;
  status: string;
  created_at: string;
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

  useEffect(() => {
    api
      .getVendorSales()
      .then((r) => setSales(r.sales))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, []);

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// espace vendeur" title="Mes clients" description="Ventes et coordonnées des acheteurs" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// espace vendeur"
        title="Mes clients"
        description={`${sales.length} vente(s)`}
        actions={
          <Button variant="outline" size="sm" render={<Link href="/vendor/products" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Mes produits
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
          <EmptyState title="Aucune vente pour le moment" description="Les ventes de vos produits apparaîtront ici avec les coordonnées de vos clients." />
        ) : (
          <div className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Date</TableHead>
                  <TableHead>Produit</TableHead>
                  <TableHead>Client</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Pays</TableHead>
                  <TableHead>Montant</TableHead>
                  <TableHead>Statut</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sales.map((sale) => (
                  <TableRow key={sale.id}>
                    <TableCell className="text-sm whitespace-nowrap">
                      {new Date(sale.created_at).toLocaleString('fr-FR')}
                    </TableCell>
                    <TableCell className="max-w-[160px] truncate">{sale.product_title}</TableCell>
                    <TableCell>{sale.buyer_name}</TableCell>
                    <TableCell className="text-sm text-green-900/70">{sale.buyer_email}</TableCell>
                    <TableCell className="whitespace-nowrap">{countryName(sale.country)}</TableCell>
                    <TableCell className="font-mono">{formatPrice(sale.amount_cfa)}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className={SALE_STATUS_BADGE[sale.status]}>
                        {ORDER_STATUS_LABELS[sale.status] || sale.status}
                      </Badge>
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
