import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, Checkmark, DocumentAdd } from '@carbon/icons-react';
import { Button, InlineLoading, Modal, Select, SelectItem, TextInput } from '@carbon/react';
import { FormBuilder } from '../../components/FormBuilder';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import { formStatusTag } from '../../lib/forms';
import { Tag } from '@carbon/react';
import type { FormDetailResponse } from '../../lib/types';

interface ProjectOption {
  id: number;
  code: string;
  name: string;
}

export function FormEditPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const toast = useToast();
  const [data, setData] = useState<FormDetailResponse['data'] | null>(null);
  const [projects, setProjects] = useState<ProjectOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'details' | 'structure'>('details');
  const [saving, setSaving] = useState(false);
  const [templateOpen, setTemplateOpen] = useState(false);
  const [templateName, setTemplateName] = useState('');
  const [savingTemplate, setSavingTemplate] = useState(false);
  const [details, setDetails] = useState({
    name: '',
    project_id: '',
    type: '',
    status: '',
    allows_multiple_responses: '0',
  });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const [detail, projectResponse] = await Promise.all([
        api.get<FormDetailResponse>(`/forms/${id}`),
        api.get<{ data: { projects: ProjectOption[] } }>('/projects').catch(() => ({ data: { projects: [] } })),
      ]);
      setData(detail.data);
      setProjects(projectResponse.data?.projects ?? []);
      const form = detail.data.form;
      setDetails({
        name: form.name,
        project_id: form.project_id ? String(form.project_id) : '',
        type: form.type,
        status: form.status,
        allows_multiple_responses: form.allows_multiple_responses ? '1' : '0',
      });
    } catch (error) {
      toast.error('Não foi possível carregar o formulário', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [id, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const onBuilderSaved = useCallback(() => {
    if (!id) return;
    void api
      .get<FormDetailResponse>(`/forms/${id}`)
      .then((detail) => setData(detail.data))
      .catch(() => undefined);
  }, [id]);

  async function saveDetails() {
    if (!id || !data) return;
    if (!details.name.trim()) {
      setFieldErrors({ name: 'Informe o nome do formulário.' });
      return;
    }
    setSaving(true);
    try {
      const response = await api.put<{ message: string }>(`/forms/${id}`, {
        name: details.name.trim(),
        project_id: details.project_id ? Number(details.project_id) : null,
        type: details.type,
        status: details.status,
        allows_multiple_responses: details.allows_multiple_responses === '1',
      });
      toast.success('Formulário atualizado', response.message);
      setFieldErrors({});
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar o formulário', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function saveTemplate() {
    if (!id || !data) return;
    if (!templateName.trim()) return;
    setSavingTemplate(true);
    try {
      const response = await api.post<{ message: string }>('/form-templates', {
        name: templateName.trim(),
        type: data.form.type,
        form_id: data.form.id,
      });
      toast.success('Template salvo', response.message);
      setTemplateOpen(false);
    } catch (error) {
      toast.error('Não foi possível salvar o template', errorMessage(error));
    } finally {
      setSavingTemplate(false);
    }
  }

  if (loading || !data) {
    return <InlineLoading description="Carregando formulário…" />;
  }

  const { form, settings } = data;
  const typeOptions = [form.type, ...(settings.types ?? [])].filter((item, index, list) => list.indexOf(item) === index);
  const statusOptions = [form.status, ...(settings.statuses ?? [])].filter(
    (item, index, list) => list.indexOf(item) === index,
  );

  return (
    <div className="page">
      <div className="form-detail-head">
        <div>
          <nav className="form-breadcrumb">
            <button type="button" onClick={() => navigate('/forms')}>Formulários</button>
            <span>/</span>
            <button type="button" onClick={() => navigate(`/forms/${form.id}`)}>{form.code}</button>
            <span>/</span>
            <span>Editar</span>
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
          <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/forms')}>Voltar</Button>
          <Button kind="tertiary" renderIcon={DocumentAdd} onClick={() => {
            setTemplateName(form.name);
            setTemplateOpen(true);
          }}>
            Salvar template
          </Button>
          <Button renderIcon={Checkmark} onClick={() => navigate(`/forms/${form.id}/evaluate`)}>
            Avaliar resposta
          </Button>
        </div>
      </div>

      <div className="form-tabs" role="tablist">
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'details'}
          className={tab === 'details' ? 'is-active' : ''}
          onClick={() => setTab('details')}
        >
          Dados
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={tab === 'structure'}
          className={tab === 'structure' ? 'is-active' : ''}
          onClick={() => setTab('structure')}
        >
          Estrutura <span>{form.steps.reduce((sum, step) => sum + step.fields.length, 0)}</span>
        </button>
      </div>

      {tab === 'details' ? (
        <div className="form-edit-grid">
          <section className="form-panel">
            <header className="form-panel__head">
              <h2>Dados do formulário</h2>
            </header>
            <div className="form-grid">
              <Select
                id="edit-project"
                labelText="Projeto (opcional)"
                value={details.project_id}
                onChange={(event) => setDetails((current) => ({ ...current, project_id: event.target.value }))}
              >
                <SelectItem value="" text="Sem projeto" />
                {projects.map((project) => (
                  <SelectItem key={project.id} value={String(project.id)} text={`[${project.code}] ${project.name}`} />
                ))}
              </Select>

              <TextInput
                id="edit-name"
                labelText="Nome do formulário"
                value={details.name}
                invalid={Boolean(fieldErrors.name)}
                invalidText={fieldErrors.name}
                onChange={(event) => setDetails((current) => ({ ...current, name: event.target.value }))}
              />

              <Select
                id="edit-type"
                labelText="Tipo"
                value={details.type}
                onChange={(event) => setDetails((current) => ({ ...current, type: event.target.value }))}
              >
                {typeOptions.map((type) => (
                  <SelectItem key={type} value={type} text={type} />
                ))}
              </Select>

              <Select
                id="edit-status"
                labelText="Status"
                value={details.status}
                onChange={(event) => setDetails((current) => ({ ...current, status: event.target.value }))}
              >
                {statusOptions.map((status) => (
                  <SelectItem key={status} value={status} text={status} />
                ))}
              </Select>

              <Select
                id="edit-multiple"
                labelText="Respostas"
                value={details.allows_multiple_responses}
                onChange={(event) =>
                  setDetails((current) => ({ ...current, allows_multiple_responses: event.target.value }))
                }
              >
                <SelectItem value="0" text="Resposta única" />
                <SelectItem value="1" text="Múltiplas respostas" />
              </Select>
            </div>
            <div className="form-panel__actions">
              <Button kind="ghost" onClick={() => navigate('/forms')}>Cancelar</Button>
              <Button onClick={() => void saveDetails()} disabled={saving}>
                {saving ? 'Salvando…' : 'Salvar alterações'}
              </Button>
            </div>
          </section>

          <aside className="form-edit-aside">
            <div className="form-side-card">
              <h3>Estrutura</h3>
              <div className="form-side-stats">
                <div><strong>{form.steps.length}</strong><span>Seções</span></div>
                <div><strong>{form.steps.reduce((sum, step) => sum + step.fields.length, 0)}</strong><span>Perguntas</span></div>
              </div>
              <Button kind="tertiary" size="sm" onClick={() => setTab('structure')}>Editar estrutura</Button>
            </div>
            <div className="form-side-card">
              <h3>Registro</h3>
              <dl className="form-side-list">
                <div><dt>Criado em</dt><dd>{new Date(form.created_at).toLocaleString('pt-BR')}</dd></div>
                <div><dt>Atualizado em</dt><dd>{new Date(form.updated_at).toLocaleString('pt-BR')}</dd></div>
                <div><dt>Respostas</dt><dd>{form.response_count}</dd></div>
                <div><dt>Avaliações</dt><dd>{form.evaluation_count}</dd></div>
              </dl>
            </div>
            <div className="form-side-card">
              <h3>Link público</h3>
              <p className="form-side-hint">Compartilhe este endereço para preenchimento externo.</p>
              <code className="form-public-link">{window.location.origin}/f/{form.public_token}</code>
              <Button
                kind="tertiary"
                size="sm"
                onClick={() => {
                  void navigator.clipboard
                    .writeText(`${window.location.origin}/f/${form.public_token}`)
                    .then(() => toast.success('Link público copiado'));
                }}
              >
                Copiar link
              </Button>
            </div>
          </aside>
        </div>
      ) : (
        <section className="form-panel">
          <header className="form-panel__head">
            <h2>Estrutura do formulário</h2>
            <p className="form-panel__hint">As alterações da estrutura são salvas automaticamente.</p>
          </header>
          <FormBuilder formId={form.id} initialSteps={form.steps} onSaved={onBuilderSaved} />
        </section>
      )}

      <Modal
        open={templateOpen}
        modalHeading="Salvar como template"
        primaryButtonText={savingTemplate ? 'Salvando…' : 'Salvar template'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={savingTemplate || !templateName.trim()}
        onRequestSubmit={() => void saveTemplate()}
        onRequestClose={() => setTemplateOpen(false)}
        size="sm"
      >
        <p className="form-panel__hint">
          As seções e perguntas atuais serão copiadas. O template fica disponível para todos os usuários.
        </p>
        <TextInput
          id="template-name"
          labelText="Nome do template"
          value={templateName}
          onChange={(event) => setTemplateName(event.target.value)}
        />
      </Modal>
    </div>
  );
}
