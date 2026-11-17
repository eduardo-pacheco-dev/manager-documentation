import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import {
  ArrowLeft,
  Checkmark,
  ChevronLeft,
  ChevronRight,
  Close,
  View,
} from '@carbon/icons-react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  InlineLoading,
  Modal,
  ProgressBar,
  RadioButton,
  RadioButtonGroup,
  Tag,
  TextArea,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
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

const DECISIONS: { name: Decision; text: string }[] = [
  { name: 'nao_aplicavel', text: 'Não aplicável' },
  { name: 'aprovado', text: 'Aprovar' },
  { name: 'reprovado', text: 'Reprovar' },
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

function EvaluationHistory({ evaluations }: { evaluations: FormEvaluationRecord[] }) {
  if (evaluations.length === 0) {
    return <p className="muted">Nenhuma avaliação registrada até o momento.</p>;
  }
  return (
    <div className="nokia-comment-list">
      {evaluations.map((evaluation) => (
        <div className="comment-row" key={evaluation.id}>
          <div>
            <strong>{evaluation.user_name ?? 'Sistema'}</strong>
            <span>{relativeTime(evaluation.created_at)}</span>
          </div>
          <div>
            <div className="tag-row">
              <Tag type={formStatusTag(evaluation.status)} size="sm">{evaluation.status}</Tag>
            </div>
            {(evaluation.changes?.length ?? 0) > 0 && (
              <ul className="form-eval-changes">
                {evaluation.changes?.map((change, index) => (
                  <li key={`${change.id}-${index}`}>
                    <code>
                      {change.id}
                      {change.type === 'image' ? ` · foto ${(change.index ?? 0) + 1}` : ''}
                    </code>
                    <span>{change.label}</span>
                    <span>{change.from ? FORM_FIELD_STATUS_LABELS[change.from] ?? change.from : '—'}</span>
                    <span>→</span>
                    <strong>{FORM_FIELD_STATUS_LABELS[change.to] ?? change.to}</strong>
                  </li>
                ))}
              </ul>
            )}
            {evaluation.notes && <p className="preserve-lines">{evaluation.notes}</p>}
          </div>
        </div>
      ))}
    </div>
  );
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
      const steps = response?.steps ?? form.steps ?? [];
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

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (rejectOpen || lightbox) return;
      if (event.key === 'ArrowRight') {
        setStepIndex((current) => Math.min(totalSteps - 1, current + 1));
      }
      if (event.key === 'ArrowLeft') {
        setStepIndex((current) => Math.max(0, current - 1));
      }
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [rejectOpen, lightbox, totalSteps]);

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

  if (loading || !data) return <InlineLoading description="Carregando avaliação…" />;

  const { form } = data;
  const isLast = stepIndex === totalSteps - 1;
  const progress = Math.round(((stepIndex + 1) / totalSteps) * 100);

  function renderField(field: FormField) {
    return (
      <div className="form-eval-field" key={field.id}>
        <div>
          <div className="tag-row">
            <strong>{field.label || field.id}</strong>
            {field.status && (
              <Tag type={fieldStatusTag(field.status)} size="sm">
                {FORM_FIELD_STATUS_LABELS[field.status] ?? field.status}
              </Tag>
            )}
          </div>
          <p className="field-help">
            {field.id} · {FORM_TYPE_LABELS[field.type] ?? field.type}
          </p>

          {field.type === 'image' ? (
            (field.images?.length ?? 0) > 0 ? (
              <div className="form-eval-image-grid">
                {field.images?.map((image, index) => {
                  const decision = imageDecisions[field.id]?.[String(index)];
                  return (
                    <div
                      className={`form-eval-image${decision === 'aprovado' ? ' is-approved' : decision === 'reprovado' ? ' is-rejected' : ''}`}
                      key={image}
                    >
                      <img
                        src={field.imageThumbnails?.[index] ?? image}
                        alt={field.label}
                        loading="lazy"
                        onClick={() => setLightbox({ fieldId: field.id, images: field.images ?? [], index })}
                      />
                      <div className="form-eval-image__body">
                        <span className="form-eval-image__status">
                          {decision === 'aprovado'
                            ? 'Aprovada'
                            : decision === 'reprovado'
                              ? 'Reprovada'
                              : 'Sem decisão'}
                        </span>
                        <div className="form-eval-image__actions">
                          <Button
                            kind={decision === 'aprovado' ? 'primary' : 'tertiary'}
                            size="xs"
                            onClick={() => setImage(field.id, index, 'aprovado')}
                          >
                            Aprovar
                          </Button>
                          <Button
                            kind={decision === 'reprovado' ? 'danger' : 'tertiary'}
                            size="xs"
                            onClick={() => setImage(field.id, index, 'reprovado')}
                          >
                            Reprovar
                          </Button>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <p className="muted">Nenhuma foto enviada.</p>
            )
          ) : field.type === 'file' ? (
            (field.files?.length ?? 0) > 0 ? (
              <div className="demand-attachments">
                {field.files?.map((file, index) => (
                  <div className="attachment-row" key={file}>
                    <a href={file} target="_blank" rel="noreferrer">
                      {field.fileNames?.[index] ?? file.split('/').pop()}
                    </a>
                    <View size={16} />
                  </div>
                ))}
              </div>
            ) : (
              <p className="muted">Nenhum arquivo enviado.</p>
            )
          ) : (
            <div className="form-answer-field__value">{field.value || '—'}</div>
          )}
        </div>

        <div className="form-eval-field__decision">
          <RadioButtonGroup
            legendText="Avaliação do campo"
            name={`decision-${field.id}`}
            orientation="horizontal"
            valueSelected={decisions[field.id] || undefined}
            onChange={(value) => decide(field.id, value as Decision)}
          >
            {DECISIONS.map((decision) => (
              <RadioButton
                key={decision.name}
                id={`decision-${field.id}-${decision.name}`}
                labelText={decision.text}
                value={decision.name}
              />
            ))}
          </RadioButtonGroup>
        </div>
      </div>
    );
  }

  return (
    <div className="page">
      <Breadcrumb noTrailingSlash className="doc-breadcrumb">
        <BreadcrumbItem><Link to="/forms">Formulários</Link></BreadcrumbItem>
        <BreadcrumbItem><Link to={`/forms/${form.id}`}>{form.code}</Link></BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>Avaliação</BreadcrumbItem>
      </Breadcrumb>

      <PageHeader
        eyebrow="Documentos"
        title="Avaliação da resposta"
        description={
          <span className="tag-row">
            <Tag type="cool-gray" size="sm">{form.code}</Tag>
            <Tag type={formStatusTag(selectedResponse?.status ?? form.status)} size="sm">
              {selectedResponse?.status ?? form.status}
            </Tag>
            <Tag type="blue" size="sm">{form.type}</Tag>
            {selectedResponse && (
              <Tag type="gray" size="sm">
                Enviada {relativeTime(selectedResponse.submitted_at ?? selectedResponse.created_at)}
              </Tag>
            )}
          </span>
        }
        actions={
          <>
            <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate(`/forms/${form.id}${selectedResponse ? `?response=${selectedResponse.id}` : ''}`)}>
              Ver formulário
            </Button>
            <Button kind="ghost" onClick={() => navigate('/forms')}>Voltar</Button>
          </>
        }
      />

      <div className="module-layout">
        <aside className="module-sidebar">
          <div className="module-sidebar__card">
            <h3>Seções</h3>
            {steps.map((step, index) => (
              <button
                type="button"
                key={step.id}
                className={`module-nav-item${stepIndex === index ? ' is-active' : ''}`}
                onClick={() => setStepIndex(index)}
              >
                <span>{index + 1}. {step.title || `Seção ${index + 1}`}</span>
                <span className="module-nav-item__count">
                  {index < stepIndex ? <Checkmark size={14} /> : `${step.fields.length}`}
                </span>
              </button>
            ))}
            <button
              type="button"
              className={`module-nav-item${stepIndex === steps.length ? ' is-active' : ''}`}
              onClick={() => setStepIndex(steps.length)}
            >
              <span>{steps.length + 1}. Parecer</span>
              <span className="module-nav-item__count">Resultado</span>
            </button>
          </div>

          <div className="module-sidebar__card">
            <h3>Progresso</h3>
            <ProgressBar label="Etapas concluídas" value={progress} max={100} hideLabel />
            <p className="muted small">
              Etapa {stepIndex + 1} de {totalSteps} · {progress}%
            </p>
            <span className={`form-save-state form-save-state--${saveState}`}>
              {saveState === 'saving' && 'Salvando…'}
              {saveState === 'saved' && `Salvo às ${savedAt}`}
              {saveState === 'error' && 'Erro ao salvar'}
            </span>
          </div>

          <div className="module-sidebar__card">
            <h3>Resposta</h3>
            <dl className="projects-summary">
              <div><dt>Status</dt><dd>{selectedResponse?.status ?? form.status}</dd></div>
              <div><dt>Projeto</dt><dd>{form.project_name ?? '—'}</dd></div>
              <div><dt>Perguntas</dt><dd>{steps.reduce((sum, step) => sum + step.fields.length, 0)}</dd></div>
            </dl>
          </div>
        </aside>

        <div className="module-main">
          {stepIndex < steps.length ? (
            <Tile className="nokia-detail-section">
              <div className="nokia-section-header">
                <div>
                  <h2>{steps[stepIndex].title || `Seção ${stepIndex + 1}`}</h2>
                  <p>{steps[stepIndex].fields.length} pergunta(s) nesta seção.</p>
                </div>
              </div>
              {steps[stepIndex].fields.length === 0 ? (
                <p className="muted">Esta seção não possui perguntas.</p>
              ) : (
                steps[stepIndex].fields.map(renderField)
              )}
            </Tile>
          ) : (
            <Tile className="nokia-detail-section">
              <div className="nokia-section-header">
                <div>
                  <h2>Parecer da avaliação</h2>
                  <p>
                    Você avaliou {steps.reduce((sum, step) => sum + step.fields.length, 0)} pergunta(s) em {steps.length}{' '}
                    seção(ões).
                  </p>
                </div>
              </div>

              <RadioButtonGroup
                legendText="Resultado geral"
                name="evaluation-status"
                orientation="horizontal"
                valueSelected={status}
                onChange={(value) => {
                  const next = value as 'Aprovado' | 'Reprovado';
                  if (next === 'Reprovado' && !notes.trim()) {
                    setRejectReason(notes);
                    setRejectOpen(true);
                  }
                  setStatus(next);
                }}
              >
                <RadioButton id="evaluation-approved" labelText="Aprovar formulário" value="Aprovado" />
                <RadioButton id="evaluation-rejected" labelText="Reprovar formulário" value="Reprovado" />
              </RadioButtonGroup>

              {status === 'Reprovado' && notes.trim() && (
                <p className="field-help">Reprovação justificada no parecer.</p>
              )}

              <TextArea
                id="evaluation-notes"
                labelText="Parecer / observações"
                placeholder="Descreva o parecer da avaliação."
                value={notes}
                rows={6}
                onChange={(event) => setNotes(event.target.value)}
              />
              <p className="field-help">Obrigatório quando o formulário for reprovado.</p>
            </Tile>
          )}

          <div className="section-actions">
            <Button
              kind="ghost"
              renderIcon={ChevronLeft}
              onClick={() => setStepIndex((current) => Math.max(0, current - 1))}
              disabled={stepIndex === 0}
            >
              Anterior
            </Button>
            {isLast ? (
              <Button renderIcon={Checkmark} onClick={() => void submit()} disabled={submitting}>
                {submitting ? 'Registrando…' : 'Registrar avaliação'}
              </Button>
            ) : (
              <Button
                renderIcon={ChevronRight}
                onClick={() => setStepIndex((current) => Math.min(totalSteps - 1, current + 1))}
              >
                Próximo
              </Button>
            )}
          </div>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div>
                <h2>Histórico de avaliações</h2>
                <p>{data.evaluations.length} avaliação(ões) registrada(s).</p>
              </div>
            </div>
            <EvaluationHistory evaluations={data.evaluations} />
          </Tile>
        </div>
      </div>

      <Modal
        open={rejectOpen}
        modalHeading="Justificar a reprovação"
        primaryButtonText="Confirmar reprovação"
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={!rejectReason.trim()}
        onRequestSubmit={() => {
          setNotes(rejectReason);
          setStatus('Reprovado');
          setRejectOpen(false);
        }}
        onRequestClose={() => setRejectOpen(false)}
        size="sm"
      >
        <div className="modal-form">
          <p className="muted">Informe o motivo da reprovação. O texto será salvo no parecer da avaliação.</p>
          <TextArea
            id="reject-reason"
            labelText="Justificativa"
            value={rejectReason}
            rows={4}
            onChange={(event) => setRejectReason(event.target.value)}
          />
        </div>
      </Modal>

      {lightbox && (
        <div className="form-lightbox" role="dialog" aria-modal="true" aria-label="Pré-visualização da imagem" onClick={() => setLightbox(null)}>
          <span className="form-lightbox__close">
            <Button kind="ghost" size="sm" hasIconOnly renderIcon={Close} iconDescription="Fechar" onClick={() => setLightbox(null)} />
          </span>
          <span className="form-lightbox__nav is-left">
            <Button
              kind="ghost"
              size="sm"
              hasIconOnly
              renderIcon={ChevronLeft}
              iconDescription="Imagem anterior"
              disabled={lightbox.index === 0}
              onClick={(event) => {
                event.stopPropagation();
                setLightbox((current) => current && { ...current, index: Math.max(0, current.index - 1) });
              }}
            />
          </span>
          <img src={lightbox.images[lightbox.index]} alt="Pré-visualização" onClick={(event) => event.stopPropagation()} />
          <span className="form-lightbox__nav is-right">
            <Button
              kind="ghost"
              size="sm"
              hasIconOnly
              renderIcon={ChevronRight}
              iconDescription="Próxima imagem"
              disabled={lightbox.index === lightbox.images.length - 1}
              onClick={(event) => {
                event.stopPropagation();
                setLightbox((current) =>
                  current && { ...current, index: Math.min(current.images.length - 1, current.index + 1) },
                );
              }}
            />
          </span>
          <div className="form-lightbox__bar" onClick={(event) => event.stopPropagation()}>
            <Button
              kind={imageDecisions[lightbox.fieldId]?.[String(lightbox.index)] === 'aprovado' ? 'primary' : 'ghost'}
              size="sm"
              renderIcon={Checkmark}
              onClick={() => setImage(lightbox.fieldId, lightbox.index, 'aprovado')}
            >
              Aprovar
            </Button>
            <Button
              kind={imageDecisions[lightbox.fieldId]?.[String(lightbox.index)] === 'reprovado' ? 'danger' : 'ghost'}
              size="sm"
              renderIcon={Close}
              onClick={() => setImage(lightbox.fieldId, lightbox.index, 'reprovado')}
            >
              Reprovar
            </Button>
            <span>
              {lightbox.index + 1} / {lightbox.images.length}
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
