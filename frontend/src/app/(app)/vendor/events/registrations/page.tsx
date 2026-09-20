import { Suspense } from 'react';
import EventRegistrations from './event-registrations';

export default function EventRegistrationsPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <EventRegistrations />
    </Suspense>
  );
}
