import { Suspense } from 'react';
import TicketDetail from './ticket-detail';

export default function SupportTicketPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <TicketDetail />
    </Suspense>
  );
}
