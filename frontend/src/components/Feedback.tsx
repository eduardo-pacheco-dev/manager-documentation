import type { ComponentType, ReactNode } from 'react';
import { InlineLoading, Tag } from '@carbon/react';
import { CheckmarkFilled, ErrorFilled, Information, WarningFilled } from '@carbon/icons-react';
import { roleLabel } from '../lib/format';

export function EmptyState({
  icon: Icon = Information,
  title,
  description,
  action,
}: {
  icon?: ComponentType<{ size?: number }>;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty-state">
      <div className="empty-state__icon"><Icon size={32} /></div>
      <h3>{title}</h3>
      {description && <p>{description}</p>}
      {action && <div className="empty-state__action">{action}</div>}
    </div>
  );
}

export function Loading({ description = 'Carregando…' }: { description?: string }) {
  return (
    <div className="centered-loading">
      <InlineLoading description={description} />
    </div>
  );
}

export function RoleTag({ isAdmin }: { isAdmin: boolean }) {
  return <Tag type={isAdmin ? 'purple' : 'gray'} size="sm">{roleLabel(isAdmin)}</Tag>;
}

export function ActiveTag({ active }: { active: boolean }) {
  return <Tag type={active ? 'green' : 'red'} size="sm">{active ? 'Ativo' : 'Inativo'}</Tag>;
}

export function PublishedTag({ published }: { published: boolean }) {
  return <Tag type={published ? 'green' : 'cool-gray'} size="sm">{published ? 'Publicado' : 'Rascunho'}</Tag>;
}

export function NotificationIcon({ level }: { level: string }) {
  switch (level) {
    case 'success':
      return <CheckmarkFilled size={20} className="notification-icon notification-icon--success" />;
    case 'warning':
      return <WarningFilled size={20} className="notification-icon notification-icon--warning" />;
    case 'danger':
      return <ErrorFilled size={20} className="notification-icon notification-icon--danger" />;
    default:
      return <Information size={20} className="notification-icon notification-icon--info" />;
  }
}
