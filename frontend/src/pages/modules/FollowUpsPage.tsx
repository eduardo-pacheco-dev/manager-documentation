import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Add, Download, Edit, Send, Settings, TrashCan, Upload } from '@carbon/icons-react';
import {
  Button,
  InlineLoading,
  Modal,
  Search,
  Select,
  SelectItem,
  Tag,
  TextArea,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState } from '../../components/Feedback';
import { useAuth } from '../../contexts/AuthContext';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';

interface Client { id: number; name: string; projects: { id: number; name: string; client_id: number }[]; project_count: number; follow_up_count: number }
interface Assignee { id: number; name: string; email: string | null; follow_up_count: number }
interface Update { id: number; body: string; created_at: string; author_name: string | null; status_from: string | null; status_to: string | null; user_id: number | null }
interface FollowUp {
  id: number; title: string; code: string | null; description: string | null; client_id: number | null;
  project_id: number | null; status: string; priority: string; due_date: string | null;
  assignee_id: number | null; completed_at: string | null; created_at: string; updated_at: string;
  client_name: string | null; project_name: string | null; assignee_name: string | null;
  assignee_email: string | null; creator_name: string | null; overdue: boolean; due_soon: boolean; updates: Update[];
}
interface Settings { per_page: number; due_soon_days: number; widget_limit: number; email_subject_prefix: string; summary_period: string; summary_times: string; summary_days: string; summary_recipients: string | null; summary_cc_recipients: string | null; email_summary_enabled: boolean }
interface Data { data: FollowUp[]; stats: { open: number; overdue: number; completed: number; mine: number }; clients: Client[]; assignees: Assignee[]; meta: { current_page: number; last_page: number; per_page: number; total: number }; settings: Settings | null }

const STATUS: Record<string, string> = { pendente: 'Pendente', em_andamento: 'Em andamento', concluido: 'Concluído', cancelado: 'Cancelado' };
const PRIORITY: Record<string, string> = { baixa: 'Baixa', media: 'Média', alta: 'Alta', urgente: 'Urgente' };
const statusType = (value: string) => value === 'concluido' ? 'green' : value === 'cancelado' ? 'red' : value === 'em_andamento' ? 'blue' : 'magenta';

