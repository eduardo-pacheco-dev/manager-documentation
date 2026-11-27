import type { ReactNode } from 'react';

type PublicShellProps = {
  meta?: string;
  children: ReactNode;
};

export function PublicShell({ meta, children }: PublicShellProps) {
  return (
    <div className="public-shell">
      <header className="public-header">
        <div className="public-brand">
          <span className="public-brand__mark">DC</span>
          Documentation Center
        </div>
        {meta && <span className="public-header__meta">{meta}</span>}
      </header>
      <main className="public-main">{children}</main>
      <footer className="public-footer">
        <span>Documentation Center</span>
        <span>Powered by IBM Carbon Design System</span>
      </footer>
    </div>
  );
}
