import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { api, getCompanyHeader, setCSRFToken, setCompanyHeader } from '../lib/api';
import { DEFAULT_COMPANY_SLUG, isAdmin, isSuperAdmin as hasSuperAdminRole } from '../lib/constants';
import type { Company, Module, User } from '../lib/types';

interface RegisterPayload {
  name: string;
  surname?: string;
  email: string;
  password: string;
  password_confirmation: string;
}

interface AuthContextValue {
  user: User | null;
  company: Company | null;
  activeCompanyID: number;
  modules: Module[];
  loading: boolean;
  unread: number;
  isSuperAdmin: boolean;
  isCompanyAdmin: boolean;
  canManageUsers: boolean;
  canManageCompanies: boolean;
  login: (company: string, email: string, password: string, remember?: boolean) => Promise<void>;
  register: (company: string, payload: RegisterPayload) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<boolean>;
  refreshUnread: () => Promise<void>;
  setActiveCompany: (identifier: string | number) => Promise<void>;
  patchUser: (partial: Partial<User>) => void;
}

interface AuthPayload {
  user: User;
  company?: Company | null;
  modules: Module[];
  csrf_token?: string;
  token?: string;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [company, setCompany] = useState<Company | null>(null);
  const [activeCompanyID, setActiveCompanyID] = useState(0);
  const [modules, setModules] = useState<Module[]>([]);
  const [loading, setLoading] = useState(true);
  const [unread, setUnread] = useState(0);

  const clearSession = useCallback(() => {
    setUser(null);
    setCompany(null);
    setActiveCompanyID(0);
    setModules([]);
    setUnread(0);
    setCSRFToken('');
    setCompanyHeader('');
  }, []);

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
      setCompany(payload.company ?? null);
      setActiveCompanyID(payload.company?.id ?? payload.user.company_id ?? 0);
      setModules(payload.modules ?? []);
      setCSRFToken(payload.csrf_token);
      if (payload.company?.slug || payload.company?.id) {
        setCompanyHeader(payload.company.slug || payload.company.id);
      }
      void refreshUnread();
    },
    [refreshUnread],
  );

  const refresh = useCallback(async () => {
    try {
      const response = await api.get<{ data: AuthPayload }>('/auth/me');
      applySession(response.data);
      return true;
    } catch {
      clearSession();
      return false;
    } finally {
      setLoading(false);
    }
  }, [applySession, clearSession]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    window.addEventListener('auth:expired', clearSession);
    const onNotificationsChanged = () => {
      void refreshUnread();
    };
    window.addEventListener('notifications:changed', onNotificationsChanged);
    return () => {
      window.removeEventListener('auth:expired', clearSession);
      window.removeEventListener('notifications:changed', onNotificationsChanged);
    };
  }, [clearSession, refreshUnread]);

  const login = useCallback(
    async (company: string, email: string, password: string, remember = false) => {
      setCompanyHeader('');
      const response = await api.post<{ data: AuthPayload }>('/auth/login', {
        company: company.trim() || DEFAULT_COMPANY_SLUG,
        email,
        password,
        remember,
      });
      applySession(response.data);
    },
    [applySession],
  );

  const register = useCallback(
    async (company: string, payload: RegisterPayload) => {
      setCompanyHeader('');
      const response = await api.post<{ data: AuthPayload }>('/auth/signup', {
        ...payload,
        company: company.trim() || DEFAULT_COMPANY_SLUG,
      });
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
      clearSession();
    }
  }, [clearSession]);

  const setActiveCompany = useCallback(
    async (identifier: string | number) => {
      const previous = getCompanyHeader();
      setCompanyHeader(identifier);
      if (await refresh()) return;
      setCompanyHeader(previous);
      await refresh();
      throw new Error('Não foi possível acessar esta empresa.');
    },
    [refresh],
  );

  const patchUser = useCallback((partial: Partial<User>) => {
    setUser((current) => (current ? { ...current, ...partial } : current));
  }, []);

  const isSuperAdmin = hasSuperAdminRole(user);
  const isCompanyAdmin = isAdmin(user);
  const canManageUsers = isCompanyAdmin;
  const canManageCompanies = isSuperAdmin;

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      company,
      activeCompanyID,
      modules,
      loading,
      unread,
      isSuperAdmin,
      isCompanyAdmin,
      canManageUsers,
      canManageCompanies,
      login,
      register,
      logout,
      refresh,
      refreshUnread,
      setActiveCompany,
      patchUser,
    }),
    [
      user,
      company,
      activeCompanyID,
      modules,
      loading,
      unread,
      isSuperAdmin,
      isCompanyAdmin,
      canManageUsers,
      canManageCompanies,
      login,
      register,
      logout,
      refresh,
      refreshUnread,
      setActiveCompany,
      patchUser,
    ],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used inside AuthProvider');
  return context;
}
