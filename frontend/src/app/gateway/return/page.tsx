import { Suspense } from 'react';
import GatewayReturnView from './return-view';

export default function GatewayReturnPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement…</main>}>
      <GatewayReturnView />
    </Suspense>
  );
}
