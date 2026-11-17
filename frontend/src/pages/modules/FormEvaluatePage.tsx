import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Checkmark, ChevronLeft, ChevronRight, Close, View } from '@carbon/icons-react';
import { Button, InlineLoading, Tag, TextArea } from '@carbon/react';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import {
  FORM_FIELD_STATUS_LABELS,
  FORM_TYPE_LABELS,
  fieldStatusTag,
  formStatusTag,
} from '../../lib/forms';
import type {
  FormDetailResponse,
  FormEvaluationRecord,
  FormField,
  FormResponseRecord,
  FormStep,
} from '../../lib/types';

type Decision = '' | 'aprovado' | 'reprovado' | 'nao_aplicavel';

const DECISION_OPTIONS: { value: Decision; label: string }[] = [
  { value: 'nao_aplicavel', label: 'Não aplicável' },
  { value: 'aprovado', label: 'Aprovar' },
  { value: 'reprovado', label: 'Reprovar' },
];

function initialDecisions(steps: FormStep[]): Record<string, Decision> {
  const result: Record<string, Decision> = {};
  for (const step of steps) {
    for (const field of step.fields) {
      result[field.id] =
        field.status === 'aprovado' || field.status === 'reprovado' || field.status === 'nao_aplicavel'
          ? field.status
          : '';
    }
  }
  return result;
}

function initialImageDecisions(steps: FormStep[]): Record<string, Record<string, string>> {
  const result: Record<string, Record<string, string>> = {};
  for (const step of steps) {
    for (const field of step.fields) {
      if (field.imageDecisions && Object.keys(field.imageDecisions).length > 0) {
        result[field.id] = { ...field.imageDecisions };
      }
    }
  }
  return result;
}

