import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Checkmark, Copy, Edit, Launch, View } from '@carbon/icons-react';
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
  formStatusTag,
  publicFormLink,
} from '../../lib/forms';
import type { FormDetailResponse, FormEvaluationRecord, FormField, FormResponseRecord, FormStep } from '../../lib/types';

function AnswerField({ field }: { field: FormField }) {
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
            {field.images?.map((image, index) => (
              <a key={image} href={image} target="_blank" rel="noopener noreferrer">
                <img src={field.imageThumbnails?.[index] ?? image} alt={field.label} loading="lazy" />
              </a>
            ))}
          </div>
        ) : (
          <p className="muted">{field.status === 'nao_aplicavel' ? 'Não aplicável.' : 'Nenhuma foto.'}</p>
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

  const selectedResponseId = searchParams.get('response');

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
  const selectedResponse: FormResponseRecord | null =
    (selectedResponseId ? responses.find((item) => String(item.id) === selectedResponseId) : responses[0]) ?? null;
  const steps: FormStep[] = selectedResponse?.steps ?? (responses.length === 0 ? form.steps : []);
  const answeredCount = countAnsweredFields(steps);
  const hasAnswers = answeredCount > 0;
  const publicLink = publicFormLink(form.public_token);
  const evaluateRoute = `/forms/${form.id}/evaluate${selectedResponse ? `?response=${selectedResponse.id}` : ''}`;

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

      <div className="station-summary-grid">
        <Tile>
          <span>Respostas</span>
          <strong>{responses.length}</strong>
        </Tile>
        <Tile>
          <span>Avaliações</span>
          <strong>{form.evaluation_count}</strong>
        </Tile>
        <Tile>
          <span>Atualizado</span>
          <strong>{relativeTime(form.updated_at)}</strong>
        </Tile>
      </div>

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
                      <AnswerField field={field} key={field.id} />
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
              <div><dt>Seções</dt><dd>{form.steps.length}</dd></div>
              <div><dt>Perguntas</dt><dd>{form.steps.reduce((sum, step) => sum + step.fields.length, 0)}</dd></div>
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
    </div>
  );
}
