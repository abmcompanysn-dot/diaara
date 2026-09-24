'use client';

// push.ts — enregistrement du Service Worker (public/sw.js) et gestion de la
// souscription aux notifications push (Web Push API + clé VAPID). Isolé du
// reste de l'app : peut échouer silencieusement sur un navigateur qui ne
// supporte pas Push (Safari desktop ancien, certains navigateurs Android
// alternatifs) sans jamais bloquer le reste du site.

// Clé publique VAPID — publique par nature (embarquée dans le bundle client,
// jamais un secret), injectée au build via une variable d'env Cloudflare
// Worker (voir README section déploiement frontend). Vide = notifications
// push désactivées côté client (dégradation silencieuse).
const VAPID_PUBLIC_KEY = process.env.NEXT_PUBLIC_VAPID_PUBLIC_KEY || '';

export function pushSupported(): boolean {
  return typeof window !== 'undefined' && 'serviceWorker' in navigator && 'PushManager' in window;
}

// registerServiceWorker — à appeler une fois au montage de l'app (voir
// ServiceWorkerRegistration dans layout.tsx). Idempotent : un second appel
// (ex. navigation client-side) réutilise l'enregistrement existant.
export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  if (typeof window === 'undefined' || !('serviceWorker' in navigator)) return null;
  try {
    return await navigator.serviceWorker.register('/sw.js');
  } catch {
    return null;
  }
}

// urlBase64ToUint8Array — la Push API attend applicationServerKey sous forme
// binaire (Uint8Array), pas la chaîne base64url renvoyée par web-push
// generate-vapid-keys — conversion standard, pas de dépendance nécessaire.
function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const rawData = atob(base64);
  const outputArray = new Uint8Array(new ArrayBuffer(rawData.length));
  for (let i = 0; i < rawData.length; i++) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

export type PushPermissionState = 'default' | 'granted' | 'denied' | 'unsupported';

export function getPushPermissionState(): PushPermissionState {
  if (!pushSupported()) return 'unsupported';
  return Notification.permission as PushPermissionState;
}

// subscribeToPush — demande la permission navigateur (si pas déjà tranchée),
// puis crée l'abonnement Push et renvoie l'objet PushSubscription brut — à
// envoyer tel quel (JSON.stringify) au backend pour qu'il puisse pousser des
// notifications plus tard. Ne fait RIEN côté serveur ici (voir api.ts,
// subscribeToPushNotifications) — cette fonction ne gère que le navigateur.
export async function subscribeToPush(): Promise<PushSubscription | null> {
  if (!pushSupported() || !VAPID_PUBLIC_KEY) return null;

  const permission = await Notification.requestPermission();
  if (permission !== 'granted') return null;

  const registration = await registerServiceWorker();
  if (!registration) return null;

  // Réutilise un abonnement existant plutôt que d'en recréer un (changer
  // d'abonnement invaliderait celui déjà connu du backend sans le prévenir).
  const existing = await registration.pushManager.getSubscription();
  if (existing) return existing;

  return registration.pushManager.subscribe({
    userVisibleOnly: true, // requis par le spec : chaque push DOIT afficher une notification visible
    applicationServerKey: urlBase64ToUint8Array(VAPID_PUBLIC_KEY),
  });
}

export async function unsubscribeFromPush(): Promise<boolean> {
  if (!pushSupported()) return false;
  const registration = await navigator.serviceWorker.getRegistration();
  const subscription = await registration?.pushManager.getSubscription();
  if (!subscription) return true;
  return subscription.unsubscribe();
}
