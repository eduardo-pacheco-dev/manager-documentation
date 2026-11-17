import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, Checkmark, Copy, DocumentAdd, Edit, Launch, View } from '@carbon/icons-react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  InlineLoading,
  Modal,
  Select,
  SelectItem,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Tag,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { FormBuilder } from '../../components/FormBuilder';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import { formStatusTag, publicFormLink } from '../../lib/forms';
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
  const [notFound, setNotFound] = useState(false);
  const [tab, setTab] = useState(0);
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
      setNotFound(true);
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

  const { form, settings } = data;
  const typeOptions = [form.type, ...(settings.types ?? [])].filter((item, index, list) => list.indexOf(item) === index);
  const statusOptions = [form.status, ...(settings.statuses ?? [])].filter(
    (item, index, list) => list.indexOf(item) === index,
  );
  const publicLink = publicFormLink(form.public_token);

  return (
    <div className="page">
      <Breadcrumb noTrailingSlash className="doc-breadcrumb">
        <BreadcrumbItem><Link to="/forms">Formulários</Link></BreadcrumbItem>
        <BreadcrumbItem><Link to={`/forms/${form.id}`}>{form.code}</Link></BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>Editar</BreadcrumbItem>
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
          </span>
        }
        actions={
          <>
            <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/forms')}>Voltar</Button>
            <Button
              kind="tertiary"
              renderIcon={DocumentAdd}
              onClick={() => {
                setTemplateName(form.name);
                setTemplateOpen(true);
              }}
            >
              Salvar template
            </Button>
            <Button renderIcon={Checkmark} onClick={() => navigate(`/forms/${form.id}/evaluate`)}>
              Avaliar resposta
            </Button>
          </>
        }
      />

      <div className="station-summary-grid">
        <Tile>
          <span>Seções</span>
          <strong>{form.steps.length}</strong>
        </Tile>
        <Tile>
          <span>Perguntas</span>
          <strong>{form.steps.reduce((sum, step) => sum + step.fields.length, 0)}</strong>
        </Tile>
        <Tile>
          <span>Respostas</span>
          <strong>{form.response_count}</strong>
        </Tile>
      </div>

      <Tabs selectedIndex={tab} onChange={({ selectedIndex }) => setTab(selectedIndex)}>
        <TabList aria-label="Configuração do formulário" contained>
          <Tab>Dados</Tab>
          <Tab>Estrutura</Tab>
        </TabList>
        <TabPanels>
          <TabPanel>
            <div className="nokia-detail-layout">
              <div className="nokia-detail-main">
                <Tile className="nokia-detail-section">
                  <div className="nokia-section-header">
                    <div>
                      <h2>Dados do formulário</h2>
                      <p>Atualize as informações gerais do formulário.</p>
                    </div>
                  </div>

                  <div className="form-grid">
                    <Select
                      id="edit-project"
                      labelText="Projeto (opcional)"
                      value={details.project_id}
                      onChange={(event) => setDetails((current) => ({ ...current, project_id: event.target.value }))}
                    >
                      <SelectItem value="" text="Sem projeto" />
                      {projects.map((project) => (
                        <SelectItem
                          key={project.id}
                          value={String(project.id)}
                          text={`[${project.code || `PRJ-${String(project.id).padStart(4, '0')}`}] ${project.name}`}
                        />
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
                  <p className="field-help">
                    Com resposta única, o mesmo registro é atualizado a cada novo envio. Com múltiplas respostas, cada
                    envio cria um registro separado no histórico.
                  </p>

                  <div className="section-actions">
                    <Button kind="ghost" onClick={() => navigate('/forms')}>Cancelar</Button>
                    <Button onClick={() => void saveDetails()} disabled={saving}>
                      {saving ? 'Salvando…' : 'Salvar alterações'}
                    </Button>
                  </div>
                </Tile>
              </div>

              <aside className="nokia-detail-aside">
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
                  <h2>Registro</h2>
                  <dl>
                    <div><dt>Criado em</dt><dd>{relativeTime(form.created_at)}</dd></div>
                    <div><dt>Atualizado em</dt><dd>{relativeTime(form.updated_at)}</dd></div>
                    <div><dt>Respostas</dt><dd>{form.response_count}</dd></div>
                    <div><dt>Avaliações</dt><dd>{form.evaluation_count}</dd></div>
                  </dl>
                </Tile>

                <Tile className="nokia-detail-aside__card">
                  <h2>Ações</h2>
                  <div className="nokia-aside-actions">
                    <Button kind="ghost" size="sm" renderIcon={View} onClick={() => navigate(`/forms/${form.id}`)}>
                      Visualizar respostas
                    </Button>
                    <Button kind="ghost" size="sm" renderIcon={Edit} onClick={() => setTab(1)}>Editar estrutura</Button>
                    <Button kind="ghost" size="sm" renderIcon={Checkmark} onClick={() => navigate(`/forms/${form.id}/evaluate`)}>
                      Avaliar resposta
                    </Button>
                  </div>
                </Tile>
              </aside>
            </div>
          </TabPanel>

          <TabPanel>
            <Tile className="nokia-detail-section">
              <div className="nokia-section-header">
                <div>
                  <h2>Estrutura do formulário</h2>
                  <p>As alterações das seções e perguntas são salvas automaticamente.</p>
                </div>
              </div>
              <FormBuilder formId={form.id} initialSteps={form.steps} onSaved={onBuilderSaved} />
            </Tile>
          </TabPanel>
        </TabPanels>
      </Tabs>

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
        <div className="modal-form">
          <p className="muted">
            As seções e perguntas atuais serão copiadas. O template fica disponível para todos os usuários.
          </p>
          <TextInput
            id="template-name"
            labelText="Nome do template"
            value={templateName}
            onChange={(event) => setTemplateName(event.target.value)}
          />
        </div>
      </Modal>
    </div>
  );
}
