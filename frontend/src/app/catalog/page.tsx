'use client';

import { useEffect, useMemo, useState, useRef } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { ProductImage } from '@/components/product-image';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { EmptyState } from '@/components/empty-state';
import {
  CartIcon,
  BookIcon,
  MonitorIcon,
  KeyIcon,
  UserIcon,
  FileIcon,
  BriefcaseIcon,
  SearchIcon,
  XIcon,
  GridIcon,
  ListIcon,
} from '@/components/icons';
import { CATEGORY_LABELS, formatPrice } from '@/lib/constants';
import { cn } from '@/lib/utils';

interface Product {
  id: string;
  title: string;
  description: string;
  price_cfa: number;
  category: string;
  moderation_status: string;
  cover_image_key?: string;
  created_at?: string;
}

// Icônes par catégorie, pour les chips de filtre et les badges des cartes.
const CATEGORY_ICONS: Record<string, typeof CartIcon> = {
  subscription: KeyIcon,
  account: UserIcon,
  ebook: BookIcon,
  pdf: FileIcon,
  software: MonitorIcon,
  service: BriefcaseIcon,
  other: CartIcon,
};

type SortKey = 'recent' | 'price_asc' | 'price_desc';
type ViewMode = 'grid' | 'list';

const SORT_LABELS: Record<SortKey, string> = {
  recent: 'Nouveautés',
  price_asc: 'Prix : croissant',
  price_desc: 'Prix : décroissant',
};

const NEW_THRESHOLD_DAYS = 7;

function isNew(product: Product): boolean {
  if (!product.created_at) return false;
  const ageMs = Date.now() - new Date(product.created_at).getTime();
  return ageMs < NEW_THRESHOLD_DAYS * 24 * 60 * 60 * 1000;
}

function CardSkeleton({ view }: { view: ViewMode }) {
  if (view === 'list') {
    return (
      <div className="flex gap-4 bg-white rounded-xl p-3 border border-green-900/5 animate-pulse">
        <div className="w-24 h-24 rounded-lg bg-green-900/10 shrink-0" />
        <div className="flex-1 space-y-2 py-1">
          <div className="h-3 w-16 bg-green-900/10 rounded" />
          <div className="h-4 w-3/4 bg-green-900/10 rounded" />
          <div className="h-4 w-1/3 bg-green-900/10 rounded" />
        </div>
      </div>
    );
  }
  return (
    <div className="bg-white rounded-xl overflow-hidden border border-green-900/5 animate-pulse">
      <div className="h-40 bg-green-900/10" />
      <div className="p-4 space-y-2">
        <div className="h-3 w-16 bg-green-900/10 rounded" />
        <div className="h-4 w-4/5 bg-green-900/10 rounded" />
        <div className="h-4 w-1/2 bg-green-900/10 rounded" />
        <div className="h-8 w-full bg-green-900/10 rounded-lg mt-3" />
      </div>
    </div>
  );
}

