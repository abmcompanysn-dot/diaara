/** @type {import('next').NextConfig} */
const nextConfig = {
  // SSR (via k3s, plus un export statique) : nécessaire pour générer des
  // balises Open Graph par produit (voir generateMetadata dans
  // src/app/product/page.tsx) — un export statique sert le même HTML à
  // tout le monde, impossible d'y injecter des données par produit.
  output: 'standalone',
  images: {
    unoptimized: true,
  },
  trailingSlash: true,
};

module.exports = nextConfig;
