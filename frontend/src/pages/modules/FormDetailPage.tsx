import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Checkmark, Edit, View } from '@carbon/icons-react';
import { Button, InlineLoading, Tag } from '@carbon/react';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import {
  FORM_FIELD_STATUS_LABELS,
  FORM_TYPE_LABELS,
  countAnsweredFields,
  fieldStatusTag,
  formStatusTag,
} from '../../lib/forms';
import type { FormDetailResponse, FormEvaluationRecord, FormField, FormResponseRecord, FormStep } from '../../lib/types';

function FieldAnswer({ field }: { field: FormField }) {
  return (
    <div className="form-answer">
      <div className="form-answer__head">
        <div>
          <p className="form-answer__label">{field.label || field.id}</p>
          <p className="form-answer__meta">
            {field.id} · {FORM_TYPE_LABELS[field.type] ?? field.type}
          </p>
        </div>
        {field.status && field.status !== 'enviado' && field.status !== 'pendente' && (
          <Tag type={fieldStatusTag(field.status)} size="sm">
            {FORM_FIELD_STATUS_LABELS[field.status] ?? field.status}
          </Tag>
        )}
      </div>

      {field.type === 'image' ? (
        (field.images?.length ?? 0) > 0 ? (
          <div className="form-image-grid">
            {field.images?.map((image, index) => (
              <a key={image} href={image} target="_blank" rel="noopener noreferrer" className="form-image">
                <img src={field.imageThumbnails?.[index] ?? image} alt={field.label} loading="lazy" />
              </a>
            ))}
          </div>
        ) : (
          <p className="form-empty-answer">{field.status === 'nao_aplicavel' ? 'Não aplicável.' : 'Nenhuma foto.'}</p>
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
          <p className="form-empty-answer">Nenhum arquivo.</p>
        )
      ) : (
        <div className="form-answer__value">{field.value || '—'}</div>
      )}
    </div>
  );
}

