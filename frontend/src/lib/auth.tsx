'use client';

import { createContext, useContext, useEffect, useState, useCallback } from 'react';
import { api } from './api';

export interface User {
  id: string;
  email: string;
  phone?: string | null;
  is_admin: boolean;
  roles?: string[];
}

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  isAdmin: boolean;
  hasRole: (role: string) => boolean;
  login: (accessToken: string, refreshToken: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue>({
  user: null,
  loading: true,
  isAdmin: false,
  hasRole: () => false,
  login: async () => {},
  logout: async () => {},
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const loadMe = useCallback(async () => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;
    if (!token) {
      setUser(null);
      setLoading(false);
      return;
    }
    try {
      const result = await api.getMe();
      setUser(result.user);
    } catch {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadMe();
  }, [loadMe]);

  const login = useCallback(async (accessToken: string, refreshToken: string) => {
    localStorage.setItem('access_token', accessToken);
    localStorage.setItem('refresh_token', refreshToken);
    await loadMe();
  }, [loadMe]);

  const logout = useCallback(async () => {
    try {
      await api.logout();
    } catch {
      // la révocation serveur échoue éventuellement : on déconnecte quand même
    }
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    setUser(null);
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
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
