export interface Env {
  ASSETS: Fetcher;
  BACKEND_URL: string;
  RATE_LIMITS: KVNamespace;
  DELIVERY_QUEUE: Queue;
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);

    const isReferral = url.pathname.startsWith('/r/');
    const isApi = url.pathname.startsWith('/api/') || url.pathname.startsWith('/ws/');
    if (isApi || isReferral) {
      const isRateLimited = await checkRateLimit(request, env);
      if (isRateLimited) {
        return new Response(JSON.stringify({ error: 'rate_limited' }), {
          status: 429,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      const backendUrl = env.BACKEND_URL || 'http://backend:8080';
      const targetUrl = `${backendUrl}${url.pathname}${url.search}`;

      const headers = new Headers(request.headers);
      headers.set('X-Forwarded-For', request.headers.get('CF-Connecting-IP') || '');
      headers.delete('CF-Connecting-IP');

      try {
        // Pour /r/ on transmet la 302 au navigateur (pas de suivi côté Worker)
        // afin que l'utilisateur atterrisse sur le produit du frontend.
        const response = await fetch(targetUrl, {
          method: request.method,
          headers,
          body: request.body,
          redirect: isReferral ? 'manual' : 'follow',
        });

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
