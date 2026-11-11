import type { ReactNode } from 'react';

interface PageHeaderProps {
  eyebrow?: string;
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
  titleAccessory?: ReactNode;
}

export function PageHeader({ eyebrow, title, description, actions, titleAccessory }: PageHeaderProps) {
  return (
    <header className="page-header">
      <div className="page-header__text">
        {eyebrow && <p className="page-header__eyebrow">{eyebrow}</p>}
        <div className="page-header__title-row">
          <h1>{title}</h1>
          {titleAccessory && <span className="page-header__title-accessory">{titleAccessory}</span>}
        </div>
        {description && <div className="page-header__description">{description}</div>}
      </div>
      {actions && <div className="page-header__actions">{actions}</div>}
    </header>
  );
}
