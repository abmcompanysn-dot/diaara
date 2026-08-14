'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { TrashIcon } from '@/components/icons';
import { formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface Bundle {
  id: string;
  title: string;
  price_cfa: number;
  created_at: string;
}

export default function VendorBundlesPage() {
  const [bundles, setBundles] = useState<Bundle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [toDelete, setToDelete] = useState<Bundle | null>(null);

  const load = () => {
    setLoading(true);
    api
      .getVendorBundles()
      .then((r) => setBundles(r.bundles))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const handleDelete = async () => {
    if (!toDelete) return;
    setDeletingId(toDelete.id);
    try {
      await api.deleteBundle(toDelete.id);
      load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setDeletingId(null);
      setToDelete(null);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader back="/vendor" eyebrow="// espace vendeur" title="Mes packs" />
        <PageLoader />
      </main>
    );

  return (
    <main className="relative min-h-screen pb-20">
      <PageHeader
        back="/vendor"
        eyebrow="// espace vendeur"
        title="Mes packs"
        description="Regroupez vos produits déjà publiés"
        actions={
          <Button size="sm" render={<Link href="/vendor/products/bundles/new" />}>
            Créer un pack
          </Button>
        }
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-6 space-y-3">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {bundles.length === 0 ? (
          <EmptyState
            title="Aucun pack pour le moment"
            description="Regroupez plusieurs de vos produits déjà publiés en un pack vendu à un prix unique."
            action={<Button render={<Link href="/vendor/products/bundles/new" />}>Créer un pack</Button>}
          />
        ) : (
          bundles.map((bundle) => (
            <Card key={bundle.id} className="border-green-900/10 rounded-2xl">
              <CardContent className="p-4 flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <p className="font-bold text-green-950 truncate">{bundle.title}</p>
                  <p className="text-xs font-mono text-green-900/50 mt-1">{formatPrice(bundle.price_cfa)}</p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={deletingId === bundle.id}
                  onClick={() => setToDelete(bundle)}
                  className="text-destructive hover:text-destructive gap-1.5"
                >
                  <TrashIcon size={14} />
                  Supprimer
                </Button>
              </CardContent>
            </Card>
          ))
        )}
      </section>

      <ConfirmDialog
        open={!!toDelete}
        title="Supprimer ce pack ?"
        description={`« ${toDelete?.title} » sera retiré de la boutique. Cette action est définitive.`}
        confirmLabel="Supprimer"
        cancelLabel="Annuler"
        danger
        onConfirm={handleDelete}
        onCancel={() => setToDelete(null)}
      />

      <Link
        href="/vendor/products/bundles/new"
        aria-label="Nouveau pack"
        className="sm:hidden fixed right-5 bottom-6 z-40 rounded-2xl bg-lime shadow-lg flex items-center justify-center"
        style={{ width: 52, height: 52, boxShadow: '0 10px 22px -6px rgba(201,242,46,.55)' }}
      >
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
          <path d="M12 5v14M5 12h14" stroke="#071F17" strokeWidth="2.4" strokeLinecap="round" />
        </svg>
      </Link>
    </main>
  );
}
