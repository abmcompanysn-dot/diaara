// Service Worker DIARRA — prérequis technique indispensable pour :
//   1. La bannière d'installation PWA (beforeinstallprompt ne se déclenche
//      JAMAIS sans un SW enregistré, quel que soit le reste du code).
//   2. La réception des notifications push (l'event 'push' n'existe que
//      dans le contexte d'un Service Worker).
//
// Volontairement minimal — pas de cache offline agressif (le site change
// souvent : catalogue, prix, statuts de commande — un cache mal invalidé
// montrerait des données périmées à l'acheteur/vendeur, pire qu'un site qui
// marche normalement en ligne).

self.addEventListener('install', () => {
  // skipWaiting : active le nouveau SW immédiatement au lieu d'attendre la
  // fermeture de tous les onglets — le site étant en export statique
  // (pas de version figée à respecter côté client), pas de risque de
  // mélanger une vieille page avec un nouveau SW incompatible.
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});

// --- Notifications push --------------------------------------------------

self.addEventListener('push', (event) => {
  if (!event.data) return;

  let payload;
  try {
    payload = event.data.json();
  } catch {
    payload = { title: 'DIARRA', body: event.data.text() };
  }

  const title = payload.title || 'DIARRA';
  const options = {
    body: payload.body || '',
    icon: payload.icon || '/brand/diarra-icon.png',
    badge: '/brand/diarra-icon.png',
    // image : grande illustration affichée dans le corps de la notification
    // (photo du produit vendu, contrairement à icon qui reste une petite
    // pastille ronde) — absente sur la plupart des notifications système
    // (Windows/macOS/Android l'ignorent hors Chrome Android), dégradation
    // silencieuse si non fournie.
    image: payload.image || undefined,
    data: { url: payload.url || '/' },
    tag: payload.tag, // regroupe/remplace les notifications du même type (ex. "order_paid")
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

// Clic sur une notification : ouvre/focus l'onglet DIARRA existant plutôt
// que d'en ouvrir un nouveau à chaque fois.
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const targetUrl = event.notification.data?.url || '/';

  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      for (const client of clientList) {
        if (client.url.includes(self.location.origin) && 'focus' in client) {
          client.navigate(targetUrl);
          return client.focus();
        }
      }
      return self.clients.openWindow(targetUrl);
    })
  );
});
