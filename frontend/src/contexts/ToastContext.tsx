import { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { ToastNotification } from '@carbon/react';

type ToastKind = 'success' | 'error' | 'info' | 'warning';

interface ToastItem {
  id: number;
  kind: ToastKind;
  title: string;
  subtitle?: string;
}

interface ToastContextValue {
  show: (kind: ToastKind, title: string, subtitle?: string) => void;
  success: (title: string, subtitle?: string) => void;
  error: (title: string, subtitle?: string) => void;
  info: (title: string, subtitle?: string) => void;
  warning: (title: string, subtitle?: string) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);
  const counter = useRef(0);

  const remove = useCallback((id: number) => {
    setItems((current) => current.filter((item) => item.id !== id));
  }, []);

  const show = useCallback(
    (kind: ToastKind, title: string, subtitle?: string) => {
      const id = ++counter.current;
      setItems((current) => [...current.slice(-4), { id, kind, title, subtitle }]);
      window.setTimeout(() => remove(id), 5500);
    },
    [remove],
  );

  const value = useMemo<ToastContextValue>(
    () => ({
      show,
      success: (title, subtitle) => show('success', title, subtitle),
      error: (title, subtitle) => show('error', title, subtitle),
      info: (title, subtitle) => show('info', title, subtitle),
      warning: (title, subtitle) => show('warning', title, subtitle),
    }),
    [show],
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-stack" aria-live="polite" aria-atomic="false">
        {items.map((item) => (
          <ToastNotification
            key={item.id}
            kind={item.kind}
            title={item.title}
            subtitle={item.subtitle}
            lowContrast
            onClose={() => remove(item.id)}
            className="toast-item"
          />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const context = useContext(ToastContext);
  if (!context) throw new Error('useToast must be used inside ToastProvider');
  return context;
}
