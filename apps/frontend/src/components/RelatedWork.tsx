import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Add, ArrowRight, Calendar, Checkmark, TrashCan } from '@carbon/icons-react';
import {
  Button,
  InlineLoading,
  Modal,
  Select,
  SelectItem,
  Tag,
  TextArea,
  TextInput,
  Tile,
} from '@carbon/react';
import { useToast } from '../contexts/ToastContext';
import { api } from '../lib/api';
import { errorMessage } from '../lib/format';

interface WorkResource {
  type: string; id: number; label: string; subtitle?: string; url: string;
  station_id?: string | null; nokia_site_id?: number | null;
}
interface WorkTask {
  id: number; link_id: number; title: string; list_name: string;
  due_at: string | null; completed_at: string | null; is_important: boolean;
}
interface WorkFollowUp {
  id: number; link_id: number; title: string; status: string; priority: string;
  due_date: string | null; assignee_name: string | null; overdue: boolean;
}
interface WorkEvent {
  id: number; link_id: number; title: string; start_at: string; end_at: string;
  location: string | null; calendar_name: string; status: string;
}
interface WorkCounts {
  tasks_open: number; tasks_overdue: number; follow_ups_open: number;
  follow_ups_overdue: number; events_today: number; events_upcoming: number;
}
interface WorkContext {
  resource: WorkResource; linked_resource: WorkResource | null;
  tasks: WorkTask[]; follow_ups: WorkFollowUp[]; events: WorkEvent[]; counts: WorkCounts;
}
interface Assignee { id: number; name: string; email: string | null }

const FOLLOWUP_STATUS: Record<string, string> = {
  pendente: 'Pendente', em_andamento: 'Em andamento', concluido: 'Concluído', cancelado: 'Cancelado',
};
const FOLLOWUP_PRIORITY: Record<string, string> = {
  baixa: 'Baixa', media: 'Média', alta: 'Alta', urgente: 'Urgente',
};
const statusType = (value: string) => value === 'concluido' ? 'green' : value === 'cancelado'
  ? 'red' : value === 'em_andamento' ? 'blue' : 'magenta';

function dateBR(value: string | null | undefined): string {
  if (!value) return '—';
  const date = value.length <= 10 ? new Date(`${value}T12:00:00`) : new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString('pt-BR');
}

