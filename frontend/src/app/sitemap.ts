import type { MetadataRoute } from 'next';

// Le projet est exporté en statique (output: 'export') — cette route doit
// donc être générée au build (à chaque déploiement) plutôt qu'à la demande.
export const dynamic = 'force-static';

const SITE_URL = (process.env.NEXT_PUBLIC_SITE_URL || 'https://diarra.abmcy.com').replace(/\/$/, '');
const API_URL = (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080').replace(/\/$/, '');

interface FeedProduct {
  id: string;
  vendor_id: string;
  cover_image_key?: string | null;
  updated_at: string;
}

// Le catalogue peut dépasser la limite d'une seule page d'API : on paginate
// jusqu'à épuisement plutôt que de fixer un nombre de pages arbitraire.
async function fetchAllApprovedProducts(): Promise<FeedProduct[]> {
  const products: FeedProduct[] = [];
  const limit = 200;
  let offset = 0;
  for (;;) {
    const res = await fetch(`${API_URL}/api/products?limit=${limit}&offset=${offset}`, { cache: 'no-store' });
    if (!res.ok) break;
    const data = await res.json();
    const batch: FeedProduct[] = data.products || [];
    products.push(...batch);
    if (batch.length < limit) break;
    offset += limit;
  }
  return products;
}

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const staticPages: MetadataRoute.Sitemap = [
    { url: `${SITE_URL}/`, changeFrequency: 'daily', priority: 1 },
    { url: `${SITE_URL}/catalog/`, changeFrequency: 'daily', priority: 0.9 },
    { url: `${SITE_URL}/how-it-works/`, changeFrequency: 'monthly', priority: 0.5 },
    { url: `${SITE_URL}/sell/`, changeFrequency: 'monthly', priority: 0.6 },
    { url: `${SITE_URL}/mentions-legales/`, changeFrequency: 'yearly', priority: 0.1 },
    { url: `${SITE_URL}/confidentialite/`, changeFrequency: 'yearly', priority: 0.1 },
    { url: `${SITE_URL}/cgu/`, changeFrequency: 'yearly', priority: 0.1 },
  ];

  let products: FeedProduct[] = [];
  try {
    products = await fetchAllApprovedProducts();
  } catch {
    products = [];
  }

  const productPages: MetadataRoute.Sitemap = products.map((p) => ({
    url: `${SITE_URL}/product/?id=${p.id}`,
    lastModified: p.updated_at,
    changeFrequency: 'weekly',
    priority: 0.8,
    images: p.cover_image_key ? [`${API_URL}/api/products/${p.id}/cover`] : undefined,
  }));

  // Une entrée par boutique (vendeur), avec la couverture d'un de ses
  // produits comme image représentative — il n'existe pas de photo de
  // profil dédiée côté vendeur.
  const vendorIds = Array.from(new Set(products.map((p) => p.vendor_id)));
  const vendorImage = new Map<string, string>();
  for (const p of products) {
    if (p.cover_image_key && !vendorImage.has(p.vendor_id)) {
      vendorImage.set(p.vendor_id, `${API_URL}/api/products/${p.id}/cover`);
    }
  }

  const boutiquePages: MetadataRoute.Sitemap = vendorIds.map((id) => ({
    url: `${SITE_URL}/boutique/?id=${id}`,
    changeFrequency: 'weekly',
    priority: 0.6,
    images: vendorImage.has(id) ? [vendorImage.get(id) as string] : undefined,
  }));

  return [...staticPages, ...productPages, ...boutiquePages];
}
