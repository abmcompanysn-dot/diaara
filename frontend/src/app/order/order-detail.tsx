'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useWebSocket } from '@/lib/use-websocket';

interface Order {
  id: string;
  product_id: string;
  amount_cfa: number;
  status: string;
  payment_reference: string;
  created_at: string;
}

const STATUS_LABELS: Record<string, { label: string; color: string }> = {
  pending: { label: 'En attente de paiement', color: 'bg-yellow-100 text-yellow-700' },
  paid: { label: 'Paiement confirmé', color: 'bg-green-100 text-green-700' },
  delivered: { label: 'Livré', color: 'bg-green-100 text-green-700' },
  failed: { label: 'Échec', color: 'bg-red-100 text-red-700' },
  refunded: { label: 'Remboursé', color: 'bg-green-100 text-green-800' },
};

export default function OrderDetailPage() {
  const searchParams = useSearchParams();
  const id = searchParams.get('id') || '';
  const [order, setOrder] = useState<Order | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [delivering, setDelivering] = useState(false);
  const [deliveryUrl, setDeliveryUrl] = useState('');

  const handleDownload = async () => {
    setDelivering(true);
    setError('');
    try {
      const result = await api.getDelivery(id);
      setDeliveryUrl(result.signed_url);
    } catch (err: any) {
      setError(err.message || 'Téléchargement indisponible');
    } finally {
      setDelivering(false);
    }
  };

  const reload = () => {
    api.getOrder(id).then((r) => setOrder(r.order)).catch(() => {});
  };

  // Mise à jour temps réel du statut via WebSocket
  const { connected } = useWebSocket(`/ws/order/${id}`, (data: any) => {
    // On recharge la commande quand une notification DB arrive pour ce canal
    reload();
  });

  useEffect(() => {
    if (id) loadOrder();
  }, [id]);

  const loadOrder = async () => {
    try {
      const result = await api.getOrder(id);
      setOrder(result.order);
    } catch (err: any) {
      setError(err.message || 'Commande introuvable');
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  if (!order) {
    return (
      <main className="p-8 text-center">
        <p className="text-red-600">{error}</p>
        <Link href="/orders" className="text-green-600">Mes commandes</Link>
      </main>
    );
  }

  const status = STATUS_LABELS[order.status] || { label: order.status, color: 'bg-green-100' };
  const date = new Date(order.created_at).toLocaleDateString('fr-FR');

  return (
    <main className="min-h-screen p-8 max-w-2xl mx-auto">
      <nav className="mb-6">
        <Link href="/orders" className="text-green-600">← Mes commandes</Link>
      </nav>

      <h1 className="text-3xl font-bold mb-6">Commande</h1>

      <div className="border rounded-lg p-8">
        <div className="flex justify-between items-center mb-6">
          <span className="text-green-900/50 text-sm">Réf. {order.payment_reference.slice(0, 12)}...</span>
          <span className={`px-3 py-1 rounded ${status.color}`}>{status.label}</span>
        </div>

        {connected && (
          <p className="mb-4 text-xs text-green-600 flex items-center gap-1">
            <span className="w-2 h-2 bg-green-500 rounded-full inline-block"></span>
            Live : mise à jour en temps réel
          </p>
        )}

        <div className="space-y-3 text-lg">
          <div className="flex justify-between">
            <span className="text-green-700">Montant</span>
            <span className="font-bold">{order.amount_cfa.toLocaleString()} FCFA</span>
          </div>
          <div className="flex justify-between">
            <span className="text-green-700">Date</span>
            <span>{date}</span>
          </div>
        </div>

        {order.status === 'pending' && (
          <div className="mt-6 p-4 bg-yellow-50 rounded text-sm text-yellow-800">
            En attente de confirmation de votre paiement mobile money.
            Cette page se met à jour automatiquement.
          </div>
        )}

        {order.status === 'paid' && (
          <div className="mt-6 p-4 bg-green-50 rounded text-sm text-green-800">
            Paiement confirmé ! La livraison de votre fichier est en cours.
          </div>
        )}

        {order.status === 'delivered' && (
          <div className="mt-6 p-4 bg-green-50 rounded text-sm text-green-800">
            Votre fichier a été livré. Retrouvez-le dans votre espace.
          </div>
        )}

        {(order.status === 'paid' || order.status === 'delivered') && (
          <div className="mt-6">
            {deliveryUrl ? (
              <a
                href={deliveryUrl}
                download
                className="block w-full py-3 bg-green-600 text-white rounded-lg text-center font-semibold hover:bg-green-700"
              >
                Télécharger mon fichier
              </a>
            ) : (
              <button
                onClick={handleDownload}
                disabled={delivering}
                className="w-full py-3 bg-green-600 text-white rounded-lg font-semibold hover:bg-green-700 disabled:opacity-50"
              >
                {delivering ? 'Préparation du téléchargement...' : 'Télécharger mon fichier'}
              </button>
            )}
            <p className="mt-2 text-xs text-green-900/50 text-center">
              Lien valable 5 minutes · 3 téléchargements maximum
            </p>
          </div>
        )}
      </div>
    </main>
  );
}
