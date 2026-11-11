import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Add,
  Checkmark,
  ChevronRight,
  Edit,
  List,
  OverflowMenuVertical,
  Star,
  Time,
  TrashCan,
  Warning,
} from '@carbon/icons-react';
import {
  Button,
  Checkbox,
  InlineLoading,
  Modal,
  OverflowMenu,
  OverflowMenuItem,
  Search,
  Select,
  SelectItem,
  TextArea,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState } from '../../components/Feedback';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';

interface TodoList { id: number; name: string; color: string; icon: string; position: number; task_count: number }
interface TodoStep { id: number; title: string; completed_at: string | null; position: number }
interface TodoTask {
  id: number; todo_list_id: number; title: string; notes: string | null; due_at: string | null;
  remind_at: string | null; repeat_type: string; is_important: boolean; my_day_on: string | null;
  completed_at: string | null; position: number; created_at: string; steps: TodoStep[];
  list_name: string; list_color: string; list_icon: string;
}
interface TasksData {
  lists: TodoList[]; tasks: TodoTask[]; completed: TodoTask[];
  counts: Record<string, number>; view: string; list_id: number; today: string;
}

const VIEWS = [
  { key: 'my_day', label: 'Meu Dia', icon: Star, count: 'my_day' },
  { key: 'important', label: 'Importante', icon: Star, count: 'important' },
  { key: 'planned', label: 'Planejado', icon: Time, count: 'planned' },
  { key: 'today', label: 'Hoje', icon: Time, count: 'today' },
  { key: 'week', label: 'Próximos 7 dias', icon: ChevronRight, count: 'week' },
  { key: 'all', label: 'Todas as tarefas', icon: List, count: 'all' },
  { key: 'completed', label: 'Concluídas', icon: Checkmark, count: 'completed' },
];

const REPEATS = [
  { value: 'none', label: 'Não se repete' }, { value: 'daily', label: 'Todos os dias' },
  { value: 'weekdays', label: 'Dias úteis' }, { value: 'weekly', label: 'Todas as semanas' },
  { value: 'monthly', label: 'Todos os meses' }, { value: 'yearly', label: 'Todos os anos' },
];

const COLORS = ['indigo', 'blue', 'sky', 'teal', 'green', 'lime', 'amber', 'orange', 'rose', 'pink', 'violet', 'slate'];

