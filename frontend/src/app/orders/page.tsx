'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';

interface Order {
  id: string;
  amount_cfa: number;
  status: string;
  payment_reference: string;
  created_at: string;
}

const STATUS_LABELS: Record<string, string> = {
  pending: 'En attente',
  paid: 'Payé',
  delivered: 'Livré',
  failed: 'Échec',
  refunded: 'Remboursé',
};

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

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-4xl mx-auto">
      <header className="mb-6">
        <h1 className="text-3xl font-bold">Mes commandes</h1>
        <p className="text-green-700">Historique de vos achats</p>
      </header>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-600 rounded">{error}</div>}

      {orders.length === 0 ? (
        <div className="text-center py-12 border rounded-lg">
          <p className="text-green-900/50 mb-4">Aucune commande pour le moment.</p>
          <Link href="/catalog" className="text-green-600">Parcourir le catalogue</Link>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border rounded">
            <thead className="bg-green-50">
              <tr>
                <th className="p-3 text-left">Référence</th>
                <th className="p-3 text-left">Montant</th>
                <th className="p-3 text-left">Date</th>
                <th className="p-3 text-left">Statut</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((order) => (
                <tr key={order.id} className="border-t">
                  <td className="p-3">
                    <Link href={`/order?id=${order.id}`} className="text-green-600 font-medium">
                      {order.payment_reference.slice(0, 16)}...
                    </Link>
                  </td>
                  <td className="p-3">{order.amount_cfa.toLocaleString()} FCFA</td>
                  <td className="p-3">
                    {new Date(order.created_at).toLocaleDateString('fr-FR')}
                  </td>
                  <td className="p-3">
                    <span className="px-2 py-1 rounded text-xs bg-green-100">
                      {STATUS_LABELS[order.status] || order.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </main>
  );
}
