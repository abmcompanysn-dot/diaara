export interface Env {
  ASSETS: Fetcher;
  BACKEND_URL: string;
  RATE_LIMITS: KVNamespace;
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);

    // /p/{id} (ProductHandler.Share) redirige les navigateurs réels vers
    // FRONTEND_URL/product?id=... — qui vaut désormais diarra.app, cette
    // MÊME zone Worker. En mode redirect:'follow' (utilisé plus bas pour
    // les autres routes), le Worker essaierait de suivre lui-même cette
    // redirection en se re-fetchant, ce qui échoue systématiquement en 522
    // (timeout). Comme /r/ (déjà en 'manual'), /p/ doit juste relayer la
    // 3xx telle quelle au navigateur, jamais la suivre côté serveur.
    const isRedirectToBrowser = url.pathname.startsWith('/r/') || url.pathname.startsWith('/p/');
    // /p/ (partage Open Graph par produit), /feed/ et /sitemap.xml sont
    // générés dynamiquement par le backend Go (voir Caddyfile du VPS) —
    // sans ça ils tombent sur env.ASSETS.fetch et cassent silencieusement
    // le partage produit + le sitemap une fois le Worker en place.
    const isBackendRoute =
      url.pathname.startsWith('/api/') ||
      url.pathname.startsWith('/ws/') ||
      url.pathname.startsWith('/p/') ||
      url.pathname.startsWith('/feed/') ||
      url.pathname === '/sitemap.xml';
    if (isBackendRoute || isRedirectToBrowser) {
      const isRateLimited = await checkRateLimit(request, env);
      if (isRateLimited) {
        return new Response(JSON.stringify({ error: 'rate_limited' }), {
          status: 429,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      const backendUrl = env.BACKEND_URL || 'https://diarra-backend.onrender.com';
      const targetUrl = `${backendUrl}${url.pathname}${url.search}`;

      const headers = new Headers(request.headers);
      headers.set('X-Forwarded-For', request.headers.get('CF-Connecting-IP') || '');
      headers.delete('CF-Connecting-IP');

      try {
        // Pour /r/ et /p/ on transmet la 3xx au navigateur (pas de suivi
        // côté Worker) — voir le commentaire sur isRedirectToBrowser plus haut.
        const response = await fetch(targetUrl, {
          method: request.method,
          headers,
          body: request.body,
          redirect: isRedirectToBrowser ? 'manual' : 'follow',
        });

        // WebSocket (101 Switching Protocols) : renvoyer la réponse telle quelle,
        // la reconstruire casserait l'upgrade.
        if (response.status === 101) {
          return response;
        }

        return new Response(response.body, {
          status: response.status,
          headers: new Headers(response.headers),
        });
      } catch (e) {
        return new Response(JSON.stringify({ error: 'backend_unavailable' }), {
          status: 502,
          headers: { 'Content-Type': 'application/json' },
        });
      }
    }

    return env.ASSETS.fetch(request);
  },

  // Garde l'instance Render (free tier, sleep après 15 min sans trafic) éveillée.
  async scheduled(_event: ScheduledEvent, env: Env): Promise<void> {
    try {
      await fetch(`${env.BACKEND_URL}/health`, { method: 'HEAD' });
    } catch {
      // silencieux : le prochain tick réessaiera
    }
  },
};

async function checkRateLimit(request: Request, env: Env): Promise<boolean> {
  const ip = request.headers.get('CF-Connecting-IP') || 'unknown';
  const url = new URL(request.url);
  const key = `rate:${url.pathname}:${ip}`;

  const current = await env.RATE_LIMITS.get(key);
  const count = current ? parseInt(current, 10) : 0;

  const limit = getRateLimit(url.pathname);
  if (count >= limit) {
    return true;
  }

  await env.RATE_LIMITS.put(key, String(count + 1), { expirationTtl: 60 });
  return false;
}

function getRateLimit(pathname: string): number {
  if (pathname.includes('/auth/login')) return 5;
  if (pathname.includes('/auth/register')) return 3;
  if (pathname.includes('/orders')) return 10;
  return 30;
}
