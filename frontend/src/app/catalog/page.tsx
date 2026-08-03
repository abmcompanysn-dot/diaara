'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ProductImage } from '@/components/product-image';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

interface Product {
  id: string;
  title: string;
  description: string;
  price_cfa: number;
  category: string;
  moderation_status: string;
  cover_image_key?: string;
}

const CATEGORY_LABELS: Record<string, string> = {
  subscription: 'Clés d\'abonnement',
  account: 'Comptes',
  ebook: 'Ebooks',
  pdf: 'PDF',
  other: 'Autres',
};

export default function CatalogPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadProducts();
  }, [category]);

  const loadProducts = async () => {
    setLoading(true);
    try {
      const params: { search?: string; category?: string } = {};
      if (search) params.search = search;
      if (category) params.category = category;
      const result = await api.getProducts(params);
      setProducts(result.products);
    } catch (err) {
      console.error('Failed to load products', err);
    } finally {
      setLoading(false);
    }
  };

  const formatPrice = (price: number) => `${price.toLocaleString()} FCFA`;

  return (
    <main className="min-h-screen">
      {/* En-tête du catalogue */}
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-6xl mx-auto px-4 py-16">
          <p className="font-mono text-sm text-green-300 mb-3 uppercase tracking-widest">
            // le catalogue
          </p>
          <h1 className="font-display text-4xl sm:text-5xl font-bold tracking-tight">
            Produits numériques
          </h1>
          <p className="mt-3 text-white/75 max-w-lg">
            Clés d&apos;abonnement, comptes, ebooks et PDF. Paiement mobile money,
            livraison instantanée.
          </p>

          <div className="mt-8 flex flex-col sm:flex-row gap-3 max-w-2xl">
            <Input
              type="text"
              placeholder="Rechercher un produit..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && loadProducts()}
              className="flex-1 rounded-full bg-white text-green-950 placeholder-green-900/40 border border-transparent focus:border-lime focus:shadow-glow outline-none"
            />
            <Select
              value={category || 'all'}
              onValueChange={(v) => setCategory(v === 'all' ? '' : (v || ''))}
            >
              <SelectTrigger className="rounded-full bg-white text-green-950 hover:bg-white">
                <SelectValue placeholder="Toutes catégories" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Toutes catégories</SelectItem>
                {Object.entries(CATEGORY_LABELS).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              onClick={loadProducts}
              className="px-6 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300"
            >
              Rechercher
            </Button>
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-4 py-12">
        {loading ? (
          <p className="text-center text-green-900/50 py-16">Chargement...</p>
        ) : products.length === 0 ? (
          <div className="text-center py-16">
            <p className="text-green-900/60 mb-2">Aucun produit trouvé.</p>
            <p className="text-sm text-green-900/40">
              Modifiez votre recherche ou parcourez une autre catégorie.
            </p>
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {products.map((product) => (
              <Link
                key={product.id}
                href={`/product?id=${product.id}`}
                className="group bg-white rounded-xl overflow-hidden shadow-card hover:shadow-lift transition-all hover:-translate-y-0.5 border border-green-900/5"
              >
                <ProductImage product={product} className="h-44" />
                <div className="p-5">
                  <div className="flex justify-between items-start mb-2">
                    <span className="px-2 py-1 bg-green-50 text-green-700 rounded text-xs font-medium">
                      {CATEGORY_LABELS[product.category] || product.category}
                    </span>
                    <span className="font-mono font-bold text-green-600">
                      {formatPrice(product.price_cfa)}
                    </span>
                  </div>
                  <h2 className="font-display font-bold text-lg text-green-950 group-hover:text-green-600 transition-colors">
                    {product.title}
                  </h2>
                  <p className="text-sm text-green-900/60 mt-1 line-clamp-2">
                    {product.description || 'Pas de description'}
                  </p>
                  <p className="mt-4 font-mono text-xs text-green-900/40 group-hover:text-green-600 transition-colors">
                    Voir le produit →
                  </p>
                </div>
              </Link>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
