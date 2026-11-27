import type { Category } from './types';

export const DOC_CATEGORIES: Category[] = [
  { key: 'getting-started', label: 'Primeiros passos' },
  { key: 'architecture', label: 'Arquitetura' },
  { key: 'database', label: 'Banco de dados' },
  { key: 'api', label: 'API' },
  { key: 'frontend', label: 'Frontend' },
  { key: 'testing', label: 'Testes' },
  { key: 'deployment', label: 'Deploy' },
  { key: 'conventions', label: 'Convenções' },
];

export const CATEGORY_LABEL = new Map(DOC_CATEGORIES.map((item) => [item.key, item.label]));

export const DEFAULT_COMPANY_SLUG = 'padrao';

export const ROLE_LABELS = {
  superAdmin: 'Super-administrador',
  admin: 'Administrador',
  user: 'Usuário',
} as const;

interface RoleSource {
  is_admin?: boolean;
  is_super_admin?: boolean;
}

export function isSuperAdmin(user: RoleSource | null | undefined): boolean {
  return Boolean(user?.is_super_admin);
}

export function isAdmin(user: RoleSource | null | undefined): boolean {
  return Boolean(user?.is_admin || user?.is_super_admin);
}
