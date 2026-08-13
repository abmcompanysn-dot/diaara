import { Suspense } from 'react';
import CheckoutReturnView from './return-view';

export default function CheckoutReturnPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <CheckoutReturnView />
    </Suspense>
  );
}
