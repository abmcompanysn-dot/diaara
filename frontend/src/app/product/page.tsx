import { Suspense } from 'react';
import ProductDetail from './product-detail';

export default function ProductPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <ProductDetail />
    </Suspense>
  );
}
