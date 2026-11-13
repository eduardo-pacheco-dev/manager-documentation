import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Add,
  ChartBar,
  Download,
  Edit,
  Milestone,
  Settings,
  TrashCan,
} from '@carbon/icons-react';
import {
  Button,
  InlineLoading,
  Modal,
  ProgressBar,
  Select,
  SelectItem,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Tag,
  TextArea,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState } from '../../components/Feedback';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage } from '../../lib/format';

interface Project {
  id: number; name: string; description: string | null; owner_id: number; owner_name: string;
  status: string; start_date: string; end_date: string; currency: string; budget: number;
  working_days: string; hours_per_day: number; archived: boolean; created_at: string; updated_at: string;
  task_count: number; done_count: number; progress: number;
}
interface ProjectTask {
  id: number; project_id: number; parent_id: number | null; wbs: string; name: string; notes: string | null;
  task_type: string; start_date: string | null; end_date: string | null; duration_days: number; progress: number;
  priority: string; effort_hours: number; fixed_cost: number; assignee_id: number | null; assignee_name: string;
  position: number; level: number; scheduled_start: string; scheduled_end: string; effective_duration: number;
  is_critical: boolean; assigned_cost: number; total_cost: number; resources: string[];
  baseline_start?: string | null; baseline_end?: string | null;
  start_variance_days: number; end_variance_days: number; cost_variance: number;
}
interface Dependency { id: number; predecessor_id: number; successor_id: number; type: string; lag_days: number }
interface Resource { id: number; collaborator_id: number | null; name: string; role: string | null; email: string | null; cost_hour: number; max_units: number; color: string }
interface CollaboratorOption {
  id: number; name: string; role: string | null; email: string | null; cost_hour: number;
  max_units: number; color: string; is_active: boolean;
}
interface Assignment { id: number; task_id: number; resource_id: number; units: number; planned_hours: number; resource_name: string; cost_hour: number; cost: number }
interface Holiday { id: number; date: string; name: string }
interface Baseline { id: number; name: string; created_at: string; task_count: number }
interface Metrics {
  task_count: number; completed: number; in_progress: number; overdue: number; milestones: number;
  critical: number; progress: number; planned_cost: number; budget: number; cost_variance: number;
  scheduled_start: string; scheduled_end: string; duration_days: number; days_remaining: number;
}
interface Bundle {
  project: Project; tasks: ProjectTask[]; dependencies: Dependency[]; resources: Resource[];
  assignments: Assignment[]; holidays: Holiday[]; baselines: Baseline[]; metrics: Metrics;
}

const STATUS: Record<string, string> = {
  planejamento: 'Planejamento', em_andamento: 'Em andamento', pausado: 'Pausado', concluido: 'Concluído', cancelado: 'Cancelado',
};
const PRIORITIES: Record<string, string> = { baixa: 'Baixa', media: 'Média', alta: 'Alta', critica: 'Crítica' };
const TASK_TYPES: Record<string, string> = { task: 'Tarefa', summary: 'Resumo', milestone: 'Marco' };
const DEP_TYPES: Record<string, string> = { FS: 'Término → Início', SS: 'Início → Início', FF: 'Término → Término', SF: 'Início → Término' };

function statusTag(value: string): 'blue' | 'green' | 'gray' | 'red' | 'purple' {
  switch (value) {
    case 'em_andamento': return 'blue';
    case 'concluido': return 'green';
    case 'cancelado': return 'red';
    case 'pausado': return 'purple';
    default: return 'gray';
  }
}

function money(value: number, currency: string) {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: currency || 'BRL' }).format(value || 0);
}

function fmtDate(value: string | null | undefined) {
  if (!value) return '—';
  const [year, month, day] = value.slice(0, 10).split('-');
  return year && month && day ? `${day}/${month}/${year}` : value;
}

const emptyProjectForm = {
  name: '', description: '', status: 'planejamento', start_date: new Date().toISOString().slice(0, 10),
  end_date: new Date().toISOString().slice(0, 10), currency: 'BRL', budget: '0', working_days: '1,2,3,4,5', hours_per_day: '8',
};
const emptyTaskForm = {
  parent_id: '', name: '', notes: '', task_type: 'task', start_date: '', duration_days: '1',
  progress: '0', priority: 'media', effort_hours: '0', fixed_cost: '0',
};