function EvaluationItem({ evaluation }: { evaluation: FormEvaluationRecord }) {
  return (
    <div className="form-evaluation-item">
      <div className="form-evaluation-item__head">
        <Tag type={formStatusTag(evaluation.status)} size="sm">
          {evaluation.status}
        </Tag>
        <strong>{evaluation.user_name ?? 'Sistema'}</strong>
        <span>{relativeTime(evaluation.created_at)}</span>
      </div>
      {(evaluation.changes?.length ?? 0) > 0 && (
        <ul className="form-evaluation-changes">
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
      {evaluation.notes && <p className="form-evaluation-notes">{evaluation.notes}</p>}
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

  const selectedResponseId = searchParams.get('response');

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const query = selectedResponseId ? `?response=${selectedResponseId}` : '';
      const response = await api.get<FormDetailResponse>(`/forms/${id}${query}`);
      setData(response.data);
    } catch (error) {
      toast.error('Não foi possível carregar o formulário', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [id, selectedResponseId, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading || !data) {
    return <InlineLoading description="Carregando formulário…" />;
  }

  const { form, responses, evaluations } = data;
  const selectedResponse: FormResponseRecord | null =
    (selectedResponseId ? responses.find((item) => String(item.id) === selectedResponseId) : responses[0]) ?? null;
  const steps: FormStep[] = selectedResponse?.steps ?? (responses.length === 0 ? form.steps : []);
  const answeredCount = countAnsweredFields(steps);
  const hasAnswers = answeredCount > 0;

  return (
    <div className="page">
      <div className="form-detail-head">
        <div>
          <nav className="form-breadcrumb">
            <Link to="/forms">Formulários</Link>
            <span>/</span>
            <span>{form.code}</span>
          </nav>
          <div className="form-detail-title">
            <h1>{form.code}</h1>
            <Tag type={formStatusTag(form.status)} size="sm">{form.status}</Tag>
          </div>
          <p className="form-detail-subtitle">{form.name}</p>
          <div className="form-detail-meta">
            <span>{form.type}</span>
            <span>{form.allows_multiple_responses ? 'Múltiplas respostas' : 'Resposta única'}</span>
            {form.project_name && <span>[{form.project_code}] {form.project_name}</span>}
            <span>Atualizado {relativeTime(form.updated_at)}</span>
          </div>
        </div>
        <div className="page-header__actions">
          <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/forms')}>
            Voltar
          </Button>
          <Button kind="tertiary" renderIcon={Checkmark} onClick={() => navigate(`/forms/${form.id}/evaluate${selectedResponse ? `?response=${selectedResponse.id}` : ''}`)}>
            Avaliar resposta
          </Button>
          <Button renderIcon={Edit} onClick={() => navigate(`/forms/${form.id}/edit`)}>
            Editar
          </Button>
        </div>
      </div>

      {responses.length > 0 && (
        <section className="form-panel">
          <header className="form-panel__head">
            <h2>Histórico de respostas</h2>
            <Tag type="gray" size="sm">{responses.length}</Tag>
          </header>
          <div className="form-response-list">
            {responses.map((response, index) => {
              const selected = selectedResponse?.id === response.id;
              return (
                <div className={`form-response-row${selected ? ' is-selected' : ''}`} key={response.id}>
                  <div>
                    <div className="form-response-row__title">
                      <Tag type={formStatusTag(response.status)} size="sm">{response.status}</Tag>
                      <strong>Resposta #{index + 1}</strong>
                      <span>{relativeTime(response.submitted_at ?? response.created_at)}</span>
                    </div>
                    <p className="form-response-row__meta">
                      {response.user_name ?? 'Link externo'} · {countAnsweredFields(response.steps)} resposta(s) preenchida(s)
                      {response.evaluations_count > 0 ? ` · ${response.evaluations_count} avaliação(ões)` : ''}
                    </p>
                  </div>
                  <div className="form-response-row__actions">
                    <Button
                      kind="ghost"
                      size="sm"
                      onClick={() => setSearchParams({ response: String(response.id) })}
                    >
                      Ver
                    </Button>
                    <Button
                      kind="tertiary"
                      size="sm"
                      onClick={() => navigate(`/forms/${form.id}/evaluate?response=${response.id}`)}
                    >
                      Avaliar
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        </section>
      )}

      <section className="form-panel">
        <header className="form-panel__head">
          <h2>Respostas do aplicativo</h2>
          {hasAnswers && <Tag type="gray" size="sm">{answeredCount} resposta(s)</Tag>}
        </header>
        <p className="form-panel__hint">
          {selectedResponse
            ? `Exibindo a resposta enviada ${relativeTime(selectedResponse.submitted_at ?? selectedResponse.created_at)}. Somente leitura.`
            : 'Valores, status e fotos preenchidos pelos técnicos. Somente leitura.'}
        </p>

        {hasAnswers ? (
          <div className="form-answer-sections">
            {steps.map((step) => {
              const answered = step.fields.filter(
                (field) => field.value || (field.images?.length ?? 0) > 0 || (field.files?.length ?? 0) > 0,
              );
              if (answered.length === 0) return null;
              return (
                <div className="form-answer-section" key={step.id}>
                  <header>
                    <h3>{step.title || 'Seção'}</h3>
                    <span>{answered.length} resposta(s)</span>
                  </header>
                  <div className="form-answer-section__body">
                    {answered.map((field) => (
                      <FieldAnswer field={field} key={field.id} />
                    ))}
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <p className="form-empty-answer">Nenhuma resposta preenchida pelo aplicativo até o momento.</p>
        )}
      </section>

      <section className="form-panel">
        <header className="form-panel__head">
          <h2>Histórico de avaliações</h2>
          <Tag type="gray" size="sm">{evaluations.length}</Tag>
        </header>
        {evaluations.length > 0 ? (
          <div className="form-evaluation-list">
            {evaluations.map((evaluation) => (
              <EvaluationItem evaluation={evaluation} key={evaluation.id} />
            ))}
          </div>
        ) : (
          <p className="form-empty-answer">Nenhuma avaliação registrada até o momento.</p>
        )}
      </section>
    </div>
  );
}
