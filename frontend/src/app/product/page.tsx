import { Suspense } from 'react';
import type { Metadata } from 'next';
import ProductDetail from './product-detail';

interface ProductForMeta {
  title: string;
  description: string | null;
  price_cfa: number;
  cover_image_key?: string | null;
}

// Génère les balises Open Graph par produit côté serveur, pour que
// n'importe quel lien /product?id=... (copié depuis la barre d'adresse,
// pas seulement le bouton "Partager" du site) montre le bon titre/prix/image
// quand il est partagé sur WhatsApp/Facebook. Même logique que
// ProductHandler.Share côté backend (backend/internal/handler/product_handler.go).
export async function generateMetadata({
  searchParams,
}: {
  searchParams: Promise<{ id?: string }>;
}): Promise<Metadata> {
  const id = (await searchParams).id;
  if (!id) return {};

  try {
    const res = await fetch(`${process.env.INTERNAL_API_URL}/api/products/${id}`, {
      cache: 'no-store',
    });
    if (!res.ok) return {};
    const { product }: { product: ProductForMeta } = await res.json();

    const price = `${product.price_cfa} FCFA`;
    const description = product.description
      ? `${price} — ${product.description}`.slice(0, 200)
      : `${price} — produit numérique sur DIARRA`;

    return {
      title: product.title,
      description,
      openGraph: {
        type: 'website',
        title: product.title,
        description,
        images: product.cover_image_key
          ? [`${process.env.NEXT_PUBLIC_API_URL}/api/products/${id}/cover`]
          : undefined,
      },
      twitter: {
        card: 'summary_large_image',
      },
      other: {
        'product:price:amount': String(product.price_cfa),
        'product:price:currency': 'XOF',
      },
    };
  } catch {
    // Pas bloquant : la page reste utilisable avec les balises génériques
    // du site (layout.tsx) si l'API est momentanément injoignable.
    return {};
  }
}

export default function ProductPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <ProductDetail />
    </Suspense>
  );
}
