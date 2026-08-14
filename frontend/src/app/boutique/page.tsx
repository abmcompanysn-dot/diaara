import { Suspense } from 'react';
import BoutiqueView from './boutique-view';

export default function BoutiquePage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <BoutiqueView />
    </Suspense>
  );
}
