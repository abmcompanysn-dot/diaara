'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { LinkIcon } from '@/components/icons';
import { CATEGORY_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface Product {
  id: string;
  title: string;
  category: string;
  affiliate_enabled: boolean;
  max_closer_commission_pct: number;
}

export default function VendorAffiliationPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [savingId, setSavingId] = useState<string | null>(null);

  useEffect(() => {
    api
      .getVendorProducts()
      .then((r) => setProducts(r.products))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, []);

  const toggle = async (product: Product) => {
    setSavingId(product.id);
    const nextEnabled = !product.affiliate_enabled;
    try {
      await api.updateProduct(product.id, {
        affiliate_enabled: nextEnabled,
        // Un plafond minimum sensé si on active l'affiliation sans commission déjà réglée.
        max_closer_commission_pct: nextEnabled ? product.max_closer_commission_pct || 10 : 0,
      });
      setProducts((list) =>
        list.map((p) =>
          p.id === product.id
            ? { ...p, affiliate_enabled: nextEnabled, max_closer_commission_pct: nextEnabled ? p.max_closer_commission_pct || 10 : 0 }
            : p
        )
      );
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSavingId(null);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader back="/vendor" eyebrow="// espace vendeur" title="Affiliation" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        back="/vendor"
        eyebrow="// espace vendeur"
        title="Affiliation"
        description="Autorisez des affiliés (closers) à promouvoir vos produits par produit"
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-6">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {products.length === 0 ? (
          <EmptyState
            icon={LinkIcon}
            title="Aucun produit"
            description="Ajoutez d'abord un produit pour pouvoir activer l'affiliation dessus."
          />
        ) : (
          <div className="space-y-3">
            {products.map((product) => (
              <div
                key={product.id}
                className="bg-white rounded-2xl border border-green-900/10 shadow-card p-3.5 flex items-center gap-3"
              >
                <div className="min-w-0 flex-1">
                  <p className="text-[13.5px] font-bold text-green-950 truncate">{product.title}</p>
                  <p className="text-xs text-green-900/50 mt-0.5">
                    {CATEGORY_LABELS[product.category] || product.category} · Commission plafonnée
                  </p>
                  <p className="font-mono text-xs font-bold text-green-700 mt-1.5">
                    Plafond : {product.max_closer_commission_pct}%
                  </p>
                </div>
                <button
                  type="button"
                  disabled={savingId === product.id}
                  onClick={() => toggle(product)}
                  className={`shrink-0 text-[10.5px] font-extrabold px-3 py-1.5 rounded-full font-mono disabled:opacity-50 ${
                    product.affiliate_enabled ? 'bg-[#EDF8F2] text-[#0E6B46]' : 'bg-[#FFF3D6] text-[#8A6300]'
                  }`}
                >
                  {product.affiliate_enabled ? 'Actif' : 'Désactivé'}
                </button>
              </div>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
