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
  sendOtp: (channel: 'email' | 'sms') =>
    fetchApi<{ status: string; channel: string; dev_code?: string }>('/api/auth/send-otp', {
      method: 'POST',
      body: JSON.stringify({ channel }),
    }),

  verifyOtp: (channel: 'email' | 'sms', code: string) =>
    fetchApi<{ status: string; channel: string }>('/api/auth/verify-otp', {
      method: 'POST',
      body: JSON.stringify({ channel, code }),
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
    fetchApi<{ product: any }>(`/api/vendor/products/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  deleteProduct: (id: string) =>
    fetchApi<{ status: string }>(`/api/vendor/products/${id}`, { method: 'DELETE' }),

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
    fetchApi<{ payout_method: { phone: string | null; operator: string | null; operator_label: string; country: string | null } }>(
      '/api/vendor/payout-method'
    ),

  setPayoutMethod: (data: { phone: string; operator: string; country: string }) =>
    fetchApi<{ payout_method: { phone: string; operator: string; operator_label: string; country: string } }>(
      '/api/vendor/payout-method',
      { method: 'PUT', body: JSON.stringify(data) }
    ),

  // Compte (libre-service, tout utilisateur connecté)
  addRole: (role: string) =>
    fetchApi<{ user: any }>('/api/account/roles', {
      method: 'POST',
      body: JSON.stringify({ role }),
    }),

  updateProfile: (data: { display_name?: string; shop_name?: string }) =>
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
    fetchApi<{ payout_method: { phone: string | null; operator: string | null; operator_label: string; country: string | null } }>(
      '/api/account/payout-method'
    ),

  setAccountPayoutMethod: (data: { phone: string; operator: string; country: string }) =>
    fetchApi<{ payout_method: { phone: string; operator: string; operator_label: string; country: string } }>(
      '/api/account/payout-method',
      { method: 'PUT', body: JSON.stringify(data) }
    ),

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

  getUsers: () => fetchApi<{ users: any[] }>('/api/admin/users'),

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

  getSales: () => fetchApi<{ sales: any[] }>('/api/admin/sales'),

  refundSale: (id: string) =>
    fetchApi<{ status: string; refund_id: string }>(`/api/admin/sales/${id}/refund`, {
      method: 'POST',
    }),

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

  retryPayout: (id: string) =>
    fetchApi<{ status: string }>(`/api/admin/payouts/${id}/retry`, { method: 'POST' }),

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
