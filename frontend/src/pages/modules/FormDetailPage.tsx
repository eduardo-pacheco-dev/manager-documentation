import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import {
  ArrowLeft,
  Checkmark,
  CheckmarkFilled,
  ChevronLeft,
  ChevronRight,
  Close,
  Copy,
  Edit,
  ErrorFilled,
  Launch,
  NotAvailable,
  Reset,
  Time,
  View,
  ZoomIn,
  ZoomOut,
} from '@carbon/icons-react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  InlineLoading,
  Tag,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import {
  FORM_FIELD_STATUS_LABELS,
  FORM_TYPE_LABELS,
  countAnsweredFields,
  fieldStatusTag,
  formImageThumbnail,
  formStatusTag,
  publicFormLink,
} from '../../lib/forms';
import type { FormDetailResponse, FormEvaluationRecord, FormField, FormResponseRecord, FormStep } from '../../lib/types';

interface PreviewHandlers {
  onPreviewImage: (images: string[], index: number, title: string) => void;
  onPreviewFile: (url: string, name: string) => void;
}

type PreviewState =
  | { type: 'image'; images: string[]; index: number; title: string }
  | { type: 'file'; url: string; name: string }
  | null;

const IMAGE_EXTENSIONS = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'svg'];
const TEXT_EXTENSIONS = ['txt', 'csv', 'json', 'xml', 'md', 'log'];
const FRAME_EXTENSIONS = ['pdf'];

function fileExtension(name: string): string {
  return name.split('.').pop()?.toLowerCase() ?? '';
}

function iconForImageStatus(status: string) {
  switch (status) {
    case 'aprovado':
    case 'enviado':
      return CheckmarkFilled;
    case 'reprovado':
      return ErrorFilled;
    case 'nao_aplicavel':
      return NotAvailable;
    default:
      return Time;
  }
}

const MIN_ZOOM = 1;
const MAX_ZOOM = 4;

function clampZoom(value: number): number {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, value));
}

function AnswerField({ field, onPreviewImage, onPreviewFile }: { field: FormField } & PreviewHandlers) {
  return (
    <div className="form-answer-field">
      <div className="form-answer-field__head">
        <div>
          <strong>{field.label || field.id}</strong>
          <span>
            {field.id} · {FORM_TYPE_LABELS[field.type] ?? field.type}
          </span>
        </div>
        {field.status && field.status !== 'enviado' && field.status !== 'pendente' && (
          <Tag type={fieldStatusTag(field.status)} size="sm">
            {FORM_FIELD_STATUS_LABELS[field.status] ?? field.status}
          </Tag>
        )}
      </div>

      {field.type === 'image' ? (
        (field.images?.length ?? 0) > 0 ? (
          <div className="form-answer-images">
            {field.images?.map((image, index) => {
              const status = field.imageDecisions?.[String(index)] || field.status || 'pendente';
              const statusLabel = FORM_FIELD_STATUS_LABELS[status] ?? status;
              const StatusIcon = iconForImageStatus(status);
              return (
                <button
                  type="button"
                  className="form-answer-image"
                  key={image}
                  onClick={() => onPreviewImage(field.images ?? [], index, field.label)}
                >
                  <img
                    src={formImageThumbnail(image, field.imageThumbnails?.[index])}
                    alt={field.label}
                    loading="lazy"
                  />
                  <span
                    className={`form-answer-image__status is-${status}`}
                    title={`Status: ${statusLabel}`}
                    role="img"
                    aria-label={`Status da imagem: ${statusLabel}`}
                  >
                    <StatusIcon size={16} />
                  </span>
                </button>
              );
            })}
          </div>
        ) : (
          <p className="muted">{field.status === 'nao_aplicavel' ? 'Não aplicável.' : 'Nenhuma foto.'}</p>
        )
      ) : field.type === 'file' ? (
        (field.files?.length ?? 0) > 0 ? (
          <div className="demand-attachments">
            {field.files?.map((file, index) => {
              const name = field.fileNames?.[index] ?? file.split('/').pop() ?? 'Arquivo';
              return (
                <div className="attachment-row" key={file}>
                  <button type="button" className="form-file-preview" onClick={() => onPreviewFile(file, name)}>
                    <span>{name}</span>
                    <View size={16} />
                  </button>
                </div>
              );
            })}
          </div>
        ) : (
          <p className="muted">Nenhum arquivo.</p>
        )
      ) : (
        <div className="form-answer-field__value">{field.value || '—'}</div>
      )}
    </div>
  );
}

