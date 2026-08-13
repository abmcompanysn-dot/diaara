import { Suspense } from 'react';
import EditProduct from './edit-product';

export default function EditProductPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <EditProduct />
    </Suspense>
  );
}