export function FormEvaluatePage() {
  const { id } = useParams();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const toast = useToast();
  const [data, setData] = useState<FormDetailResponse['data'] | null>(null);
  const [loading, setLoading] = useState(true);
  const [stepIndex, setStepIndex] = useState(0);
  const [decisions, setDecisions] = useState<Record<string, Decision>>({});
  const [imageDecisions, setImageDecisions] = useState<Record<string, Record<string, string>>>({});
  const [status, setStatus] = useState<'Aprovado' | 'Reprovado' | ''>('');
  const [notes, setNotes] = useState('');
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [savedAt, setSavedAt] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [lightbox, setLightbox] = useState<{ fieldId: string; images: string[]; index: number } | null>(null);
  const timer = useRef<number | null>(null);
  const skipAutosave = useRef(true);

  const responseId = searchParams.get('response');

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const query = responseId ? `?response=${responseId}` : '';
      let detail = await api.get<FormDetailResponse>(`/forms/${id}${query}`);
      if (!responseId && detail.data.responses[0]) {
        detail = await api.get<FormDetailResponse>(`/forms/${id}?response=${detail.data.responses[0].id}`);
      }
      setData(detail.data);
      const form = detail.data.form;
      const response = responseId
        ? detail.data.responses.find((item) => String(item.id) === responseId) ?? detail.data.responses[0]
        : detail.data.responses[0];
      const steps = response?.steps ?? form.steps;
      setDecisions(initialDecisions(steps));
      setImageDecisions(initialImageDecisions(steps));
      const currentStatus = response?.status ?? form.status;
      setStatus(currentStatus === 'Aprovado' || currentStatus === 'Reprovado' ? currentStatus : '');
      setNotes('');
      setStepIndex(0);
      skipAutosave.current = true;
    } catch (error) {
      toast.error('Não foi possível carregar a avaliação', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [id, responseId, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const selectedResponse: FormResponseRecord | null = useMemo(() => {
    if (!data) return null;
    if (responseId) return data.responses.find((item) => String(item.id) === responseId) ?? data.responses[0] ?? null;
    return data.responses[0] ?? null;
  }, [data, responseId]);

  const steps = selectedResponse?.steps ?? data?.form.steps ?? [];
  const totalSteps = steps.length + 1;

  const payload = useCallback(
    () => ({
      form_response_id: selectedResponse?.id ?? null,
      status: status || null,
      notes,
      decisions,
      image_decisions: imageDecisions,
    }),
    [selectedResponse?.id, status, notes, decisions, imageDecisions],
  );

  const autosave = useCallback(async () => {
    if (!id) return;
    setSaveState('saving');
    try {
      const response = await api.post<{ saved_at: string }>(`/forms/${id}/evaluate/autosave`, payload());
      setSavedAt(response.saved_at);
      setSaveState('saved');
    } catch {
      setSaveState('error');
    }
  }, [id, payload]);

  useEffect(() => {
    if (loading) return;
    if (skipAutosave.current) {
      skipAutosave.current = false;
      return;
    }
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => void autosave(), 900);
    return () => {
      if (timer.current) window.clearTimeout(timer.current);
    };
  }, [decisions, imageDecisions, status, notes, autosave, loading]);

  function decide(fieldId: string, decision: Decision) {
    setDecisions((current) => ({ ...current, [fieldId]: decision }));
  }

  function setImage(fieldId: string, index: number, decision: 'aprovado' | 'reprovado') {
    setImageDecisions((current) => ({
      ...current,
      [fieldId]: { ...(current[fieldId] ?? {}), [String(index)]: decision },
    }));
  }

  async function submit() {
    if (!id) return;
    if (!status) {
      toast.error('Selecione o resultado geral', 'Aprove ou reprove o formulário para registrar a avaliação.');
      return;
    }
    if (status === 'Reprovado' && !notes.trim()) {
      setRejectReason(notes);
      setRejectOpen(true);
      return;
    }
    setSubmitting(true);
    try {
      await api.post(`/forms/${id}/evaluate`, payload());
      toast.success('Avaliação registrada', 'O parecer foi salvo com sucesso.');
      navigate(`/forms/${id}${selectedResponse ? `?response=${selectedResponse.id}` : ''}`);
    } catch (error) {
      toast.error('Não foi possível registrar a avaliação', errorMessage(error));
    } finally {
      setSubmitting(false);
    }
  }

  if (loading || !data) {
    return <InlineLoading description="Carregando avaliação…" />;
  }

  const { form } = data;
  const isLast = stepIndex === totalSteps - 1;
  const evaluations: FormEvaluationRecord[] = data.evaluations;

  function renderField(field: FormField) {
    return (
      <div className="form-eval-field" key={field.id}>
        <div className="form-eval-field__content">
          <div className="form-eval-field__title">
            <p>{field.label || field.id}</p>
            {field.status && (
              <Tag type={fieldStatusTag(field.status)} size="sm">
                {FORM_FIELD_STATUS_LABELS[field.status] ?? field.status}
              </Tag>
            )}
          </div>
          <p className="form-answer__meta">
            {field.id} · {FORM_TYPE_LABELS[field.type] ?? field.type}
          </p>

          {field.type === 'image' ? (
            (field.images?.length ?? 0) > 0 ? (
              <div className="form-eval-images">
                {field.images?.map((image, index) => {
                  const decision = imageDecisions[field.id]?.[String(index)];
                  return (
                    <div className={`form-eval-image${decision ? ` is-${decision}` : ''}`} key={image}>
                      <button
                        type="button"
                        className="form-eval-image__preview"
                        onClick={() => setLightbox({ fieldId: field.id, images: field.images ?? [], index })}
                      >
                        <img src={field.imageThumbnails?.[index] ?? image} alt={field.label} loading="lazy" />
                        {decision && (
                          <span className={`form-eval-image__badge is-${decision}`}>
                            {decision === 'aprovado' ? 'Aprovada' : 'Reprovada'}
                          </span>
                        )}
                      </button>
                      <div className="form-eval-image__actions">
                        <button
                          type="button"
                          className={decision === 'aprovado' ? 'is-approved' : ''}
                          onClick={() => setImage(field.id, index, 'aprovado')}
                        >
                          Aprovar
                        </button>
                        <button
                          type="button"
                          className={decision === 'reprovado' ? 'is-rejected' : ''}
                          onClick={() => setImage(field.id, index, 'reprovado')}
                        >
                          Reprovar
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <p className="form-empty-answer">Nenhuma foto enviada.</p>
            )
          ) : field.type === 'file' ? (
            (field.files?.length ?? 0) > 0 ? (
              <div className="form-files">
                {field.files?.map((file, index) => (
                  <a key={file} href={file} target="_blank" rel="noopener noreferrer" className="form-file">
                    <span>{field.fileNames?.[index] ?? file.split('/').pop()}</span>
                    <View size={16} />
                  </a>
                ))}
              </div>
            ) : (
              <p className="form-empty-answer">Nenhum arquivo enviado.</p>
            )
          ) : (
            <div className="form-answer__value">{field.value || '—'}</div>
          )}
        </div>

        <div className="form-eval-field__decision">
          <span>Avaliação do campo</span>
          <div className="form-decision-group">
            {DECISION_OPTIONS.map((option) => (
              <button
                type="button"
                key={option.value}
                className={decisions[field.id] === option.value ? `is-active is-${option.value}` : ''}
                onClick={() => decide(field.id, option.value)}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="page form-eval-page">
      <div className="form-detail-head">
        <div>
          <nav className="form-breadcrumb">
            <button type="button" onClick={() => navigate('/forms')}>Formulários</button>
            <span>/</span>
            <button type="button" onClick={() => navigate(`/forms/${form.id}`)}>{form.code}</button>
            <span>/</span>
            <span>Avaliação</span>
          </nav>
          <div className="form-detail-title">
            <h1>Avaliação da resposta</h1>
            <Tag type={formStatusTag(selectedResponse?.status ?? form.status)} size="sm">
              {selectedResponse?.status ?? form.status}
            </Tag>
          </div>
          <p className="form-detail-subtitle">{form.code} · {form.name}</p>
          <div className="form-detail-meta">
            <span>{form.type}</span>
            {selectedResponse && <span>Resposta enviada {relativeTime(selectedResponse.submitted_at ?? selectedResponse.created_at)}</span>}
            {form.project_name && <span>[{form.project_code}] {form.project_name}</span>}
          </div>
        </div>
        <div className="page-header__actions">
          <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate(`/forms/${form.id}${selectedResponse ? `?response=${selectedResponse.id}` : ''}`)}>
            Ver formulário
          </Button>
          <Button kind="ghost" onClick={() => navigate('/forms')}>Voltar</Button>
        </div>
      </div>

      <div className="form-eval-layout">
        <aside className="form-eval-nav">
          <p>Seções</p>
          {steps.map((step, index) => (
            <button
              type="button"
              key={step.id}
              className={stepIndex === index ? 'is-active' : index < stepIndex ? 'is-done' : ''}
              onClick={() => setStepIndex(index)}
            >
              <span>{index + 1}</span>
              <div>
                <strong>{step.title || `Seção ${index + 1}`}</strong>
                <small>{step.fields.length} pergunta(s)</small>
              </div>
            </button>
          ))}
          <button
            type="button"
            className={stepIndex === steps.length ? 'is-active' : ''}
            onClick={() => setStepIndex(steps.length)}
          >
            <span>{steps.length + 1}</span>
            <div>
              <strong>Parecer</strong>
              <small>Resultado e observações</small>
            </div>
          </button>
          <div className="form-eval-progress">
            <div>
              <span>Progresso</span>
              <span>{Math.round(((stepIndex + 1) / totalSteps) * 100)}%</span>
            </div>
            <div className="form-eval-progress__bar">
              <span style={{ width: `${((stepIndex + 1) / totalSteps) * 100}%` }} />
            </div>
          </div>
        </aside>

        <div className="form-eval-content">
          {stepIndex < steps.length ? (
            <section className="form-panel">
              <header className="form-panel__head">
                <h2>{steps[stepIndex].title || `Seção ${stepIndex + 1}`}</h2>
                <Tag type="gray" size="sm">{steps[stepIndex].fields.length} pergunta(s)</Tag>
              </header>
              <div className="form-eval-fields">
                {steps[stepIndex].fields.length === 0 ? (
                  <p className="form-empty-answer">Esta seção não possui perguntas.</p>
                ) : (
                  steps[stepIndex].fields.map(renderField)
                )}
              </div>
            </section>
          ) : (
            <section className="form-panel">
              <header className="form-panel__head">
                <h2>Parecer da avaliação</h2>
              </header>
              <div className="form-eval-verdict">
                <p className="form-panel__hint">
                  Você avaliou {steps.reduce((sum, step) => sum + step.fields.length, 0)} pergunta(s) em {steps.length} seção(ões).
                </p>
                <span className="form-eval-label">Resultado geral</span>
                <div className="form-verdict-options">
                  <button
                    type="button"
                    className={status === 'Aprovado' ? 'is-approved' : ''}
                    onClick={() => setStatus('Aprovado')}
                  >
                    <Checkmark size={18} />
                    Aprovar formulário
                  </button>
                  <button
                    type="button"
                    className={status === 'Reprovado' ? 'is-rejected' : ''}
                    onClick={() => {
                      setRejectReason(notes);
                      setRejectOpen(true);
                    }}
                  >
                    <Close size={18} />
                    Reprovar formulário
                  </button>
                </div>
                {status === 'Reprovado' && notes.trim() && (
                  <p className="form-eval-ok">Reprovação justificada no parecer.</p>
                )}

                <TextArea
                  id="evaluation-notes"
                  labelText="Parecer / observações"
                  placeholder="Descreva o parecer da avaliação."
                  value={notes}
                  rows={6}
                  onChange={(event) => setNotes(event.target.value)}
                />
                <p className="form-panel__hint">Obrigatório quando o formulário for reprovado.</p>
              </div>
            </section>
          )}

          <div className="form-eval-actions">
            <Button kind="ghost" onClick={() => setStepIndex((current) => Math.max(0, current - 1))} disabled={stepIndex === 0}>
              Anterior
            </Button>
            <span className={`form-save-state form-save-state--${saveState}`}>
              {saveState === 'saving' && 'Salvando…'}
              {saveState === 'saved' && `Salvo às ${savedAt}`}
              {saveState === 'error' && 'Erro ao salvar'}
            </span>
            {isLast ? (
              <Button renderIcon={Checkmark} onClick={() => void submit()} disabled={submitting}>
                {submitting ? 'Registrando…' : 'Registrar avaliação'}
              </Button>
            ) : (
              <Button renderIcon={ChevronRight} onClick={() => setStepIndex((current) => Math.min(totalSteps - 1, current + 1))}>
                Próximo
              </Button>
            )}
          </div>
        </div>
      </div>

      <section className="form-panel">
        <header className="form-panel__head">
          <h2>Histórico de avaliações</h2>
          <Tag type="gray" size="sm">{evaluations.length}</Tag>
        </header>
        {evaluations.length > 0 ? (
          <div className="form-evaluation-list">
            {evaluations.map((evaluation) => (
              <div className="form-evaluation-item" key={evaluation.id}>
                <div className="form-evaluation-item__head">
                  <Tag type={formStatusTag(evaluation.status)} size="sm">{evaluation.status}</Tag>
                  <strong>{evaluation.user_name ?? 'Sistema'}</strong>
                  <span>{relativeTime(evaluation.created_at)}</span>
                </div>
                {(evaluation.changes?.length ?? 0) > 0 && (
                  <ul className="form-evaluation-changes">
                    {evaluation.changes?.map((change, index) => (
                      <li key={`${change.id}-${index}`}>
                        <code>{change.id}{change.type === 'image' ? ` · foto ${(change.index ?? 0) + 1}` : ''}</code>
                        <span>{change.label}</span>
                        <span>{change.from ? FORM_FIELD_STATUS_LABELS[change.from] ?? change.from : '—'}</span>
                        <span>→</span>
                        <strong>{FORM_FIELD_STATUS_LABELS[change.to] ?? change.to}</strong>
                      </li>
                    ))}
                  </ul>
                )}
                {evaluation.notes && <p className="form-evaluation-notes">{evaluation.notes}</p>}
              </div>
            ))}
          </div>
        ) : (
          <p className="form-empty-answer">Nenhuma avaliação registrada até o momento.</p>
        )}
      </section>

      {rejectOpen && (
        <div className="form-modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="reject-title">
          <div className="form-modal">
            <h2 id="reject-title">Justificar a reprovação</h2>
            <p className="form-panel__hint">Informe o motivo da reprovação. O texto será salvo no parecer da avaliação.</p>
            <TextArea
              id="reject-reason"
              labelText="Justificativa"
              value={rejectReason}
              rows={4}
              onChange={(event) => setRejectReason(event.target.value)}
            />
            <div className="form-modal__actions">
              <Button kind="ghost" onClick={() => setRejectOpen(false)}>Cancelar</Button>
              <Button
                kind="danger"
                disabled={!rejectReason.trim()}
                onClick={() => {
                  setNotes(rejectReason);
                  setStatus('Reprovado');
                  setRejectOpen(false);
                }}
              >
                Confirmar reprovação
              </Button>
            </div>
          </div>
        </div>
      )}

      {lightbox && (
        <div className="form-lightbox" role="dialog" aria-modal="true" aria-label="Pré-visualização da imagem">
          <button type="button" className="form-lightbox__close" aria-label="Fechar" onClick={() => setLightbox(null)}>
            <Close size={20} />
          </button>
          <button
            type="button"
            className="form-lightbox__nav is-left"
            aria-label="Imagem anterior"
            disabled={lightbox.index === 0}
            onClick={() => setLightbox((current) => current && { ...current, index: Math.max(0, current.index - 1) })}
          >
            <ChevronLeft size={24} />
          </button>
          <img src={lightbox.images[lightbox.index]} alt="Pré-visualização" />
          <button
            type="button"
            className="form-lightbox__nav is-right"
            aria-label="Próxima imagem"
            disabled={lightbox.index === lightbox.images.length - 1}
            onClick={() =>
              setLightbox((current) =>
                current && { ...current, index: Math.min(current.images.length - 1, current.index + 1) },
              )
            }
          >
            <ChevronRight size={24} />
          </button>
          <div className="form-lightbox__actions">
            <button
              type="button"
              className={imageDecisions[lightbox.fieldId]?.[String(lightbox.index)] === 'aprovado' ? 'is-approved' : ''}
              onClick={() => setImage(lightbox.fieldId, lightbox.index, 'aprovado')}
            >
              <Checkmark size={20} /> Aprovar
            </button>
            <button
              type="button"
              className={imageDecisions[lightbox.fieldId]?.[String(lightbox.index)] === 'reprovado' ? 'is-rejected' : ''}
              onClick={() => setImage(lightbox.fieldId, lightbox.index, 'reprovado')}
            >
              <Close size={20} /> Reprovar
            </button>
            <span>{lightbox.index + 1} / {lightbox.images.length}</span>
          </div>
        </div>
      )}
    </div>
  );
}
