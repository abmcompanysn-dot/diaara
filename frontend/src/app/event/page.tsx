import { Suspense } from 'react';
import EventDetail from './event-detail';

export default function EventPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <EventDetail />
    </Suspense>
  );
}
