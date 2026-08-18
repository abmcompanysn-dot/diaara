import { Suspense } from 'react';
import ProductDetail from './product-detail';

// Pas de generateMetadata ici : le site est en export statique
// (output: 'export'), la même page HTML est servie à tout le monde quel
// que soit ?id=. L'Open Graph par produit passe par la route backend
// /p/{id} (ProductHandler.Share, voir backend/internal/handler/product_handler.go)
// et le bouton "Partager" / QR code de la fiche produit pointent vers elle.
export default function ProductPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <ProductDetail />
    </Suspense>
  );
}
