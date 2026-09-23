// Vide par défaut (donc relatif/same-origin) : nginx (VPS) et le Worker
// Cloudflare (worker/src/index.ts) proxient déjà /api/* vers le backend sur
// le même domaine que sert la page — pas besoin de connaître l'URL finale
// (workers.dev en test, diarra.app en prod) au moment du build.
const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

// Origine absolue : sert à construire les liens partagés hors-page (ex.
// /r/{slug} envoyé par WhatsApp/SMS), où une URL relative ne veut rien dire.
// Repli sur l'origine réelle du navigateur si non fournie au build.
export const apiOrigin =
  process.env.NEXT_PUBLIC_API_URL || (typeof window !== 'undefined' ? window.location.origin : '');

interface FetchOptions extends RequestInit {
  skipAuth?: boolean;
}

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

// Endpoints exclus de la logique de refresh/déconnexion automatique : y
// appliquer le refresh créerait une boucle (refresh qui échoue en 401
// déclenche un nouveau refresh...).
const AUTH_ENDPOINTS = ['/api/auth/refresh', '/api/auth/login', '/api/auth/register', '/api/auth/logout'];

// Un seul refresh en vol à la fois : si plusieurs requêtes expirent en même
// temps, elles partagent la même tentative de refresh au lieu d'en déclencher
// une chacune.
let refreshInFlight: Promise<string | null> | null = null;

async function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      try {
        const res = await fetch(`${API_BASE}/api/auth/refresh`, { method: 'POST', credentials: 'include' });
        if (!res.ok) return null;
        const data = await res.json();
        localStorage.setItem('access_token', data.access_token);
        return data.access_token as string;
      } catch {
        return null;
      } finally {
        refreshInFlight = null;
      }
    })();
  }
  return refreshInFlight;
}

// toEventDateISO convertit la valeur brute d'un <input type="date">
// ("YYYY-MM-DD") en horodatage RFC3339 complet : le backend Go décode
// event_date en time.Time, qui rejette un format de date seule ("cannot
// parse... as T" — vérifié : "2026-11-26" échoue le décodage JSON). Minuit
// UTC est arbitraire mais suffisant, la date seule (pas l'heure précise) est
// ce qui est affiché sur la fiche événement.
function toEventDateISO(dateOnly?: string): string | undefined {
  if (!dateOnly) return undefined;
  return `${dateOnly}T00:00:00Z`;
}

// Session expirée et non récupérable : déconnexion réelle (pas juste un
// message affiché pendant que l'utilisateur reste "connecté" côté client).
// Redirection dure (pas de router.push) pour repartir d'un état React propre.
function forceLogout() {
  localStorage.removeItem('access_token');
  if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/auth/')) {
    window.location.href = '/auth/login';
  }
}

async function fetchApi<T>(endpoint: string, options: FetchOptions = {}, isRetry = false): Promise<T> {
  const { skipAuth = false, ...fetchOptions } = options;

  const isFormData = typeof FormData !== 'undefined' && fetchOptions.body instanceof FormData;

  const headers: HeadersInit = {
    ...(isFormData ? {} : { 'Content-Type': 'application/json' }),
    ...fetchOptions.headers,
  };

  if (!skipAuth) {
    const token = localStorage.getItem('access_token');
    if (token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
    }
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...fetchOptions,
    headers,
    credentials: 'include',
  });

  if (!response.ok) {
    if (response.status === 401 && !skipAuth && !isRetry && !AUTH_ENDPOINTS.includes(endpoint)) {
      const newToken = await refreshAccessToken();
      if (newToken) {
        return fetchApi<T>(endpoint, options, true);
      }
      forceLogout();
    }
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new ApiError(response.status, error.error || 'Request failed');
  }

  return response.json();
}

