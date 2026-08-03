const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

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

  const headers: HeadersInit = {
    'Content-Type': 'application/json',
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
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new ApiError(response.status, error.error || 'Request failed');
  }

  return response.json();
}

export const api = {
  // Auth
  register: (data: { email: string; password: string; phone?: string }) =>
    fetchApi<{ user: any; access_token: string; refresh_token: string }>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
      skipAuth: true,
    }),

  login: (data: { email: string; password: string }) =>
    fetchApi<{ access_token: string; refresh_token: string }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
      skipAuth: true,
    }),

  logout: () => fetchApi<void>('/api/auth/logout', { method: 'POST' }),

  refreshToken: (refreshToken: string) =>
    fetchApi<{ access_token: string }>('/api/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
      skipAuth: true,
    }),

  forgotPassword: (email: string) =>
    fetchApi<{ message: string }>('/api/auth/forgot-password', {
      method: 'POST',
      body: JSON.stringify({ email }),
      skipAuth: true,
    }),

  // Products
  getProducts: (params?: { category?: string; search?: string }) => {
    const query = new URLSearchParams(params as Record<string, string>).toString();
    return fetchApi<{ products: any[] }>(`/api/products${query ? `?${query}` : ''}`);
  },

  getProduct: (id: string) => fetchApi<{ product: any }>(`/api/products/${id}`),

  createProduct: (data: FormData) =>
    fetchApi<{ product: any }>('/api/vendor/products', {
      method: 'POST',
      body: data as any,
      headers: {}, // Let browser set Content-Type for multipart
    }),

  updateProduct: (id: string, data: FormData) =>
    fetchApi<{ product: any }>(`/api/vendor/products/${id}`, {
      method: 'PUT',
      body: data as any,
      headers: {},
    }),

  deleteProduct: (id: string) =>
    fetchApi<void>(`/api/vendor/products/${id}`, { method: 'DELETE' }),

  // Orders
  createOrder: (data: { product_id: string }) =>
    fetchApi<{ order: any; payment_url: string }>('/api/orders', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  getOrder: (id: string) => fetchApi<{ order: any }>(`/api/orders/${id}`),

  getOrders: () => fetchApi<{ orders: any[] }>('/api/orders'),

  // Delivery
  getDelivery: (orderId: string) =>
    fetchApi<{ delivery: any; signed_url: string }>(`/api/orders/${orderId}/delivery`, {
      method: 'POST',
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

  // Admin
  getPendingProducts: () => fetchApi<{ products: any[] }>('/api/admin/products/pending'),

  moderateProduct: (id: string, data: { status: string; note?: string }) =>
    fetchApi<void>(`/api/admin/products/${id}/moderate`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  getUsers: () => fetchApi<{ users: any[] }>('/api/admin/users'),

  suspendUser: (id: string) =>
    fetchApi<void>(`/api/admin/users/${id}/suspend`, { method: 'PUT' }),

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
