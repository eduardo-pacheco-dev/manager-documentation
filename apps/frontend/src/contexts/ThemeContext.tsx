import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { Theme } from '@carbon/react';
import { api } from '../lib/api';
import { useAuth } from './AuthContext';
import type { ThemePreference } from '../lib/types';

interface ThemeContextValue {
  preference: ThemePreference;
  effective: 'light' | 'dark';
  carbonTheme: 'g10' | 'g100';
  setPreference: (preference: ThemePreference, persist?: boolean) => Promise<void>;
  cyclePreference: () => Promise<void>;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() =>
    typeof window === 'undefined' ? false : window.matchMedia(query).matches,
  );
  useEffect(() => {
    const media = window.matchMedia(query);
    const listener = (event: MediaQueryListEvent) => setMatches(event.matches);
    setMatches(media.matches);
    media.addEventListener('change', listener);
    return () => media.removeEventListener('change', listener);
  }, [query]);
  return matches;
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const { user, patchUser } = useAuth();
  const [override, setOverride] = useState<ThemePreference | null>(null);
  const systemDark = useMediaQuery('(prefers-color-scheme: dark)');

  const preference = override ?? user?.theme ?? 'system';
  const effective: 'light' | 'dark' =
    preference === 'system' ? (systemDark ? 'dark' : 'light') : preference;
  const carbonTheme = effective === 'dark' ? 'g100' : 'g10';

  useEffect(() => {
    document.documentElement.dataset.theme = effective;
    document.documentElement.style.colorScheme = effective;
  }, [effective]);

  useEffect(() => {
    if (!user) setOverride(null);
  }, [user]);

  const setPreference = useCallback(
    async (next: ThemePreference, persist = true) => {
      setOverride(next);
      if (persist && user) {
        try {
          await api.put<{ data: { theme: ThemePreference } }>('/settings/theme', { theme: next });
          patchUser({ theme: next });
        } catch {
          // Mantém a preferência visual local; a próxima sincronização corrige.
        }
      }
    },
    [patchUser, user],
  );

  const cyclePreference = useCallback(async () => {
    const sequence: ThemePreference[] = ['system', 'light', 'dark'];
    const next = sequence[(sequence.indexOf(preference) + 1) % sequence.length];
    await setPreference(next);
  }, [preference, setPreference]);

  const value = useMemo<ThemeContextValue>(
    () => ({ preference, effective, carbonTheme, setPreference, cyclePreference }),
    [preference, effective, carbonTheme, setPreference, cyclePreference],
  );

  return (
    <ThemeContext.Provider value={value}>
      <Theme theme={carbonTheme}>{children}</Theme>
    </ThemeContext.Provider>
  );
}

export function useAppTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) throw new Error('useAppTheme must be used inside ThemeProvider');
  return context;
}
