/** @type {import('next').NextConfig} */
const nextConfig = {
  // Export statique : servi par nginx sur le VPS ET par Cloudflare Worker
  // (env.ASSETS) pour diarra.app. L'Open Graph par produit passe par la
  // route backend /p/{id} (ProductHandler.Share) plutôt que par du SSR
  // Next.js — un export statique ne peut pas générer de métadonnées par
  // requête (voir historique git pour la tentative SSR abandonnée ici).
  output: 'export',
  images: {
    unoptimized: true,
  },
  trailingSlash: true,
};

module.exports = nextConfig;