export function ProjectsPage() {
  const toast = useToast();
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedID, setSelectedID] = useState<number | null>(null);
  const [bundle, setBundle] = useState<Bundle | null>(null);
  const [loading, setLoading] = useState(true);
  const [bundleLoading, setBundleLoading] = useState(false);
  const [tab, setTab] = useState(0);
  const [saving, setSaving] = useState(false);

  const [projectModal, setProjectModal] = useState<'create' | 'edit' | null>(null);
  const [projectForm, setProjectForm] = useState(emptyProjectForm);

  const [taskModal, setTaskModal] = useState(false);
  const [editingTask, setEditingTask] = useState<ProjectTask | null>(null);
  const [taskForm, setTaskForm] = useState(emptyTaskForm);

  const [dependencyForm, setDependencyForm] = useState({ predecessor_id: '', successor_id: '', type: 'FS', lag_days: '0' });

  const [resourceForm, setResourceForm] = useState({ collaborator_id: '', name: '', role: '', email: '', cost_hour: '0', max_units: '100', color: 'blue' });
  const [collaborators, setCollaborators] = useState<CollaboratorOption[]>([]);
  const [assignmentForm, setAssignmentForm] = useState({ task_id: '', resource_id: '', units: '100', planned_hours: '0' });

  const [holidayForm, setHolidayForm] = useState({ date: '', name: '' });
  const [baselineName, setBaselineName] = useState('');

  const project = bundle?.project;
  const tasks = bundle?.tasks ?? [];
  const leafTasks = useMemo(() => tasks.filter((task) => task.task_type !== 'summary'), [tasks]);
  const taskName = useMemo(() => {
    const map: Record<number, string> = {};
    tasks.forEach((task) => { map[task.id] = `${task.wbs} ${task.name}`; });
    return map;
  }, [tasks]);

  const loadProjects = useCallback(async (preferred?: number) => {
    try {
      const response = await api.get<{ data: { projects: Project[] } }>('/projects?archived=1');
      const items = response.data.projects;
      setProjects(items);
      setSelectedID((current) => {
        const wanted = preferred ?? current;
        if (wanted && items.some((item) => item.id === wanted)) return wanted;
        return items.find((item) => !item.archived)?.id ?? items[0]?.id ?? null;
      });
    } catch (error) {
      toast.error('Não foi possível carregar os projetos', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [toast]);

  const loadBundle = useCallback(async (id: number) => {
    setBundleLoading(true);
    try {
      const response = await api.get<{ data: Bundle }>(`/projects/${id}`);
      setBundle(response.data);
    } catch (error) {
      toast.error('Não foi possível carregar o projeto', errorMessage(error));
      setBundle(null);
    } finally {
      setBundleLoading(false);
    }
  }, [toast]);

  useEffect(() => { void loadProjects(); }, [loadProjects]);
  useEffect(() => { if (selectedID) void loadBundle(selectedID); }, [selectedID, loadBundle]);
  useEffect(() => {
    api.get<{ data: CollaboratorOption[] }>('/collaborators?status=active&per_page=200')
      .then((response) => setCollaborators(response.data))
      .catch(() => setCollaborators([]));
  }, []);

  async function refresh() {
    await loadProjects(selectedID ?? undefined);
    if (selectedID) await loadBundle(selectedID);
  }

  function openProjectCreate() {
    setProjectForm(emptyProjectForm);
    setProjectModal('create');
  }

  function openProjectEdit() {
    if (!project) return;
    setProjectForm({
      name: project.name, description: project.description ?? '', status: project.status,
      start_date: project.start_date, end_date: project.end_date, currency: project.currency,
      budget: String(project.budget), working_days: project.working_days, hours_per_day: String(project.hours_per_day),
    });
    setProjectModal('edit');
  }

  async function submitProject() {
    if (!projectForm.name.trim()) {
      toast.error('Informe o nome do projeto');
      return;
    }
    setSaving(true);
    try {
      const payload = {
        ...projectForm,
        budget: Number(projectForm.budget) || 0,
        hours_per_day: Number(projectForm.hours_per_day) || 8,
      };
      if (projectModal === 'edit' && project) {
        await api.put(`/projects/${project.id}`, payload);
        toast.success('Projeto atualizado');
        setProjectModal(null);
        await refresh();
      } else {
        const response = await api.post<{ data: { project: Project } }>('/projects', payload);
        toast.success('Projeto criado');
        setProjectModal(null);
        await loadProjects(response.data.project.id);
      }
    } catch (error) {
      toast.error('Não foi possível salvar o projeto', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function archiveProject() {
    if (!project) return;
    try {
      await api.patch(`/projects/${project.id}/archive`);
      toast.success(project.archived ? 'Projeto restaurado' : 'Projeto arquivado');
      await refresh();
    } catch (error) {
      toast.error('Não foi possível arquivar o projeto', errorMessage(error));
    }
  }

  async function deleteProject() {
    if (!project || !window.confirm(`Excluir o projeto "${project.name}" e todas as tarefas?`)) return;
    try {
      await api.delete(`/projects/${project.id}`);
      toast.success('Projeto excluído');
      setSelectedID(null);
      setBundle(null);
      await loadProjects();
    } catch (error) {
      toast.error('Não foi possível excluir o projeto', errorMessage(error));
    }
  }

  function openTaskCreate() {
    setEditingTask(null);
    setTaskForm({ ...emptyTaskForm, start_date: project?.start_date ?? '' });
    setTaskModal(true);
  }

  function openTaskEdit(task: ProjectTask) {
    setEditingTask(task);
    setTaskForm({
      parent_id: task.parent_id ? String(task.parent_id) : '',
      name: task.name, notes: task.notes ?? '', task_type: task.task_type,
      start_date: task.start_date ?? '', duration_days: String(task.duration_days),
      progress: String(task.progress), priority: task.priority,
      effort_hours: String(task.effort_hours), fixed_cost: String(task.fixed_cost),
    });
    setTaskModal(true);
  }

  async function submitTask() {
    if (!project) return;
    if (!taskForm.name.trim()) {
      toast.error('Informe o nome da tarefa');
      return;
    }
    setSaving(true);
    try {
      const payload = {
        ...taskForm,
        parent_id: Number(taskForm.parent_id) || 0,
        duration_days: Number(taskForm.duration_days) || 0,
        progress: Number(taskForm.progress) || 0,
        effort_hours: Number(taskForm.effort_hours) || 0,
        fixed_cost: Number(taskForm.fixed_cost) || 0,
      };
      if (editingTask) {
        await api.put(`/projects/${project.id}/tasks/${editingTask.id}`, payload);
        toast.success('Tarefa atualizada');
      } else {
        await api.post(`/projects/${project.id}/tasks`, payload);
        toast.success('Tarefa criada');
      }
      setTaskModal(false);
      await refresh();
    } catch (error) {
      toast.error('Não foi possível salvar a tarefa', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function deleteTask(task: ProjectTask) {
    if (!project || !window.confirm(`Excluir a tarefa "${task.name}"?`)) return;
    try {
      await api.delete(`/projects/${project.id}/tasks/${task.id}`);
      toast.success('Tarefa excluída');
      await refresh();
    } catch (error) {
      toast.error('Não foi possível excluir a tarefa', errorMessage(error));
    }
  }

  async function submitDependency() {
    if (!project) return;
    try {
      await api.post(`/projects/${project.id}/dependencies`, {
        predecessor_id: Number(dependencyForm.predecessor_id) || 0,
        successor_id: Number(dependencyForm.successor_id) || 0,
        type: dependencyForm.type,
        lag_days: Number(dependencyForm.lag_days) || 0,
      });
      toast.success('Dependência criada');
      setDependencyForm({ predecessor_id: '', successor_id: '', type: 'FS', lag_days: '0' });
      await refresh();
    } catch (error) {
      toast.error('Não foi possível criar a dependência', errorMessage(error));
    }
  }

  async function deleteDependency(id: number) {
    if (!project) return;
    try {
      await api.delete(`/projects/${project.id}/dependencies/${id}`);
      toast.success('Dependência removida');
      await refresh();
    } catch (error) {
      toast.error('Não foi possível remover a dependência', errorMessage(error));
    }
  }

  function selectCollaborator(value: string) {
    const collaborator = collaborators.find((item) => String(item.id) === value);
    if (!collaborator) {
      setResourceForm((current) => ({ ...current, collaborator_id: '' }));
      return;
    }
    setResourceForm({
      collaborator_id: value,
      name: collaborator.name,
      role: collaborator.role ?? '',
      email: collaborator.email ?? '',
      cost_hour: String(collaborator.cost_hour),
      max_units: String(collaborator.max_units),
      color: collaborator.color,
    });
  }

  async function submitResource() {
    if (!project) return;
    if (!resourceForm.collaborator_id && !resourceForm.name.trim()) {
      toast.error('Selecione um colaborador ou informe o nome do recurso');
      return;
    }
    try {
      await api.post(`/projects/${project.id}/resources`, {
        ...resourceForm,
        collaborator_id: Number(resourceForm.collaborator_id) || 0,
        cost_hour: Number(resourceForm.cost_hour) || 0,
        max_units: Number(resourceForm.max_units) || 100,
      });
      toast.success('Recurso criado');
      setResourceForm({ collaborator_id: '', name: '', role: '', email: '', cost_hour: '0', max_units: '100', color: 'blue' });
      await refresh();
    } catch (error) {
      toast.error('Não foi possível criar o recurso', errorMessage(error));
    }
  }

  async function deleteResource(id: number) {
    if (!project) return;
    try {
      await api.delete(`/projects/${project.id}/resources/${id}`);
      toast.success('Recurso removido');
      await refresh();
    } catch (error) {
      toast.error('Não foi possível remover o recurso', errorMessage(error));
    }
  }

  async function submitAssignment() {
    if (!project) return;
    try {
      await api.post(`/projects/${project.id}/assignments`, {
        task_id: Number(assignmentForm.task_id) || 0,
        resource_id: Number(assignmentForm.resource_id) || 0,
        units: Number(assignmentForm.units) || 100,
        planned_hours: Number(assignmentForm.planned_hours) || 0,
      });
      toast.success('Recurso alocado');
      setAssignmentForm({ task_id: '', resource_id: '', units: '100', planned_hours: '0' });
      await refresh();
    } catch (error) {
      toast.error('Não foi possível alocar o recurso', errorMessage(error));
    }
  }

  async function deleteAssignment(id: number) {
    if (!project) return;
    try {
      await api.delete(`/projects/${project.id}/assignments/${id}`);
      toast.success('Alocação removida');
      await refresh();
    } catch (error) {
      toast.error('Não foi possível remover a alocação', errorMessage(error));
    }
  }

  async function submitHoliday() {
    if (!project) return;
    try {
      await api.post(`/projects/${project.id}/holidays`, holidayForm);
      toast.success('Feriado cadastrado');
      setHolidayForm({ date: '', name: '' });
      await refresh();
    } catch (error) {
      toast.error('Não foi possível cadastrar o feriado', errorMessage(error));
    }
  }

  async function deleteHoliday(id: number) {
    if (!project) return;
    try {
      await api.delete(`/projects/${project.id}/holidays/${id}`);
      toast.success('Feriado removido');
      await refresh();
    } catch (error) {
      toast.error('Não foi possível remover o feriado', errorMessage(error));
    }
  }

  async function createBaseline() {
    if (!project) return;
    try {
      await api.post(`/projects/${project.id}/baselines`, { name: baselineName });
      toast.success('Linha de base criada');
      setBaselineName('');
      await refresh();
    } catch (error) {
      toast.error('Não foi possível criar a linha de base', errorMessage(error));
    }
  }

  async function deleteBaseline(id: number) {
    if (!project) return;
    try {
      await api.delete(`/projects/${project.id}/baselines/${id}`);
      toast.success('Linha de base removida');
      await refresh();
    } catch (error) {
      toast.error('Não foi possível remover a linha de base', errorMessage(error));
    }
  }

  async function toggleTaskProgress(task: ProjectTask) {
    if (!project) return;
    const next = task.progress >= 100 ? 0 : 100;
    try {
      await api.put(`/projects/${project.id}/tasks/${task.id}`, {
        parent_id: task.parent_id ?? 0, name: task.name, notes: task.notes ?? '', task_type: task.task_type,
        start_date: task.start_date ?? '', end_date: task.end_date ?? '', duration_days: task.duration_days,
        progress: next, priority: task.priority, effort_hours: task.effort_hours, fixed_cost: task.fixed_cost,
        assignee_id: task.assignee_id ?? 0, position: task.position,
      });
      await refresh();
    } catch (error) {
      toast.error('Não foi possível atualizar o progresso', errorMessage(error));
    }
  }

  if (loading) {
    return <InlineLoading description="Carregando projetos…" />;
  }

  return (
    <div className="projects-page">
      <PageHeader
        eyebrow="Portfólio"
        title="Projetos"
        description="Planejamento no estilo Microsoft Project: EAP, dependências, recursos, custos, linha de base e caminho crítico."
        actions={<Button renderIcon={Add} onClick={openProjectCreate}>Novo projeto</Button>}
      />

      <div className="module-layout projects-layout">
        <aside className="module-sidebar">
          <div className="module-sidebar__card">
            <div className="module-sidebar__heading">
              <h3>Projetos</h3>
              <span className="module-nav-item__count">{projects.length}</span>
            </div>
            {projects.length === 0 && <p className="projects-muted">Nenhum projeto ainda.</p>}
            {projects.map((item) => (
              <button
                key={item.id}
                type="button"
                className={`module-nav-item projects-nav ${selectedID === item.id ? 'is-active' : ''} ${item.archived ? 'projects-nav--archived' : ''}`}
                onClick={() => setSelectedID(item.id)}
              >
                <span className="projects-nav__body">
                  <span className="projects-nav__name">{item.name}</span>
                  <span className="projects-nav__meta">
                    {STATUS[item.status] ?? item.status} · {item.done_count}/{item.task_count} tarefas
                  </span>
                  <span className="projects-nav__bar"><span style={{ width: `${item.progress}%` }} /></span>
                </span>
              </button>
            ))}
          </div>
        </aside>

        <section className="module-main">
          {bundleLoading && <InlineLoading description="Carregando projeto…" />}
          {!bundle && !bundleLoading && (
            <EmptyState
              icon={ChartBar}
              title="Selecione um projeto"
              description="Crie um projeto para montar a EAP, definir dependências, alocar recursos e acompanhar o caminho crítico."
              action={<Button renderIcon={Add} onClick={openProjectCreate}>Novo projeto</Button>}
            />
          )}
          {bundle && project && (
            <>
              <div className="module-main__header projects-header">
                <div>
                  <div className="projects-header__title">
                    <h2>{project.name}</h2>
                    <Tag type={statusTag(project.status)}>{STATUS[project.status] ?? project.status}</Tag>
                    {project.archived && <Tag type="gray">Arquivado</Tag>}
                  </div>
                  <p>
                    {fmtDate(project.start_date)} – {fmtDate(project.end_date)} · {project.owner_name} · {project.working_days.split(',').length} dias úteis/semana
                  </p>
                </div>
                <div className="projects-header__actions">
                  <Button kind="ghost" size="sm" renderIcon={Download} href={`/api/v1/projects/${project.id}/export`}>Exportar CSV</Button>
                  <Button kind="ghost" size="sm" renderIcon={Settings} onClick={openProjectEdit}>Editar</Button>
                  <Button kind="ghost" size="sm" onClick={() => void archiveProject()}>{project.archived ? 'Restaurar' : 'Arquivar'}</Button>
                  <Button kind="danger--ghost" size="sm" renderIcon={TrashCan} onClick={() => void deleteProject()}>Excluir</Button>
                </div>
              </div>

              <div className="projects-metrics">
                <Tile className="projects-metric">
                  <span className="projects-metric__label">Progresso</span>
                  <strong>{bundle.metrics.progress}%</strong>
                  <ProgressBar label="Progresso" hideLabel value={bundle.metrics.progress} />
                </Tile>
                <Tile className="projects-metric">
                  <span className="projects-metric__label">Tarefas</span>
                  <strong>{bundle.metrics.completed}/{bundle.metrics.task_count}</strong>
                  <span className="projects-metric__hint">{bundle.metrics.in_progress} em andamento · {bundle.metrics.overdue} atrasadas</span>
                </Tile>
                <Tile className="projects-metric">
                  <span className="projects-metric__label">Caminho crítico</span>
                  <strong>{bundle.metrics.critical}</strong>
                  <span className="projects-metric__hint">{bundle.metrics.milestones} marcos no projeto</span>
                </Tile>
                <Tile className="projects-metric">
                  <span className="projects-metric__label">Custo planejado</span>
                  <strong>{money(bundle.metrics.planned_cost, project.currency)}</strong>
                  <span className={`projects-metric__hint ${bundle.metrics.cost_variance < 0 ? 'is-danger' : ''}`}>
                    Orçamento {money(bundle.metrics.budget, project.currency)} · saldo {money(bundle.metrics.cost_variance, project.currency)}
                  </span>
                </Tile>
                <Tile className="projects-metric">
                  <span className="projects-metric__label">Prazo</span>
                  <strong>{fmtDate(bundle.metrics.scheduled_end)}</strong>
                  <span className="projects-metric__hint">{bundle.metrics.duration_days} dias úteis · {bundle.metrics.days_remaining} restantes</span>
                </Tile>
              </div>

              <Tabs selectedIndex={tab} onChange={({ selectedIndex }) => setTab(selectedIndex)}>
                <TabList aria-label="Seções do projeto" contained>
                  <Tab>Visão geral</Tab>
                  <Tab>Tarefas e EAP</Tab>
                  <Tab>Gantt</Tab>
                  <Tab>Recursos</Tab>
                  <Tab>Linha de base</Tab>
                  <Tab>Configurações</Tab>
                </TabList>
                <TabPanels>
                  <TabPanel>
                    <OverviewPanel bundle={bundle} />
                  </TabPanel>
                  <TabPanel>
                    <div className="projects-toolbar">
                      <Button size="sm" renderIcon={Add} onClick={openTaskCreate}>Nova tarefa</Button>
                      <span className="projects-muted">Clique no progresso para concluir/reabrir uma tarefa.</span>
                    </div>
                    <div className="table-scroll">
                      <table className="data-table projects-task-table">
                        <thead>
                          <tr>
                            <th>EAP</th><th>Tarefa</th><th>Tipo</th><th>Início</th><th>Fim</th>
                            <th>Dur.</th><th>Progresso</th><th>Responsável / recursos</th><th>Custo</th><th />
                          </tr>
                        </thead>
                        <tbody>
                          {tasks.length === 0 && (
                            <tr><td colSpan={10}><span className="projects-muted">Nenhuma tarefa. Comece pela EAP.</span></td></tr>
                          )}
                          {tasks.map((task) => (
                            <tr key={task.id} className={task.is_critical ? 'projects-task--critical' : ''}>
                              <td className="projects-wbs">{task.wbs}</td>
                              <td>
                                <span style={{ paddingLeft: `${task.level * 1.1}rem` }} className="projects-task-name">
                                  {task.task_type === 'milestone' && <Milestone size={14} />}
                                  {task.name}
                                  {task.is_critical && <Tag type="red" size="sm">crítica</Tag>}
                                </span>
                                {task.notes && <span className="projects-task-notes">{task.notes}</span>}
                              </td>
                              <td>{TASK_TYPES[task.task_type] ?? task.task_type}</td>
                              <td>{fmtDate(task.scheduled_start)}</td>
                              <td>{fmtDate(task.scheduled_end)}</td>
                              <td>{task.effective_duration}d</td>
                              <td>
                                <button type="button" className="projects-progress" title="Alternar conclusão" onClick={() => void toggleTaskProgress(task)}>
                                  <span style={{ width: `${task.progress}%` }} />
                                  <em>{task.progress}%</em>
                                </button>
                              </td>
                              <td>{task.resources.length > 0 ? task.resources.join(', ') : task.assignee_name || '—'}</td>
                              <td>{money(task.total_cost, project.currency)}</td>
                              <td className="projects-row-actions">
                                <Button kind="ghost" size="sm" hasIconOnly renderIcon={Edit} iconDescription="Editar" onClick={() => openTaskEdit(task)} />
                                <Button kind="ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={() => void deleteTask(task)} />
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>

                    <div className="module-subcard projects-dependencies">
                      <h3>Dependências</h3>
                      <div className="projects-inline-form">
                        <Select id="dep-predecessor" labelText="Predecessora" value={dependencyForm.predecessor_id} onChange={(event) => setDependencyForm({ ...dependencyForm, predecessor_id: event.target.value })}>
                          <SelectItem value="" text="Selecione" />
                          {leafTasks.map((task) => <SelectItem key={task.id} value={String(task.id)} text={`${task.wbs} ${task.name}`} />)}
                        </Select>
                        <Select id="dep-successor" labelText="Sucessora" value={dependencyForm.successor_id} onChange={(event) => setDependencyForm({ ...dependencyForm, successor_id: event.target.value })}>
                          <SelectItem value="" text="Selecione" />
                          {leafTasks.map((task) => <SelectItem key={task.id} value={String(task.id)} text={`${task.wbs} ${task.name}`} />)}
                        </Select>
                        <Select id="dep-type" labelText="Tipo" value={dependencyForm.type} onChange={(event) => setDependencyForm({ ...dependencyForm, type: event.target.value })}>
                          {Object.entries(DEP_TYPES).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
                        </Select>
                        <TextInput id="dep-lag" labelText="Lag (dias)" type="number" value={dependencyForm.lag_days} onChange={(event) => setDependencyForm({ ...dependencyForm, lag_days: event.target.value })} />
                        <Button size="sm" onClick={() => void submitDependency()}>Adicionar</Button>
                      </div>
                      <div className="projects-dependency-list">
                        {bundle.dependencies.length === 0 && <span className="projects-muted">Sem dependências — o cronograma usa apenas as datas informadas.</span>}
                        {bundle.dependencies.map((dependency) => (
                          <span key={dependency.id} className="projects-dependency">
                            {taskName[dependency.predecessor_id]} <b>{DEP_TYPES[dependency.type] ?? dependency.type}</b> {taskName[dependency.successor_id]}
                            {dependency.lag_days !== 0 && ` (lag ${dependency.lag_days}d)`}
                            <button type="button" onClick={() => void deleteDependency(dependency.id)} aria-label="Remover dependência"><TrashCan size={12} /></button>
                          </span>
                        ))}
                      </div>
                    </div>
                  </TabPanel>

                  <TabPanel>
                    <Gantt tasks={tasks} project={project} />
                  </TabPanel>

                  <TabPanel>
                    <div className="projects-two-columns">
                      <div className="module-subcard">
                        <h3>Recursos</h3>
                        <div className="projects-form-grid">
                          <Select id="res-collaborator" labelText="Colaborador" value={resourceForm.collaborator_id} onChange={(event) => selectCollaborator(event.target.value)}>
                            <SelectItem value="" text="Recurso avulso" />
                            {collaborators.map((collaborator) => (
                              <SelectItem
                                key={collaborator.id}
                                value={String(collaborator.id)}
                                text={`${collaborator.name}${collaborator.role ? ` · ${collaborator.role}` : ''}`}
                              />
                            ))}
                          </Select>
                          <TextInput id="res-name" labelText="Nome" value={resourceForm.name} readOnly={Boolean(resourceForm.collaborator_id)} onChange={(event) => setResourceForm({ ...resourceForm, name: event.target.value })} />
                          <TextInput id="res-role" labelText="Função" value={resourceForm.role} readOnly={Boolean(resourceForm.collaborator_id)} onChange={(event) => setResourceForm({ ...resourceForm, role: event.target.value })} />
                          <TextInput id="res-email" labelText="E-mail" value={resourceForm.email} readOnly={Boolean(resourceForm.collaborator_id)} onChange={(event) => setResourceForm({ ...resourceForm, email: event.target.value })} />
                          <TextInput id="res-cost" labelText="Custo/hora" type="number" value={resourceForm.cost_hour} onChange={(event) => setResourceForm({ ...resourceForm, cost_hour: event.target.value })} />
                          <TextInput id="res-units" labelText="Capacidade (%)" type="number" value={resourceForm.max_units} onChange={(event) => setResourceForm({ ...resourceForm, max_units: event.target.value })} />
                          <Button size="sm" onClick={() => void submitResource()}>Adicionar recurso</Button>
                        </div>
                        {resourceForm.collaborator_id && (
                          <p className="projects-muted">
                            Dados herdados do cadastro de colaboradores. Custo/hora e capacidade podem ser ajustados para este projeto.
                          </p>
                        )}
                        <div className="projects-list">
                          {bundle.resources.length === 0 && <span className="projects-muted">Nenhum recurso cadastrado.</span>}
                          {bundle.resources.map((resource) => (
                            <div key={resource.id} className="projects-list-row">
                              <span className="color-dot" style={{ background: resource.color }} />
                              <span>
                                <strong>{resource.name}</strong>
                                {resource.role ? ` · ${resource.role}` : ''}
                                {resource.collaborator_id && <Tag type="blue" size="sm">colaborador</Tag>}
                              </span>
                              <span className="projects-muted">{money(resource.cost_hour, project.currency)}/h</span>
                              <Button kind="ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Remover" onClick={() => void deleteResource(resource.id)} />
                            </div>
                          ))}
                        </div>
                      </div>

                      <div className="module-subcard">
                        <h3>Alocações</h3>
                        <div className="projects-form-grid">
                          <Select id="as-task" labelText="Tarefa" value={assignmentForm.task_id} onChange={(event) => setAssignmentForm({ ...assignmentForm, task_id: event.target.value })}>
                            <SelectItem value="" text="Selecione" />
                            {leafTasks.map((task) => <SelectItem key={task.id} value={String(task.id)} text={`${task.wbs} ${task.name}`} />)}
                          </Select>
                          <Select id="as-resource" labelText="Recurso" value={assignmentForm.resource_id} onChange={(event) => setAssignmentForm({ ...assignmentForm, resource_id: event.target.value })}>
                            <SelectItem value="" text="Selecione" />
                            {bundle.resources.map((resource) => <SelectItem key={resource.id} value={String(resource.id)} text={resource.name} />)}
                          </Select>
                          <TextInput id="as-units" labelText="Unidades (%)" type="number" value={assignmentForm.units} onChange={(event) => setAssignmentForm({ ...assignmentForm, units: event.target.value })} />
                          <TextInput id="as-hours" labelText="Horas planejadas" type="number" value={assignmentForm.planned_hours} onChange={(event) => setAssignmentForm({ ...assignmentForm, planned_hours: event.target.value })} />
                          <Button size="sm" onClick={() => void submitAssignment()}>Alocar</Button>
                        </div>
                        <div className="table-scroll">
                          <table className="data-table data-table--compact">
                            <thead><tr><th>Tarefa</th><th>Recurso</th><th>Un.</th><th>Horas</th><th>Custo</th><th /></tr></thead>
                            <tbody>
                              {bundle.assignments.length === 0 && <tr><td colSpan={6}><span className="projects-muted">Nenhuma alocação.</span></td></tr>}
                              {bundle.assignments.map((assignment) => (
                                <tr key={assignment.id}>
                                  <td>{taskName[assignment.task_id]}</td>
                                  <td>{assignment.resource_name}</td>
                                  <td>{assignment.units}%</td>
                                  <td>{assignment.planned_hours}h</td>
                                  <td>{money(assignment.cost, project.currency)}</td>
                                  <td><Button kind="ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Remover" onClick={() => void deleteAssignment(assignment.id)} /></td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </div>
                    </div>
                  </TabPanel>

                  <TabPanel>
                    <div className="projects-toolbar">
                      <TextInput id="baseline-name" labelText="Nome da linha de base" value={baselineName} onChange={(event) => setBaselineName(event.target.value)} placeholder="Ex.: Aprovação do cliente" />
                      <Button size="sm" renderIcon={Add} onClick={() => void createBaseline()}>Capturar linha de base</Button>
                    </div>
                    <div className="projects-list">
                      {bundle.baselines.length === 0 && <span className="projects-muted">Nenhuma linha de base. Capture uma para comparar prazos e custos.</span>}
                      {bundle.baselines.map((baseline) => (
                        <div key={baseline.id} className="projects-list-row">
                          <strong>{baseline.name}</strong>
                          <span className="projects-muted">{baseline.task_count} tarefas · {new Date(baseline.created_at).toLocaleString('pt-BR')}</span>
                          <Button kind="ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Remover" onClick={() => void deleteBaseline(baseline.id)} />
                        </div>
                      ))}
                    </div>
                    {bundle.baselines.length > 0 && (
                      <div className="table-scroll">
                        <table className="data-table">
                          <thead><tr><th>EAP</th><th>Tarefa</th><th>Início base</th><th>Fim base</th><th>Início atual</th><th>Fim atual</th><th>Variação</th><th>Custo var.</th></tr></thead>
                          <tbody>
                            {tasks.filter((task) => task.task_type !== 'summary').map((task) => (
                              <tr key={task.id}>
                                <td>{task.wbs}</td><td>{task.name}</td>
                                <td>{fmtDate(task.baseline_start)}</td><td>{fmtDate(task.baseline_end)}</td>
                                <td>{fmtDate(task.scheduled_start)}</td><td>{fmtDate(task.scheduled_end)}</td>
                                <td className={task.end_variance_days > 0 ? 'projects-late' : ''}>{task.end_variance_days > 0 ? `+${task.end_variance_days}d` : `${task.end_variance_days}d`}</td>
                                <td className={task.cost_variance > 0 ? 'projects-late' : ''}>{money(task.cost_variance, project.currency)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </TabPanel>

                  <TabPanel>
                    <div className="projects-two-columns">
                      <div className="module-subcard">
                        <h3>Calendário</h3>
                        <p className="projects-muted">Dias úteis: {project.working_days}. Jornada: {project.hours_per_day}h/dia.</p>
                        <div className="projects-form-grid">
                          <TextInput id="holiday-date" labelText="Feriado" type="date" value={holidayForm.date} onChange={(event) => setHolidayForm({ ...holidayForm, date: event.target.value })} />
                          <TextInput id="holiday-name" labelText="Descrição" value={holidayForm.name} onChange={(event) => setHolidayForm({ ...holidayForm, name: event.target.value })} />
                          <Button size="sm" onClick={() => void submitHoliday()}>Adicionar feriado</Button>
                        </div>
                        <div className="projects-list">
                          {bundle.holidays.length === 0 && <span className="projects-muted">Nenhum feriado cadastrado.</span>}
                          {bundle.holidays.map((holiday) => (
                            <div key={holiday.id} className="projects-list-row">
                              <span>{fmtDate(holiday.date)}</span><span>{holiday.name}</span>
                              <Button kind="ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Remover" onClick={() => void deleteHoliday(holiday.id)} />
                            </div>
                          ))}
                        </div>
                      </div>
                      <div className="module-subcard">
                        <h3>Resumo do cronograma</h3>
                        <dl className="projects-summary">
                          <div><dt>Início</dt><dd>{fmtDate(bundle.metrics.scheduled_start)}</dd></div>
                          <div><dt>Término calculado</dt><dd>{fmtDate(bundle.metrics.scheduled_end)}</dd></div>
                          <div><dt>Duração</dt><dd>{bundle.metrics.duration_days} dias úteis</dd></div>
                          <div><dt>Caminho crítico</dt><dd>{bundle.metrics.critical} tarefas</dd></div>
                          <div><dt>Custo planejado</dt><dd>{money(bundle.metrics.planned_cost, project.currency)}</dd></div>
                          <div><dt>Saldo do orçamento</dt><dd>{money(bundle.metrics.cost_variance, project.currency)}</dd></div>
                        </dl>
                      </div>
                    </div>
                  </TabPanel>
                </TabPanels>
              </Tabs>
            </>
          )}
        </section>
      </div>

      <Modal
        open={projectModal !== null}
        modalHeading={projectModal === 'edit' ? 'Editar projeto' : 'Novo projeto'}
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving}
        onRequestSubmit={() => void submitProject()}
        onRequestClose={() => setProjectModal(null)}
      >
        <div className="projects-modal-grid">
          <TextInput id="project-name" labelText="Nome" value={projectForm.name} onChange={(event) => setProjectForm({ ...projectForm, name: event.target.value })} />
          <Select id="project-status" labelText="Status" value={projectForm.status} onChange={(event) => setProjectForm({ ...projectForm, status: event.target.value })}>
            {Object.entries(STATUS).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
          </Select>
          <TextInput id="project-start" labelText="Início" type="date" value={projectForm.start_date} onChange={(event) => setProjectForm({ ...projectForm, start_date: event.target.value })} />
          <TextInput id="project-end" labelText="Término planejado" type="date" value={projectForm.end_date} onChange={(event) => setProjectForm({ ...projectForm, end_date: event.target.value })} />
          <TextInput id="project-currency" labelText="Moeda" value={projectForm.currency} onChange={(event) => setProjectForm({ ...projectForm, currency: event.target.value })} />
          <TextInput id="project-budget" labelText="Orçamento" type="number" value={projectForm.budget} onChange={(event) => setProjectForm({ ...projectForm, budget: event.target.value })} />
          <TextInput id="project-working" labelText="Dias úteis (0=dom … 6=sáb)" value={projectForm.working_days} onChange={(event) => setProjectForm({ ...projectForm, working_days: event.target.value })} />
          <TextInput id="project-hours" labelText="Horas por dia" type="number" value={projectForm.hours_per_day} onChange={(event) => setProjectForm({ ...projectForm, hours_per_day: event.target.value })} />
          <TextArea id="project-description" labelText="Descrição" value={projectForm.description} onChange={(event) => setProjectForm({ ...projectForm, description: event.target.value })} />
        </div>
      </Modal>

      <Modal
        open={taskModal}
        modalHeading={editingTask ? 'Editar tarefa' : 'Nova tarefa'}
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving}
        onRequestSubmit={() => void submitTask()}
        onRequestClose={() => setTaskModal(false)}
      >
        <div className="projects-modal-grid">
          <TextInput id="task-name" labelText="Nome" value={taskForm.name} onChange={(event) => setTaskForm({ ...taskForm, name: event.target.value })} />
          <Select id="task-parent" labelText="Tarefa pai (EAP)" value={taskForm.parent_id} onChange={(event) => setTaskForm({ ...taskForm, parent_id: event.target.value })}>
            <SelectItem value="" text="Nenhuma (nível raiz)" />
            {tasks.filter((task) => task.id !== editingTask?.id).map((task) => (
              <SelectItem key={task.id} value={String(task.id)} text={`${task.wbs} ${task.name}`} />
            ))}
          </Select>
          <Select id="task-type" labelText="Tipo" value={taskForm.task_type} onChange={(event) => setTaskForm({ ...taskForm, task_type: event.target.value })}>
            {Object.entries(TASK_TYPES).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
          </Select>
          <Select id="task-priority" labelText="Prioridade" value={taskForm.priority} onChange={(event) => setTaskForm({ ...taskForm, priority: event.target.value })}>
            {Object.entries(PRIORITIES).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
          </Select>
          <TextInput id="task-start" labelText="Início" type="date" value={taskForm.start_date} onChange={(event) => setTaskForm({ ...taskForm, start_date: event.target.value })} />
          <TextInput id="task-duration" labelText="Duração (dias úteis)" type="number" value={taskForm.duration_days} onChange={(event) => setTaskForm({ ...taskForm, duration_days: event.target.value })} />
          <TextInput id="task-progress" labelText="Progresso (%)" type="number" value={taskForm.progress} onChange={(event) => setTaskForm({ ...taskForm, progress: event.target.value })} />
          <TextInput id="task-effort" labelText="Esforço (horas)" type="number" value={taskForm.effort_hours} onChange={(event) => setTaskForm({ ...taskForm, effort_hours: event.target.value })} />
          <TextInput id="task-cost" labelText="Custo fixo" type="number" value={taskForm.fixed_cost} onChange={(event) => setTaskForm({ ...taskForm, fixed_cost: event.target.value })} />
          <TextArea id="task-notes" labelText="Anotações" value={taskForm.notes} onChange={(event) => setTaskForm({ ...taskForm, notes: event.target.value })} />
        </div>
      </Modal>
    </div>
  );
}

function OverviewPanel({ bundle }: { bundle: Bundle }) {
  const tasks = bundle.tasks;
  const critical = tasks.filter((task) => task.is_critical && task.task_type !== 'summary').slice(0, 6);
  const milestones = tasks.filter((task) => task.task_type === 'milestone').sort((a, b) => a.scheduled_end.localeCompare(b.scheduled_end)).slice(0, 6);
  const upcoming = tasks
    .filter((task) => task.task_type !== 'summary' && task.progress < 100)
    .sort((a, b) => a.scheduled_end.localeCompare(b.scheduled_end))
    .slice(0, 6);

  return (
    <div className="projects-two-columns">
      <div className="module-subcard">
        <h3>Próximos trabalhos</h3>
        {upcoming.length === 0 && <span className="projects-muted">Nenhuma tarefa pendente.</span>}
        {upcoming.map((task) => (
          <div key={task.id} className="projects-list-row">
            <span className="projects-wbs">{task.wbs}</span>
            <span><strong>{task.name}</strong><br /><span className="projects-muted">{fmtDate(task.scheduled_start)} – {fmtDate(task.scheduled_end)}</span></span>
            <Tag type={task.is_critical ? 'red' : 'gray'} size="sm">{task.is_critical ? 'crítica' : `${task.progress}%`}</Tag>
          </div>
        ))}
      </div>
      <div className="module-subcard">
        <h3>Caminho crítico</h3>
        {critical.length === 0 && <span className="projects-muted">Sem tarefas críticas — aumente as dependências para calcular o caminho.</span>}
        {critical.map((task) => (
          <div key={task.id} className="projects-list-row">
            <span className="projects-wbs">{task.wbs}</span>
            <span>{task.name}</span>
            <span className="projects-muted">{task.effective_duration}d</span>
          </div>
        ))}
        <h3 className="projects-subheading">Marcos</h3>
        {milestones.length === 0 && <span className="projects-muted">Nenhum marco cadastrado.</span>}
        {milestones.map((task) => (
          <div key={task.id} className="projects-list-row">
            <Milestone size={14} />
            <span>{task.name}</span>
            <span className="projects-muted">{fmtDate(task.scheduled_end)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function Gantt({ tasks, project }: { tasks: ProjectTask[]; project: Project }) {
  const visible = tasks.filter((task) => task.scheduled_start && task.scheduled_end);
  if (visible.length === 0) {
    return <EmptyState icon={ChartBar} title="Gantt vazio" description="Cadastre tarefas para visualizar o cronograma." />;
  }
  const parse = (value: string) => new Date(`${value.slice(0, 10)}T00:00:00`).getTime();
  const start = Math.min(...visible.map((task) => parse(task.scheduled_start)));
  const end = Math.max(...visible.map((task) => parse(task.scheduled_end)));
  const span = Math.max(end - start, 24 * 60 * 60 * 1000);
  const months: { label: string; left: number; width: number }[] = [];
  const cursor = new Date(start);
  cursor.setDate(1);
  while (cursor.getTime() <= end) {
    const monthStart = Math.max(cursor.getTime(), start);
    const next = new Date(cursor.getFullYear(), cursor.getMonth() + 1, 1).getTime();
    const monthEnd = Math.min(next, end + 24 * 60 * 60 * 1000);
    months.push({
      label: cursor.toLocaleDateString('pt-BR', { month: 'short', year: '2-digit' }),
      left: ((monthStart - start) / span) * 100,
      width: ((monthEnd - monthStart) / span) * 100,
    });
    cursor.setMonth(cursor.getMonth() + 1);
  }
  return (
    <div className="projects-gantt">
      <div className="projects-gantt__legend">
        <span><i className="projects-legend projects-legend--task" /> Tarefa</span>
        <span><i className="projects-legend projects-legend--critical" /> Crítica</span>
        <span><i className="projects-legend projects-legend--summary" /> Resumo</span>
        <span><i className="projects-legend projects-legend--milestone" /> Marco</span>
        <span className="projects-muted">Moeda: {project.currency}</span>
      </div>
      <div className="projects-gantt__head">
        <div className="projects-gantt__label">EAP / Tarefa</div>
        <div className="projects-gantt__timeline">
          {months.map((month) => (
            <span key={`${month.label}-${month.left}`} style={{ left: `${month.left}%`, width: `${month.width}%` }}>{month.label}</span>
          ))}
        </div>
      </div>
      {visible.map((task) => {
        const taskStart = parse(task.scheduled_start);
        const taskEnd = parse(task.scheduled_end);
        const left = ((taskStart - start) / span) * 100;
        const width = Math.max(((taskEnd - taskStart) / span) * 100, 0.6);
        return (
          <div key={task.id} className="projects-gantt__row">
            <div className="projects-gantt__label" style={{ paddingLeft: `${0.5 + task.level * 1.1}rem` }}>
              <span className="projects-wbs">{task.wbs}</span> {task.name}
            </div>
            <div className="projects-gantt__timeline">
              <span className="projects-gantt__grid" />
              {task.task_type === 'milestone' ? (
                <span
                  className="projects-gantt__milestone"
                  style={{ left: `${left}%` }}
                  title={`${task.name} · ${fmtDate(task.scheduled_end)}`}
                />
              ) : (
                <span
                  className={`projects-gantt__bar ${task.task_type === 'summary' ? 'is-summary' : ''} ${task.is_critical ? 'is-critical' : ''}`}
                  style={{ left: `${left}%`, width: `${width}%` }}
                  title={`${task.name} · ${fmtDate(task.scheduled_start)} – ${fmtDate(task.scheduled_end)} · ${task.progress}%`}
                >
                  <span style={{ width: `${task.progress}%` }} />
                  <em>{task.progress}%</em>
                </span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
