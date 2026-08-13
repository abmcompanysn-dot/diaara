'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useWebSocket } from '@/lib/use-websocket';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ArrowLeftIcon, DownloadIcon } from '@/components/icons';
import { ORDER_STATUS_LABELS, formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface Order {
  id: string;
  product_id: string;
  amount_cfa: number;
  status: string;
  payment_reference: string;
  created_at: string;
}

const STATUS_COLOR: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  paid: 'bg-green-100 text-green-700',
  delivered: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
  refunded: 'bg-green-100 text-green-800',
};

export default function OrderDetailPage() {
  const searchParams = useSearchParams();
  const id = searchParams.get('id') || '';
  const [order, setOrder] = useState<Order | null>(null);
  const [productTitle, setProductTitle] = useState('');
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
      setError(friendlyError(err));
    } finally {
      setDelivering(false);
    }
  };

  const reload = () => {
    api
      .getOrder(id)
      .then((r) => setOrder(r.order))
      .catch(() => {});
  };

  // Mise à jour temps réel du statut via WebSocket
  const { connected } = useWebSocket(`/ws/order/${id}`, () => reload());

  useEffect(() => {
    if (id) loadOrder();
  }, [id]);

  const loadOrder = async () => {
    try {
      const result = await api.getOrder(id);
      setOrder(result.order);
      const pid = result.order?.product_id;
      if (pid) {
        api
          .getProduct(pid)
          .then((r) => setProductTitle(r.product?.title || ''))
          .catch(() => {});
      }
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// mon espace" title="Commande" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader eyebrow="// mon espace" title="Commande" />

      <section className="max-w-2xl mx-auto px-4 sm:px-6 py-10">
        <nav className="mb-6">
          <Button variant="outline" size="sm" render={<Link href="/orders" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Mes commandes
          </Button>
        </nav>

        {!order ? (
          <div className="text-center">
            <p className="text-destructive">{error}</p>
          </div>
        ) : (
          <div className="bg-white rounded-xl border border-green-900/10 shadow-card p-6 sm:p-8">
            <div className="flex justify-between items-center mb-6">
              <span className="text-muted-foreground text-sm font-mono">
                Réf. {order.payment_reference.slice(0, 12)}…
              </span>
              <Badge className={STATUS_COLOR[order.status] || 'bg-green-100'}>
                {ORDER_STATUS_LABELS[order.status] || order.status}
              </Badge>
            </div>

            {connected && (
              <p className="mb-4 text-xs text-green-600 flex items-center gap-1" aria-live="polite">
                <span className="w-2 h-2 bg-green-500 rounded-full inline-block" />
                Live : mise à jour en temps réel
              </p>
            )}

            {productTitle && (
              <p className="mb-4 text-sm text-green-900/70">
                Produit : <span className="font-medium text-green-950">{productTitle}</span>
              </p>
            )}

            <div className="space-y-3 text-lg">
              <div className="flex justify-between">
                <span className="text-green-700">Montant</span>
                <span className="font-bold font-mono">{formatPrice(order.amount_cfa)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-green-700">Date</span>
                <span>{new Date(order.created_at).toLocaleDateString('fr-FR')}</span>
              </div>
            </div>

            {order.status === 'pending' && (
              <div className="mt-6 p-4 bg-yellow-50 rounded text-sm text-yellow-800" aria-live="polite">
                En attente de confirmation de votre paiement mobile money.
                Cette page se met à jour automatiquement.
              </div>
            )}

            {order.status === 'paid' && (
              <div className="mt-6 p-4 bg-green-50 rounded text-sm text-green-800" aria-live="polite">
                Paiement confirmé ! La livraison de votre fichier est en cours.
              </div>
            )}

            {order.status === 'delivered' && (
              <div className="mt-6 p-4 bg-green-50 rounded text-sm text-green-800" aria-live="polite">
                Votre fichier a été livré. Retrouvez-le dans votre espace.
              </div>
            )}

            {(order.status === 'paid' || order.status === 'delivered') && (
              <div className="mt-6">
                {deliveryUrl ? (
                  <Button
                    render={
                      <a href={deliveryUrl} download className="block w-full">
                        <DownloadIcon size={18} className="mr-2" />
                        Télécharger mon fichier
                      </a>
                    }
                    className="w-full h-11 font-semibold"
                  />
                ) : (
                  <Button
                    onClick={handleDownload}
                    disabled={delivering}
                    className="w-full h-11 font-semibold"
                  >
                    <DownloadIcon size={18} className="mr-2" />
                    {delivering ? 'Préparation du téléchargement…' : 'Télécharger mon fichier'}
                  </Button>
                )}
                <p className="mt-2 text-xs text-muted-foreground text-center">
                  Lien valable 5 minutes · 3 téléchargements maximum
                </p>
              </div>
            )}
          </div>
        )}
      </section>
    </main>
  );
}