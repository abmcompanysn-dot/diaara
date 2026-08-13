'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api, apiOrigin } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ProductImage } from '@/components/product-image';
import { ArrowLeftIcon, CheckIcon, LockIcon, ZapIcon, LinkIcon } from '@/components/icons';
import { CATEGORY_LABELS, PAYMENTS } from '@/lib/constants';

interface Product {
  id: string;
  title: string;
  description: string;
  price_cfa: number;
  category: string;
  moderation_status: string;
  cover_image_key?: string;
  preview_keys?: string[];
  preview_status?: string;
}

const previewKind = (key: string): 'image' | 'audio' | 'video' => {
  const ext = key.split('.').pop()?.toLowerCase() || '';
  if (['mp3', 'wav', 'm4a', 'ogg', 'flac'].includes(ext)) return 'audio';
  if (['mp4', 'mov', 'webm', 'mkv', 'avi'].includes(ext)) return 'video';
  return 'image';
};

export default function ProductDetailPage() {
  const searchParams = useSearchParams();
  const id = searchParams.get('id') || '';
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState(false);

  const handleShare = async () => {
    const url = `${window.location.origin}/p/${id}`;
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      window.prompt('Copiez ce lien :', url);
    }
  };

  useEffect(() => {
    if (id) loadProduct();
  }, [id]);

  const loadProduct = async () => {
    try {
      const result = await api.getProduct(id);
      setProduct(result.product);
    } catch (err) {
      setError('Produit introuvable');
    } finally {
      setLoading(false);
    }
  };

  if (loading)
    return (
      <main className="min-h-[calc(100vh-4rem)] flex items-center justify-center">
        <p className="font-mono text-green-900/50">chargement…</p>
      </main>
    );

  if (!product) {
    return (
      <main className="min-h-[calc(100vh-4rem)] flex flex-col items-center justify-center gap-4">
        <p className="text-destructive font-semibold">{error}</p>
        <Button variant="outline" size="sm" render={<Link href="/catalog" />}>
          <ArrowLeftIcon size={16} className="mr-2" />
          Retour au catalogue
        </Button>
      </main>
    );
  }

  const formatPrice = (price: number) => `${price.toLocaleString()} FCFA`;

  return (
    <main className="bg-paper pb-24 sm:pb-0">
      {/* Bande produit */}
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="grid-pattern absolute inset-0" aria-hidden />
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="absolute top-0 right-0 w-80 h-80 bg-lime/15 rounded-full blur-3xl" aria-hidden />

        <div className="relative max-w-6xl mx-auto px-4 py-12">
          <Link
            href="/catalog"
            className="inline-flex items-center gap-2 font-mono text-sm text-white/60 hover:text-lime transition-colors"
          >
            <ArrowLeftIcon size={16} />
            catalogue
          </Link>

          <p className="font-mono text-sm text-green-300 mt-8 uppercase tracking-widest">
            // produit
          </p>
          <h1 className="font-display text-3xl sm:text-5xl font-bold tracking-tight leading-tight mt-3 max-w-3xl">
            {product.title}
          </h1>

          <div className="mt-5 flex flex-wrap items-center gap-2">
            <Badge className="bg-lime text-green-950 hover:bg-lime border-0">
              {CATEGORY_LABELS[product.category] || product.category}
            </Badge>
            <span className="inline-flex items-center gap-1.5 text-sm text-white/70">
              <ZapIcon size={14} className="text-lime" />
              Livraison instantanée
            </span>
            <span className="inline-flex items-center gap-1.5 text-sm text-white/70">
              <LockIcon size={14} className="text-lime" />
              Paiement sécurisé
            </span>
          </div>
        </div>
      </section>

      {/* Corps */}
      <section className="max-w-6xl mx-auto px-4 py-10 grid lg:grid-cols-5 gap-8">
        <div className="lg:col-span-3">
          <div className="bg-white rounded-xl overflow-hidden shadow-card border border-green-900/5">
            <div className="relative">
              <ProductImage product={product} className="h-64 sm:h-80" />
              <button
                type="button"
                onClick={handleShare}
                aria-label="Partager ce produit"
                className="absolute top-3 right-3 w-10 h-10 rounded-full bg-white/90 backdrop-blur text-green-950 flex items-center justify-center shadow-card hover:bg-white transition-colors"
              >
                <LinkIcon size={17} />
              </button>
              {copied && (
                <span className="absolute top-3 right-14 px-2.5 py-1.5 rounded-lg bg-green-950 text-white text-xs font-medium shadow-lift">
                  Lien copié !
                </span>
              )}
            </div>
            <div className="p-6 sm:p-8">
              <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-3">
                // description
              </p>
              <p className="text-green-900/80 leading-relaxed whitespace-pre-line">
                {product.description || 'Pas de description pour ce produit.'}
              </p>
            </div>
          </div>

          {product.preview_status === 'pending' && (
            <div className="mt-6 bg-white rounded-xl p-6 sm:p-8 shadow-card border border-green-900/5">
              <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-4">
                // aperçu
              </p>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3" aria-hidden>
                {[0, 1, 2].map((i) => (
                  <div key={i} className="h-28 rounded-lg bg-green-900/5 animate-pulse" />
                ))}
              </div>
              <p className="text-sm text-green-900/50 mt-4">
                Génération de l&apos;aperçu en cours, réessayez dans un instant…
              </p>
            </div>
          )}

          {product.preview_status === 'ready' && !!product.preview_keys?.length && (
            <div className="mt-6 bg-white rounded-xl p-6 sm:p-8 shadow-card border border-green-900/5">
              <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-4">
                // aperçu avant achat
              </p>
              {(() => {
                const kind = previewKind(product.preview_keys![0]);
                if (kind === 'audio') {
                  return (
                    <audio
                      controls
                      className="w-full"
                      src={`${apiOrigin}/api/products/${product.id}/preview/0`}
                    />
                  );
                }
                if (kind === 'video') {
                  return (
                    <video
                      controls
                      className="w-full rounded-lg bg-black"
                      src={`${apiOrigin}/api/products/${product.id}/preview/0`}
                    />
                  );
                }
                return (
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    {product.preview_keys!.map((_, idx) => (
                      <img
                        key={idx}
                        src={`${apiOrigin}/api/products/${product.id}/preview/${idx}`}
                        alt={`Aperçu ${idx + 1}`}
                        loading="lazy"
                        className="w-full rounded-lg border border-green-900/10 object-cover"
                      />
                    ))}
                  </div>
                );
              })()}
            </div>
          )}

          <div className="mt-6 bg-white rounded-xl p-6 sm:p-8 shadow-card border border-green-900/5">
            <p className="font-mono text-sm text-green-700/60 uppercase tracking-widest mb-4">
              // livraison
            </p>
            <ul className="grid gap-3 sm:grid-cols-3">
              {[
                { title: 'Instantannée', desc: 'Lien de téléchargement généré immédiatement après paiement.' },
                { title: 'Sécurisée', desc: 'Lien unique, limité à 3 téléchargements.' },
                { title: 'Encadrée', desc: 'Le vendeur est payé après votre confirmation.' },
              ].map((item) => (
                <li key={item.title} className="rounded-lg bg-paper p-4">
                  <p className="font-display font-bold text-green-950">{item.title}</p>
                  <p className="text-sm text-green-900/60 mt-1">{item.desc}</p>
                </li>
              ))}
            </ul>
          </div>
        </div>

        <aside className="lg:col-span-2">
          <div className="lg:sticky lg:top-20 bg-white rounded-xl shadow-lift border border-green-900/5 p-6 sm:p-8">
            <p className="font-mono text-xs text-green-900/50 uppercase tracking-widest">
              // prix
            </p>
            <p className="font-display text-3xl font-bold text-green-700 mt-2">
              {formatPrice(product.price_cfa)}
            </p>

            <Button
              render={<Link href={`/checkout?product=${product.id}`} />}
              className="w-full h-11 mt-6 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300 text-base"
            >
              Acheter maintenant
            </Button>

            <ul className="mt-6 space-y-2.5">
              {[
                'Paiement mobile money vérifié (PawaPay)',
                'Livraison instantanée par lien sécurisé',
                'Confirmation par code PIN sur votre téléphone',
              ].map((item) => (
                <li key={item} className="flex items-start gap-2.5 text-sm text-green-900/70">
                  <span className="mt-0.5 w-5 h-5 rounded-full bg-green-500 text-white flex items-center justify-center shrink-0" aria-hidden>
                    <CheckIcon size={13} />
                  </span>
                  <span>{item}</span>
                </li>
              ))}
            </ul>

            <div className="mt-6 pt-5 border-t border-green-900/10">
              <p className="text-xs text-green-900/50 mb-2.5">Paiements acceptés :</p>
              <div className="flex flex-wrap gap-2">
                {PAYMENTS.map((p) => (
                  <span
                    key={p.name}
                    className={`px-2.5 py-1 rounded ${p.color} ${p.text} text-xs font-bold`}
                  >
                    {p.name}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </aside>
      </section>

      {/* Barre d'achat fixe (mobile uniquement) : le prix et le CTA restent accessibles
          à tout moment du défilement, sans dupliquer la carte sticky desktop. */}
      <div className="sm:hidden fixed bottom-0 inset-x-0 z-50 bg-white border-t border-green-900/10 shadow-lift px-4 py-3 flex items-center gap-3 pb-[calc(env(safe-area-inset-bottom)+0.75rem)]">
        <div className="min-w-0">
          <p className="text-[10px] text-green-900/50 uppercase tracking-wide">Prix</p>
          <p className="font-display font-bold text-lg text-green-950 truncate">
            {formatPrice(product.price_cfa)}
          </p>
        </div>
        <Button
          render={<Link href={`/checkout?product=${product.id}`} />}
          className="flex-1 h-12 rounded-full bg-lime text-green-950 font-semibold hover:bg-green-300 text-base"
        >
          🛒 Acheter maintenant
        </Button>
      </div>
    </main>
  );
}