function TaskRow({ task, onRefresh, onOpen }: { task: TodoTask; onRefresh: () => void; onOpen: (task: TodoTask) => void }) {
  const toast = useToast();
  const [busy, setBusy] = useState(false);
  async function action(path: string, method: 'patch' | 'delete' = 'patch') {
    setBusy(true);
    try {
      if (method === 'delete') await api.delete(path);
      else await api.patch(path);
      onRefresh();
    } catch (error) {
      toast.error('Não foi possível atualizar a tarefa', errorMessage(error));
    } finally {
      setBusy(false);
    }
  }
  const completedSteps = task.steps.filter((step) => step.completed_at).length;
  async function moveTask(direction: 'up' | 'down') {
    setBusy(true);
    try {
      await api.patch(`/tasks/${task.id}/move`, { direction });
      onRefresh();
    } catch (error) {
      toast.error('Não foi possível mover a tarefa', errorMessage(error));
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className={`task-row ${task.completed_at ? 'task-row--done' : ''}`}>
      <button
        type="button"
        className="task-check"
        aria-label={task.completed_at ? 'Reabrir tarefa' : 'Concluir tarefa'}
        onClick={() => void action(`/tasks/${task.id}/toggle`)}
        disabled={busy}
      >
        {task.completed_at ? <Checkmark size={14} /> : null}
      </button>
      <button type="button" className="task-main" onClick={() => onOpen(task)}>
        <span className="task-title">{task.title}</span>
        {task.notes && <span className="task-notes">{task.notes}</span>}
        <span className="task-badges">
          {task.due_at && <span className={`task-badge ${task.due_at < new Date().toISOString().slice(0, 10) && !task.completed_at ? 'task-badge--danger' : ''}`}>{task.due_at}</span>}
          {task.repeat_type !== 'none' && <span className="task-badge">{REPEATS.find((item) => item.value === task.repeat_type)?.label}</span>}
          {task.steps.length > 0 && <span className="task-badge">{completedSteps}/{task.steps.length} etapas</span>}
          {!task.list_name.includes('undefined') && <span className="task-list-dot" style={{ background: colorFor(task.list_color) }} />}
          {task.list_name}
        </span>
      </button>
      <div className="task-actions">
        <Button
          kind="ghost"
          size="sm"
          hasIconOnly
          renderIcon={Star}
          iconDescription="Meu Dia"
          onClick={() => void action(`/tasks/${task.id}/my-day`)}
        />
        <Button
          kind={task.is_important ? 'primary' : 'ghost'}
          size="sm"
          hasIconOnly
          renderIcon={Star}
          iconDescription="Importante"
          onClick={() => void action(`/tasks/${task.id}/important`)}
        />
        <OverflowMenu iconDescription="Ações da tarefa" renderIcon={OverflowMenuVertical} size="sm" flipped>
          <OverflowMenuItem itemText="Detalhes" onClick={() => onOpen(task)} />
          <OverflowMenuItem itemText="Mover para cima" onClick={() => void moveTask('up')} />
          <OverflowMenuItem itemText="Mover para baixo" onClick={() => void moveTask('down')} />
          <OverflowMenuItem itemText="Excluir" isDelete hasDivider onClick={() => void action(`/tasks/${task.id}`, 'delete')} />
        </OverflowMenu>
      </div>
    </div>
  );
}

function colorFor(value: string): string {
  const map: Record<string, string> = {
    indigo: '#6366f1', blue: '#3b82f6', sky: '#0ea5e9', teal: '#14b8a6', green: '#22c55e',
    lime: '#84cc16', amber: '#f59e0b', orange: '#f97316', rose: '#f43f5e', pink: '#ec4899',
    violet: '#8b5cf6', slate: '#64748b',
  };
  return map[value] ?? '#6366f1';
}

export function TasksPage() {
  const toast = useToast();
  const [searchParams] = useSearchParams();
  const [data, setData] = useState<TasksData | null>(null);
  const [loading, setLoading] = useState(true);
  const [view, setView] = useState('my_day');
  const [listId, setListId] = useState(0);
  const [search, setSearch] = useState('');
  const [sort, setSort] = useState('manual');
  const [taskModal, setTaskModal] = useState<{ open: boolean; task: TodoTask | null }>({ open: false, task: null });
  const [listModal, setListModal] = useState<{ open: boolean; list: TodoList | null }>({ open: false, list: null });
  const [taskForm, setTaskForm] = useState({ title: '', todo_list_id: 0, due_at: '', remind_at: '', notes: '', repeat_type: 'none', is_important: false, my_day: false });
  const [listForm, setListForm] = useState({ name: '', color: 'indigo', icon: 'check-circle' });
  const [detail, setDetail] = useState<TodoTask | null>(null);
  const [stepTitle, setStepTitle] = useState('');
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({ view, sort });
    if (listId > 0) params.set('list', String(listId));
    if (search) params.set('search', search);
    try {
      const response = await api.get<{ data: TasksData }>(`/tasks?${params.toString()}`);
      setData(response.data);
      if (response.data.list_id && response.data.list_id !== listId) setListId(response.data.list_id);
    } catch (error) {
      toast.error('Não foi possível carregar as tarefas', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [view, listId, sort, search, toast]);

  useEffect(() => { void load(); }, [load]);

  useEffect(() => {
    const id = Number(searchParams.get('task'));
    if (!Number.isFinite(id) || id <= 0) return;
    api.get<{ data: { task: TodoTask } }>(`/tasks/${id}`)
      .then((response) => openEditTask(response.data.task))
      .catch((error) => toast.error('Não foi possível abrir a tarefa', errorMessage(error)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const lists = data?.lists ?? [];
  const activeList = lists.find((list) => list.id === listId);

  function openCreateTask() {
    const defaultList = listId || lists[0]?.id || 0;
    setTaskForm({
      title: '', todo_list_id: defaultList, due_at: '', remind_at: '', notes: '',
      repeat_type: 'none', is_important: view === 'important', my_day: view === 'my_day',
    });
    setTaskModal({ open: true, task: null });
  }

  function openEditTask(task: TodoTask) {
    setTaskForm({
      title: task.title, todo_list_id: task.todo_list_id, due_at: task.due_at ?? '',
      remind_at: task.remind_at ? task.remind_at.slice(0, 16) : '', notes: task.notes ?? '',
      repeat_type: task.repeat_type, is_important: task.is_important,
      my_day: Boolean(task.my_day_on && data && task.my_day_on === data.today),
    });
    setTaskModal({ open: true, task });
  }

  async function saveTask() {
    setSaving(true);
    try {
      const payload = { ...taskForm, remind_at: taskForm.remind_at || '' };
      if (taskModal.task) await api.put(`/tasks/${taskModal.task.id}`, payload);
      else await api.post('/tasks', payload);
      toast.success(taskModal.task ? 'Tarefa atualizada.' : 'Tarefa criada com sucesso.');
      setTaskModal({ open: false, task: null });
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar a tarefa', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function saveList() {
    setSaving(true);
    try {
      if (listModal.list) await api.put(`/tasks/lists/${listModal.list.id}`, listForm);
      else {
        const response = await api.post<{ data: { list: TodoList } }>('/tasks/lists', listForm);
        setListId(response.data.list.id);
        setView('list');
      }
      toast.success(listModal.list ? 'Lista atualizada.' : 'Lista criada.');
      setListModal({ open: false, list: null });
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar a lista', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function deleteList(list: TodoList) {
    if (!window.confirm(`Excluir a lista “${list.name}” e todas as suas tarefas?`)) return;
    try {
      await api.delete(`/tasks/lists/${list.id}`);
      setListId(0);
      setView('my_day');
      toast.success('Lista excluída.');
      await load();
    } catch (error) {
      toast.error('Não foi possível excluir a lista', errorMessage(error));
    }
  }

  async function addStep() {
    if (!detail || !stepTitle.trim()) return;
    try {
      await api.post(`/tasks/${detail.id}/steps`, { title: stepTitle });
      setStepTitle('');
      const response = await api.get<{ data: { task: TodoTask } }>(`/tasks/${detail.id}`);
      setDetail(response.data.task);
      await load();
    } catch (error) {
      toast.error('Não foi possível adicionar a etapa', errorMessage(error));
    }
  }

  async function toggleStep(step: TodoStep) {
    if (!detail) return;
    try {
      await api.put(`/tasks/${detail.id}/steps/${step.id}`, { completed: !step.completed_at });
      const response = await api.get<{ data: { task: TodoTask } }>(`/tasks/${detail.id}`);
      setDetail(response.data.task);
      await load();
    } catch (error) {
      toast.error('Não foi possível atualizar a etapa', errorMessage(error));
    }
  }

  async function deleteStep(step: TodoStep) {
    if (!detail) return;
    await api.delete(`/tasks/${detail.id}/steps/${step.id}`);
    const response = await api.get<{ data: { task: TodoTask } }>(`/tasks/${detail.id}`);
    setDetail(response.data.task);
    await load();
  }

  const tasks = data?.tasks ?? [];
  const activeView = VIEWS.find((item) => item.key === view);

  return (
    <div className="page">
      <PageHeader
        eyebrow="Produtividade"
        title="Tarefas"
        description="Organize listas, planeje o dia e acompanhe suas etapas."
        actions={
          <>
            <Button kind="tertiary" renderIcon={Add} onClick={() => { setListForm({ name: '', color: 'indigo', icon: 'check-circle' }); setListModal({ open: true, list: null }); }}>
              Nova lista
            </Button>
            <Button renderIcon={Add} onClick={openCreateTask} disabled={lists.length === 0}>Nova tarefa</Button>
          </>
        }
      />

      <div className="module-layout">
        <aside className="module-sidebar">
          <Tile className="module-sidebar__card">
            <h3>Visualizações</h3>
            {VIEWS.map((item) => {
              const Icon = item.icon;
              const active = view === item.key && listId === 0;
              const count = data?.counts[item.count] ?? 0;
              return (
                <button
                  key={item.key}
                  type="button"
                  className={`module-nav-item ${active ? 'is-active' : ''}`}
                  onClick={() => { setListId(0); setView(item.key); }}
                >
                  <Icon size={16} />
                  <span>{item.label}</span>
                  {count > 0 && <span className="module-nav-item__count">{count}</span>}
                </button>
              );
            })}
          </Tile>
          <Tile className="module-sidebar__card">
            <div className="module-sidebar__heading">
              <h3>Minhas listas</h3>
              <Button kind="ghost" size="sm" hasIconOnly renderIcon={Add} iconDescription="Nova lista" onClick={() => { setListForm({ name: '', color: 'indigo', icon: 'check-circle' }); setListModal({ open: true, list: null }); }} />
            </div>
            {lists.length === 0 && <p className="muted small">Nenhuma lista ainda. Crie a primeira para começar.</p>}
            {lists.map((list) => (
              <div key={list.id} className={`module-nav-row ${view === 'list' && listId === list.id ? 'is-active' : ''}`}>
                <button type="button" onClick={() => { setView('list'); setListId(list.id); }}>
                  <span className="color-dot" style={{ background: colorFor(list.color) }} />
                  <span>{list.name}</span>
                  {list.task_count > 0 && <span className="module-nav-item__count">{list.task_count}</span>}
                </button>
                <OverflowMenu iconDescription="Ações da lista" renderIcon={OverflowMenuVertical} size="xs" flipped>
                  <OverflowMenuItem itemText="Renomear" onClick={() => { setListForm({ name: list.name, color: list.color, icon: list.icon }); setListModal({ open: true, list }); }} />
                  <OverflowMenuItem itemText="Excluir" isDelete hasDivider onClick={() => void deleteList(list)} />
                </OverflowMenu>
              </div>
            ))}
          </Tile>
        </aside>

        <div className="module-main">
          <Tile className="module-main__header">
            <div>
              <h2>{view === 'list' && activeList ? activeList.name : activeView?.label ?? 'Tarefas'}</h2>
              <p>{tasks.length} tarefa(s) em exibição</p>
            </div>
            <div className="module-toolbar">
              <Search size="lg" labelText="Pesquisar tarefas" placeholder="Pesquisar tarefas..." value={search} onChange={(event) => setSearch(event.target.value)} onClear={() => setSearch('')} />
              <Select id="task-sort" labelText="Ordenar" value={sort} size="sm" onChange={(event) => setSort(event.target.value)}>
                <SelectItem value="manual" text="Ordem manual" />
                <SelectItem value="due" text="Data de vencimento" />
                <SelectItem value="important" text="Importância" />
                <SelectItem value="title" text="Nome" />
                <SelectItem value="created" text="Data de criação" />
              </Select>
            </div>
          </Tile>

          {loading ? <InlineLoading description="Carregando tarefas…" /> : tasks.length === 0 ? (
            <EmptyState icon={Checkmark} title="Nenhuma tarefa nesta visão" description="Crie uma tarefa para começar." action={<Button renderIcon={Add} onClick={openCreateTask} disabled={lists.length === 0}>Nova tarefa</Button>} />
          ) : (
            <div className="task-list">
              {tasks.map((task) => <TaskRow key={task.id} task={task} onRefresh={() => void load()} onOpen={(item) => setDetail(item)} />)}
            </div>
          )}

          {(data?.completed.length ?? 0) > 0 && view !== 'completed' && (
            <Tile className="module-subcard">
              <h3>Concluídas ({data?.completed.length})</h3>
              {data?.completed.map((task) => <TaskRow key={task.id} task={task} onRefresh={() => void load()} onOpen={(item) => setDetail(item)} />)}
            </Tile>
          )}
        </div>
      </div>

      <Modal
        open={taskModal.open}
        modalHeading={taskModal.task ? 'Editar tarefa' : 'Nova tarefa'}
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving}
        onRequestSubmit={() => void saveTask()}
        onRequestClose={() => setTaskModal({ open: false, task: null })}
        size="sm"
      >
        <div className="modal-form">
          <TextInput id="task-title" labelText="Tarefa" value={taskForm.title} onChange={(event) => setTaskForm({ ...taskForm, title: event.target.value })} required />
          <Select id="task-list" labelText="Lista" value={String(taskForm.todo_list_id)} onChange={(event) => setTaskForm({ ...taskForm, todo_list_id: Number(event.target.value) })}>
            {lists.map((list) => <SelectItem key={list.id} value={String(list.id)} text={list.name} />)}
          </Select>
          <div className="form-row">
            <TextInput id="task-due" type="date" labelText="Vencimento" value={taskForm.due_at} onChange={(event) => setTaskForm({ ...taskForm, due_at: event.target.value })} />
            <TextInput id="task-remind" type="datetime-local" labelText="Lembrete" value={taskForm.remind_at} onChange={(event) => setTaskForm({ ...taskForm, remind_at: event.target.value })} />
          </div>
          <Select id="task-repeat" labelText="Repetição" value={taskForm.repeat_type} onChange={(event) => setTaskForm({ ...taskForm, repeat_type: event.target.value })}>
            {REPEATS.map((item) => <SelectItem key={item.value} value={item.value} text={item.label} />)}
          </Select>
          <TextArea id="task-notes" labelText="Anotações" value={taskForm.notes} onChange={(event) => setTaskForm({ ...taskForm, notes: event.target.value })} rows={3} />
          <Checkbox id="task-important" labelText="Importante" checked={taskForm.is_important} onChange={(_e, { checked }) => setTaskForm({ ...taskForm, is_important: checked })} />
          <Checkbox id="task-myday" labelText="Adicionar ao Meu Dia" checked={taskForm.my_day} onChange={(_e, { checked }) => setTaskForm({ ...taskForm, my_day: checked })} />
        </div>
      </Modal>

      <Modal
        open={listModal.open}
        modalHeading={listModal.list ? 'Editar lista' : 'Nova lista'}
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving}
        onRequestSubmit={() => void saveList()}
        onRequestClose={() => setListModal({ open: false, list: null })}
        size="sm"
      >
        <div className="modal-form">
          <TextInput id="list-name" labelText="Nome" value={listForm.name} onChange={(event) => setListForm({ ...listForm, name: event.target.value })} required />
          <div>
            <p className="field-label">Cor</p>
            <div className="color-options">
              {COLORS.map((color) => (
                <button key={color} type="button" className={`color-option ${listForm.color === color ? 'is-selected' : ''}`} style={{ background: colorFor(color) }} aria-label={color} onClick={() => setListForm({ ...listForm, color })} />
              ))}
            </div>
          </div>
        </div>
      </Modal>

      <Modal
        open={Boolean(detail)}
        modalHeading={detail?.title ?? ''}
        primaryButtonText="Fechar"
        onRequestClose={() => setDetail(null)}
        onRequestSubmit={() => setDetail(null)}
        size="sm"
      >
        {detail && (
          <div className="modal-form">
            <div className="tag-row">
              <span className="task-badge">{detail.list_name}</span>
              {detail.due_at && <span className="task-badge">{detail.due_at}</span>}
            </div>
            {detail.notes && <p>{detail.notes}</p>}
            <h4>Etapas</h4>
            {detail.steps.map((step) => (
              <div key={step.id} className="step-row">
                <Checkbox id={`step-${step.id}`} labelText={step.title} checked={Boolean(step.completed_at)} onChange={() => void toggleStep(step)} />
                <Button kind="ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Remover etapa" onClick={() => void deleteStep(step)} />
              </div>
            ))}
            <div className="inline-form">
              <TextInput id="step-title" labelText="Nova etapa" value={stepTitle} onChange={(event) => setStepTitle(event.target.value)} />
              <Button kind="tertiary" onClick={() => void addStep()}>Adicionar</Button>
            </div>
            <div className="section-actions">
              <Button kind="tertiary" renderIcon={Edit} onClick={() => { setDetail(null); openEditTask(detail); }}>Editar tarefa</Button>
              <Button kind="danger--ghost" renderIcon={TrashCan} onClick={async () => { await api.delete(`/tasks/${detail.id}`); setDetail(null); await load(); }}>Excluir</Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
