'use client';

import { createContext, useContext, useEffect, useState, useCallback } from 'react';
import { api } from './api';

export interface User {
  id: string;
  email: string;
  phone?: string | null;
  is_admin: boolean;
  roles?: string[];
  email_verified_at?: string | null;
  phone_verified_at?: string | null;
}

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  isAdmin: boolean;
  hasRole: (role: string) => boolean;
  login: (accessToken: string) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<boolean>;
}

const AuthContext = createContext<AuthContextValue>({
  user: null,
  loading: true,
  isAdmin: false,
  hasRole: () => false,
  login: async () => {},
  logout: async () => {},
  refresh: async () => false,
});

const ACCESS_TOKEN_KEY = 'access_token';

// Restaure la session : access token en localStorage (court), sinon on tente
// un refresh via le cookie httpOnly posé par le backend.
async function restoreSession(): Promise<User | null> {
  const token = typeof window !== 'undefined' ? localStorage.getItem(ACCESS_TOKEN_KEY) : null;
  if (token) {
    try {
      const result = await api.getMe();
      return result.user;
    } catch {
      localStorage.removeItem(ACCESS_TOKEN_KEY);
      // cookie encore valide → on retente par refresh
    }
  }
  try {
    const result = await api.refreshToken();
    localStorage.setItem(ACCESS_TOKEN_KEY, result.access_token);
    const me = await api.getMe();
    return me.user;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const loadMe = useCallback(async () => {
    const u = await restoreSession();
    if (!u) localStorage.removeItem(ACCESS_TOKEN_KEY);
    setUser(u);
    setLoading(false);
  }, []);

  useEffect(() => {
    loadMe();
  }, [loadMe]);

  const login = useCallback(async (accessToken: string) => {
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
    const me = await api.getMe();
    setUser(me.user);
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.logout();
    } catch {
      // la révocation serveur échoue éventuellement : on déconnecte quand même
    }
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    setUser(null);
  }, []);

  const refresh = useCallback(async () => {
    try {
      const result = await api.refreshToken();
      localStorage.setItem(ACCESS_TOKEN_KEY, result.access_token);
      const me = await api.getMe();
      setUser(me.user);
      return true;
    } catch {
      localStorage.removeItem(ACCESS_TOKEN_KEY);
      setUser(null);
      return false;
    }
  }, []);

  const hasRole = useCallback((role: string) => user?.roles?.includes(role) ?? false, [user]);

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        isAdmin: user?.is_admin ?? false,
        hasRole,
        login,
        logout,
        refresh,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
