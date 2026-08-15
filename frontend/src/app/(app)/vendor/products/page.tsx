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
import { ConfirmDialog } from '@/components/confirm-dialog';
import { ProductImage } from '@/components/product-image';
import { SegmentedTabs } from '@/components/segmented-tabs';
import { StoreIcon, TrashIcon, EditIcon } from '@/components/icons';
import { formatPrice, PRODUCT_STATUS_BADGE, PRODUCT_STATUS_LABELS, CATEGORY_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

type StatusFilter = 'all' | 'approved' | 'pending' | 'rejected';

const FILTERS: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'Tous' },
  { value: 'approved', label: 'Approuvés' },
  { value: 'pending', label: 'En attente' },
  { value: 'rejected', label: 'Refusés' },
];

interface Product {
  id: string;
  title: string;
  price_cfa: number;
  price_mode?: string;
  min_price_cfa?: number | null;
  category: string;
  cover_image_key?: string;
  moderation_status: string;
  moderation_note?: string | null;
  preview_status?: string;
  deletion_requested?: boolean;
}

export default function VendorProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toDelete, setToDelete] = useState<Product | null>(null);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');

  useEffect(() => {
    loadProducts();
  }, []);

  const loadProducts = async () => {
    setLoading(true);
    try {
      const result = await api.getVendorProducts();
      setProducts(result.products);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!toDelete) return;
    try {
      await api.deleteProduct(toDelete.id);
      setProducts(products.map((p) => (p.id === toDelete.id ? { ...p, deletion_requested: true } : p)));
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToDelete(null);
    }
  };

  const pendingCount = products.filter((p) => p.moderation_status === 'pending').length;
  const filteredProducts = statusFilter === 'all' ? products : products.filter((p) => p.moderation_status === statusFilter);

  if (loading)
    return (
      <main>
        <PageHeader
          back="/vendor"
          eyebrow="// espace vendeur"
          title="Mes produits"
          description="Gérez votre boutique"
        />
        <PageLoader />
      </main>
    );

  return (
    <main className="relative min-h-screen pb-20">
      <PageHeader
        back="/vendor"
        eyebrow="// espace vendeur"
        title="Mes produits"
        description={`${products.length} produit(s)${pendingCount > 0 ? ` · ${pendingCount} en attente de modération` : ''}`}
        actions={
          <>
            <Button variant="outline" size="sm" render={<Link href="/vendor/earnings" />}>
              Revenus
            </Button>
            <Button variant="outline" size="sm" render={<Link href="/vendor/sales" />}>
              Mes clients
            </Button>
            <Button variant="outline" size="sm" render={<Link href="/vendor/products/bundles" />}>
              Mes packs
            </Button>
            <Button variant="outline" size="sm" render={<Link href="/vendor/messages" />}>
              Messages
            </Button>
            <Button size="sm" render={<Link href="/vendor/products/new" />}>
              + Nouveau produit
            </Button>
          </>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-6">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {products.length > 0 && (
          <div className="mb-5">
            <SegmentedTabs options={FILTERS} value={statusFilter} onChange={setStatusFilter} />
          </div>
        )}

        {products.length === 0 ? (
          <EmptyState
            icon={StoreIcon}
            title="Aucun produit pour le moment"
            description="Déposez votre premier fichier pour ouvrir votre boutique."
            action={
              <Button render={<Link href="/vendor/products/new" />}>Déposer mon premier produit</Button>
            }
          />
        ) : filteredProducts.length === 0 ? (
          <p className="text-sm text-green-900/50 text-center py-10">Aucun produit dans ce filtre.</p>
        ) : (
          <>
            {/* Vue mobile : cartes empilées, pas de scroll horizontal */}
            <div className="sm:hidden space-y-2">
              {filteredProducts.map((product) => (
                <div key={product.id} className="bg-white rounded-xl border border-green-900/10 shadow-card p-3.5">
                  <Link href={`/product?id=${product.id}`} className="flex items-center gap-3 group min-w-0">
                    <div className="w-12 h-12 rounded-lg overflow-hidden shrink-0 border border-green-900/10">
                      <ProductImage product={product} className="h-12" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="font-medium text-primary group-hover:underline line-clamp-2 text-sm">
                        {product.title}
                      </p>
                      <p className="text-xs text-green-900/50">{CATEGORY_LABELS[product.category] || product.category}</p>
                    </div>
                    <div className="text-right shrink-0 font-mono text-sm">
                      {product.price_mode === 'flexible' ? (
                        <p className="text-xs text-green-900/70">dès {formatPrice(product.min_price_cfa || 0)}</p>
                      ) : (
                        formatPrice(product.price_cfa)
                      )}
                    </div>
                  </Link>
                  <div className="flex items-center justify-between gap-2 mt-3 pt-3 border-t border-green-900/5">
                    <div>
                      <Badge className={PRODUCT_STATUS_BADGE[product.moderation_status]}>
                        {PRODUCT_STATUS_LABELS[product.moderation_status] || product.moderation_status}
                      </Badge>
                      {product.deletion_requested && (
                        <Badge className="ml-1 bg-amber-100 text-amber-800 hover:bg-amber-100">
                          Suppression demandée
                        </Badge>
                      )}
                      {product.preview_status === 'pending' && (
                        <p className="text-[11px] text-green-900/40 mt-1">Aperçu en cours…</p>
                      )}
                      {product.moderation_status === 'rejected' && product.moderation_note && (
                        <p className="text-[11px] text-red-600 mt-1 max-w-[200px]">{product.moderation_note}</p>
                      )}
                    </div>
                    <div className="flex gap-1 shrink-0">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="min-h-9"
                        render={<Link href={`/vendor/products/edit?id=${product.id}`} />}
                      >
                        <EditIcon size={16} />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={product.deletion_requested}
                        className="text-destructive hover:text-destructive hover:bg-destructive/10 min-h-9 disabled:opacity-40"
                        onClick={() => setToDelete(product)}
                      >
                        <TrashIcon size={16} />
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* Vue desktop : tableau */}
            <div className="hidden sm:block rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Produit</TableHead>
                    <TableHead>Prix</TableHead>
                    <TableHead>Catégorie</TableHead>
                    <TableHead>Statut</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredProducts.map((product) => (
                    <TableRow key={product.id}>
                      <TableCell>
                        <Link href={`/product?id=${product.id}`} className="flex items-center gap-3 group min-w-0">
                          <div className="w-12 h-12 rounded-lg overflow-hidden shrink-0 border border-green-900/10">
                            <ProductImage product={product} className="h-12" />
                          </div>
                          <span className="font-medium text-primary group-hover:underline line-clamp-2 min-w-0">
                            {product.title}
                          </span>
                        </Link>
                      </TableCell>
                      <TableCell className="font-mono whitespace-nowrap">
                        {product.price_mode === 'flexible' ? (
                          <div>
                            <Badge className="bg-lime/20 text-green-800 hover:bg-lime/20 mb-1">Prix libre</Badge>
                            <p className="text-xs text-green-900/50">dès {formatPrice(product.min_price_cfa || 0)}</p>
                          </div>
                        ) : (
                          formatPrice(product.price_cfa)
                        )}
                      </TableCell>
                      <TableCell>{CATEGORY_LABELS[product.category] || product.category}</TableCell>
                      <TableCell>
                        <Badge className={PRODUCT_STATUS_BADGE[product.moderation_status]}>
                          {PRODUCT_STATUS_LABELS[product.moderation_status] || product.moderation_status}
                        </Badge>
                        {product.deletion_requested && (
                          <Badge className="ml-1 bg-amber-100 text-amber-800 hover:bg-amber-100">
                            Suppression demandée
                          </Badge>
                        )}
                        {product.preview_status === 'pending' && (
                          <p className="text-[11px] text-green-900/40 mt-1">Aperçu en cours…</p>
                        )}
                        {product.moderation_status === 'rejected' && product.moderation_note && (
                          <p className="text-[11px] text-red-600 mt-1 max-w-[200px]">{product.moderation_note}</p>
                        )}
                      </TableCell>
                      <TableCell className="text-right space-x-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="min-h-9"
                          render={<Link href={`/vendor/products/edit?id=${product.id}`} />}
                        >
                          <EditIcon size={16} className="mr-1.5" />
                          Modifier
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={product.deletion_requested}
                          className="text-destructive hover:text-destructive hover:bg-destructive/10 min-h-9 disabled:opacity-40"
                          onClick={() => setToDelete(product)}
                        >
                          <TrashIcon size={16} className="mr-1.5" />
                          Supprimer
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </>
        )}
      </section>

      <ConfirmDialog
        open={!!toDelete}
        title="Demander la suppression ?"
        description={`« ${toDelete?.title} » sera retiré de la boutique une fois la suppression confirmée par un admin.`}
        confirmLabel="Demander la suppression"
        cancelLabel="Annuler"
        danger
        onConfirm={handleDelete}
        onCancel={() => setToDelete(null)}
      />

      <Link
        href="/vendor/products/new"
        aria-label="Nouveau produit"
        className="sm:hidden fixed right-5 bottom-6 z-40 w-13 h-13 rounded-2xl bg-lime shadow-lg flex items-center justify-center"
        style={{ width: 52, height: 52, boxShadow: '0 10px 22px -6px rgba(201,242,46,.55)' }}
      >
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
          <path d="M12 5v14M5 12h14" stroke="#071F17" strokeWidth="2.4" strokeLinecap="round" />
        </svg>
      </Link>
    </main>
  );
}