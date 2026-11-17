import { useCallback, useEffect, useRef, useState } from 'react';
import { Add, ChevronDown, ChevronUp, Copy, TrashCan } from '@carbon/icons-react';
import { Button, Checkbox, Select, SelectItem, TextInput } from '@carbon/react';
import { api } from '../lib/api';
import { errorMessage } from '../lib/format';
import type { FormField, FormFieldType, FormStep } from '../lib/types';

const FIELD_TYPES: { value: FormFieldType; label: string }[] = [
  { value: 'text', label: 'Texto curto' },
  { value: 'textarea', label: 'Parágrafo' },
  { value: 'number', label: 'Número' },
  { value: 'date', label: 'Data' },
  { value: 'time', label: 'Hora' },
  { value: 'radio', label: 'Múltipla escolha' },
  { value: 'checkbox', label: 'Caixas de seleção' },
  { value: 'dropdown', label: 'Lista suspensa' },
  { value: 'scale', label: 'Escala linear' },
  { value: 'image', label: 'Imagem / foto' },
  { value: 'file', label: 'Arquivo' },
];

function isChoice(type: FormFieldType) {
  return type === 'radio' || type === 'checkbox' || type === 'dropdown';
}

function nextStepID(steps: FormStep[]) {
  const used = new Set(steps.map((step) => step.id));
  let number = 1;
  while (used.has(`S${number}`)) number += 1;
  return `S${number}`;
}

function nextFieldID(step: FormStep, index: number) {
  const used = new Set((step.fields ?? []).map((field) => field.id));
  const stepNumber = step.id.replace(/\D/g, '') || String(index + 1);
  let number = 1;
  while (used.has(`F${stepNumber}-${number}`)) number += 1;
  return `F${stepNumber}-${number}`;
}

