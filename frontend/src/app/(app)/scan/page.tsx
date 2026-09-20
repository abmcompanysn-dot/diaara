import { Suspense } from 'react';
import TicketScan from './ticket-scan';

// /scan (pas /admin/scan) : accessible à tout utilisateur connecté — le
// layout (app) exige seulement RequireAuth, pas RequireAdmin (voir
// src/app/(app)/admin/layout.tsx qui, lui, bloquerait un vendeur non-admin).
// L'autorisation réelle (admin OU vendeur propriétaire de l'événement) est
// vérifiée côté backend (EventTicketHandler.scan), cette page affiche juste
// ce que l'API renvoie.
export default function ScanPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <TicketScan />
    </Suspense>
  );
}
