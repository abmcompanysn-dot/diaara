import type { MetadataRoute } from 'next';

// Le projet est exporté en statique (output: 'export') — cette route doit
// donc être générée au build plutôt qu'à la demande.
export const dynamic = 'force-static';

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: '*',
      allow: '/',
      // Zones privées ou sans intérêt pour l'indexation : espaces connectés,
      // API, flux d'achat/inscription (contenu dupliqué par produit/paramètre).
      disallow: ['/dashboard', '/admin', '/vendor', '/closer', '/account', '/orders', '/order', '/checkout', '/api'],
    },
  };
}
