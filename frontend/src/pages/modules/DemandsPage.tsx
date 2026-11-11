import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Add, Download, Edit, TrashCan, UserMultiple } from '@carbon/icons-react';
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
import { ConfirmModal } from '../../components/ConfirmModal';
import { useAuth } from '../../contexts/AuthContext';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import { StationSearch } from '../../components/StationSearch';

interface Collaborator { id: number; name: string; email: string }
interface Demand {
  id: number; requester_id: number; assignee_id: number; title: string;
  description: string | null; status: string; priority: string; due_at: string | null;
  completed_at: string | null; created_at: string; updated_at: string;
  client_name: string | null; client_email: string | null;
  station_id: number | null; station_site_id: string | null; station_municipio: string | null; station_estado: string | null;
  operadora: string | null; demand_type: string | null; oc_tim: string | null;
  client_os: string | null; key_number: string | null; activity_scope: string | null;
  requester_name: string | null; assignee_name: string | null; assignee_email: string | null;
  overdue: boolean; update_count: number;
}
interface Stats {
  open: number; in_progress: number; waiting: number; completed: number;
  overdue: number; mine: number; created: number;
}
interface ListData {
  data: Demand[]; stats: Stats; collaborators: Collaborator[];
  meta: { current_page: number; last_page: number; per_page: number; total: number };
  filters: { search: string; status: string; priority: string; assignee_id: number; scope: string };
}

export const DEMAND_STATUS: Record<string, string> = {
  aberta: 'Aberta', em_andamento: 'Em andamento', aguardando: 'Aguardando',
  concluida: 'Concluída', cancelada: 'Cancelada',
};
export const DEMAND_PRIORITY: Record<string, string> = {
  baixa: 'Baixa', media: 'Média', alta: 'Alta', urgente: 'Urgente',
};
export const demandStatusType = (value: string) => value === 'concluida' ? 'green' : value === 'cancelada'
  ? 'red' : value === 'em_andamento' ? 'blue' : value === 'aguardando' ? 'magenta' : 'gray';
export const demandPriorityType = (value: string) => value === 'urgente' ? 'red' : value === 'alta'
  ? 'magenta' : value === 'media' ? 'blue' : 'gray';

const EMPTY_FORM = {
  title: '', description: '', assignee_id: '', priority: 'media', due_at: '', status: 'aberta',
  client_name: '', client_email: '', station_id: '', operadora: '', demand_type: '',
  oc_tim: '', client_os: '', key_number: '', activity_scope: '',
};