export function FollowUpsPage() {
  const { user } = useAuth();
  const toast = useToast();
  const [searchParams] = useSearchParams();
  const [data, setData] = useState<Data | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [priority, setPriority] = useState('');
  const [assigned, setAssigned] = useState('');
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<FollowUp | null>(null);
  const [detail, setDetail] = useState<FollowUp | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [updateBody, setUpdateBody] = useState('');
  const [form, setForm] = useState({
    title: '', code: '', description: '', client_id: '', project_id: '', status: 'pendente',
    priority: 'media', due_date: '', completed_at: '', assignee_id: '',
  });
  const [settingsForm, setSettingsForm] = useState<Settings | null>(null);
  const [newAssignee, setNewAssignee] = useState({ name: '', email: '' });
  const [newClient, setNewClient] = useState('');
  const [newProject, setNewProject] = useState({ client_id: '', name: '' });

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({ page: String(page) });
    if (search) params.set('search', search);
    if (status) params.set('status', status);
    if (priority) params.set('priority', priority);
    if (assigned) params.set('assigned', assigned);
    try {
      const response = await api.get<Data>(`/follow-ups?${params.toString()}`);
      setData(response);
      if (response.settings) setSettingsForm(response.settings);
    } catch (error) {
      toast.error('Não foi possível carregar os follow-ups', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [page, search, status, priority, assigned, toast]);

  useEffect(() => { void load(); }, [load]);

  async function openDetailById(id: number) {
    try {
      const response = await api.get<{ data: { follow_up: FollowUp } }>(`/follow-ups/${id}`);
      setDetail(response.data.follow_up);
    } catch (error) {
      toast.error('Não foi possível abrir o follow-up', errorMessage(error));
    }
  }

  useEffect(() => {
    const deepLink = Number(searchParams.get('followup'));
    if (Number.isFinite(deepLink) && deepLink > 0) void openDetailById(deepLink);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  function openCreate() {
    setEditing(null);
    setForm({ title: '', code: '', description: '', client_id: '', project_id: '', status: 'pendente', priority: 'media', due_date: '', completed_at: '', assignee_id: '' });
    setModalOpen(true);
  }

  function openEdit(item: FollowUp) {
    setEditing(item);
    setForm({
      title: item.title, code: item.code ?? '', description: item.description ?? '', client_id: item.client_id ? String(item.client_id) : '',
      project_id: item.project_id ? String(item.project_id) : '', status: item.status, priority: item.priority,
      due_date: item.due_date ?? '', completed_at: item.completed_at ? item.completed_at.slice(0, 10) : '',
      assignee_id: item.assignee_id ? String(item.assignee_id) : '',
    });
    setModalOpen(true);
  }

  async function save() {
    setSaving(true);
    try {
      const payload = {
        ...form, client_id: form.client_id ? Number(form.client_id) : null, project_id: form.project_id ? Number(form.project_id) : null,
        assignee_id: form.assignee_id ? Number(form.assignee_id) : null,
      };
      if (editing) await api.put(`/follow-ups/${editing.id}`, payload);
      else await api.post('/follow-ups', payload);
      toast.success(editing ? 'Follow-up atualizado.' : 'Follow-up criado.');
      setModalOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function openDetail(item: FollowUp) {
    await openDetailById(item.id);
  }

  async function addUpdate() {
    if (!detail || !updateBody.trim()) return;
    try {
      await api.post(`/follow-ups/${detail.id}/updates`, { body: updateBody });
      setUpdateBody('');
      await openDetail(detail);
    } catch (error) {
      toast.error('Não foi possível registrar', errorMessage(error));
    }
  }

  async function saveSettings() {
    if (!settingsForm) return;
    setSaving(true);
    try {
      await api.put('/follow-ups/settings', settingsForm);
      toast.success('Configurações salvas.');
      setSettingsOpen(false);
    } catch (error) {
      toast.error('Não foi possível salvar as configurações', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  const items = data?.data ?? [];
  const clients = data?.clients ?? [];
  const assignees = data?.assignees ?? [];
  const projects = clients.flatMap((client) => client.projects.map((project) => ({ ...project, clientName: client.name })));
  const filteredProjects = form.client_id ? projects.filter((project) => project.client_id === Number(form.client_id)) : projects;

  return (
    <div className="page">
      <PageHeader
        eyebrow="Acompanhamento"
        title="Follow-ups"
        description="Acompanhe pendências, prazos, responsáveis e histórico."
        actions={
          <>
            <Button kind="ghost" renderIcon={Download} onClick={() => window.open('/api/v1/follow-ups/export', '_blank')}>Exportar</Button>
            <Button kind="ghost" renderIcon={Upload} onClick={() => { const input = document.createElement('input'); input.type = 'file'; input.accept = '.csv'; input.onchange = async () => { if (!input.files?.[0]) return; const formData = new FormData(); formData.append('file', input.files[0]); try { const response = await api.upload<{ message: string }>('/follow-ups/import', formData); toast.success(response.message); await load(); } catch (error) { toast.error('Falha na importação', errorMessage(error)); } }; input.click(); }}>Importar</Button>
            {user?.is_admin && <Button kind="ghost" renderIcon={Settings} onClick={() => { setSettingsForm(data?.settings ?? null); setSettingsOpen(true); }}>Configurações</Button>}
            <Button renderIcon={Add} onClick={openCreate}>Novo follow-up</Button>
          </>
        }
      />

      {data && (
        <div className="stats-grid">
          <Tile className="stat-card stat-card--blue"><span className="stat-card__label">Em aberto</span><strong className="stat-card__value">{data.stats.open}</strong></Tile>
          <Tile className="stat-card stat-card--red"><span className="stat-card__label">Atrasados</span><strong className="stat-card__value">{data.stats.overdue}</strong></Tile>
          <Tile className="stat-card stat-card--green"><span className="stat-card__label">Concluídos</span><strong className="stat-card__value">{data.stats.completed}</strong></Tile>
          <Tile className="stat-card stat-card--purple"><span className="stat-card__label">Atribuídos a mim</span><strong className="stat-card__value">{data.stats.mine}</strong></Tile>
        </div>
      )}

      <Tile className="module-main__header">
        <div className="module-toolbar">
          <Search size="lg" labelText="Buscar follow-ups" placeholder="Buscar por título, código, cliente..." value={search} onChange={(event) => { setSearch(event.target.value); setPage(1); }} onClear={() => setSearch('')} />
          <Select id="fu-status" labelText="Status" size="sm" value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todos" />
            {Object.entries(STATUS).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
          </Select>
          <Select id="fu-priority" labelText="Prioridade" size="sm" value={priority} onChange={(event) => { setPriority(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todas" />
            {Object.entries(PRIORITY).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
          </Select>
          <Select id="fu-assigned" labelText="Responsável" size="sm" value={assigned} onChange={(event) => { setAssigned(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todos" />
            <SelectItem value="me" text="Atribuídos a mim" />
            {assignees.map((assignee) => <SelectItem key={assignee.id} value={String(assignee.id)} text={assignee.name} />)}
          </Select>
        </div>
      </Tile>

      {loading ? <InlineLoading description="Carregando follow-ups…" /> : items.length === 0 ? (
        <EmptyState icon={Add} title="Nenhum follow-up encontrado" description="Crie um novo acompanhamento ou ajuste os filtros." action={<Button renderIcon={Add} onClick={openCreate}>Novo follow-up</Button>} />
      ) : (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table">
              <thead><tr><th>Follow-up</th><th>Código</th><th>Status</th><th>Prioridade</th><th className="hide-sm">Responsável</th><th>Prazo</th><th>Ações</th></tr></thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id}>
                    <td><button type="button" className="link-button" onClick={() => void openDetail(item)}>{item.title}</button>{item.client_name && <span className="cell-secondary">{item.client_name}{item.project_name ? ` · ${item.project_name}` : ''}</span>}</td>
                    <td>{item.code ?? '—'}</td>
                    <td><Tag type={statusType(item.status)} size="sm">{STATUS[item.status]}</Tag></td>
                    <td><Tag type={item.priority === 'urgente' ? 'red' : item.priority === 'alta' ? 'magenta' : 'gray'} size="sm">{PRIORITY[item.priority]}</Tag>{item.overdue && <Tag type="red" size="sm">Atrasado</Tag>}</td>
                    <td className="hide-sm">{item.assignee_name ?? 'Não atribuído'}</td>
                    <td className={item.overdue ? 'text-danger' : ''}>{item.due_date ?? '—'}</td>
                    <td className="cell-actions">
                      <Button kind="ghost" size="sm" hasIconOnly renderIcon={Edit} iconDescription="Editar" onClick={() => openEdit(item)} />
                      <Button kind="danger--ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={async () => { if (!window.confirm(`Excluir “${item.title}”?`)) return; await api.delete(`/follow-ups/${item.id}`); toast.success('Follow-up excluído.'); await load(); }} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="table-footer"><span className="table-footer__order">{data ? `${data.meta.total} follow-up(s)` : ''}</span></div>
        </div>
      )}

      {data && data.meta.last_page > 1 && (
        <div className="pager">
          <Button kind="ghost" size="sm" disabled={page <= 1} onClick={() => setPage((current) => Math.max(1, current - 1))}>Anterior</Button>
          <span>Página {data.meta.current_page} de {data.meta.last_page}</span>
          <Button kind="ghost" size="sm" disabled={page >= data.meta.last_page} onClick={() => setPage((current) => current + 1)}>Próxima</Button>
        </div>
      )}

      <Modal open={modalOpen} modalHeading={editing ? 'Editar follow-up' : 'Novo follow-up'} primaryButtonText={saving ? 'Salvando…' : 'Salvar'} secondaryButtonText="Cancelar" primaryButtonDisabled={saving} onRequestSubmit={() => void save()} onRequestClose={() => setModalOpen(false)} size="md">
        <div className="modal-form">
          <TextInput id="fu-title" labelText="Título" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} required />
          <div className="form-row">
            <TextInput id="fu-code" labelText="Código" value={form.code} onChange={(event) => setForm({ ...form, code: event.target.value })} />
            <TextInput id="fu-due" type="date" labelText="Prazo" value={form.due_date} onChange={(event) => setForm({ ...form, due_date: event.target.value })} />
          </div>
          <TextArea id="fu-description" labelText="Descrição" value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} rows={3} />
          <div className="form-row">
            <Select id="fu-client" labelText="Cliente" value={form.client_id} onChange={(event) => setForm({ ...form, client_id: event.target.value, project_id: '' })}>
              <SelectItem value="" text="Sem cliente" />
              {clients.map((client) => <SelectItem key={client.id} value={String(client.id)} text={client.name} />)}
            </Select>
            <Select id="fu-project" labelText="Projeto" value={form.project_id} onChange={(event) => setForm({ ...form, project_id: event.target.value })}>
              <SelectItem value="" text="Sem projeto" />
              {filteredProjects.map((project) => <SelectItem key={project.id} value={String(project.id)} text={project.name} />)}
            </Select>
          </div>
          <div className="form-row">
            <Select id="fu-status-field" labelText="Status" value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })}>
              {Object.entries(STATUS).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
            </Select>
            <Select id="fu-priority-field" labelText="Prioridade" value={form.priority} onChange={(event) => setForm({ ...form, priority: event.target.value })}>
              {Object.entries(PRIORITY).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
            </Select>
          </div>
          <div className="form-row">
            <Select id="fu-assignee" labelText="Responsável" value={form.assignee_id} onChange={(event) => setForm({ ...form, assignee_id: event.target.value })}>
              <SelectItem value="" text="Não atribuído" />
              {assignees.map((assignee) => <SelectItem key={assignee.id} value={String(assignee.id)} text={assignee.name} />)}
            </Select>
            <TextInput id="fu-completed" type="date" labelText="Concluído em" value={form.completed_at} onChange={(event) => setForm({ ...form, completed_at: event.target.value })} disabled={form.status !== 'concluido'} />
          </div>
        </div>
      </Modal>

      <Modal open={Boolean(detail)} modalHeading={detail?.title ?? ''} primaryButtonText="Fechar" onRequestClose={() => setDetail(null)} onRequestSubmit={() => setDetail(null)} size="md">
        {detail && (
          <div className="modal-form">
            <div className="tag-row">
              <Tag type={statusType(detail.status)} size="sm">{STATUS[detail.status]}</Tag>
              <Tag type="gray" size="sm">{PRIORITY[detail.priority]}</Tag>
              {detail.overdue && <Tag type="red" size="sm">Atrasado</Tag>}
              {detail.code && <Tag type="blue" size="sm">{detail.code}</Tag>}
            </div>
            <p>{detail.description}</p>
            <div className="detail-grid">
              <div><span>Cliente</span><strong>{detail.client_name ?? 'Não informado'}</strong></div>
              <div><span>Projeto</span><strong>{detail.project_name ?? 'Não informado'}</strong></div>
              <div><span>Responsável</span><strong>{detail.assignee_name ?? 'Não atribuído'}</strong></div>
              <div><span>Prazo</span><strong>{detail.due_date ?? '—'}</strong></div>
            </div>
            <h4>Histórico</h4>
            <div className="inline-form">
              <TextInput id="fu-update" labelText="Nova atualização" value={updateBody} onChange={(event) => setUpdateBody(event.target.value)} />
              <Button kind="tertiary" onClick={() => void addUpdate()}>Registrar</Button>
            </div>
            <div className="timeline">
              {detail.updates.map((update) => (
                <div key={update.id} className="timeline-item">
                  <div><strong>{update.author_name ?? 'Usuário removido'}</strong> <span>{relativeTime(update.created_at)}</span></div>
                  {update.status_from && update.status_to && <p className="muted small">{STATUS[update.status_from]} → {STATUS[update.status_to]}</p>}
                  <p>{update.body}</p>
                  {(user?.is_admin || update.user_id === user?.id) && <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover" onClick={async () => { await api.delete(`/follow-ups/${detail.id}/updates/${update.id}`); await openDetail(detail); }} />}
                </div>
              ))}
            </div>
          </div>
        )}
      </Modal>

      {settingsForm && (
        <Modal open={settingsOpen} modalHeading="Configurações do Follow-up" primaryButtonText={saving ? 'Salvando…' : 'Salvar'} secondaryButtonText="Cancelar" onRequestSubmit={() => void saveSettings()} onRequestClose={() => setSettingsOpen(false)} size="sm">
          <div className="modal-form">
            <TextInput id="set-per-page" type="number" labelText="Itens por página" value={String(settingsForm.per_page)} onChange={(event) => setSettingsForm({ ...settingsForm, per_page: Number(event.target.value) })} />
            <TextInput id="set-due" type="number" labelText="Dias para 'vence em breve'" value={String(settingsForm.due_soon_days)} onChange={(event) => setSettingsForm({ ...settingsForm, due_soon_days: Number(event.target.value) })} />
            <TextInput id="set-widget" type="number" labelText="Itens no widget" value={String(settingsForm.widget_limit)} onChange={(event) => setSettingsForm({ ...settingsForm, widget_limit: Number(event.target.value) })} />
            <TextInput id="set-prefix" labelText="Prefixo do assunto" value={settingsForm.email_subject_prefix} onChange={(event) => setSettingsForm({ ...settingsForm, email_subject_prefix: event.target.value })} />
            <TextInput id="set-recipients" labelText="Destinatários do resumo" value={settingsForm.summary_recipients ?? ''} onChange={(event) => setSettingsForm({ ...settingsForm, summary_recipients: event.target.value })} />
            <h4>Responsáveis</h4>
            {assignees.map((assignee) => (
              <div key={assignee.id} className="sharing-row">
                <span>{assignee.name}{assignee.email ? ` · ${assignee.email}` : ''}</span>
                <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover" onClick={async () => { await api.delete(`/follow-ups/settings/assignees/${assignee.id}`); await load(); }} />
              </div>
            ))}
            <div className="inline-form">
              <TextInput id="new-assignee-name" labelText="Nome" value={newAssignee.name} onChange={(event) => setNewAssignee({ ...newAssignee, name: event.target.value })} />
              <TextInput id="new-assignee-email" labelText="E-mail" value={newAssignee.email} onChange={(event) => setNewAssignee({ ...newAssignee, email: event.target.value })} />
              <Button kind="tertiary" onClick={async () => { await api.post('/follow-ups/settings/assignees', newAssignee); setNewAssignee({ name: '', email: '' }); await load(); }}>Adicionar</Button>
            </div>
            <h4>Clientes e projetos</h4>
            {clients.map((client) => (
              <div key={client.id} className="sharing-row">
                <span>{client.name} · {client.project_count} projeto(s)</span>
                <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover" onClick={async () => { await api.delete(`/follow-ups/settings/clients/${client.id}`); await load(); }} />
              </div>
            ))}
            <div className="inline-form">
              <TextInput id="new-client" labelText="Novo cliente" value={newClient} onChange={(event) => setNewClient(event.target.value)} />
              <Button kind="tertiary" onClick={async () => { await api.post('/follow-ups/settings/clients', { client_name: newClient }); setNewClient(''); await load(); }}>Adicionar</Button>
            </div>
            <div className="inline-form">
              <Select id="new-project-client" labelText="Cliente" value={newProject.client_id} onChange={(event) => setNewProject({ ...newProject, client_id: event.target.value })}>
                <SelectItem value="" text="Selecione" />
                {clients.map((client) => <SelectItem key={client.id} value={String(client.id)} text={client.name} />)}
              </Select>
              <TextInput id="new-project" labelText="Novo projeto" value={newProject.name} onChange={(event) => setNewProject({ ...newProject, name: event.target.value })} />
              <Button kind="tertiary" onClick={async () => { await api.post('/follow-ups/settings/projects', { client_id: Number(newProject.client_id), project_name: newProject.name }); setNewProject({ client_id: '', name: '' }); await load(); }}>Adicionar</Button>
            </div>
            <div className="section-actions"><Button kind="tertiary" renderIcon={Send} onClick={() => toast.info('Envio de e-mail usa o SMTP configurado no servidor.')}>Testar e-mail</Button></div>
          </div>
        </Modal>
      )}
    </div>
  );
}
