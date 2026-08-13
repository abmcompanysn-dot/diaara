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
import { StoreIcon, TrashIcon, EditIcon } from '@/components/icons';
import { formatPrice, PRODUCT_STATUS_BADGE, PRODUCT_STATUS_LABELS, CATEGORY_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface Product {
  id: string;
  title: string;
  price_cfa: number;
  category: string;
  moderation_status: string;
}

export default function VendorProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toDelete, setToDelete] = useState<Product | null>(null);

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
      setProducts(products.filter((p) => p.id !== toDelete.id));
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToDelete(null);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader
          eyebrow="// espace vendeur"
          title="Mes produits"
          description="Gérez votre boutique"
        />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// espace vendeur"
        title="Mes produits"
        description={`${products.length} produit(s) en boutique`}
        actions={
          <>
            <Button variant="outline" size="sm" render={<Link href="/vendor/earnings" />}>
              Revenus
            </Button>
            <Button size="sm" render={<Link href="/vendor/products/new" />}>
              + Nouveau produit
            </Button>
          </>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
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
        ) : (
          <div className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden">
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
                {products.map((product) => (
                  <TableRow key={product.id}>
                    <TableCell>
                      <Link href={`/product?id=${product.id}`} className="font-medium text-primary">
                        {product.title}
                      </Link>
                    </TableCell>
                    <TableCell className="font-mono">{formatPrice(product.price_cfa)}</TableCell>
                    <TableCell>{CATEGORY_LABELS[product.category] || product.category}</TableCell>
                    <TableCell>
                      <Badge className={PRODUCT_STATUS_BADGE[product.moderation_status]}>
                        {PRODUCT_STATUS_LABELS[product.moderation_status] || product.moderation_status}
                      </Badge>
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
                        className="text-destructive hover:text-destructive hover:bg-destructive/10 min-h-9"
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
        )}
      </section>

      <ConfirmDialog
        open={!!toDelete}
        title="Supprimer ce produit ?"
        description={`« ${toDelete?.title} » sera retiré de la boutique. Cette action est définitive.`}
        confirmLabel="Supprimer"
        cancelLabel="Annuler"
        danger
        onConfirm={handleDelete}
        onCancel={() => setToDelete(null)}
      />
    </main>
  );
}