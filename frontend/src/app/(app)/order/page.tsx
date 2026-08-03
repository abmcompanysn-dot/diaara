import { Suspense } from 'react';
import OrderDetail from './order-detail';

export default function OrderPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <OrderDetail />
    </Suspense>
  );
}
