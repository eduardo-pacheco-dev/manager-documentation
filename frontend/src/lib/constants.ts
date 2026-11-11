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
