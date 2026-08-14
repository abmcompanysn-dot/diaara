'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { ProductImage } from '@/components/product-image';
import { EmptyState } from '@/components/empty-state';
import { PageLoader } from '@/components/page-loader';
import { StoreIcon } from '@/components/icons';
import { CATEGORY_LABELS, formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface Product {
  id: string;
  title: string;
  description: string;
  price_cfa: number;
  price_mode?: string;
  min_price_cfa?: number | null;
  category: string;
  cover_image_key?: string;
}

interface Vendor {
  id: string;
  display_name: string | null;
  shop_name: string | null;
}

export default function BoutiqueView() {
  const searchParams = useSearchParams();
  const vendorId = searchParams.get('id') || '';

  const [vendor, setVendor] = useState<Vendor | null>(null);
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!vendorId) {
      setError('Boutique introuvable.');
      setLoading(false);
      return;
    }
    api
      .getVendorShop(vendorId)
      .then((r) => {
        setVendor(r.vendor);
        setProducts(r.products);
      })
      .catch((err) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, [vendorId]);

  if (loading)
    return (
      <main className="min-h-screen">
        <PageLoader />
      </main>
    );

  if (error || !vendor) {
    return (
      <main className="min-h-[calc(100vh-4rem)] flex items-center justify-center px-4">
        <EmptyState icon={StoreIcon} title="Boutique introuvable" description={error || "Cette boutique n'existe pas."} />
      </main>
    );
  }

  const shopTitle = vendor.shop_name || vendor.display_name || 'Boutique';

  return (
    <main className="min-h-screen">
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-6xl mx-auto px-4 py-10 sm:py-14 text-center">
          <span className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-white/15 mb-4">
            <StoreIcon size={28} />
          </span>
          <p className="font-mono text-sm text-green-300 mb-2 uppercase tracking-widest">// boutique DIARRA</p>
          <h1 className="font-display text-3xl sm:text-4xl font-bold">{shopTitle}</h1>
          <p className="mt-2 text-white/70 text-sm">{products.length} produit(s) disponible(s)</p>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        {products.length === 0 ? (
          <EmptyState
            icon={StoreIcon}
            title="Aucun produit pour le moment"
            description="Cette boutique n'a pas encore de produit publié."
          />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {products.map((product) => (
              <div
                key={product.id}
                className="bg-white rounded-xl overflow-hidden border border-green-900/5 shadow-card hover:shadow-lift transition-all"
              >
                <Link href={`/product?id=${product.id}`}>
                  <div className="h-40 overflow-hidden">
                    <ProductImage product={product} className="h-40" />
                  </div>
                </Link>
                <div className="p-4">
                  <span className="text-[11px] font-semibold text-green-700 uppercase tracking-wide">
                    {CATEGORY_LABELS[product.category] || product.category}
                  </span>
                  <Link href={`/product?id=${product.id}`}>
                    <h2 className="font-display font-bold text-green-950 line-clamp-1 mt-1 hover:text-green-600 transition-colors">
                      {product.title}
                    </h2>
                  </Link>
                  <p className="text-sm text-green-900/60 line-clamp-2 mt-1">
                    {product.description || 'Pas de description'}
                  </p>
                  <div className="flex items-center justify-between mt-3 gap-2">
                    <span className="font-mono font-bold text-green-700">
                      {product.price_mode === 'flexible'
                        ? `dès ${formatPrice(product.min_price_cfa || 0)}`
                        : formatPrice(product.price_cfa)}
                    </span>
                    <Button
                      render={<Link href={`/product?id=${product.id}`} />}
                      className="h-9 rounded-full bg-green-950 text-white hover:bg-green-900 px-3.5"
                    >
                      Voir
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