function timeBR(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function localInput(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

type Kind = 'task' | 'follow_up' | 'agenda_event';

const EMPTY_FORM = {
  title: '', notes: '', due_date: '', priority: 'media', assignee_id: '',
  start_at: '', duration_minutes: '60',
};

export function RelatedWork({ resourceType, resourceId }: { resourceType: 'station' | 'nokia_site'; resourceId: number }) {
  const navigate = useNavigate();
  const toast = useToast();
  const [context, setContext] = useState<WorkContext | null>(null);
  const [loading, setLoading] = useState(true);
  const [assignees, setAssignees] = useState<Assignee[]>([]);
  const [kind, setKind] = useState<Kind | null>(null);
  const [form, setForm] = useState({ ...EMPTY_FORM });
  const [saving, setSaving] = useState(false);
  const [linkOpen, setLinkOpen] = useState(false);
  const [linkKind, setLinkKind] = useState<'follow_up' | 'task'>('follow_up');
  const [linkEntityID, setLinkEntityID] = useState('');
  const [linkBusy, setLinkBusy] = useState(false);
  const [linkOptions, setLinkOptions] = useState<{
    follow_up: Array<{ id: number; label: string; secondary: string }>;
    task: Array<{ id: number; label: string; secondary: string }>;
  }>({ follow_up: [], task: [] });

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.get<{ data: WorkContext }>(`/work-context?resource_type=${resourceType}&resource_id=${resourceId}`);
      setContext(response.data);
    } catch (error) {
      toast.error('Não foi possível carregar o trabalho relacionado', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [resourceType, resourceId, toast]);

  useEffect(() => { void load(); }, [load]);

  async function openCreate(next: Kind) {
    setForm({ ...EMPTY_FORM, start_at: next === 'agenda_event' ? localInput(new Date(Date.now() + 60 * 60 * 1000)) : '' });
    setKind(next);
    if (next === 'follow_up' && assignees.length === 0) {
      try {
        const response = await api.get<{ assignees: Assignee[] }>('/follow-ups?page=1&per_page=5');
        setAssignees(response.assignees ?? []);
      } catch {
        setAssignees([]);
      }
    }
  }

  async function submit() {
    if (!kind) return;
    setSaving(true);
    try {
      const response = await api.post<{ message: string }>('/work-items', {
        resource_type: resourceType,
        resource_id: resourceId,
        kind,
        title: form.title,
        notes: form.notes,
        due_date: form.due_date,
        priority: form.priority,
        assignee_id: form.assignee_id ? Number(form.assignee_id) : null,
        start_at: form.start_at,
        duration_minutes: Number(form.duration_minutes) || 60,
      });
      toast.success(response.message ?? 'Item criado e vinculado.');
      setKind(null);
      await load();
    } catch (error) {
      toast.error('Não foi possível criar o item', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function unlink(linkId: number) {
    try {
      await api.delete(`/work-links/${linkId}`);
      await load();
    } catch (error) {
      toast.error('Não foi possível remover o vínculo', errorMessage(error));
    }
  }

  async function openLinkExisting() {
    setLinkOpen(true);
    setLinkEntityID('');
    setLinkBusy(true);
    try {
      const [followUps, tasks] = await Promise.all([
        api.get<{ data: Array<{ id: number; title: string; assignee_name: string | null }> }>('/follow-ups?page=1&per_page=100'),
        api.get<{ data: { tasks: Array<{ id: number; title: string; list_name: string }>; completed: Array<{ id: number; title: string; list_name: string }> } }>('/tasks?view=all'),
      ]);
      setLinkOptions({
        follow_up: (followUps.data ?? []).map((item) => ({
          id: item.id, label: item.title, secondary: item.assignee_name ?? 'Não atribuído',
        })),
        task: [...(tasks.data?.tasks ?? []), ...(tasks.data?.completed ?? [])].map((item) => ({
          id: item.id, label: item.title, secondary: item.list_name,
        })),
      });
    } catch (error) {
      toast.error('Não foi possível carregar os itens', errorMessage(error));
    } finally {
      setLinkBusy(false);
    }
  }

  async function submitLink() {
    if (!linkEntityID) return;
    setLinkBusy(true);
    try {
      await api.post('/work-links', {
        module: linkKind,
        entity_id: Number(linkEntityID),
        resource_type: resourceType,
        resource_id: resourceId,
      });
      toast.success('Item vinculado ao recurso.');
      setLinkOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível vincular o item', errorMessage(error));
    } finally {
      setLinkBusy(false);
    }
  }

  const counts = context?.counts;
  return (
    <Tile className="nokia-detail-section related-work">
      <div className="nokia-section-header">
        <div>
          <h2>Trabalho relacionado</h2>
          <p>Tarefas, follow-ups e compromissos vinculados a este recurso.</p>
        </div>
        <div className="related-work__actions">
          <Button kind="ghost" size="sm" renderIcon={Checkmark} onClick={() => void openCreate('task')}>Nova tarefa</Button>
          <Button kind="ghost" size="sm" renderIcon={Add} onClick={() => void openCreate('follow_up')}>Novo follow-up</Button>
          <Button kind="ghost" size="sm" renderIcon={Calendar} onClick={() => void openCreate('agenda_event')}>Agendar visita</Button>
          <Button kind="ghost" size="sm" renderIcon={ArrowRight} onClick={() => void openLinkExisting()}>Vincular existente</Button>
        </div>
      </div>

      {context?.linked_resource && (
        <button type="button" className="related-work__crosslink" onClick={() => navigate(context.linked_resource!.url)}>
          <span>
            {context.linked_resource.type === 'nokia_site' ? 'Projeto Nokia vinculado' : 'Estação vinculada'}
          </span>
          <strong>{context.linked_resource.label}</strong>
          {context.linked_resource.subtitle && <em>{context.linked_resource.subtitle}</em>}
          <ArrowRight size={16} />
        </button>
      )}

      {loading ? <InlineLoading description="Carregando trabalho relacionado…" /> : (
        <>
          <div className="related-work__summary">
            <span><strong>{counts?.follow_ups_open ?? 0}</strong> follow-up(s) em aberto</span>
            <span><strong>{counts?.tasks_open ?? 0}</strong> tarefa(s) pendente(s)</span>
            <span><strong>{counts?.events_upcoming ?? 0}</strong> compromisso(s) futuro(s)</span>
            {(counts?.follow_ups_overdue ?? 0) + (counts?.tasks_overdue ?? 0) > 0 && (
              <Tag type="red" size="sm">{(counts?.follow_ups_overdue ?? 0) + (counts?.tasks_overdue ?? 0)} atrasado(s)</Tag>
            )}
          </div>

          <div className="related-work__groups">
            <div className="related-work__group">
              <p className="field-label">Follow-ups</p>
              {context && context.follow_ups.length > 0 ? context.follow_ups.map((item) => (
                <div key={item.id} className="related-work__row">
                  <button type="button" className="related-work__link" onClick={() => navigate(`/follow-ups?followup=${item.id}`)}>
                    <strong>{item.title}</strong>
                    <span>{item.assignee_name ?? 'Não atribuído'} · prazo {dateBR(item.due_date)}</span>
                  </button>
                  <Tag type={statusType(item.status)} size="sm">{FOLLOWUP_STATUS[item.status] ?? item.status}</Tag>
                  <Tag type={item.priority === 'urgente' ? 'red' : item.priority === 'alta' ? 'magenta' : 'gray'} size="sm">
                    {FOLLOWUP_PRIORITY[item.priority] ?? item.priority}
                  </Tag>
                  <Button kind="ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Desvincular" onClick={() => void unlink(item.link_id)} />
                </div>
              )) : <p className="muted">Nenhum follow-up vinculado.</p>}
            </div>

            <div className="related-work__group">
              <p className="field-label">Minhas tarefas</p>
              {context && context.tasks.length > 0 ? context.tasks.map((item) => (
                <div key={item.id} className="related-work__row">
                  <button type="button" className="related-work__link" onClick={() => navigate(`/tasks?task=${item.id}`)}>
                    <strong>{item.title}</strong>
                    <span>{item.list_name} · {item.due_at ? `vence ${dateBR(item.due_at)}` : 'sem prazo'}</span>
                  </button>
                  {item.completed_at
                    ? <Tag type="green" size="sm">Concluída</Tag>
                    : <Tag type={item.is_important ? 'red' : 'gray'} size="sm">{item.is_important ? 'Importante' : 'Aberta'}</Tag>}
                  <Button kind="ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Desvincular" onClick={() => void unlink(item.link_id)} />
                </div>
              )) : <p className="muted">Nenhuma tarefa vinculada.</p>}
            </div>

            <div className="related-work__group">
              <p className="field-label">Agenda</p>
              {context && context.events.length > 0 ? context.events.map((item) => (
                <div key={item.id} className="related-work__row">
                  <button type="button" className="related-work__link" onClick={() => navigate(`/agenda?event=${item.id}`)}>
                    <strong>{item.title}</strong>
                    <span>{timeBR(item.start_at)} · {item.calendar_name}</span>
                  </button>
                  <Tag type="blue" size="sm">{item.location ?? 'Sem local'}</Tag>
                  <Button kind="ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Desvincular" onClick={() => void unlink(item.link_id)} />
                </div>
              )) : <p className="muted">Nenhum compromisso vinculado.</p>}
            </div>
          </div>
        </>
      )}

      <Modal
        open={kind !== null}
        modalHeading={kind === 'task' ? 'Nova tarefa vinculada' : kind === 'follow_up' ? 'Novo follow-up vinculado' : 'Agendar visita'}
        primaryButtonText={saving ? 'Salvando…' : 'Criar e vincular'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving || !form.title.trim()}
        onRequestSubmit={() => void submit()}
        onRequestClose={() => setKind(null)}
        size="md"
      >
        <div className="modal-form">
          <TextInput
            id="related-work-title"
            labelText={kind === 'agenda_event' ? 'Assunto' : 'Título'}
            value={form.title}
            onChange={(event) => setForm({ ...form, title: event.target.value })}
            required
          />
          {kind === 'follow_up' && (
            <div className="form-row">
              <Select id="related-work-priority" labelText="Prioridade" value={form.priority} onChange={(event) => setForm({ ...form, priority: event.target.value })}>
                {Object.entries(FOLLOWUP_PRIORITY).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
              </Select>
              <Select id="related-work-assignee" labelText="Responsável" value={form.assignee_id} onChange={(event) => setForm({ ...form, assignee_id: event.target.value })}>
                <SelectItem value="" text="Não atribuído" />
                {assignees.map((assignee) => <SelectItem key={assignee.id} value={String(assignee.id)} text={assignee.name} />)}
              </Select>
            </div>
          )}
          {kind === 'agenda_event' ? (
            <div className="form-row">
              <TextInput id="related-work-start" type="datetime-local" labelText="Início" value={form.start_at} onChange={(event) => setForm({ ...form, start_at: event.target.value })} />
              <TextInput id="related-work-duration" type="number" labelText="Duração (min)" value={form.duration_minutes} onChange={(event) => setForm({ ...form, duration_minutes: event.target.value })} />
            </div>
          ) : (
            <TextInput id="related-work-due" type="date" labelText="Prazo" value={form.due_date} onChange={(event) => setForm({ ...form, due_date: event.target.value })} />
          )}
          <TextArea id="related-work-notes" labelText="Observações" rows={3} value={form.notes} onChange={(event) => setForm({ ...form, notes: event.target.value })} />
        </div>
      </Modal>

      <Modal
        open={linkOpen}
        modalHeading="Vincular item existente"
        primaryButtonText={linkBusy ? 'Vinculando…' : 'Vincular'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={linkBusy || !linkEntityID}
        onRequestSubmit={() => void submitLink()}
        onRequestClose={() => setLinkOpen(false)}
        size="md"
      >
        <div className="modal-form">
          <Select
            id="related-work-link-kind"
            labelText="Tipo de item"
            value={linkKind}
            onChange={(event) => { setLinkKind(event.target.value as 'follow_up' | 'task'); setLinkEntityID(''); }}
          >
            <SelectItem value="follow_up" text="Follow-up" />
            <SelectItem value="task" text="Tarefa" />
          </Select>
          {linkBusy ? <InlineLoading description="Carregando itens…" /> : (
            <Select id="related-work-link-item" labelText="Item" value={linkEntityID} onChange={(event) => setLinkEntityID(event.target.value)}>
              <SelectItem value="" text="Selecione" />
              {linkOptions[linkKind].map((item) => (
                <SelectItem key={item.id} value={String(item.id)} text={`${item.label} — ${item.secondary}`} />
              ))}
            </Select>
          )}
          {!linkBusy && linkOptions[linkKind].length === 0 && (
            <p className="muted">Nenhum item disponível para vincular.</p>
          )}
        </div>
      </Modal>
    </Tile>
  );
}
