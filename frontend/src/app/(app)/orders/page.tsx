'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { ProductImage } from '@/components/product-image';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { PackageIcon, SearchIcon, XIcon, DownloadIcon, HeadsetIcon, FileIcon } from '@/components/icons';
import { ORDER_STATUS_LABELS, SALE_STATUS_BADGE, formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';
import { openSaleReceipt } from '@/lib/sale-receipt';
import { cn } from '@/lib/utils';

interface Order {
  id: string;
  product_id: string;
  amount_cfa: number;
  status: string;
  buyer_name?: string;
  payment_reference: string;
  created_at: string;
}

interface ProductInfo {
  id: string;
  title: string;
  category: string;
  cover_image_key?: string;
}

type FilterKey = 'all' | 'available' | 'pending';

const FILTERS: { key: FilterKey; label: string }[] = [
  { key: 'all', label: 'Toutes' },
  { key: 'available', label: 'Disponibles' },
  { key: 'pending', label: 'En attente' },
];

const STATUS_DOT: Record<string, string> = {
  pending: 'bg-yellow-500',
  paid: 'bg-green-500',
  delivered: 'bg-green-500',
  failed: 'bg-red-500',
  refunded: 'bg-gray-400',
};

export default function OrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [products, setProducts] = useState<Record<string, ProductInfo>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<FilterKey>('all');
  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const [downloadError, setDownloadError] = useState<Record<string, string>>({});

  useEffect(() => {
    loadOrders();
  }, []);

  const loadOrders = async () => {
    setLoading(true);
    try {
      const result = await api.getOrders();
      setOrders(result.orders);
      const ids = Array.from(new Set(result.orders.map((o: Order) => o.product_id).filter(Boolean)));
      const entries = await Promise.all(
        ids.map((id) =>
          api
            .getProduct(id)
            .then((r) => [id, r.product] as const)
            .catch(() => [id, null] as const)
        )
      );
      const map: Record<string, ProductInfo> = {};
      entries.forEach(([id, product]) => {
        if (product) map[id] = product;
      });
      setProducts(map);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleDownload = async (order: Order) => {
    setDownloadingId(order.id);
    setDownloadError((prev) => ({ ...prev, [order.id]: '' }));
    try {
      const result = await api.getDelivery(order.id);
      window.open(result.signed_url, '_blank', 'noopener,noreferrer');
    } catch (err: any) {
      setDownloadError((prev) => ({ ...prev, [order.id]: friendlyError(err) }));
    } finally {
      setDownloadingId(null);
    }
  };

  const filteredOrders = useMemo(() => {
    return orders.filter((order) => {
      if (filter === 'available' && !['paid', 'delivered'].includes(order.status)) return false;
      if (filter === 'pending' && order.status !== 'pending') return false;
      if (search) {
        const title = products[order.product_id]?.title || '';
        const haystack = `${title} ${order.payment_reference}`.toLowerCase();
        if (!haystack.includes(search.toLowerCase())) return false;
      }
      return true;
    });
  }, [orders, products, filter, search]);

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// mon espace" title="Mes achats" description="Historique de vos achats" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader eyebrow="// mon espace" title="Mes achats" description="Historique de vos achats" />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-8 sm:py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {orders.length === 0 ? (
          <EmptyState
            icon={PackageIcon}
            title="Vous n'avez pas encore effectué d'achat"
            description="Dès que vous achèterez un produit, il apparaîtra ici."
            action={
              <Button render={<Link href="/catalog" />}>Explorer le catalogue</Button>
            }
          />
        ) : (
          <>
            {/* Recherche + filtres */}
            <div className="relative max-w-md mb-3">
              <SearchIcon
                size={16}
                className="absolute left-3.5 top-1/2 -translate-y-1/2 text-green-900/40 pointer-events-none"
              />
              <input
                type="text"
                placeholder="Rechercher dans mes achats..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full h-10 pl-9 pr-9 rounded-full bg-white border border-green-900/10 text-sm outline-none focus:border-green-600/40 placeholder-green-900/40"
              />
              {search && (
                <button
                  type="button"
                  onClick={() => setSearch('')}
                  aria-label="Effacer"
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 w-6 h-6 rounded-full flex items-center justify-center text-green-900/40 hover:bg-green-900/5"
                >
                  <XIcon size={14} />
                </button>
              )}
            </div>
            <div className="flex gap-2 mb-6">
              {FILTERS.map((f) => (
                <button
                  key={f.key}
                  type="button"
                  onClick={() => setFilter(f.key)}
                  className={cn(
                    'px-3.5 py-1.5 rounded-full text-sm font-medium border transition-colors',
                    filter === f.key
                      ? 'bg-green-950 text-white border-green-950'
                      : 'bg-white text-green-900/60 border-green-900/10 hover:bg-green-900/5'
                  )}
                >
                  {f.label}
                </button>
              ))}
            </div>

            {filteredOrders.length === 0 ? (
              <EmptyState
                icon={PackageIcon}
                title="Aucun achat ne correspond"
                description="Modifiez votre recherche ou changez de filtre."
              />
            ) : (
              <div className="grid gap-4 sm:grid-cols-2">
                {filteredOrders.map((order) => {
                  const product = products[order.product_id];
                  const available = order.status === 'paid' || order.status === 'delivered';
                  return (
                    <div
                      key={order.id}
                      className="bg-white rounded-xl border border-green-900/10 shadow-card overflow-hidden"
                    >
                      <div className="flex items-center justify-between px-4 pt-3.5 pb-2 text-xs text-green-900/50">
                        <span>
                          📅 {new Date(order.created_at).toLocaleDateString('fr-FR')} · #
                          {order.payment_reference.slice(0, 8)}
                        </span>
                        <span
                          className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[11px] font-medium ${SALE_STATUS_BADGE[order.status] || 'bg-gray-100 text-gray-600'}`}
                        >
                          <span className={`w-1.5 h-1.5 rounded-full ${STATUS_DOT[order.status] || 'bg-gray-400'}`} />
                          {ORDER_STATUS_LABELS[order.status] || order.status}
                        </span>
                      </div>

                      <Link
                        href={`/order?id=${order.id}`}
                        className="flex items-center gap-3 px-4 pb-3 hover:bg-green-900/2 transition-colors"
                      >
                        <div className="w-14 h-14 rounded-lg overflow-hidden shrink-0">
                          {product ? (
                            <ProductImage product={product} className="h-14" />
                          ) : (
                            <div className="w-14 h-14 rounded-lg bg-green-900/5" />
                          )}
                        </div>
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-medium text-green-950 line-clamp-1">
                            {product?.title || 'Produit'}
                          </p>
                          <p className="text-xs font-mono text-green-900/50 mt-0.5">
                            {formatPrice(order.amount_cfa)}
                          </p>
                        </div>
                      </Link>

                      <div className="flex items-center gap-2 px-4 pb-4">
                        {available ? (
                          <>
                            <Button
                              size="sm"
                              onClick={() => handleDownload(order)}
                              disabled={downloadingId === order.id}
                              className="flex-1 gap-1.5"
                            >
                              <DownloadIcon size={14} />
                              {downloadingId === order.id ? 'Préparation…' : 'Télécharger'}
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => openSaleReceipt(order, product ? { title: product.title } : null)}
                              className="gap-1.5"
                            >
                              <FileIcon size={14} />
                              Reçu
                            </Button>
                          </>
                        ) : (
                          <p className="flex-1 text-xs text-yellow-700 bg-yellow-50 rounded-lg px-3 py-2">
                            {order.status === 'pending'
                              ? 'Validation de votre paiement en cours…'
                              : 'Fichier indisponible pour cette commande.'}
                          </p>
                        )}
                        <Button
                          size="sm"
                          variant="outline"
                          render={<Link href="/support" />}
                          className="gap-1.5"
                        >
                          <HeadsetIcon size={14} />
                          Aide
                        </Button>
                      </div>
                      {downloadError[order.id] && (
                        <p className="px-4 pb-3 -mt-2 text-xs text-red-600">{downloadError[order.id]}</p>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </>
        )}
      </section>
    </main>
  );
}
