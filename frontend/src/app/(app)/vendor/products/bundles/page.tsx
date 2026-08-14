'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ArrowLeftIcon, TrashIcon } from '@/components/icons';
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

  const handleDelete = async (id: string) => {
    if (!confirm('Supprimer ce pack ?')) return;
    setDeletingId(id);
    try {
      await api.deleteBundle(id);
      load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setDeletingId(null);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// espace vendeur" title="Mes packs" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// espace vendeur"
        title="Mes packs"
        description={`${bundles.length} pack(s)`}
        actions={
          <div className="flex gap-2">
            <Button variant="outline" size="sm" render={<Link href="/vendor/products" />}>
              <ArrowLeftIcon size={16} className="mr-2" />
              Mes produits
            </Button>
            <Button size="sm" render={<Link href="/vendor/products/bundles/new" />}>
              Créer un pack
            </Button>
          </div>
        }
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-10 space-y-3">
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
            <Card key={bundle.id} className="border-green-900/5">
              <CardContent className="p-4 flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <p className="font-medium text-green-950 truncate">{bundle.title}</p>
                  <p className="text-xs font-mono text-green-900/50 mt-0.5">{formatPrice(bundle.price_cfa)}</p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={deletingId === bundle.id}
                  onClick={() => handleDelete(bundle.id)}
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
    </main>
  );
}
