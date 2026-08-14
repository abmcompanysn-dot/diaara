const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Origine de l'API : sert à construire les liens de partage /r/{slug} côté closer.
export const apiOrigin = API_BASE;

interface FetchOptions extends RequestInit {
  skipAuth?: boolean;
}

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

async function fetchApi<T>(endpoint: string, options: FetchOptions = {}): Promise<T> {
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

  getProduct: (id: string) => fetchApi<{ product: any }>(`/api/products/${id}`),

  // Boutique publique d'un vendeur (partageable via QR code)
  getVendorShop: (vendorId: string) =>
    fetchApi<{ vendor: { id: string; display_name: string | null; shop_name: string | null }; products: any[] }>(
      `/api/vendors/${vendorId}/shop`,
      { skipAuth: true }
    ),

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
      affiliate_enabled?: boolean;
      max_closer_commission_pct?: number;
    }
  ) =>
    fetchApi<{ product: any }>(`/api/vendor/products/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  deleteProduct: (id: string) =>
    fetchApi<void>(`/api/vendor/products/${id}`, { method: 'DELETE' }),

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
    fetchApi<{ total_earned: number; available: number; pending: number; history: any[] }>('/api/vendor/earnings'),

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
