import type { FormFieldType, FormStep } from './types';

export const FORM_TYPE_LABELS: Record<FormFieldType, string> = {
  text: 'Texto curto',
  textarea: 'Parágrafo',
  number: 'Número',
  date: 'Data',
  time: 'Hora',
  radio: 'Múltipla escolha',
  checkbox: 'Caixas de seleção',
  dropdown: 'Lista suspensa',
  scale: 'Escala linear',
  image: 'Imagem / foto',
  file: 'Arquivo',
};

export const FORM_FIELD_STATUS_LABELS: Record<string, string> = {
  pendente: 'Pendente',
  enviado: 'Enviado',
  aprovado: 'Aprovado',
  reprovado: 'Reprovado',
  nao_aplicavel: 'Não aplicável',
};

export function formStatusTag(status: string): 'red' | 'green' | 'blue' | 'gray' | 'purple' | 'teal' | 'warm-gray' {
  switch (status) {
    case 'Aprovado':
      return 'green';
    case 'Reprovado':
      return 'red';
    case 'Preenchido':
      return 'blue';
    case 'Em preenchimento':
      return 'teal';
    case 'Pendente':
      return 'warm-gray';
    default:
      return 'purple';
  }
}

export function fieldStatusTag(status?: string): 'red' | 'green' | 'blue' | 'gray' | 'warm-gray' {
  switch (status) {
    case 'aprovado':
      return 'green';
    case 'reprovado':
      return 'red';
    case 'enviado':
      return 'blue';
    case 'nao_aplicavel':
      return 'gray';
    default:
      return 'warm-gray';
  }
}

export function countAnsweredFields(steps: FormStep[] | undefined): number {
  let total = 0;
  for (const step of steps ?? []) {
    for (const field of step.fields ?? []) {
      if (field.value || (field.images?.length ?? 0) > 0 || (field.files?.length ?? 0) > 0) total += 1;
    }
  }
  return total;
}

export function totalFields(steps: FormStep[] | undefined): number {
  return (steps ?? []).reduce((sum, step) => sum + (step.fields?.length ?? 0), 0);
}

export function publicFormLink(token: string): string {
  return `${window.location.origin}/f/${token}`;
}

// formImageThumbnail prefers the generated low-resolution thumbnail. Seed and
// legacy images may only have the full-size URL; Lorem Picsum URLs are reduced
// on the fly so the demo does not download 640px photos for 128px previews.
export function formImageThumbnail(image: string, thumbnail?: string): string {
  if (thumbnail && thumbnail !== image) return thumbnail;
  const picsum = image.match(/^https:\/\/picsum\.photos\/seed\/([^/]+)\/\d+\/\d+$/);
  if (picsum) return `https://picsum.photos/seed/${picsum[1]}/320/240`;
  return thumbnail || image;
}
