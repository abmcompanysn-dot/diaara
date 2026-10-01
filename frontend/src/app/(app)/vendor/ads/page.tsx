import { Suspense } from 'react';
import VendorAds from './vendor-ads';

export default function VendorAdsPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <VendorAds />
    </Suspense>
  );
}
