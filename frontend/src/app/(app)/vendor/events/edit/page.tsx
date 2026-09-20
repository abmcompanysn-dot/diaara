import { Suspense } from 'react';
import EditEvent from './edit-event';

export default function EditEventPage() {
  return (
    <Suspense fallback={<main className="p-8 text-center">Chargement...</main>}>
      <EditEvent />
    </Suspense>
  );
}
