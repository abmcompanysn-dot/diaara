'use client';

import { useState } from 'react';
import { apiOrigin } from '@/lib/api';
import { cn } from '@/lib/utils';
import { CATEGORY_LABELS } from '@/lib/constants';

interface ProductImageProps {
  product: { id: string; cover_image_key?: string; title: string; category: string };
  className?: string;
}

// Affiche l'image de couverture d'un produit (URL signée via /api/products/{id}/cover).
// Sans couverture (ou en cas d'erreur) : placeholder dégradé garanti, donc
// chaque produit est toujours une carte avec une image.
export function ProductImage({ product, className }: ProductImageProps) {
  const [broken, setBroken] = useState(false);
  const showImage = !!product.cover_image_key && !broken;
  const initial = (product.title || 'P').trim().charAt(0).toUpperCase() || 'P';

  if (showImage) {
    return (
      <img
        src={`${apiOrigin}/api/products/${product.id}/cover`}
        alt={product.title}
        loading="lazy"
        onError={() => setBroken(true)}
        className={cn('w-full object-cover', className)}
      />
    );
  }

  return (
    <div
      className={cn(
        'w-full gradient-green-soft flex flex-col items-center justify-center gap-2',
        className
      )}
      aria-hidden
    >
      <span className="w-12 h-12 rounded-lg bg-white/70 flex items-center justify-center text-green-700 font-display text-xl font-bold">
        {initial}
      </span>
      <span className="text-xs text-green-800/80 font-medium uppercase tracking-wider">
        {CATEGORY_LABELS[product.category] || product.category}
      </span>
    </div>
  );
}