function EvaluationItem({ evaluation }: { evaluation: FormEvaluationRecord }) {
  return (
    <div className="comment-row">
      <div>
        <strong>{evaluation.user_name ?? 'Sistema'}</strong>
        <span>{relativeTime(evaluation.created_at)}</span>
      </div>
      <div>
        <div className="tag-row">
          <Tag type={formStatusTag(evaluation.status)} size="sm">
            {evaluation.status}
          </Tag>
        </div>
        {(evaluation.changes?.length ?? 0) > 0 && (
          <ul className="form-eval-changes">
            {evaluation.changes?.map((change, index) => (
              <li key={`${change.id}-${change.type}-${index}`}>
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
  );
}

export function FormDetailPage() {
  const { id } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const toast = useToast();
  const [data, setData] = useState<FormDetailResponse['data'] | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [preview, setPreview] = useState<PreviewState>(null);
  const [textPreview, setTextPreview] = useState('');
  const [textLoading, setTextLoading] = useState(false);
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [dragging, setDragging] = useState(false);
  const dragRef = useRef<{ x: number; y: number; originX: number; originY: number } | null>(null);
  const lightboxRef = useRef<HTMLDivElement | null>(null);

  const selectedResponseId = searchParams.get('response');
  const previewImageKey = preview?.type === 'image' ? preview.index : -1;

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const query = selectedResponseId ? `?response=${selectedResponseId}` : '';
      const response = await api.get<FormDetailResponse>(`/forms/${id}${query}`);
      setData(response.data);
    } catch (error) {
      setNotFound(true);
      toast.error('Não foi possível carregar o formulário', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [id, selectedResponseId, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    setTextPreview('');
    if (preview?.type !== 'file' || !TEXT_EXTENSIONS.includes(fileExtension(preview.name))) return;
    let active = true;
    setTextLoading(true);
    fetch(preview.url, { credentials: 'include' })
      .then((response) => {
        if (!response.ok) throw new Error('preview failed');
        return response.text();
      })
      .then((text) => {
        if (active) setTextPreview(text);
      })
      .catch(() => undefined)
      .finally(() => {
        if (active) setTextLoading(false);
      });
    return () => {
      active = false;
    };
  }, [preview]);

  useEffect(() => {
    setZoom(1);
    setPan({ x: 0, y: 0 });
    setDragging(false);
    dragRef.current = null;
  }, [previewImageKey]);

  useEffect(() => {
    if (zoom <= MIN_ZOOM && (pan.x !== 0 || pan.y !== 0)) {
      setPan({ x: 0, y: 0 });
    }
  }, [zoom, pan.x, pan.y]);

  useEffect(() => {
    if (preview?.type !== 'image') return;
    const element = lightboxRef.current;
    if (!element) return;
    function onWheel(event: WheelEvent) {
      event.preventDefault();
      const step = event.deltaY < 0 ? 0.25 : -0.25;
      setZoom((current) => clampZoom(current + step));
    }
    element.addEventListener('wheel', onWheel, { passive: false });
    return () => element.removeEventListener('wheel', onWheel);
  }, [preview?.type]);

  useEffect(() => {
    if (!preview) return;
    const currentPreview = preview;
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setPreview(null);
        return;
      }
      if (currentPreview.type === 'image') {
        if (event.key === 'ArrowRight') {
          setPreview((current) =>
            current && current.type === 'image'
              ? { ...current, index: Math.min(current.images.length - 1, current.index + 1) }
              : current,
          );
        }
        if (event.key === 'ArrowLeft') {
          setPreview((current) =>
            current && current.type === 'image'
              ? { ...current, index: Math.max(0, current.index - 1) }
              : current,
          );
        }
        if (event.key === '+' || event.key === '=') {
          setZoom((current) => clampZoom(current + 0.5));
        }
        if (event.key === '-') {
          setZoom((current) => clampZoom(current - 0.5));
        }
        if (event.key === '0') {
          setZoom(MIN_ZOOM);
          setPan({ x: 0, y: 0 });
        }
      }
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [preview]);

  async function copyPublicLink() {
    if (!data) return;
    const link = publicFormLink(data.form.public_token);
    try {
      await navigator.clipboard.writeText(link);
      toast.success('Link público copiado.');
    } catch {
      toast.error('Não foi possível copiar o link', link);
    }
  }

  if (loading) return <InlineLoading description="Carregando formulário…" />;

  if (notFound || !data) {
    return (
      <div className="status-page">
        <p className="status-page__code">404</p>
        <h1>Formulário não encontrado</h1>
        <p>O formulário pode ter sido excluído ou o endereço está incorreto.</p>
        <Link to="/forms"><Button>Voltar para Formulários</Button></Link>
      </div>
    );
  }

  const { form, responses, evaluations, settings } = data;
  const formSteps = form.steps ?? [];
  const selectedResponse: FormResponseRecord | null =
    (selectedResponseId ? responses.find((item) => String(item.id) === selectedResponseId) : responses[0]) ?? null;
  const steps: FormStep[] = selectedResponse?.steps ?? (responses.length === 0 ? formSteps : []);
  const answeredCount = countAnsweredFields(steps);
  const hasAnswers = answeredCount > 0;
  const publicLink = publicFormLink(form.public_token);
  const evaluateRoute = `/forms/${form.id}/evaluate${selectedResponse ? `?response=${selectedResponse.id}` : ''}`;
  const previewFile = preview?.type === 'file' ? preview : null;
  const previewExtension = previewFile ? fileExtension(previewFile.name) : '';
  const previewIsImage = IMAGE_EXTENSIONS.includes(previewExtension);
  const previewIsFrame = FRAME_EXTENSIONS.includes(previewExtension);
  const previewIsText = TEXT_EXTENSIONS.includes(previewExtension);

  return (
    <div className="page">
      <Breadcrumb noTrailingSlash className="doc-breadcrumb">
        <BreadcrumbItem><Link to="/forms">Formulários</Link></BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>{form.code}</BreadcrumbItem>
      </Breadcrumb>

      <PageHeader
        eyebrow="Documentos"
        title={form.name}
        description={
          <span className="tag-row">
            <Tag type="cool-gray" size="sm">{form.code}</Tag>
            <Tag type={formStatusTag(form.status)} size="sm">{form.status}</Tag>
            <Tag type="blue" size="sm">{form.type}</Tag>
            <Tag type="gray" size="sm">{form.allows_multiple_responses ? 'Múltiplas respostas' : 'Resposta única'}</Tag>
            {form.project_name && (
              <Tag type="purple" size="sm">[{form.project_code}] {form.project_name}</Tag>
            )}
          </span>
        }
        actions={
          <>
            <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/forms')}>Voltar</Button>
            <Button kind="tertiary" renderIcon={Checkmark} onClick={() => navigate(evaluateRoute)}>Avaliar resposta</Button>
            <Button renderIcon={Edit} onClick={() => navigate(`/forms/${form.id}/edit`)}>Editar</Button>
          </>
        }
      />

      <div className="nokia-detail-layout">
        <div className="nokia-detail-main">
          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div>
                <h2>Respostas do aplicativo</h2>
                <p>
                  {selectedResponse
                    ? `Resposta enviada ${relativeTime(selectedResponse.submitted_at ?? selectedResponse.created_at)}. Somente leitura.`
                    : 'Valores, status e fotos preenchidos pelos técnicos. Somente leitura.'}
                </p>
              </div>
              {hasAnswers && <Tag type="gray" size="sm">{answeredCount} resposta(s)</Tag>}
            </div>

            {hasAnswers ? (
              steps.map((step) => {
                const answered = step.fields.filter(
                  (field) => field.value || (field.images?.length ?? 0) > 0 || (field.files?.length ?? 0) > 0,
                );
                if (answered.length === 0) return null;
                return (
                  <div key={step.id} className="module-subcard">
                    <h3>{step.title || 'Seção'}</h3>
                    {answered.map((field) => (
                      <AnswerField
                        field={field}
                        key={field.id}
                        onPreviewImage={(images, index, title) => setPreview({ type: 'image', images, index, title })}
                        onPreviewFile={(url, name) => setPreview({ type: 'file', url, name })}
                      />
                    ))}
                  </div>
                );
              })
            ) : (
              <p className="muted">Nenhuma resposta preenchida pelo aplicativo até o momento.</p>
            )}
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div>
                <h2>Histórico de avaliações</h2>
                <p>
                  {evaluations.length} avaliação(ões) {selectedResponse ? 'desta resposta' : 'registrada(s)'}.
                </p>
              </div>
            </div>
            {evaluations.length === 0 ? (
              <p className="muted">Nenhuma avaliação registrada até o momento.</p>
            ) : (
              <div className="nokia-comment-list">
                {evaluations.map((evaluation) => (
                  <EvaluationItem evaluation={evaluation} key={evaluation.id} />
                ))}
              </div>
            )}
          </Tile>
        </div>

        <aside className="nokia-detail-aside">
          <Tile className="nokia-detail-aside__card">
            <h2>Resumo</h2>
            <dl>
              <div><dt>Status</dt><dd>{form.status}</dd></div>
              <div><dt>Tipo</dt><dd>{form.type}</dd></div>
              <div><dt>Projeto</dt><dd>{form.project_name ?? '—'}</dd></div>
              <div><dt>Seções</dt><dd>{formSteps.length}</dd></div>
              <div><dt>Perguntas</dt><dd>{formSteps.reduce((sum, step) => sum + step.fields.length, 0)}</dd></div>
              <div><dt>Respostas</dt><dd>{form.response_count}</dd></div>
              <div><dt>Avaliações</dt><dd>{form.evaluation_count}</dd></div>
              <div><dt>Criado em</dt><dd>{relativeTime(form.created_at)}</dd></div>
              <div><dt>Atualizado em</dt><dd>{relativeTime(form.updated_at)}</dd></div>
            </dl>
          </Tile>

          <Tile className="nokia-detail-aside__card">
            <h2>Link público</h2>
            <p className="muted">Compartilhe este endereço para preenchimento externo.</p>
            <div className="demand-public-link">
              <span>Preenchimento</span>
              <code>{publicLink}</code>
              <div>
                <Button kind="ghost" size="xs" hasIconOnly renderIcon={Copy} iconDescription="Copiar link público" onClick={() => void copyPublicLink()} />
                <Button kind="ghost" size="xs" hasIconOnly renderIcon={Launch} iconDescription="Abrir link público" onClick={() => window.open(publicLink, '_blank')} />
              </div>
            </div>
          </Tile>

          <Tile className="nokia-detail-aside__card">
            <h2>Histórico de respostas</h2>
            {responses.length === 0 ? (
              <p className="muted">Nenhuma resposta recebida.</p>
            ) : (
              <div className="module-main">
                {responses.map((response, index) => {
                  const selected = selectedResponse?.id === response.id;
                  return (
                    <div className={`module-nav-row${selected ? ' is-active' : ''}`} key={response.id}>
                      <button
                        type="button"
                        onClick={() => setSearchParams(selected ? {} : { response: String(response.id) })}
                      >
                        <span>
                          Resposta #{index + 1} · {relativeTime(response.submitted_at ?? response.created_at)}
                        </span>
                        <span className="module-nav-item__count">{response.status}</span>
                      </button>
                      <span className="module-nav-row__action">
                        <Button
                          kind="ghost"
                          size="sm"
                          hasIconOnly
                          renderIcon={Checkmark}
                          iconDescription={`Avaliar resposta #${index + 1}`}
                          onClick={() => navigate(`/forms/${form.id}/evaluate?response=${response.id}`)}
                        />
                      </span>
                    </div>
                  );
                })}
              </div>
            )}
          </Tile>

          <Tile className="nokia-detail-aside__card">
            <h2>Ações</h2>
            <div className="nokia-aside-actions">
              <Button kind="ghost" size="sm" renderIcon={Checkmark} onClick={() => navigate(evaluateRoute)}>Avaliar resposta</Button>
              <Button kind="ghost" size="sm" renderIcon={Edit} onClick={() => navigate(`/forms/${form.id}/edit`)}>Editar estrutura</Button>
              <Button kind="ghost" size="sm" renderIcon={View} onClick={() => navigate('/forms/settings')}>
                Configurações
              </Button>
            </div>
          </Tile>

          {settings.default_template_id && (
            <Tile className="nokia-detail-aside__card">
              <h2>Template padrão</h2>
              <p className="muted">
                {data.templates.find((template) => template.id === settings.default_template_id)?.name ?? '—'}
              </p>
            </Tile>
          )}
        </aside>
      </div>

      {preview?.type === 'image' && (
        <div
          className="form-lightbox"
          role="dialog"
          aria-modal="true"
          aria-label={`Pré-visualização: ${preview.title}`}
          onClick={() => setPreview(null)}
          ref={lightboxRef}
        >
          <span className="form-lightbox__close">
            <Button
              kind="ghost"
              size="sm"
              hasIconOnly
              renderIcon={Close}
              iconDescription="Fechar"
              onClick={() => setPreview(null)}
            />
          </span>
          {preview.index > 0 && (
            <span className="form-lightbox__nav is-left">
              <Button
                kind="ghost"
                size="sm"
                hasIconOnly
                renderIcon={ChevronLeft}
                iconDescription="Imagem anterior"
                onClick={(event) => {
                  event.stopPropagation();
                  setPreview((current) =>
                    current && current.type === 'image'
                      ? { ...current, index: Math.max(0, current.index - 1) }
                      : current,
                  );
                }}
              />
            </span>
          )}
          <img
            className={`form-lightbox__image${zoom > MIN_ZOOM ? ' is-zoomed' : ''}${dragging ? ' is-dragging' : ''}`}
            src={preview.images[preview.index]}
            alt={preview.title}
            draggable={false}
            onDragStart={(event) => event.preventDefault()}
            style={{ transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})` }}
            onClick={(event) => event.stopPropagation()}
            onDoubleClick={(event) => {
              event.stopPropagation();
              if (zoom > MIN_ZOOM) {
                setZoom(MIN_ZOOM);
                setPan({ x: 0, y: 0 });
              } else {
                setZoom(2.5);
              }
            }}
            onPointerDown={(event) => {
              if (zoom <= MIN_ZOOM) return;
              event.currentTarget.setPointerCapture(event.pointerId);
              dragRef.current = {
                x: event.clientX,
                y: event.clientY,
                originX: pan.x,
                originY: pan.y,
              };
              setDragging(true);
            }}
            onPointerMove={(event) => {
              const drag = dragRef.current;
              if (!drag) return;
              setPan({
                x: drag.originX + event.clientX - drag.x,
                y: drag.originY + event.clientY - drag.y,
              });
            }}
            onPointerUp={() => {
              dragRef.current = null;
              setDragging(false);
            }}
            onPointerCancel={() => {
              dragRef.current = null;
              setDragging(false);
            }}
          />
          {preview.index < preview.images.length - 1 && (
            <span className="form-lightbox__nav is-right">
              <Button
                kind="ghost"
                size="sm"
                hasIconOnly
                renderIcon={ChevronRight}
                iconDescription="Próxima imagem"
                onClick={(event) => {
                  event.stopPropagation();
                  setPreview((current) =>
                    current && current.type === 'image'
                      ? { ...current, index: Math.min(current.images.length - 1, current.index + 1) }
                      : current,
                  );
                }}
              />
            </span>
          )}
          <div className="form-lightbox__bar" onClick={(event) => event.stopPropagation()}>
            <span>
              {preview.title} · {preview.index + 1} / {preview.images.length}
            </span>
            <div className="form-zoom-controls" role="group" aria-label="Zoom da imagem">
              <Button
                kind="ghost"
                size="sm"
                hasIconOnly
                renderIcon={ZoomOut}
                iconDescription="Diminuir zoom"
                disabled={zoom <= MIN_ZOOM}
                onClick={() => setZoom((current) => clampZoom(current - 0.5))}
              />
              <span className="form-zoom-value">{Math.round(zoom * 100)}%</span>
              <Button
                kind="ghost"
                size="sm"
                hasIconOnly
                renderIcon={ZoomIn}
                iconDescription="Aumentar zoom"
                disabled={zoom >= MAX_ZOOM}
                onClick={() => setZoom((current) => clampZoom(current + 0.5))}
              />
              <Button
                kind="ghost"
                size="sm"
                hasIconOnly
                renderIcon={Reset}
                iconDescription="Restaurar zoom"
                disabled={zoom <= MIN_ZOOM && pan.x === 0 && pan.y === 0}
                onClick={() => {
                  setZoom(MIN_ZOOM);
                  setPan({ x: 0, y: 0 });
                }}
              />
            </div>
            <Button
              kind="ghost"
              size="sm"
              renderIcon={Launch}
              onClick={() => window.open(preview.images[preview.index], '_blank')}
            >
              Abrir original
            </Button>
          </div>
        </div>
      )}

      {previewFile && (
        <div
          className="form-lightbox"
          role="dialog"
          aria-modal="true"
          aria-label={`Pré-visualização: ${previewFile.name}`}
          onClick={() => setPreview(null)}
        >
          <div className="form-preview-panel" onClick={(event) => event.stopPropagation()}>
            <header className="form-preview-header">
              <div>
                <strong>{previewFile.name}</strong>
                <span className="muted">{previewExtension ? previewExtension.toUpperCase() : 'Arquivo'}</span>
              </div>
              <div className="tag-row">
                <Button
                  kind="ghost"
                  size="sm"
                  renderIcon={Launch}
                  onClick={() => window.open(previewFile.url, '_blank')}
                >
                  Abrir
                </Button>
                <Button
                  kind="ghost"
                  size="sm"
                  hasIconOnly
                  renderIcon={Close}
                  iconDescription="Fechar"
                  onClick={() => setPreview(null)}
                />
              </div>
            </header>
            <div className="form-preview-body">
              {previewIsImage ? (
                <img src={previewFile.url} alt={previewFile.name} />
              ) : previewIsFrame ? (
                <iframe src={previewFile.url} title={previewFile.name} />
              ) : previewIsText ? (
                textLoading ? (
                  <InlineLoading description="Carregando pré-visualização…" />
                ) : textPreview ? (
                  <pre>{textPreview}</pre>
                ) : (
                  <p className="muted">Não foi possível carregar a pré-visualização. Use o botão Abrir.</p>
                )
              ) : (
                <p className="muted">
                  Pré-visualização não disponível para este formato. Use o botão Abrir para visualizar ou baixar o
                  arquivo.
                </p>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
