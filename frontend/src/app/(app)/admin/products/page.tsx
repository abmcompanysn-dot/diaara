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
import { cn } from '@/lib/utils';

interface Product {
  id: string;
  title: string;
  description: string;
  price_cfa: number;
  category: string;
  vendor_id: string;
  vendor_email?: string;
  moderation_status: string;
  moderation_note?: string | null;
  created_at: string;
}

type Tab = 'pending' | 'approved' | 'rejected' | 'all';

const TABS: { key: Tab; label: string }[] = [
  { key: 'pending', label: 'En attente' },
  { key: 'approved', label: 'Approuvés' },
  { key: 'rejected', label: 'Refusés' },
  { key: 'all', label: 'Tous' },
];

export default function AdminProductsPage() {
  const [tab, setTab] = useState<Tab>('pending');
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toModerate, setToModerate] = useState<{ id: string; status: string } | null>(null);
  const [rejectNote, setRejectNote] = useState('');

  useEffect(() => {
    loadProducts(tab);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab]);

  const loadProducts = async (t: Tab) => {
    setLoading(true);
    try {
      const result = await api.getAdminProducts(t === 'all' ? undefined : t);
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
      // Le produit change de statut : il ne fait peut-être plus partie de l'onglet actif.
      await loadProducts(tab);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToModerate(null);
      setRejectNote('');
    }
  };

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Modération"
        description="Produits soumis par les vendeurs"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            ← Dashboard
          </Button>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        <div className="flex gap-1.5 mb-6 border-b border-green-900/10 overflow-x-auto">
          {TABS.map((t) => (
            <button
              key={t.key}
              type="button"
              onClick={() => setTab(t.key)}
              className={cn(
                'px-4 py-2.5 text-sm font-medium whitespace-nowrap border-b-2 -mb-px transition-colors',
                tab === t.key
                  ? 'border-lime text-green-950'
                  : 'border-transparent text-green-900/50 hover:text-green-900'
              )}
            >
              {t.label}
            </button>
          ))}
        </div>

        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {loading ? (
          <PageLoader />
        ) : products.length === 0 ? (
          <EmptyState
            title="Aucun produit"
            description={tab === 'pending' ? 'Toutes les soumissions ont été traitées.' : 'Rien à afficher dans cet onglet.'}
          />
        ) : (
          <div className="space-y-4">
            {products.map((product) => (
              <div key={product.id} className="p-6 border rounded-xl bg-white shadow-card border-green-900/10">
                <div className="flex justify-between items-start mb-2 gap-3">
                  <h2 className="font-display font-bold text-lg text-green-950">{product.title}</h2>
                  <div className="flex items-center gap-2 shrink-0">
                    <Badge className={PRODUCT_STATUS_BADGE[product.moderation_status]}>
                      {PRODUCT_STATUS_LABELS[product.moderation_status] || product.moderation_status}
                    </Badge>
                    <span className="font-mono font-bold text-green-600">
                      {product.price_cfa.toLocaleString('fr-FR')} FCFA
                    </span>
                  </div>
                </div>
                <p className="text-green-900/70 text-sm mb-4 leading-relaxed">
                  {product.description || 'Pas de description'}
                </p>
                {product.moderation_status === 'rejected' && product.moderation_note && (
                  <p className="text-sm text-red-600 mb-4">Raison du refus : {product.moderation_note}</p>
                )}
                <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground mb-6">
                  <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                    {CATEGORY_LABELS[product.category] || product.category}
                  </Badge>
                  <span>Vendeur : {product.vendor_email || product.vendor_id.slice(0, 8) + '...'}</span>
                  <span>{new Date(product.created_at).toLocaleDateString('fr-FR')}</span>
                </div>
                <div className="flex gap-3">
                  {product.moderation_status !== 'approved' && (
                    <Button
                      onClick={() => setToModerate({ id: product.id, status: 'approved' })}
                      className="bg-green-600 text-white hover:bg-green-500"
                    >
                      Approuver
                    </Button>
                  )}
                  {product.moderation_status !== 'rejected' && (
                    <Button
                      variant="destructive"
                      onClick={() => setToModerate({ id: product.id, status: 'rejected' })}
                    >
                      Refuser
                    </Button>
                  )}
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
