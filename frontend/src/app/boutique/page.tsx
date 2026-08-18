import { Suspense } from 'react';
import BoutiqueView from './boutique-view';

// Page par vendeur (paramétrée par ?vendor=), export statique : un seul
// titre générique ici, comme product/page.tsx — pas de generateMetadata
// possible sans SSR.
export const metadata = {
  title: 'Boutique',
  description: 'La boutique de ce vendeur sur DIARRA : tous ses produits numériques en un seul endroit.',
};

export default function BoutiquePage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <BoutiqueView />
    </Suspense>
  );
}
