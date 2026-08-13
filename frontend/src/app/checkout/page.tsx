import { Suspense } from 'react';
import CheckoutView from './checkout-view';

export default function CheckoutPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <CheckoutView />
    </Suspense>
  );
}
