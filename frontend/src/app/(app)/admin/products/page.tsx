'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { Textarea } from '@/components/ui/textarea';
import { PRODUCT_STATUS_BADGE, PRODUCT_STATUS_LABELS, CATEGORY_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface Product {
  id: string;
  title: string;
  description: string;
  price_cfa: number;
  category: string;
  vendor_id: string;
  moderation_status: string;
  created_at: string;
}

export default function AdminProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toModerate, setToModerate] = useState<{id: string, status: string} | null>(null);
  const [rejectNote, setRejectNote] = useState('');

  useEffect(() => {
    loadProducts();
  }, []);

  const loadProducts = async () => {
    setLoading(true);
    try {
      const result = await api.getPendingProducts();
      setProducts(result.products);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleModerate = async () => {
    if (!toModerate) return;
    try {
      await api.moderateProduct(toModerate.id, {
        status: toModerate.status,
        ...(toModerate.status === 'rejected' && rejectNote.trim() ? { note: rejectNote.trim() } : {}),
      });
      setProducts(products.filter((p) => p.id !== toModerate.id));
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToModerate(null);
      setRejectNote('');
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader
          eyebrow="// administration"
          title="Modération"
          description="Produits en attente de validation"
        />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Modération"
        description="Produits en attente de validation"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            ← Dashboard
          </Button>
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
            title="Aucun produit en attente"
            description="Toutes les soumissions ont été traitées."
          />
        ) : (
          <div className="space-y-4">
            {products.map((product) => (
              <div key={product.id} className="p-6 border rounded-xl bg-white shadow-card border-green-900/10">
                <div className="flex justify-between items-start mb-2">
                  <h2 className="font-display font-bold text-lg text-green-950">{product.title}</h2>
                  <span className="font-mono font-bold text-green-600">
                    {product.price_cfa.toLocaleString('fr-FR')} FCFA
                  </span>
                </div>
                <p className="text-green-900/70 text-sm mb-4 leading-relaxed">
                  {product.description || 'Pas de description'}
                </p>
                <div className="flex items-center gap-4 text-xs text-muted-foreground mb-6">
                  <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                    {CATEGORY_LABELS[product.category] || product.category}
                  </Badge>
                  <span className="flex items-center gap-1">
                    Vendeur : <span className="font-mono">{product.vendor_id.slice(0, 8)}...</span>
                  </span>
                  <span>{new Date(product.created_at).toLocaleDateString('fr-FR')}</span>
                </div>
                <div className="flex gap-3">
                  <Button
                    onClick={() => setToModerate({ id: product.id, status: 'approved' })}
                    className="bg-green-600 text-white hover:bg-green-500"
                  >
                    Approuver
                  </Button>
                  <Button
                    variant="destructive"
                    onClick={() => setToModerate({ id: product.id, status: 'rejected' })}
                  >
                    Refuser
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <ConfirmDialog
        open={!!toModerate}
        title={toModerate?.status === 'approved' ? "Approuver ce produit ?" : "Refuser ce produit ?"}
        description={toModerate?.status === 'approved' 
          ? "Le produit sera publié et deviendra visible pour tous les acheteurs." 
          : "Le produit sera marqué comme refusé et le vendeur sera notifié."}
        confirmLabel={toModerate?.status === 'approved' ? "Approuver" : "Refuser"}
        cancelLabel="Annuler"
        danger={toModerate?.status === 'rejected'}
        onConfirm={handleModerate}
        onCancel={() => {
          setToModerate(null);
          setRejectNote('');
        }}
      >
        {toModerate?.status === 'rejected' && (
          <div className="space-y-1.5">
            <label htmlFor="reject-note" className="text-xs font-medium text-green-900/70">
              Raison du refus (envoyée au vendeur)
            </label>
            <Textarea
              id="reject-note"
              value={rejectNote}
              onChange={(e) => setRejectNote(e.target.value)}
              placeholder="Ex : image de couverture floue, description incomplète..."
              className="min-h-20 text-sm"
            />
          </div>
        )}
      </ConfirmDialog>
    </main>
  );
}