export const api = {
  // Auth
  register: (data: { email: string; password: string; phone?: string; roles?: string[] }) =>
    fetchApi<{
      user: any;
      access_token: string;
      pending_verifications: string[];
      dev_email_otp?: string;
      dev_phone_otp?: string;
    }>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
      skipAuth: true,
    }),

  login: (data: { email: string; password: string }) =>
    fetchApi<{ access_token: string }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
      skipAuth: true,
    }),

  googleLogin: (idToken: string) =>
    fetchApi<{ access_token: string }>('/api/auth/google', {
      method: 'POST',
      body: JSON.stringify({ id_token: idToken }),
      skipAuth: true,
    }),

  getMe: () => fetchApi<{ user: any }>('/api/auth/me'),

  verifyPhoneFirebase: (idToken: string) =>
    fetchApi<{ user: any }>('/api/auth/verify-phone-firebase', {
      method: 'POST',
      body: JSON.stringify({ id_token: idToken }),
    }),

  logout: () =>
    fetchApi<void>('/api/auth/logout', {
      method: 'POST',
      skipAuth: true,
    }),

  // OTP : envoi (resend) et vérification. Le refresh token vit en cookie httpOnly,
  // l'access token dans localStorage.
  // purpose : optionnel, force l'usage du code plutôt que de le déduire du
  // canal — sert au repli "confirmer le téléphone par email" (channel:
  // 'email', purpose: 'phone_verify') quand Firebase Phone Auth échoue.
  sendOtp: (channel: 'email' | 'sms', purpose?: 'email_verify' | 'phone_verify') =>
    fetchApi<{ status: string; channel: string; dev_code?: string }>('/api/auth/send-otp', {
      method: 'POST',
      body: JSON.stringify({ channel, purpose }),
    }),

  verifyOtp: (channel: 'email' | 'sms', code: string, purpose?: 'email_verify' | 'phone_verify') =>
    fetchApi<{ status: string; channel: string }>('/api/auth/verify-otp', {
      method: 'POST',
      body: JSON.stringify({ channel, code, purpose }),
    }),

  refreshToken: () =>
    fetchApi<{ access_token: string }>('/api/auth/refresh', {
      method: 'POST',
      skipAuth: true,
    }),

  forgotPassword: (email: string) =>
    fetchApi<{ message: string }>('/api/auth/forgot-password', {
      method: 'POST',
      body: JSON.stringify({ email }),
      skipAuth: true,
    }),

  resetPassword: (token: string, newPassword: string) =>
    fetchApi<{ status: string }>('/api/auth/reset-password', {
      method: 'POST',
      body: JSON.stringify({ token, new_password: newPassword }),
      skipAuth: true,
    }),

  // Checkout : indique si le bouton "Carte bancaire / PayPal" doit être
  // affiché (interrupteur admin, voir /admin/settings et
  // model.SettingCardPaymentEnabled côté backend).
  getCheckoutConfig: () =>
    fetchApi<{ card_payment_enabled: boolean }>('/api/checkout/config', { skipAuth: true }),

  // Products
  getProducts: (params?: { category?: string; search?: string }) => {
    const query = new URLSearchParams(params as Record<string, string>).toString();
    return fetchApi<{ products: any[] }>(`/api/products${query ? `?${query}` : ''}`);
  },

  getProduct: (id: string) =>
    fetchApi<{
      product: any;
      vendor_tracking?: { facebook_pixel_id?: string | null; google_tag_id?: string | null };
    }>(`/api/products/${id}`),

  // Boutique publique d'un vendeur (partageable via QR code)
  getVendorShop: (vendorId: string) =>
    fetchApi<{
      vendor: {
        id: string;
        display_name: string | null;
        shop_name: string | null;
        tier?: string;
        facebook_pixel_id?: string | null;
        google_tag_id?: string | null;
      };
      products: any[];
    }>(`/api/vendors/${vendorId}/shop`, { skipAuth: true }),

  createProduct: (data: {
    title: string;
    description: string;
    price_cfa: number;
    price_mode?: 'fixed' | 'flexible';
    min_price_cfa?: number;
    category: string;
    file_key: string;
    cover_image_key?: string;
    affiliate_enabled?: boolean;
    max_closer_commission_pct?: number;
  }) =>
    fetchApi<{ product: any }>('/api/vendor/products', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  updateProduct: (
    id: string,
    data: {
      title?: string;
      description?: string;
      price_cfa?: number;
      price_mode?: 'fixed' | 'flexible';
      min_price_cfa?: number;
      category?: string;
      file_key?: string;
      cover_image_key?: string;
      affiliate_enabled?: boolean;
      max_closer_commission_pct?: number;
    }
  ) =>
    fetchApi<{ product: any; reverted_to_pending?: boolean }>(`/api/vendor/products/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  deleteProduct: (id: string) =>
    fetchApi<{ status: string }>(`/api/vendor/products/${id}`, { method: 'DELETE' }),

  // Événements vendeur — 1 à 3 offres, chacune payante ou gratuite.
  createEvent: (data: {
    title: string;
    description?: string;
    cover_image_key?: string;
    event_date?: string; // "YYYY-MM-DD" (valeur brute d'un <input type="date">)
    meeting_link?: string;
    offers: { title: string; is_free: boolean; price_cfa?: number }[];
  }) =>
    fetchApi<{ event: any }>('/api/vendor/events', {
      method: 'POST',
      body: JSON.stringify({ ...data, event_date: toEventDateISO(data.event_date) }),
    }),

  updateEvent: (
    id: string,
    data: { title?: string; description?: string; cover_image_key?: string; event_date?: string; meeting_link?: string }
  ) =>
    fetchApi<{ event: any }>(`/api/vendor/events/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ ...data, event_date: toEventDateISO(data.event_date) }),
    }),

  getVendorEvents: () => fetchApi<{ events: any[] }>('/api/vendor/events'),

  deleteEvent: (id: string) => fetchApi<void>(`/api/vendor/events/${id}`, { method: 'DELETE' }),

  getEventRegistrations: (id: string) =>
    fetchApi<{ registrations: any[] }>(`/api/vendor/events/${id}/registrations`),

  // Offres d'un événement (jusqu'à 3) : ajout, modification (titre et/ou
  // prix d'une offre payante), suppression (refusée s'il ne reste qu'une
  // seule offre — voir EventHandler.DeleteOffer côté backend).
  addEventOffer: (eventId: string, data: { title: string; is_free: boolean; price_cfa?: number }) =>
    fetchApi<any>(`/api/vendor/events/${eventId}/offers`, { method: 'POST', body: JSON.stringify(data) }),

  updateEventOffer: (offerId: string, data: { title?: string; price_cfa?: number }) =>
    fetchApi<any>(`/api/vendor/events/offers/${offerId}`, { method: 'PUT', body: JSON.stringify(data) }),

  deleteEventOffer: (offerId: string) =>
    fetchApi<void>(`/api/vendor/events/offers/${offerId}`, { method: 'DELETE' }),

  getEvents: () => fetchApi<{ events: any[] }>('/api/events', { skipAuth: true }),

  getEvent: (idOrSlug: string) =>
    fetchApi<{ event: any; offers: any[] }>(`/api/events/${idOrSlug}`, { skipAuth: true }),

  registerFreeEventOffer: (offerId: string, data: { full_name: string; email: string; phone: string }) =>
    fetchApi<{ id: string }>(`/api/events/offers/${offerId}/register`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Billet PDF (QR de vérification) d'une commande payée — pas un fetchApi
  // JSON classique : c'est un lien direct vers un flux PDF binaire, à
  // utiliser dans un <a href> / window.open, pas un appel await.
  ticketPdfUrl: (checkoutToken: string) => `${apiOrigin}/api/orders/${encodeURIComponent(checkoutToken)}/ticket.pdf`,

  // Scan à l'entrée (admin ou vendeur propriétaire de l'événement — filtré
  // côté backend). status: false = consulter sans marquer utilisé.
  getTicketScanStatus: (token: string) =>
    fetchApi<{ buyer_name: string; event_title: string; offer_title: string; already_used: boolean; checked_in_at: string | null }>(
      `/api/events/scan?token=${encodeURIComponent(token)}`
    ),

  confirmTicketScan: (token: string) =>
    fetchApi<{ buyer_name: string; event_title: string; offer_title: string; already_used: boolean; checked_in_at: string | null }>(
      `/api/events/scan?token=${encodeURIComponent(token)}`,
      { method: 'POST' }
    ),

  // Orders
  createOrder: (data: {
    product_id: string;
    buyer_name: string;
    buyer_email?: string;
    country: string;
    referral_link_id?: string;
    // amount_cfa : uniquement pris en compte si le produit est en prix libre
    // (price_mode "flexible") — ignoré côté serveur pour un produit à prix fixe.
    amount_cfa?: number;
    // payment_method : "mobile_money" (défaut si absent) | "card" | "paypal".
    // Carte et PayPal passent toujours par KPay côté serveur (PawaPay ne les
    // supporte pas) ; mobile_money est routé PawaPay/KPay selon le réglage
    // admin par pays (voir CheckoutProviderSettingKey côté backend).
    payment_method?: 'mobile_money' | 'card' | 'paypal';
  }) =>
    // Pas de skipAuth ici : la route accepte les invités (guest checkout)
    // mais doit recevoir le token si l'acheteur est connecté, sinon le
    // backend le traite toujours comme un invité et exige un email.
    fetchApi<{ order: any; checkout: { deposit_id: string; redirect_url: string } }>('/api/orders', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Suivi public d'une commande (acheteur invité) par checkout_token
  getCheckoutStatus: (token: string) =>
    fetchApi<{ order: any }>(`/api/orders/status?token=${encodeURIComponent(token)}`, {
      skipAuth: true,
    }),

  getOrder: (id: string) => fetchApi<{ order: any }>(`/api/orders/${id}`),

  getOrders: () => fetchApi<{ orders: any[] }>('/api/orders'),

  // Delivery
  getDelivery: (orderId: string) =>
    fetchApi<{ delivery: any; signed_url: string }>(`/api/orders/${orderId}/delivery`, {
      method: 'POST',
    }),

  // Livraison pour un acheteur invité (sans session), via checkout_token
  getDeliveryByToken: (token: string) =>
    fetchApi<{ delivery: any; signed_url: string }>(`/api/orders/delivery?token=${encodeURIComponent(token)}`, {
      method: 'POST',
      skipAuth: true,
    }),

  // Packs de produits
  getBundle: (id: string) => fetchApi<{ bundle: any; products: any[] }>(`/api/bundles/${id}`),

  getVendorBundles: () => fetchApi<{ bundles: any[] }>('/api/vendor/bundles'),

  createBundle: (data: { title: string; description?: string; price_cfa: number; product_ids: string[] }) =>
    fetchApi<{ bundle: any }>('/api/vendor/bundles', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  deleteBundle: (id: string) => fetchApi<void>(`/api/vendor/bundles/${id}`, { method: 'DELETE' }),

  // Vendor
  getVendorProducts: () => fetchApi<{ products: any[] }>('/api/vendor/products'),

  getVendorSales: () => fetchApi<{ sales: any[] }>('/api/vendor/sales'),

  // Relance par email d'un acheteur dont la commande est restée "en attente"
  remindVendorSale: (id: string) =>
    fetchApi<{ ok: boolean }>(`/api/vendor/sales/${id}/remind`, { method: 'POST' }),

  // "Dernières mises à jour" — annonces admin visibles sur le dashboard vendeur
  getVendorAnnouncements: () =>
    fetchApi<{ announcements: { id: string; title: string; body: string; created_at: string }[] }>(
      '/api/vendor/announcements'
    ),

  getAdminAnnouncements: () =>
    fetchApi<{ announcements: { id: string; title: string; body: string; created_at: string }[] }>(
      '/api/admin/announcements'
    ),

  createAnnouncement: (data: { title: string; body: string }) =>
    fetchApi<{ announcement: { id: string; title: string; body: string; created_at: string } }>(
      '/api/admin/announcements',
      { method: 'POST', body: JSON.stringify(data) }
    ),

  deleteAnnouncement: (id: string) =>
    fetchApi<{ status: string }>(`/api/admin/announcements/${id}`, { method: 'DELETE' }),

  // Notifications in-app
  getNotifications: () => fetchApi<{ notifications: any[] }>('/api/notifications'),

  getUnreadNotificationCount: () => fetchApi<{ count: number }>('/api/notifications/unread-count'),

  markNotificationRead: (id: string) =>
    fetchApi<{ status: string }>(`/api/notifications/${id}/read`, { method: 'PUT' }),

  markAllNotificationsRead: () =>
    fetchApi<{ status: string }>('/api/notifications/read-all', { method: 'PUT' }),

  uploadFile: (formData: FormData) =>
    fetchApi<{ file_key: string; size: string }>('/api/vendor/products/upload', {
      method: 'POST',
      body: formData as any,
      headers: {},
    }),

  getVendorEarnings: () =>
    fetchApi<{ total_earned: number; available: number; pending: number; history: any[]; tier?: string }>(
      '/api/vendor/earnings'
    ),

  // Limites min/max réelles par opérateur PawaPay pour les versements
  // (source : Active Configuration PawaPay, mis en cache côté serveur).
  getPayoutLimits: () =>
    fetchApi<{ limits: Record<string, { min: number; max: number }> }>('/api/vendor/payout-limits'),

  requestPayout: (amount: number) =>
    fetchApi<{ payout: any }>('/api/vendor/payouts', {
      method: 'POST',
      body: JSON.stringify({ amount }),
    }),

  getPayoutMethod: () =>
    fetchApi<{
      payout_method: {
        active_channel: 'mobile_money' | 'paypal';
        phone: string | null;
        operator: string | null;
        operator_label: string;
        country: string | null;
        paypal_email: string | null;
      };
    }>('/api/vendor/payout-method'),

  // Canal "mobile_money" : phone + operator + country. Canal "paypal" :
  // paypal_email. Les deux coexistent, PayPal prioritaire au règlement.
  setPayoutMethod: (
    data:
      | { channel: 'mobile_money'; phone: string; operator: string; country: string }
      | { channel: 'paypal'; paypal_email: string }
  ) =>
    fetchApi<{ payout_method: Record<string, unknown> }>('/api/vendor/payout-method', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Compte (libre-service, tout utilisateur connecté)
  addRole: (role: string) =>
    fetchApi<{ user: any }>('/api/account/roles', {
      method: 'POST',
      body: JSON.stringify({ role }),
    }),

  // phone : numéro du compte (distinct du numéro du moyen de retrait). Le
  // fournir remet phone_verified_at à NULL côté backend s'il change — il faut
  // alors le re-vérifier via PhoneVerifyForm avant tout versement.
  updateProfile: (data: { display_name?: string; shop_name?: string; phone?: string }) =>
    fetchApi<{ user: any }>('/api/account/profile', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  updateAdTracking: (data: { facebook_pixel_id?: string; google_tag_id?: string }) =>
    fetchApi<{ user: any }>('/api/account/ad-tracking', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Moyen de versement/retrait — accessible à tout utilisateur connecté
  // (client ou vendeur), contrairement à /api/vendor/payout-method.
  getAccountPayoutMethod: () =>
    fetchApi<{
      payout_method: {
        active_channel: 'mobile_money' | 'paypal';
        phone: string | null;
        operator: string | null;
        operator_label: string;
        country: string | null;
        paypal_email: string | null;
      };
    }>('/api/account/payout-method'),

  setAccountPayoutMethod: (
    data:
      | { channel: 'mobile_money'; phone: string; operator: string; country: string }
      | { channel: 'paypal'; paypal_email: string }
  ) =>
    fetchApi<{ payout_method: Record<string, unknown> }>('/api/account/payout-method', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Closer (affiliation)
  getCloserLinks: () => fetchApi<{ links: any[] }>('/api/closer/links'),

  createReferralLink: (data: { product_id: string; commission_pct: number }) =>
    fetchApi<{ link: any }>('/api/closer/links', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Admin
  getPendingProducts: () => fetchApi<{ products: any[] }>('/api/admin/products/pending'),

  getAdminProducts: (status?: string) =>
    fetchApi<{ products: any[] }>(`/api/admin/products${status ? `?status=${status}` : ''}`),

  moderateProduct: (id: string, data: { status: string; note?: string }) =>
    fetchApi<void>(`/api/admin/products/${id}/moderate`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  adminListPendingEvents: () => fetchApi<{ events: any[] }>('/api/admin/events/pending'),

  adminModerateEvent: (id: string, data: { status: string; note?: string }) =>
    fetchApi<void>(`/api/admin/events/${id}/moderate`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Sponsors DIARRA Summit — lecture publique, gestion admin.
  getSummitSponsors: () => fetchApi<{ tiers: any[]; sponsors: any[] }>('/api/summit/sponsors', { skipAuth: true }),

  adminListSponsorTiers: () => fetchApi<{ tiers: any[] }>('/api/admin/summit/sponsor-tiers'),

  adminCreateSponsorTier: (data: { name: string; price_cfa: number; perks: string[]; highlight?: boolean; sort_order?: number }) =>
    fetchApi<any>('/api/admin/summit/sponsor-tiers', { method: 'POST', body: JSON.stringify(data) }),

  adminUpdateSponsorTier: (
    id: string,
    data: { name?: string; price_cfa?: number; perks?: string[]; highlight?: boolean; sort_order?: number }
  ) => fetchApi<any>(`/api/admin/summit/sponsor-tiers/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

  adminDeleteSponsorTier: (id: string) =>
    fetchApi<void>(`/api/admin/summit/sponsor-tiers/${id}`, { method: 'DELETE' }),

  adminListSponsors: () => fetchApi<{ sponsors: any[] }>('/api/admin/summit/sponsors'),

  adminCreateSponsor: (data: {
    tier_id?: string;
    name: string;
    logo_key?: string;
    website_url?: string;
    published?: boolean;
    sort_order?: number;
  }) => fetchApi<any>('/api/admin/summit/sponsors', { method: 'POST', body: JSON.stringify(data) }),

  adminUpdateSponsor: (
    id: string,
    data: { tier_id?: string; name?: string; logo_key?: string; website_url?: string; published?: boolean; sort_order?: number }
  ) => fetchApi<any>(`/api/admin/summit/sponsors/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

  adminDeleteSponsor: (id: string) => fetchApi<void>(`/api/admin/summit/sponsors/${id}`, { method: 'DELETE' }),

  adminUploadSponsorLogo: (formData: FormData) =>
    fetchApi<{ file_key: string; size: string }>('/api/admin/summit/sponsors/upload', {
      method: 'POST',
      body: formData as any,
      headers: {},
    }),

  // Renvoie une URL signée de courte durée vers le fichier livrable du
  // produit, pour le vérifier avant de le modérer.
  getAdminProductDownloadUrl: (id: string) =>
    fetchApi<{ signed_url: string }>(`/api/admin/products/${id}/download`),

  confirmProductDeletion: (id: string) =>
    fetchApi<{ status: string }>(`/api/admin/products/${id}`, { method: 'DELETE' }),

  cancelProductDeletion: (id: string) =>
    fetchApi<{ product: any }>(`/api/admin/products/${id}/cancel-deletion`, { method: 'PUT' }),

  attachProductFile: (id: string, file: File) => {
    const form = new FormData();
    form.append('file', file);
    return fetchApi<{ product: any }>(`/api/admin/products/${id}/file`, {
      method: 'PUT',
      body: form as any,
      headers: {},
    });
  },

  // country: ISO3 (ex "SEN"), "UNKNOWN" (pays non déterminé), ou omis = tous.
  // Le pays est déduit de l'indicatif du téléphone côté backend.
  getUsers: (country?: string) =>
    fetchApi<{ users: any[] }>(`/api/admin/users${country ? `?country=${encodeURIComponent(country)}` : ''}`),

  // Répartition des comptes par pays (en-tête de la vue « Utilisateurs par pays »
  // + options du message groupé).
  getUsersByCountry: () =>
    fetchApi<{
      countries: {
        country: string;
        country_label: string;
        users: number;
        vendors: number;
        with_phone: number;
      }[];
    }>('/api/admin/users/by-country'),

  setRole: (id: string, role: string, action: 'grant' | 'revoke') =>
    fetchApi<void>(`/api/admin/users/${id}/role`, {
      method: 'PUT',
      body: JSON.stringify({ role, action }),
    }),

  suspendUser: (id: string) =>
    fetchApi<void>(`/api/admin/users/${id}/suspend`, { method: 'PUT' }),

  reactivateUser: (id: string) =>
    fetchApi<void>(`/api/admin/users/${id}/reactivate`, { method: 'PUT' }),

  // Gestion des droits admin (réservé aux admins à accès complet)
  getAdmins: () => fetchApi<{ admins: any[] }>('/api/admin/admins'),

  setAdminStatus: (id: string, action: 'grant' | 'revoke') =>
    fetchApi<void>(`/api/admin/users/${id}/admin`, {
      method: 'PUT',
      body: JSON.stringify({ action }),
    }),

  setAdminPermission: (id: string, permission: string, action: 'grant' | 'revoke') =>
    fetchApi<void>(`/api/admin/admins/${id}/permission`, {
      method: 'PUT',
      body: JSON.stringify({ permission, action }),
    }),

  // Clé d'automatisation (création de produit via IA/script)
  getAutomationKey: () => fetchApi<{ key: string }>('/api/admin/automation/key'),

  regenerateAutomationKey: () =>
    fetchApi<{ key: string }>('/api/admin/automation/key/regenerate', { method: 'POST' }),

  // Passerelle de paiement — clients externes (ex. ABMCY Core) qui appellent
  // /api/gateway/v1/* . api_key + hmac_secret ne sont renvoyés QU'À la création.
  getGatewayClients: () =>
    fetchApi<{ clients: any[] }>('/api/admin/gateway/clients'),

  createGatewayClient: (name: string, defaultCallbackUrl?: string) =>
    fetchApi<{ client: any; api_key: string; hmac_secret: string }>('/api/admin/gateway/clients', {
      method: 'POST',
      body: JSON.stringify({ name, default_callback_url: defaultCallbackUrl || undefined }),
    }),

  setGatewayClientActive: (id: string, active: boolean) =>
    fetchApi<{ ok: boolean }>(`/api/admin/gateway/clients/${id}/active`, {
      method: 'PUT',
      body: JSON.stringify({ active }),
    }),

  // Plafonds de versement (voir migration 031) — sans eux, une clé API +
  // secret HMAC compromis pouvait vider tout le solde PawaPay de DIARRA en
  // un seul appel POST /api/gateway/v1/payouts (audit sécurité 2026-09-11).
  setGatewayClientLimits: (id: string, maxPayoutCfa: number, dailyPayoutCapCfa: number) =>
    fetchApi<{ ok: boolean }>(`/api/admin/gateway/clients/${id}/limits`, {
      method: 'PUT',
      body: JSON.stringify({ max_payout_cfa: maxPayoutCfa, daily_payout_cap_cfa: dailyPayoutCapCfa }),
    }),

  getGatewayTransactions: (params?: {
    client_id?: string;
    type?: string;
    status?: string;
    limit?: number;
    offset?: number;
  }) => {
    const q = new URLSearchParams();
    Object.entries(params || {}).forEach(([k, v]) => {
      if (v !== undefined && v !== '') q.set(k, String(v));
    });
    const qs = q.toString();
    return fetchApi<{ transactions: any[] }>(
      `/api/admin/gateway/transactions${qs ? `?${qs}` : ''}`,
    );
  },

  getGatewayStats: (sinceDays: number) =>
    fetchApi<any>(`/api/admin/gateway/stats?since=${sinceDays}`),

  checkGatewayTransaction: (id: string) =>
    fetchApi<{ transaction: any; provider_status?: string }>(
      `/api/admin/gateway/transactions/${id}/check-provider`,
      { method: 'POST' },
    ),

  getSales: () => fetchApi<{ sales: any[] }>('/api/admin/sales'),

  // Commandes non abouties (pending/failed) avec contact acheteur complet
  getPendingSales: () => fetchApi<{ sales: any[] }>('/api/admin/sales/pending'),

  refundSale: (id: string) =>
    fetchApi<{ status: string; refund_id: string }>(`/api/admin/sales/${id}/refund`, {
      method: 'POST',
    }),

  // Relance par email l'acheteur d'une commande "en attente"
  remindSale: (id: string) =>
    fetchApi<{ ok: boolean }>(`/api/admin/sales/${id}/remind`, { method: 'POST' }),

  // Confirme À LA MAIN un paiement (webhook jamais arrivé) : la vente passe
  // "payée", l'acheteur reçoit son fichier, le vendeur est crédité.
  markSalePaid: (id: string) =>
    fetchApi<{ status: string }>(`/api/admin/sales/${id}/mark-paid`, { method: 'POST' }),

  // « Vérifier chez PawaPay » : interroge le prestataire sur le vrai statut du
  // dépôt. Si COMPLETED, la vente est confirmée automatiquement (comme le
  // webhook). Ne force rien si le prestataire ne confirme pas.
  checkSaleProvider: (id: string) =>
    fetchApi<{
      provider: string;
      provider_status: string; // COMPLETED | FAILED | PROCESSING | NOT_FOUND | ...
      provider_transaction_id?: string;
      sale_status: string;
    }>(`/api/admin/sales/${id}/check-provider`, { method: 'POST' }),

  getStats: () =>
    fetchApi<{
      total_sales: number;
      total_products: number;
      pending_moderation: number;
      active_products: number;
      total_users: number;
      total_vendors: number;
      total_closers: number;
      total_admins: number;
      total_revenue: number;
      gmv: number;
      revenue_this_month: number;
      revenue_last_month: number;
      revenue_growth_pct: number;
    }>('/api/admin/stats'),

  getAnalytics: (days?: 7 | 30 | 365) =>
    fetchApi<{
      sales_by_day: { day: string; sales: number; revenue_cfa: number }[];
      top_products: any[];
      top_vendors: any[];
      top_closers: any[];
    }>(`/api/admin/analytics${days ? `?days=${days}` : ''}`),

  getAdminSettings: () => fetchApi<{ settings: Record<string, string> }>('/api/admin/settings'),

  updateAdminSettings: (values: Record<string, string>) =>
    fetchApi<{ settings: Record<string, string> }>('/api/admin/settings', {
      method: 'PUT',
      body: JSON.stringify(values),
    }),

  getAdminPayouts: () => fetchApi<{ payouts: any[] }>('/api/admin/payouts'),

  // Règle AUTOMATIQUEMENT une demande de versement « en attente » : déclenche
  // le versement chez PawaPay/KPay. La demande passe « en traitement ».
  settlePayoutAuto: (id: string) =>
    fetchApi<{ status: string }>(`/api/admin/payouts/${id}/settle-auto`, { method: 'POST' }),

  retryPayout: (id: string) =>
    fetchApi<{ status: string }>(`/api/admin/payouts/${id}/retry`, { method: 'POST' }),

  // Interroge le prestataire sur le vrai statut d'un versement « en traitement »
  // et applique payé/échec en conséquence (comme checkSaleProvider).
  checkPayoutProvider: (id: string) =>
    fetchApi<{ provider: string; provider_status: string; payout_status: string }>(
      `/api/admin/payouts/${id}/check-provider`,
      { method: 'POST' }
    ),

  // Marque un versement "payé" à la main (argent envoyé hors PawaPay/KPay).
  settlePayoutManual: (id: string, data: { note: string; fee_cfa: number }) =>
    fetchApi<{ status: string }>(`/api/admin/payouts/${id}/settle-manual`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Reconnaît qu'un versement déjà payé (ou échoué) a été remboursé — l'argent
  // a été rendu hors plateforme. Le solde disponible du vendeur redescend
  // aussitôt (ce montant sort de "requested", voir PayoutHandler.Earnings).
  refundPayout: (id: string, note?: string) =>
    fetchApi<{ status: string }>(`/api/admin/payouts/${id}/refund`, {
      method: 'POST',
      body: JSON.stringify({ note: note || '' }),
    }),

  // Crée un versement manuel de toutes pièces pour un vendeur.
  createManualPayout: (data: {
    user_id: string;
    amount: number;
    fee_cfa: number;
    phone: string;
    note: string;
  }) =>
    fetchApi<{ payout: any }>('/api/admin/payouts/manual', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Versement direct (argent réellement envoyé via PawaPay ou PayPal, vers
  // n'importe quel destinataire) — step-up OTP obligatoire avant l'envoi.
  // channel: "mobile_money" (défaut) exige country/operator/phone ; "paypal"
  // exige paypal_email.
  sendDirectPayoutOtp: () =>
    fetchApi<{ status: string }>('/api/admin/payouts/send-otp', { method: 'POST' }),

  createDirectPayout: (data: {
    amount_cfa: number;
    channel?: 'mobile_money' | 'paypal';
    country?: string;
    operator?: string;
    phone?: string;
    paypal_email?: string;
    note?: string;
    otp_code: string;
  }) =>
    fetchApi<{ payout: any; status: string }>('/api/admin/payouts/direct', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Programme de reversement automatique ("Fidélisation")
  getDonations: () =>
    fetchApi<{
      pool: { balance_cfa: number; updated_at: string };
      recipients: {
        id: string;
        name: string;
        phone_number: string;
        operator: string;
        country: string;
        active: boolean;
      }[];
      payouts: {
        id: string;
        recipient_id: string;
        recipient_name: string;
        recipient_phone: string;
        amount_cfa: number;
        status: string;
        failure_reason?: string;
        requested_at: string;
        paid_at?: string;
      }[];
      settings: { share_pct: number; threshold_cfa: number; enabled: boolean };
    }>('/api/admin/donations'),

  createDonationRecipient: (input: { name: string; phone: string; operator: string; country: string }) =>
    fetchApi<{ recipient: any }>('/api/admin/donations/recipients', {
      method: 'POST',
      body: JSON.stringify(input),
    }),

  updateDonationRecipient: (id: string, input: { name?: string; active?: boolean }) =>
    fetchApi<{ recipient: any }>(`/api/admin/donations/recipients/${id}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    }),

  deleteDonationRecipient: (id: string) =>
    fetchApi<void>(`/api/admin/donations/recipients/${id}`, { method: 'DELETE' }),

  retryDonationPayout: (id: string) =>
    fetchApi<{ ok: boolean }>(`/api/admin/donations/payouts/${id}/retry`, { method: 'POST' }),

  // Widget de contact support public (visiteur non authentifié)
  submitSupportContact: (input: { name: string; contact_method: 'email' | 'whatsapp'; contact_value: string; message: string }) =>
    fetchApi<{ ok: boolean }>('/api/support/contact', {
      method: 'POST',
      body: JSON.stringify(input),
    }),

  // Inscription publique au DIARRA Summit (26 octobre 2026, en ligne).
  submitSummitRegistration: (input: { full_name: string; email: string; phone: string; profile: 'vendeur' | 'acheteur' | 'entrepreneur' | 'curieux' }) =>
    fetchApi<{ id: string }>('/api/summit/register', {
      method: 'POST',
      body: JSON.stringify(input),
    }),

  // Admin : liste des inscrits au DIARRA Summit.
  getSummitRegistrations: () =>
    fetchApi<{ registrations: any[]; count: number }>('/api/admin/summit/registrations'),

  // Diffusion email (admin, scope "users") : subject + html composés par
  // l'admin. test_only n'envoie qu'à l'admin connecté, pour prévisualiser.
  // country (ISO3 ou "UNKNOWN") restreint la diffusion aux comptes de ce pays
  // (déduit de l'indicatif téléphone) ; un bloc « Rejoindre la communauté
  // WhatsApp » est ajouté en pied si un lien est configuré.
  sendBroadcast: (data: {
    subject: string;
    html: string;
    test_only?: boolean;
    country?: string;
  }) =>
    fetchApi<{ status: string; recipients?: number; to?: string }>('/api/admin/broadcast', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Message direct à un utilisateur précis (admin, scope "users") — ex:
  // expliquer un incident de paiement à un vendeur, sans ticket préalable.
  sendUserMessage: (userId: string, data: { subject: string; message: string }) =>
    fetchApi<{ ok: boolean }>(`/api/admin/users/${userId}/message`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Agents support (admin, tout admin)
  getSupportAgents: () =>
    fetchApi<{
      agents: {
        id: string;
        name: string;
        email: string;
        phone?: string;
        callmebot_apikey?: string;
        active: boolean;
        created_at: string;
      }[];
    }>('/api/admin/support-agents'),

  createSupportAgent: (input: { name: string; email: string; phone?: string; callmebot_apikey?: string }) =>
    fetchApi<{ agent: any }>('/api/admin/support-agents', {
      method: 'POST',
      body: JSON.stringify(input),
    }),

  updateSupportAgent: (
    id: string,
    input: { name?: string; email?: string; phone?: string; callmebot_apikey?: string; active?: boolean }
  ) =>
    fetchApi<{ agent: any }>(`/api/admin/support-agents/${id}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    }),

  deleteSupportAgent: (id: string) =>
    fetchApi<void>(`/api/admin/support-agents/${id}`, { method: 'DELETE' }),

  getSupportContacts: () =>
    fetchApi<{
      requests: { id: string; name: string; contact_method: string; contact_value: string; message: string; created_at: string }[];
    }>('/api/admin/support-contacts'),

  getActivityFeed: () =>
    fetchApi<{ activity: { kind: string; id: string; at: string; data: any }[] }>(
      '/api/admin/activity'
    ),

  // Journal des actions admin (backoffice 360°) — distinct de getActivityFeed
  // ci-dessus (événements plateforme sans auteur).
  getActivityLog: (page = 1) =>
    fetchApi<{
      logs: {
        id: string;
        admin_id?: string;
        admin_email?: string;
        action: string;
        target_type: string;
        target_id?: string;
        description: string;
        created_at: string;
      }[];
      total: number;
      page: number;
      per_page: number;
    }>(`/api/admin/activity-log?page=${page}`),

  getAdminNotifications: () =>
    fetchApi<{
      pending_moderation: number;
      open_tickets: number;
      failed_sales_24h: number;
      total: number;
    }>('/api/admin/notifications'),

  // Support
  createTicket: (data: { sale_id?: string; subject: string; message: string }) =>
    fetchApi<{ ticket: any }>('/api/tickets', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  getTickets: () => fetchApi<{ tickets: any[] }>('/api/tickets'),

  getTicketMessages: (id: string) =>
    fetchApi<{ ticket: any; messages: any[] }>(`/api/tickets/${id}/messages`),

  addTicketMessage: (id: string, data: { body: string }) =>
    fetchApi<{ message: any }>(`/api/tickets/${id}/messages`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  claimTicket: (id: string) =>
    fetchApi<{ ticket: any }>(`/api/admin/tickets/${id}/claim`, { method: 'PUT' }),

  assignTicket: (id: string, adminId: string) =>
    fetchApi<{ ticket: any }>(`/api/admin/tickets/${id}/assign`, {
      method: 'PUT',
      body: JSON.stringify({ admin_id: adminId }),
    }),

  getTicketAssignees: () => fetchApi<{ admins: any[] }>('/api/admin/tickets/assignees'),
};

export { ApiError };