export function DemandsPage() {
  const { user } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [data, setData] = useState<ListData | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [priority, setPriority] = useState('');
  const [assignee, setAssignee] = useState('');
  const [scope, setScope] = useState(searchParams.get('scope') ?? '');
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Demand | null>(null);
  const [form, setForm] = useState({ ...EMPTY_FORM });
  const [saving, setSaving] = useState(false);
  const [deleteDemand, setDeleteDemand] = useState<Demand | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({ page: String(page) });
    if (search) params.set('search', search);
    if (status) params.set('status', status);
    if (priority) params.set('priority', priority);
    if (assignee) params.set('assignee_id', assignee);
    if (scope) params.set('scope', scope);
    try {
      const response = await api.get<ListData>(`/demands?${params.toString()}`);
      setData(response);
    } catch (error) {
      toast.error('Não foi possível carregar as demandas', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [page, search, status, priority, assignee, scope, toast]);

  useEffect(() => { void load(); }, [load]);

  useEffect(() => {
    const id = Number(searchParams.get('demand'));
    if (Number.isFinite(id) && id > 0) {
      navigate(`/demands/${id}`, { replace: true });
    }
  }, [searchParams, navigate]);

  function openCreate() {
    setEditing(null);
    setForm({ ...EMPTY_FORM, assignee_id: data?.collaborators[0] ? String(data.collaborators[0].id) : '' });
    setModalOpen(true);
  }

  function openEdit(demand: Demand) {
    setEditing(demand);
    setForm({
      title: demand.title, description: demand.description ?? '', assignee_id: String(demand.assignee_id),
      priority: demand.priority, due_at: demand.due_at ?? '', status: demand.status,
      client_name: demand.client_name ?? '', client_email: demand.client_email ?? '',
      station_id: demand.station_id ? String(demand.station_id) : '',
      operadora: demand.operadora ?? '', demand_type: demand.demand_type ?? '',
      oc_tim: demand.oc_tim ?? '', client_os: demand.client_os ?? '',
      key_number: demand.key_number ?? '', activity_scope: demand.activity_scope ?? '',
    });
    setModalOpen(true);
  }

  async function save() {
    setSaving(true);
    try {
      const payload = {
        ...form,
        assignee_id: form.assignee_id ? Number(form.assignee_id) : 0,
        station_id: form.station_id ? Number(form.station_id) : null,
        due_at: form.due_at || '',
      };
      if (editing) {
        await api.put(`/demands/${editing.id}`, payload);
        toast.success('Demanda atualizada com sucesso.');
        setModalOpen(false);
        await load();
      } else {
        const response = await api.post<{ data: { demand: Demand } }>('/demands', payload);
        toast.success('Demanda criada e encaminhada.');
        setModalOpen(false);
        navigate(`/demands/${response.data.demand.id}`);
        return;
      }
    } catch (error) {
      toast.error('Não foi possível salvar a demanda', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function removeDemand() {
    if (!deleteDemand) return;
    setDeleting(true);
    try {
      await api.delete(`/demands/${deleteDemand.id}`);
      toast.success('Demanda excluída.');
      setDeleteDemand(null);
      await load();
    } catch (error) {
      toast.error('Não foi possível excluir a demanda', errorMessage(error));
    } finally {
      setDeleting(false);
    }
  }

  function exportDemands() {
    const params = new URLSearchParams();
    if (search) params.set('search', search);
    if (status) params.set('status', status);
    if (priority) params.set('priority', priority);
    if (assignee) params.set('assignee_id', assignee);
    if (scope) params.set('scope', scope);
    window.open(`/api/v1/demands/export?${params.toString()}`, '_blank');
  }

  const items = data?.data ?? [];
  const collaborators = data?.collaborators ?? [];
  const stats = data?.stats;
  const isAdmin = Boolean(user?.is_admin);
  const canManage = (demand: Demand) => isAdmin || demand.requester_id === user?.id;

  return (
    <div className="page">
      <PageHeader
        eyebrow="Colaboração"
        title="Demandas"
        description="Acione colaboradores, defina prazos e acompanhe cada etapa até a conclusão."
        actions={
          <>
            <Button kind="ghost" renderIcon={Download} onClick={exportDemands}>Exportar</Button>
            <Button renderIcon={Add} onClick={openCreate}>Nova demanda</Button>
          </>
        }
      />

      {stats && (
        <div className="stats-grid">
          <Tile className="stat-card stat-card--blue"><span className="stat-card__label">Em aberto</span><strong className="stat-card__value">{stats.open}</strong></Tile>
          <Tile className="stat-card stat-card--purple"><span className="stat-card__label">Em andamento</span><strong className="stat-card__value">{stats.in_progress}</strong></Tile>
          <Tile className="stat-card stat-card--amber"><span className="stat-card__label">Aguardando</span><strong className="stat-card__value">{stats.waiting}</strong></Tile>
          <Tile className="stat-card stat-card--red"><span className="stat-card__label">Atrasadas</span><strong className="stat-card__value">{stats.overdue}</strong></Tile>
        </div>
      )}

      <Tile className="module-main__header">
        <div className="module-toolbar demand-toolbar">
          <Search
            size="lg"
            labelText="Buscar demandas"
            placeholder="Buscar por título, descrição ou colaborador..."
            value={search}
            onChange={(event) => { setSearch(event.target.value); setPage(1); }}
            onClear={() => setSearch('')}
          />
          <Select id="demand-status" labelText="Status" size="sm" value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todos" />
            {Object.entries(DEMAND_STATUS).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
          </Select>
          <Select id="demand-priority" labelText="Prioridade" size="sm" value={priority} onChange={(event) => { setPriority(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todas" />
            {Object.entries(DEMAND_PRIORITY).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
          </Select>
          <Select id="demand-assignee" labelText="Responsável" size="sm" value={assignee} onChange={(event) => { setAssignee(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todos" />
            {collaborators.map((collaborator) => <SelectItem key={collaborator.id} value={String(collaborator.id)} text={collaborator.name} />)}
          </Select>
          <Select id="demand-scope" labelText="Visão" size="sm" value={scope} onChange={(event) => { setScope(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todas" />
            <SelectItem value="mine" text="Atribuídas a mim" />
            <SelectItem value="created" text="Criadas por mim" />
          </Select>
        </div>
      </Tile>

      {loading ? <InlineLoading description="Carregando demandas…" /> : items.length === 0 ? (
        <EmptyState
          icon={UserMultiple}
          title="Nenhuma demanda encontrada"
          description="Crie uma demanda e acione um colaborador responsável."
          action={<Button renderIcon={Add} onClick={openCreate}>Nova demanda</Button>}
        />
      ) : (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Demanda</th>
                  <th className="hide-sm">Responsável</th>
                  <th>Prioridade</th>
                  <th>Status</th>
                  <th>Prazo</th>
                  <th className="hide-sm">Atualizada</th>
                  <th>Ações</th>
                </tr>
              </thead>
              <tbody>
                {items.map((demand) => (
                  <tr key={demand.id}>
                    <td>
                      <button type="button" className="link-button" onClick={() => navigate(`/demands/${demand.id}`)}>{demand.title}</button>
                      <span className="cell-secondary">Solicitante: {demand.requester_name ?? '—'}</span>
                    </td>
                    <td className="hide-sm">{demand.assignee_name ?? '—'}</td>
                    <td><Tag type={demandPriorityType(demand.priority)} size="sm">{DEMAND_PRIORITY[demand.priority] ?? demand.priority}</Tag></td>
                    <td><Tag type={demandStatusType(demand.status)} size="sm">{DEMAND_STATUS[demand.status] ?? demand.status}</Tag></td>
                    <td className={demand.overdue ? 'text-danger' : ''}>
                      {demand.due_at ?? '—'}
                      {demand.overdue && <span className="cell-secondary">Atrasada</span>}
                    </td>
                    <td className="hide-sm">{relativeTime(demand.updated_at)}</td>
                    <td className="cell-actions">
                      <Button kind="ghost" size="sm" hasIconOnly renderIcon={Edit} iconDescription="Editar" onClick={() => openEdit(demand)} disabled={!canManage(demand)} />
                      <Button kind="danger--ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={() => setDeleteDemand(demand)} disabled={!canManage(demand)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="table-footer"><span className="table-footer__order">{data ? `${data.meta.total} demanda(s)` : ''}</span></div>
        </div>
      )}

      {data && data.meta.last_page > 1 && (
        <div className="pager">
          <Button kind="ghost" size="sm" disabled={page <= 1} onClick={() => setPage((current) => Math.max(1, current - 1))}>Anterior</Button>
          <span>Página {data.meta.current_page} de {data.meta.last_page}</span>
          <Button kind="ghost" size="sm" disabled={page >= data.meta.last_page} onClick={() => setPage((current) => current + 1)}>Próxima</Button>
        </div>
      )}

      <Modal
        open={modalOpen}
        modalHeading={editing ? 'Editar demanda' : 'Nova demanda'}
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving || !form.title.trim() || !form.assignee_id}
        onRequestSubmit={() => void save()}
        onRequestClose={() => setModalOpen(false)}
        size="lg"
        hasScrollingContent
        className="demand-modal"
      >
        <div className="modal-form demand-form">
          <section className="demand-form-section">
            <p className="demand-form-section__title">Identificação</p>
            <TextInput id="demand-title" labelText="Título *" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} required />
            <StationSearch
              id="demand-station"
              value={form.station_id ? Number(form.station_id) : null}
              valueLabel={editing?.station_site_id}
              onChange={(stationId) => setForm({ ...form, station_id: stationId ? String(stationId) : '' })}
            />
            <div className="form-row">
              <TextInput id="demand-operadora" labelText="Operadora" value={form.operadora} onChange={(event) => setForm({ ...form, operadora: event.target.value })} />
              <TextInput id="demand-type" labelText="Tipo de demanda" value={form.demand_type} onChange={(event) => setForm({ ...form, demand_type: event.target.value })} />
            </div>
          </section>

          <section className="demand-form-section">
            <p className="demand-form-section__title">Referências do acionamento</p>
            <div className="form-row">
              <TextInput id="demand-oc-tim" labelText="Número OC TIM" value={form.oc_tim} onChange={(event) => setForm({ ...form, oc_tim: event.target.value })} />
              <TextInput id="demand-client-os" labelText="Número OS do Cliente" value={form.client_os} onChange={(event) => setForm({ ...form, client_os: event.target.value })} />
            </div>
            <TextInput id="demand-key" labelText="Número da chave" value={form.key_number} onChange={(event) => setForm({ ...form, key_number: event.target.value })} />
          </section>

          <section className="demand-form-section">
            <p className="demand-form-section__title">Cliente</p>
            <div className="form-row">
              <TextInput id="demand-client-name" labelText="Cliente" value={form.client_name} onChange={(event) => setForm({ ...form, client_name: event.target.value })} />
              <TextInput id="demand-client-email" type="email" labelText="E-mail do cliente" value={form.client_email} onChange={(event) => setForm({ ...form, client_email: event.target.value })} />
            </div>
          </section>

          <section className="demand-form-section">
            <p className="demand-form-section__title">Acionamento</p>
            <div className="form-row">
              <Select id="demand-form-assignee" labelText="Responsável *" value={form.assignee_id} onChange={(event) => setForm({ ...form, assignee_id: event.target.value })}>
                <SelectItem value="" text="Selecione" />
                {collaborators.map((collaborator) => <SelectItem key={collaborator.id} value={String(collaborator.id)} text={`${collaborator.name} — ${collaborator.email}`} />)}
              </Select>
              <Select id="demand-form-priority" labelText="Prioridade" value={form.priority} onChange={(event) => setForm({ ...form, priority: event.target.value })}>
                {Object.entries(DEMAND_PRIORITY).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
              </Select>
            </div>
            <div className="form-row">
              <TextInput id="demand-due" type="date" labelText="Prazo" value={form.due_at} onChange={(event) => setForm({ ...form, due_at: event.target.value })} />
              {editing && (
                <Select id="demand-form-status" labelText="Status" value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })}>
                  {Object.entries(DEMAND_STATUS).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
                </Select>
              )}
            </div>
          </section>

          <section className="demand-form-section">
            <p className="demand-form-section__title">Detalhamento</p>
            <TextArea id="demand-description" labelText="Descrição" rows={3} value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} />
            <TextArea id="demand-activity-scope" labelText="Escopo da atividade" rows={3} value={form.activity_scope} onChange={(event) => setForm({ ...form, activity_scope: event.target.value })} />
          </section>
        </div>
      </Modal>

      <ConfirmModal
        open={deleteDemand !== null}
        title="Excluir demanda"
        message={deleteDemand ? `Excluir a demanda "${deleteDemand.title}"? O histórico e os anexos também serão removidos.` : ''}
        confirmLabel="Excluir demanda"
        loading={deleting}
        onCancel={() => setDeleteDemand(null)}
        onConfirm={() => void removeDemand()}
      />
    </div>
  );
}
