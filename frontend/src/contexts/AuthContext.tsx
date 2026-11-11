import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { api, setCSRFToken } from '../lib/api';
import type { Module, User } from '../lib/types';

interface AuthContextValue {
  user: User | null;
  modules: Module[];
  loading: boolean;
  unread: number;
  login: (email: string, password: string, remember?: boolean) => Promise<void>;
  register: (payload: {
    name: string;
    email: string;
    password: string;
    password_confirmation: string;
  }) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
  refreshUnread: () => Promise<void>;
  patchUser: (partial: Partial<User>) => void;
}

interface AuthPayload {
  user: User;
  modules: Module[];
  csrf_token?: string;
  token?: string;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [modules, setModules] = useState<Module[]>([]);
  const [loading, setLoading] = useState(true);
  const [unread, setUnread] = useState(0);

  const refreshUnread = useCallback(async () => {
    try {
      const response = await api.get<{ unread: number }>('/notifications?per_page=1');
      setUnread(response.unread);
    } catch {
      // A sessão pode ter expirado; o interceptor da API cuida disso.
    }
  }, []);

  const applySession = useCallback(
    (payload: AuthPayload) => {
      setUser(payload.user);
      setModules(payload.modules ?? []);
      setCSRFToken(payload.csrf_token);
      void refreshUnread();
    },
    [refreshUnread],
  );

  const refresh = useCallback(async () => {
    try {
      const response = await api.get<{ data: AuthPayload }>('/auth/me');
      applySession(response.data);
    } catch {
      setUser(null);
      setModules([]);
      setCSRFToken('');
    } finally {
      setLoading(false);
    }
  }, [applySession]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    const onExpired = () => {
      setUser(null);
      setModules([]);
      setCSRFToken('');
      setUnread(0);
    };
    const onNotificationsChanged = () => {
      void refreshUnread();
    };
    window.addEventListener('auth:expired', onExpired);
    window.addEventListener('notifications:changed', onNotificationsChanged);
    return () => {
      window.removeEventListener('auth:expired', onExpired);
      window.removeEventListener('notifications:changed', onNotificationsChanged);
    };
  }, [refreshUnread]);

  const login = useCallback(
    async (email: string, password: string, remember = false) => {
      const response = await api.post<{ data: AuthPayload }>('/auth/login', { email, password, remember });
      applySession(response.data);
    },
    [applySession],
  );

  const register = useCallback(
    async (payload: { name: string; email: string; password: string; password_confirmation: string }) => {
      const response = await api.post<{ data: AuthPayload }>('/auth/signup', payload);
      applySession(response.data);
    },
    [applySession],
  );

  const logout = useCallback(async () => {
    try {
      await api.post('/auth/logout');
    } catch {
      // A sessão já pode ter expirado ou sido removida.
    } finally {
      setUser(null);
      setModules([]);
      setUnread(0);
      setCSRFToken('');
    }
  }, []);

  const patchUser = useCallback((partial: Partial<User>) => {
    setUser((current) => (current ? { ...current, ...partial } : current));
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      modules,
      loading,
      unread,
      login,
      register,
      logout,
      refresh,
      refreshUnread,
      patchUser,
    }),
    [user, modules, loading, unread, login, register, logout, refresh, refreshUnread, patchUser],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used inside AuthProvider');
  return context;
}
