'use client';

import { useEffect } from 'react';
import { registerServiceWorker } from '@/lib/push';

// ServiceWorkerRegistration — enregistre public/sw.js au montage de l'app.
// Composant sans rendu (retourne null) : uniquement un point d'accroche
// useEffect, monté une fois dans layout.tsx.
export function ServiceWorkerRegistration() {
  useEffect(() => {
    registerServiceWorker();
  }, []);
  return null;
}
