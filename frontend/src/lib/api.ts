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

  createProduct: (data: {
    title: string;
    description: string;
    price_cfa: number;
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
      category?: string;
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
  }) =>
    fetchApi<{ order: any; checkout: { deposit_id: string; redirect_url: string } }>('/api/orders', {
      method: 'POST',
      body: JSON.stringify(data),
      skipAuth: true,
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

  // Vendor
  getVendorProducts: () => fetchApi<{ products: any[] }>('/api/vendor/products'),

  uploadFile: (formData: FormData) =>
    fetchApi<{ file_key: string; size: string }>('/api/vendor/products/upload', {
      method: 'POST',
      body: formData as any,
      headers: {},
    }),

  getVendorEarnings: () =>
    fetchApi<{ total_earned: number; available: number; pending: number; history: any[] }>('/api/vendor/earnings'),

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

  // Closer (affiliation)
  getCloserLinks: () => fetchApi<{ links: any[] }>('/api/closer/links'),

  createReferralLink: (data: { product_id: string; commission_pct: number }) =>
    fetchApi<{ link: any }>('/api/closer/links', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Admin
  getPendingProducts: () => fetchApi<{ products: any[] }>('/api/admin/products/pending'),

  moderateProduct: (id: string, data: { status: string; note?: string }) =>
    fetchApi<void>(`/api/admin/products/${id}/moderate`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  getUsers: () => fetchApi<{ users: any[] }>('/api/admin/users'),

  setRole: (id: string, role: string, action: 'grant' | 'revoke') =>
    fetchApi<void>(`/api/admin/users/${id}/role`, {
      method: 'PUT',
      body: JSON.stringify({ role, action }),
    }),

  suspendUser: (id: string) =>
    fetchApi<void>(`/api/admin/users/${id}/suspend`, { method: 'PUT' }),

  getSales: () => fetchApi<{ sales: any[] }>('/api/admin/sales'),

  getStats: () =>
    fetchApi<{
      total_sales: number;
      total_users: number;
      total_products: number;
      total_revenue: number;
      pending_moderation: number;
    }>('/api/admin/stats'),

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
};

export { ApiError };