export default function CatalogPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState('');
  const [sort, setSort] = useState<SortKey>('recent');
  const [view, setView] = useState<ViewMode>('grid');
  const [loading, setLoading] = useState(true);
  const debounceRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    loadProducts();
  }, [category]);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      loadProducts();
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [search]);

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

  const sortedProducts = useMemo(() => {
    const list = [...products];
    if (sort === 'price_asc') list.sort((a, b) => a.price_cfa - b.price_cfa);
    else if (sort === 'price_desc') list.sort((a, b) => b.price_cfa - a.price_cfa);
    else
      list.sort(
        (a, b) => new Date(b.created_at || 0).getTime() - new Date(a.created_at || 0).getTime()
      );
    return list;
  }, [products, sort]);

  return (
    <main className="min-h-screen">
      {/* En-tête du catalogue */}
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-6xl mx-auto px-4 py-10 sm:py-16">
          <p className="font-mono text-sm text-green-300 mb-3 uppercase tracking-widest">
            // le catalogue
          </p>
          <h1 className="font-display text-3xl sm:text-5xl font-bold tracking-tight">
            Produits numériques
          </h1>
          <p className="mt-3 text-white/75 max-w-lg text-sm sm:text-base">
            Clés d&apos;abonnement, comptes, ebooks et PDF. Paiement mobile money,
            livraison instantanée.
          </p>

          {/* Barre de recherche */}
          <div className="mt-8 relative max-w-2xl">
            <SearchIcon
              size={18}
              className="absolute left-4 top-1/2 -translate-y-1/2 text-green-900/40 pointer-events-none"
            />
            <input
              type="text"
              placeholder="Rechercher un ebook, script, formation..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full h-12 pl-11 pr-11 rounded-full bg-white text-green-950 placeholder-green-900/40 border border-transparent focus:border-lime focus:shadow-glow outline-none text-sm"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearch('')}
                aria-label="Effacer la recherche"
                className="absolute right-3 top-1/2 -translate-y-1/2 w-7 h-7 rounded-full flex items-center justify-center text-green-900/40 hover:text-green-900 hover:bg-green-900/5 transition-colors"
              >
                <XIcon size={16} />
              </button>
            )}
          </div>

          {/* Chips de catégories, défilement horizontal */}
          <div className="mt-4 flex gap-2 overflow-x-auto no-scrollbar -mx-4 px-4 pb-1">
            <button
              type="button"
              onClick={() => setCategory('')}
              className={cn(
                'shrink-0 inline-flex items-center gap-1.5 px-3.5 py-2 rounded-full text-sm font-medium transition-colors border',
                category === ''
                  ? 'bg-lime text-green-950 border-lime'
                  : 'bg-white/10 text-white/80 border-white/15 hover:bg-white/15'
              )}
            >
              🔥 Tout
            </button>
            {Object.entries(CATEGORY_LABELS).map(([value, label]) => {
              const Icon = CATEGORY_ICONS[value] || CartIcon;
              const active = category === value;
              return (
                <button
                  key={value}
                  type="button"
                  onClick={() => setCategory(value)}
                  className={cn(
                    'shrink-0 inline-flex items-center gap-1.5 px-3.5 py-2 rounded-full text-sm font-medium transition-colors border',
                    active
                      ? 'bg-lime text-green-950 border-lime'
                      : 'bg-white/10 text-white/80 border-white/15 hover:bg-white/15'
                  )}
                >
                  <Icon size={14} className="shrink-0" />
                  {label}
                </button>
              );
            })}
          </div>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-4 py-8 sm:py-12">
        {/* Barre de résultats : compte, tri, disposition */}
        <div className="flex items-center justify-between gap-3 mb-5">
          <p className="text-sm text-green-900/60">
            {loading ? 'Recherche…' : `Résultats (${sortedProducts.length})`}
          </p>
          <div className="flex items-center gap-2">
            <Select value={sort} onValueChange={(v) => setSort((v as SortKey) || 'recent')}>
              <SelectTrigger className="h-9 rounded-full bg-white border-green-900/10 text-xs sm:text-sm">
                <SelectValue placeholder="Trier" />
              </SelectTrigger>
              <SelectContent>
                {(Object.keys(SORT_LABELS) as SortKey[]).map((key) => (
                  <SelectItem key={key} value={key}>
                    {SORT_LABELS[key]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <div className="hidden sm:flex items-center gap-1 bg-white rounded-full border border-green-900/10 p-1">
              <button
                type="button"
                onClick={() => setView('grid')}
                aria-label="Vue grille"
                aria-pressed={view === 'grid'}
                className={cn(
                  'w-8 h-8 rounded-full flex items-center justify-center transition-colors',
                  view === 'grid' ? 'bg-green-950 text-white' : 'text-green-900/40 hover:text-green-900'
                )}
              >
                <GridIcon size={16} />
              </button>
              <button
                type="button"
                onClick={() => setView('list')}
                aria-label="Vue liste"
                aria-pressed={view === 'list'}
                className={cn(
                  'w-8 h-8 rounded-full flex items-center justify-center transition-colors',
                  view === 'list' ? 'bg-green-950 text-white' : 'text-green-900/40 hover:text-green-900'
                )}
              >
                <ListIcon size={16} />
              </button>
            </div>
          </div>
        </div>

        {loading ? (
          <div
            className={cn(
              view === 'grid' ? 'grid gap-4 sm:grid-cols-2 lg:grid-cols-3' : 'flex flex-col gap-3'
            )}
          >
            {Array.from({ length: 6 }).map((_, i) => (
              <CardSkeleton key={i} view={view} />
            ))}
          </div>
        ) : sortedProducts.length === 0 ? (
          <EmptyState
            icon={CartIcon}
            title="Aucun produit trouvé"
            description="Modifiez votre recherche ou parcourez une autre catégorie."
          />
        ) : view === 'grid' ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {sortedProducts.map((product) => (
              <div
                key={product.id}
                className="group bg-white rounded-xl overflow-hidden shadow-card hover:shadow-lift transition-all hover:-translate-y-0.5 border border-green-900/5 flex flex-col"
              >
                <Link href={`/product?id=${product.id}`} className="relative block">
                  <ProductImage product={product} className="h-40 sm:h-44" />
                  <span className="absolute top-2.5 left-2.5 px-2 py-1 bg-white/95 text-green-700 rounded-md text-[11px] font-semibold shadow-sm">
                    {CATEGORY_LABELS[product.category] || product.category}
                  </span>
                  {isNew(product) && (
                    <span className="absolute top-2.5 right-2.5 px-2 py-1 bg-lime text-green-950 rounded-md text-[11px] font-bold shadow-sm">
                      Nouveau
                    </span>
                  )}
                </Link>
                <div className="p-4 flex flex-col flex-1">
                  <Link href={`/product?id=${product.id}`}>
                    <h2 className="font-display font-bold text-base text-green-950 group-hover:text-green-600 transition-colors line-clamp-2">
                      {product.title}
                    </h2>
                  </Link>
                  <p className="text-xs text-green-900/50 mt-1 line-clamp-2 flex-1">
                    {product.description || 'Pas de description'}
                  </p>
                  <div className="flex items-center justify-between mt-3 gap-2">
                    <span className="font-mono font-bold text-green-700">
                      {formatPrice(product.price_cfa)}
                    </span>
                    <Button
                      size="sm"
                      render={<Link href={`/product?id=${product.id}`} />}
                      className="rounded-full bg-green-950 text-white hover:bg-green-900 px-3.5"
                    >
                      Voir
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {sortedProducts.map((product) => (
              <Link
                key={product.id}
                href={`/product?id=${product.id}`}
                className="group flex gap-4 bg-white rounded-xl p-3 shadow-card hover:shadow-lift transition-all border border-green-900/5"
              >
                <div className="relative w-24 h-24 sm:w-28 sm:h-28 shrink-0 rounded-lg overflow-hidden">
                  <ProductImage product={product} className="h-full" />
                </div>
                <div className="flex-1 min-w-0 flex flex-col py-0.5">
                  <span className="text-[11px] font-semibold text-green-700 uppercase tracking-wide">
                    {CATEGORY_LABELS[product.category] || product.category}
                    {isNew(product) && <span className="ml-2 text-lime-dark">• Nouveau</span>}
                  </span>
                  <h2 className="font-display font-bold text-sm sm:text-base text-green-950 group-hover:text-green-600 transition-colors line-clamp-1 mt-0.5">
                    {product.title}
                  </h2>
                  <p className="text-xs text-green-900/50 line-clamp-1 mt-0.5">
                    {product.description || 'Pas de description'}
                  </p>
                  <div className="mt-auto flex items-center justify-between pt-1.5">
                    <span className="font-mono font-bold text-sm text-green-700">
                      {formatPrice(product.price_cfa)}
                    </span>
                    <span className="text-xs font-medium text-green-700 group-hover:text-green-600">
                      Voir →
                    </span>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