function hydrateField(field: FormField): FormField {
  const hydrated: FormField = {
    ...field,
    type: field.type ?? 'text',
    label: field.label ?? '',
    required: Boolean(field.required),
  };
  if (isChoice(hydrated.type)) {
    hydrated.options = Array.isArray(field.options) ? [...field.options] : [''];
  }
  if (hydrated.type === 'scale') {
    hydrated.scaleMin = field.scaleMin ?? 1;
    hydrated.scaleMax = field.scaleMax ?? 5;
    hydrated.scaleMinLabel = field.scaleMinLabel ?? '';
    hydrated.scaleMaxLabel = field.scaleMaxLabel ?? '';
  }
  return hydrated;
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

interface Props {
  formId: number;
  initialSteps: FormStep[];
  onSaved?: () => void;
}

export function FormBuilder({ formId, initialSteps, onSaved }: Props) {
  const [steps, setSteps] = useState<FormStep[]>(() =>
    (initialSteps ?? []).map((step, index) => ({
      id: step.id || `S${index + 1}`,
      title: step.title || '',
      fields: (step.fields ?? []).map(hydrateField),
    })),
  );
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [saveState, setSaveState] = useState<'idle' | 'pending' | 'saving' | 'saved' | 'error'>('idle');
  const [saveMessage, setSaveMessage] = useState('');
  const [confirmSection, setConfirmSection] = useState<number | null>(null);
  const skipSave = useRef(true);
  const timer = useRef<number | null>(null);
  const inFlight = useRef(false);
  const lastSaved = useRef(JSON.stringify(initialSteps ?? []));

  const update = (next: FormStep[]) => setSteps(next);

  const save = useCallback(async () => {
    const payload = JSON.stringify(steps);
    if (payload === lastSaved.current || inFlight.current) return;
    inFlight.current = true;
    setSaveState('saving');
    try {
      await api.patch(`/forms/${formId}/steps`, { steps });
      lastSaved.current = payload;
      setSaveState('saved');
      onSaved?.();
    } catch (error) {
      setSaveState('error');
      setSaveMessage(errorMessage(error));
    } finally {
      inFlight.current = false;
    }
  }, [formId, steps, onSaved]);

  useEffect(() => {
    if (skipSave.current) {
      skipSave.current = false;
      return;
    }
    setSaveState('pending');
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => void save(), 900);
    return () => {
      if (timer.current) window.clearTimeout(timer.current);
    };
  }, [steps, save]);

  function patchStep(index: number, patch: Partial<FormStep>) {
    update(steps.map((step, position) => (position === index ? { ...step, ...patch } : step)));
  }

  function patchField(stepIndex: number, fieldIndex: number, patch: Partial<FormField>) {
    update(
      steps.map((step, position) =>
        position === stepIndex
          ? {
              ...step,
              fields: step.fields.map((field, fieldPosition) =>
                fieldPosition === fieldIndex ? { ...field, ...patch } : field,
              ),
            }
          : step,
      ),
    );
  }

  function addStep() {
    const step: FormStep = { id: nextStepID(steps), title: '', fields: [] };
    update([...steps, step]);
  }

  function duplicateStep(index: number) {
    const copy = clone(steps[index]);
    copy.id = nextStepID(steps);
    copy.title = `${copy.title || 'Seção'} (cópia)`;
    const stepNumber = copy.id.replace(/\D/g, '');
    copy.fields = copy.fields.map((field, fieldIndex) => ({
      ...field,
      id: `F${stepNumber}-${fieldIndex + 1}`,
    }));
    update([...steps.slice(0, index + 1), copy, ...steps.slice(index + 1)]);
  }

  function moveStep(index: number, direction: number) {
    const target = index + direction;
    if (target < 0 || target >= steps.length) return;
    const next = [...steps];
    const [item] = next.splice(index, 1);
    next.splice(target, 0, item);
    update(next);
  }

  function removeStep(index: number) {
    update(steps.filter((_, position) => position !== index));
    setConfirmSection(null);
  }

  function addField(stepIndex: number) {
    const next = steps.map((step, index) => {
      if (index !== stepIndex) return step;
      const field: FormField = {
        id: nextFieldID(step, index),
        type: 'text',
        label: '',
        required: false,
      };
      return { ...step, fields: [...(step.fields ?? []), field] };
    });
    update(next);
  }

  function duplicateField(stepIndex: number, fieldIndex: number) {
    const step = steps[stepIndex];
    const copy = clone(step.fields[fieldIndex]);
    copy.id = nextFieldID(step, stepIndex);
    const fields = [...step.fields];
    fields.splice(fieldIndex + 1, 0, copy);
    patchStep(stepIndex, { fields });
  }

  function removeField(stepIndex: number, fieldIndex: number) {
    patchStep(stepIndex, { fields: steps[stepIndex].fields.filter((_, index) => index !== fieldIndex) });
  }

  function moveField(stepIndex: number, fieldIndex: number, direction: number) {
    const fields = [...steps[stepIndex].fields];
    const target = fieldIndex + direction;
    if (target < 0 || target >= fields.length) return;
    const [item] = fields.splice(fieldIndex, 1);
    fields.splice(target, 0, item);
    patchStep(stepIndex, { fields });
  }

  function changeType(stepIndex: number, fieldIndex: number, type: FormFieldType) {
    const field = steps[stepIndex].fields[fieldIndex];
    const patch: Partial<FormField> = { type };
    if (isChoice(type) && (!field.options || field.options.length === 0)) patch.options = [''];
    if (type === 'scale' && field.scaleMin == null) {
      patch.scaleMin = 1;
      patch.scaleMax = 5;
      patch.scaleMinLabel = '';
      patch.scaleMaxLabel = '';
    }
    patchField(stepIndex, fieldIndex, patch);
  }

  const totalFields = steps.reduce((sum, step) => sum + (step.fields?.length ?? 0), 0);

  return (
    <div className="form-builder">
      <div className="form-builder__head">
        <p className="form-builder__count">
          {steps.length} seção(ões) · {totalFields} pergunta(s)
        </p>
        <span className={`form-save-state form-save-state--${saveState}`}>
          {saveState === 'pending' && 'Alterações não salvas'}
          {saveState === 'saving' && 'Salvando…'}
          {saveState === 'saved' && 'Salvo automaticamente'}
          {saveState === 'error' && (saveMessage || 'Não foi possível salvar')}
        </span>
        <div className="form-builder__tools">
          <Button kind="ghost" size="sm" onClick={() => setCollapsed(Object.fromEntries(steps.map((step) => [step.id, true])))}>
            Recolher tudo
          </Button>
          <Button kind="ghost" size="sm" onClick={() => setCollapsed({})}>
            Expandir tudo
          </Button>
        </div>
      </div>

      <div className="form-builder__steps">
        {steps.map((step, stepIndex) => {
          const isCollapsed = Boolean(collapsed[step.id]);
          return (
            <section className="form-builder-step" key={step.id}>
              <header className="form-builder-step__head">
                <button
                  type="button"
                  className="form-icon-button"
                  aria-label={isCollapsed ? 'Expandir seção' : 'Recolher seção'}
                  onClick={() => setCollapsed((current) => ({ ...current, [step.id]: !isCollapsed }))}
                >
                  {isCollapsed ? <ChevronDown size={16} /> : <ChevronUp size={16} />}
                </button>
                <span className="form-builder-step__index">{stepIndex + 1}</span>
                <input
                  className="form-builder-step__title"
                  value={step.title}
                  placeholder="Título da seção"
                  onChange={(event) => patchStep(stepIndex, { title: event.target.value })}
                />
                <span className="form-badge">{step.fields.length} pergunta(s)</span>
                {confirmSection === stepIndex ? (
                  <span className="form-confirm">
                    Excluir seção?
                    <button type="button" className="form-confirm__danger" onClick={() => removeStep(stepIndex)}>
                      Excluir
                    </button>
                    <button type="button" className="form-confirm__cancel" onClick={() => setConfirmSection(null)}>
                      Cancelar
                    </button>
                  </span>
                ) : (
                  <div className="form-builder-step__actions">
                    <button type="button" className="form-icon-button" title="Mover para cima" disabled={stepIndex === 0} onClick={() => moveStep(stepIndex, -1)}>
                      <ChevronUp size={16} />
                    </button>
                    <button type="button" className="form-icon-button" title="Mover para baixo" disabled={stepIndex === steps.length - 1} onClick={() => moveStep(stepIndex, 1)}>
                      <ChevronDown size={16} />
                    </button>
                    <button type="button" className="form-icon-button" title="Duplicar seção" onClick={() => duplicateStep(stepIndex)}>
                      <Copy size={16} />
                    </button>
                    <button
                      type="button"
                      className="form-icon-button form-icon-button--danger"
                      title="Excluir seção"
                      onClick={() => (step.fields.length > 0 ? setConfirmSection(stepIndex) : removeStep(stepIndex))}
                    >
                      <TrashCan size={16} />
                    </button>
                  </div>
                )}
              </header>

              {!isCollapsed && (
                <div className="form-builder-step__body">
                  {step.fields.map((field, fieldIndex) => (
                    <div className="form-builder-field" key={field.id}>
                      <div className="form-builder-field__row">
                        <span className="form-builder-field__index">{fieldIndex + 1}.</span>
                        <TextInput
                          id={`field-label-${field.id}`}
                          labelText=""
                          hideLabel
                          placeholder="Pergunta"
                          value={field.label}
                          onChange={(event) => patchField(stepIndex, fieldIndex, { label: event.target.value })}
                        />
                        <Select
                          id={`field-type-${field.id}`}
                          labelText=""
                          hideLabel
                          value={field.type}
                          onChange={(event) => changeType(stepIndex, fieldIndex, event.target.value as FormFieldType)}
                        >
                          {FIELD_TYPES.map((type) => (
                            <SelectItem key={type.value} value={type.value} text={type.label} />
                          ))}
                        </Select>
                        <div className="form-builder-field__actions">
                          <button type="button" className="form-icon-button" title="Mover para cima" disabled={fieldIndex === 0} onClick={() => moveField(stepIndex, fieldIndex, -1)}>
                            <ChevronUp size={16} />
                          </button>
                          <button type="button" className="form-icon-button" title="Mover para baixo" disabled={fieldIndex === step.fields.length - 1} onClick={() => moveField(stepIndex, fieldIndex, 1)}>
                            <ChevronDown size={16} />
                          </button>
                          <button type="button" className="form-icon-button" title="Duplicar pergunta" onClick={() => duplicateField(stepIndex, fieldIndex)}>
                            <Copy size={16} />
                          </button>
                          <button type="button" className="form-icon-button form-icon-button--danger" title="Excluir pergunta" onClick={() => removeField(stepIndex, fieldIndex)}>
                            <TrashCan size={16} />
                          </button>
                        </div>
                      </div>

                      <TextInput
                        id={`field-help-${field.id}`}
                        labelText=""
                        hideLabel
                        placeholder="Texto de ajuda (opcional)"
                        value={field.help ?? ''}
                        onChange={(event) => patchField(stepIndex, fieldIndex, { help: event.target.value })}
                      />

                      {field.type === 'scale' && (
                        <div className="form-builder-scale">
                          <Select
                            id={`scale-min-${field.id}`}
                            labelText="Mínimo"
                            value={String(field.scaleMin ?? 1)}
                            onChange={(event) => patchField(stepIndex, fieldIndex, { scaleMin: Number(event.target.value) })}
                          >
                            <SelectItem value="0" text="0" />
                            <SelectItem value="1" text="1" />
                          </Select>
                          <Select
                            id={`scale-max-${field.id}`}
                            labelText="Máximo"
                            value={String(field.scaleMax ?? 5)}
                            onChange={(event) => patchField(stepIndex, fieldIndex, { scaleMax: Number(event.target.value) })}
                          >
                            {[2, 3, 4, 5, 6, 7, 8, 9, 10].map((value) => (
                              <SelectItem key={value} value={String(value)} text={String(value)} />
                            ))}
                          </Select>
                          <TextInput
                            id={`scale-min-label-${field.id}`}
                            labelText="Rótulo mínimo"
                            value={field.scaleMinLabel ?? ''}
                            onChange={(event) => patchField(stepIndex, fieldIndex, { scaleMinLabel: event.target.value })}
                          />
                          <TextInput
                            id={`scale-max-label-${field.id}`}
                            labelText="Rótulo máximo"
                            value={field.scaleMaxLabel ?? ''}
                            onChange={(event) => patchField(stepIndex, fieldIndex, { scaleMaxLabel: event.target.value })}
                          />
                        </div>
                      )}

                      {isChoice(field.type) && (
                        <div className="form-builder-options">
                          {(field.options ?? []).map((option, optionIndex) => (
                            <div className="form-builder-option" key={optionIndex}>
                              <span>{field.type === 'checkbox' ? '▢' : field.type === 'dropdown' ? '▾' : '○'}</span>
                              <TextInput
                                id={`field-option-${field.id}-${optionIndex}`}
                                labelText=""
                                hideLabel
                                placeholder="Opção"
                                value={option}
                                onChange={(event) =>
                                  patchField(stepIndex, fieldIndex, {
                                    options: (field.options ?? []).map((item, index) =>
                                      index === optionIndex ? event.target.value : item,
                                    ),
                                  })
                                }
                              />
                              <button
                                type="button"
                                className="form-icon-button"
                                title="Remover opção"
                                onClick={() =>
                                  patchField(stepIndex, fieldIndex, {
                                    options: (field.options ?? []).filter((_, index) => index !== optionIndex),
                                  })
                                }
                              >
                                <TrashCan size={16} />
                              </button>
                            </div>
                          ))}
                          <Button
                            kind="ghost"
                            size="sm"
                            renderIcon={Add}
                            onClick={() =>
                              patchField(stepIndex, fieldIndex, { options: [...(field.options ?? []), ''] })
                            }
                          >
                            Adicionar opção
                          </Button>
                        </div>
                      )}

                      <div className="form-builder-field__footer">
                        <Checkbox
                          id={`field-required-${field.id}`}
                          labelText="Obrigatória"
                          checked={Boolean(field.required)}
                          onChange={(_, { checked }) => patchField(stepIndex, fieldIndex, { required: checked })}
                        />
                        <span className="form-field-id">{field.id}</span>
                      </div>
                    </div>
                  ))}

                  <Button kind="tertiary" size="sm" renderIcon={Add} onClick={() => addField(stepIndex)}>
                    Adicionar pergunta
                  </Button>
                </div>
              )}
            </section>
          );
        })}

        {steps.length === 0 && <p className="form-builder__empty">Nenhuma seção ainda. Adicione a primeira seção para começar.</p>}

        <Button kind="ghost" renderIcon={Add} className="form-builder__add" onClick={addStep}>
          Adicionar seção
        </Button>
      </div>
    </div>
  );
}
