'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { PackageIcon } from '@/components/icons';
import { ORDER_STATUS_LABELS, SALE_STATUS_BADGE, formatPrice } from '@/lib/constants';

interface Order {
  id: string;
  amount_cfa: number;
  status: string;
  payment_reference: string;
  created_at: string;
}

export default function OrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadOrders();
  }, []);

  const loadOrders = async () => {
    setLoading(true);
    try {
      const result = await api.getOrders();
      setOrders(result.orders);
    } catch (err: any) {
      setError(err.message || 'Impossible de charger les commandes');
    } finally {
      setLoading(false);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// mon espace" title="Mes commandes" description="Historique de vos achats" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader eyebrow="// mon espace" title="Mes commandes" description="Historique de vos achats" />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {orders.length === 0 ? (
          <EmptyState
            icon={PackageIcon}
            title="Aucune commande pour le moment"
            description="Dès que vous achèterez un produit, il apparaîtra ici."
            action={
              <Button render={<Link href="/catalog" />}>Parcourir le catalogue</Button>
            }
          />
        ) : (
          <div className="overflow-x-auto rounded-xl border border-green-900/10 bg-white shadow-card">
            <table className="w-full">
              <thead className="bg-green-50/60">
                <tr>
                  <th className="p-3 text-left text-sm font-semibold text-green-900/70">Référence</th>
                  <th className="p-3 text-left text-sm font-semibold text-green-900/70">Montant</th>
                  <th className="p-3 text-left text-sm font-semibold text-green-900/70">Date</th>
                  <th className="p-3 text-left text-sm font-semibold text-green-900/70">Statut</th>
                </tr>
              </thead>
              <tbody>
                {orders.map((order) => (
                  <tr key={order.id} className="border-t border-green-900/10">
                    <td className="p-3">
                      <Link href={`/order?id=${order.id}`} className="text-green-600 font-medium hover:text-green-500">
                        {order.payment_reference.slice(0, 16)}…
                      </Link>
                    </td>
                    <td className="p-3 font-mono">{formatPrice(order.amount_cfa)}</td>
                    <td className="p-3 text-sm">{new Date(order.created_at).toLocaleDateString('fr-FR')}</td>
                    <td className="p-3">
                      <span
                        className={`px-2 py-1 rounded text-xs font-medium ${SALE_STATUS_BADGE[order.status] || 'bg-gray-100 text-gray-600'}`}
                      >
                        {ORDER_STATUS_LABELS[order.status] || order.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </main>
  );
}