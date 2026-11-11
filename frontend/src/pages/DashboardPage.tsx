import { Fragment, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Add,
  ArrowDown,
  ArrowUp,
  Checkmark,
  Close,
  Draggable,
  Edit,
  Time,
  TrashCan,
  UserMultiple,
} from '@carbon/icons-react';
import {
  Button,
  ClickableTile,
  InlineLoading,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tag,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../components/PageHeader';
import { EmptyState, Loading } from '../components/Feedback';
import { useAuth } from '../contexts/AuthContext';
import { useToast } from '../contexts/ToastContext';
import { api } from '../lib/api';
import { errorMessage, formatNumber, greetingDate, relativeTime, roleLabel } from '../lib/format';
import type { DashboardResponse, DashboardWidget } from '../lib/types';

function WidgetIcon({ id }: { id: string }) {
  if (id === 'core.users.recent') return <Time size={20} />;
  if (id === 'core.users') return <UserMultiple size={20} />;
  return <Checkmark size={20} />;
}

function UserCountWidget({ widget, onOpen }: { widget: DashboardWidget; onOpen: () => void }) {
  return (
    <ClickableTile className="widget-stat" onClick={onOpen}>
      <div className="widget-stat__top">
        <span className="widget-stat__label">{widget.title}</span>
        <UserMultiple size={20} />
      </div>
      <strong className="widget-stat__value">{formatNumber(widget.data?.total ?? 0)}</strong>
      <span className="widget-stat__hint">contas cadastradas</span>
      <span className="widget-stat__link">Ver usuários →</span>
    </ClickableTile>
  );
}

function RecentUsersWidget({ widget, onOpen }: { widget: DashboardWidget; onOpen: () => void }) {
  const users = widget.data?.users ?? [];
  return (
    <div className="widget-recent">
      <div className="widget-recent__header">
        <div>
          <h3>{widget.title}</h3>
          <p>{widget.description}</p>
        </div>
        <Button kind="ghost" size="sm" onClick={onOpen}>Ver todos</Button>
      </div>
      {users.length === 0 ? (
        <p className="muted">Nenhum usuário cadastrado.</p>
      ) : (
        <div className="table-scroll">
          <Table size="sm" aria-label="Usuários recentes">
            <TableHead>
              <TableRow>
                <TableHeader>Nome</TableHeader>
                <TableHeader className="hide-sm">E-mail</TableHeader>
                <TableHeader className="hide-md">Perfil</TableHeader>
                <TableHeader className="hide-lg">Criado em</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              {users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell>
                    <span className="cell-primary">{user.full_name}</span>
                    <span className="cell-secondary show-sm">{user.email}</span>
                  </TableCell>
                  <TableCell className="hide-sm">{user.email}</TableCell>
                  <TableCell className="hide-md">
                    <Tag type={user.is_admin ? 'purple' : 'gray'} size="sm">{roleLabel(user.is_admin)}</Tag>
                  </TableCell>
                  <TableCell className="hide-lg">{relativeTime(user.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

function WidgetBody({ widget, onOpen }: { widget: DashboardWidget; onOpen: () => void }) {
  switch (widget.id) {
    case 'work.center':
      return <WorkCenterWidget widget={widget} />;
    case 'core.users':
      return <UserCountWidget widget={widget} onOpen={onOpen} />;
    case 'core.users.recent':
      return <RecentUsersWidget widget={widget} onOpen={onOpen} />;
    case 'stations.sites':
      return <StationsWidget widget={widget} />;
    case 'follow-up.overdue':
      return <FollowUpsWidget widget={widget} />;
    case 'demands.mine':
      return <DemandsWidget widget={widget} />;
    case 'follow-up.charts':
      return <FollowUpsChartWidget widget={widget} />;
    case 'nokia.charts':
      return <NokiaWidget widget={widget} />;
    default:
      return <Tile className="widget-generic">{widget.description}</Tile>;
  }
}

function WorkCenterWidget({ widget }: { widget: DashboardWidget }) {
  const navigate = useNavigate();
  const data = (widget.data ?? {}) as {
    tasks?: Array<{ id: number; title: string; list_name: string; due_at: string | null }>;
    follow_ups?: Array<{ id: number; title: string; status: string; due_date: string | null; assignee_name: string | null; overdue: boolean }>;
    events?: Array<{ id: number; title: string; start_at: string; calendar_name: string }>;
    counts?: { tasks_open?: number; tasks_overdue?: number; follow_ups_open?: number; follow_ups_overdue?: number; events_today?: number; events_upcoming?: number };
  };
  const tasks = data.tasks ?? [];
  const followUps = data.follow_ups ?? [];
  const events = data.events ?? [];
  const counts = data.counts ?? {};
  const overdue = (counts.tasks_overdue ?? 0) + (counts.follow_ups_overdue ?? 0);
  return (
    <div className="widget-recent work-center">
      <div className="widget-recent__header">
        <div>
          <h3>{widget.title}</h3>
          <p>{widget.description}</p>
        </div>
        <div className="tag-row">
          <Tag type={overdue > 0 ? 'red' : 'green'} size="sm">{overdue > 0 ? `${overdue} atrasado(s)` : 'Em dia'}</Tag>
          <Tag type="blue" size="sm">{counts.events_today ?? 0} hoje</Tag>
        </div>
      </div>
      <div className="work-center__grid">
        <section>
          <header><strong>Follow-ups</strong><span>{counts.follow_ups_open ?? 0} abertos</span></header>
          {followUps.length === 0 ? <p className="muted">Nada atribuído a você.</p> : followUps.map((item) => (
            <button key={item.id} type="button" className="work-center__row" onClick={() => navigate(`/follow-ups?followup=${item.id}`)}>
              <strong>{item.title}</strong>
              <span>{item.assignee_name ?? 'Não atribuído'} · {item.due_date ?? 'sem prazo'}</span>
              {item.overdue && <Tag type="red" size="sm">Atrasado</Tag>}
            </button>
          ))}
          <Button kind="ghost" size="sm" onClick={() => navigate('/follow-ups')}>Abrir follow-ups</Button>
        </section>
        <section>
          <header><strong>Tarefas</strong><span>{counts.tasks_open ?? 0} pendentes</span></header>
          {tasks.length === 0 ? <p className="muted">Nenhuma tarefa com prazo próximo.</p> : tasks.map((item) => (
            <button key={item.id} type="button" className="work-center__row" onClick={() => navigate(`/tasks?task=${item.id}`)}>
              <strong>{item.title}</strong>
              <span>{item.list_name} · vence {item.due_at ? new Date(`${item.due_at}T12:00:00`).toLocaleDateString('pt-BR') : 'sem prazo'}</span>
            </button>
          ))}
          <Button kind="ghost" size="sm" onClick={() => navigate('/tasks')}>Abrir tarefas</Button>
        </section>
        <section>
          <header><strong>Agenda</strong><span>{counts.events_upcoming ?? 0} próximos</span></header>
          {events.length === 0 ? <p className="muted">Nenhum compromisso nos próximos dias.</p> : events.map((item) => (
            <button key={item.id} type="button" className="work-center__row" onClick={() => navigate(`/agenda?event=${item.id}`)}>
              <strong>{item.title}</strong>
              <span>{new Date(item.start_at).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })} · {item.calendar_name}</span>
            </button>
          ))}
          <Button kind="ghost" size="sm" onClick={() => navigate('/agenda')}>Abrir agenda</Button>
        </section>
      </div>
    </div>
  );
}

function StationsWidget({ widget }: { widget: DashboardWidget }) {
  const sites = ((widget.data as Record<string, unknown> | undefined)?.sites as Array<Record<string, unknown>> | undefined) ?? [];
  return (
    <div className="widget-recent">
      <div className="widget-recent__header">
        <div><h3>{widget.title}</h3><p>{widget.description}</p></div>
        <Tag type="blue" size="sm">{String((widget.data as Record<string, unknown> | undefined)?.total ?? 0)} sites</Tag>
      </div>
      {sites.length === 0 ? <p className="muted">Nenhuma estação cadastrada.</p> : sites.map((site) => (
        <div key={String(site.site_id)} className="widget-list-row">
          <strong>{String(site.site_id)}</strong>
          <span>{String(site.municipio ?? 'Município não informado')}{site.estado ? `/${String(site.estado)}` : ''}</span>
          <Tag type="gray" size="sm">{String(site.stations_count)} estação(ões)</Tag>
        </div>
      ))}
    </div>
  );
}

const FOLLOWUP_STATUS_LABELS: Record<string, string> = { pendente: 'Pendente', em_andamento: 'Em andamento', concluido: 'Concluído', cancelado: 'Cancelado' };
const FOLLOWUP_PRIORITY_LABELS: Record<string, string> = { baixa: 'Baixa', media: 'Média', alta: 'Alta', urgente: 'Urgente' };

function FollowUpsWidget({ widget }: { widget: DashboardWidget }) {
  const data = (widget.data as Record<string, unknown> | undefined) ?? {};
  const items = (data.items as Array<Record<string, unknown>> | undefined) ?? [];
  const overdue = Number(data.overdue_count ?? 0);
  return (
    <div className="widget-recent">
      <div className="widget-recent__header">
        <div><h3>Meus follow-ups</h3><p>{String(data.open_count ?? 0)} em aberto</p></div>
        <Tag type={overdue > 0 ? 'red' : 'green'} size="sm">{overdue > 0 ? `${overdue} atrasado(s)` : 'Em dia'}</Tag>
      </div>
      {items.length === 0 ? <p className="muted">Nenhum follow-up atribuído a você.</p> : items.map((item) => (
        <div key={String(item.id)} className="widget-list-row">
          <strong>{String(item.title)}</strong>
          <span>{String(item.assignee_name ?? 'Não atribuído')} · {String(item.due_date ?? 'sem prazo')}</span>
          <Tag type={item.priority === 'urgente' ? 'red' : item.priority === 'alta' ? 'magenta' : 'gray'} size="sm">{FOLLOWUP_PRIORITY_LABELS[String(item.priority)] ?? String(item.priority)}</Tag>
        </div>
      ))}
    </div>
  );
}

function DemandsWidget({ widget }: { widget: DashboardWidget }) {
  const navigate = useNavigate();
  const data = (widget.data as Record<string, unknown> | undefined) ?? {};
  const items = (data.items as Array<Record<string, unknown>> | undefined) ?? [];
  const stats = (data.stats as Record<string, unknown> | undefined) ?? {};
  const overdue = Number(stats.overdue ?? 0);
  return (
    <div className="widget-recent">
      <div className="widget-recent__header">
        <div><h3>Minhas demandas</h3><p>{String(stats.mine ?? 0)} atribuída(s) em aberto</p></div>
        <Tag type={overdue > 0 ? 'red' : 'green'} size="sm">{overdue > 0 ? `${overdue} atrasada(s)` : 'Em dia'}</Tag>
      </div>
      {items.length === 0 ? <p className="muted">Nenhuma demanda atribuída a você.</p> : items.map((item) => (
        <button key={String(item.id)} type="button" className="work-center__row" onClick={() => navigate(`/demands/${item.id}`)}>
          <strong>{String(item.title)}</strong>
          <span>{String(item.requester_name ?? 'Solicitante')} · prazo {String(item.due_at ?? 'sem prazo')}</span>
          {item.overdue === true && <Tag type="red" size="sm">Atrasada</Tag>}
        </button>
      ))}
      <Button kind="ghost" size="sm" onClick={() => navigate('/demands?scope=mine')}>Abrir demandas</Button>
    </div>
  );
}

function FollowUpsChartWidget({ widget }: { widget: DashboardWidget }) {  const data = (widget.data as Record<string, unknown> | undefined) ?? {};
  const statuses = (data.statuses as Array<Record<string, unknown>> | undefined) ?? [];
  return (
    <div className="widget-recent">
      <div className="widget-recent__header"><div><h3>Painel de follow-ups</h3><p>{widget.description}</p></div></div>
      <div className="mini-stats">
        <div><strong>{String(data.total ?? 0)}</strong><span>Total</span></div>
        <div><strong>{String(data.open ?? 0)}</strong><span>Em aberto</span></div>
        <div><strong>{String(data.overdue ?? 0)}</strong><span>Atrasados</span></div>
        <div><strong>{String(data.completed ?? 0)}</strong><span>Concluídos</span></div>
      </div>
      <div className="bar-list">
        {statuses.map((status) => (
          <div key={String(status.status)} className="bar-row">
            <span>{FOLLOWUP_STATUS_LABELS[String(status.status)] ?? String(status.status)}</span>
            <div><i style={{ width: `${Math.min(100, Number(status.count ?? 0) * 10)}%` }} /></div>
            <strong>{String(status.count)}</strong>
          </div>
        ))}
      </div>
    </div>
  );
}

function NokiaWidget({ widget }: { widget: DashboardWidget }) {
  const data = (widget.data as Record<string, unknown> | undefined) ?? {};
  const statuses = (data.statuses as Array<Record<string, unknown>> | undefined) ?? [];
  const regional = (data.regional as Array<Record<string, unknown>> | undefined) ?? [];
  const pending = (data.pending_items as Array<Record<string, unknown>> | undefined) ?? [];
  return (
    <div className="widget-recent">
      <div className="widget-recent__header">
        <div><h3>Projetos Nokia</h3><p>{widget.description}</p></div>
        <Tag type={Number(data.acceptance ?? 0) >= 80 ? 'green' : 'red'} size="sm">Aceite {Number(data.acceptance ?? 0).toFixed(1)}%</Tag>
      </div>
      <div className="mini-stats">
        <div><strong>{String(data.integrated ?? data.total ?? 0)}</strong><span>Sites integrados</span></div>
        <div><strong>{String(data.accepted ?? 0)}</strong><span>Aceitos</span></div>
        <div><strong>{String(data.pending ?? 0)}</strong><span>Pendências</span></div>
        <div><strong>{Number(data.projected ?? 0).toFixed(1)}%</strong><span>Projeção</span></div>
      </div>
      {statuses.length > 0 && (
        <div className="bar-list">
          {statuses.map((status) => (
            <div key={String(status.key)} className="bar-row">
              <span>{String(status.label)}</span>
              <div><i style={{ width: `${Math.min(100, Number(status.percent ?? 0))}%`, background: String(status.color ?? '#4589ff') }} /></div>
              <strong>{String(status.count)}</strong>
            </div>
          ))}
        </div>
      )}
      {regional.length > 0 && (
        <div className="nokia-widget-columns">
          <div>
            <p className="field-label">Regionais</p>
            {regional.slice(0, 5).map((item) => (
              <div key={String(item.name)} className="widget-list-row">
                <span>{String(item.name)}</span>
                <strong>{Number(item.percent ?? 0).toFixed(1)}%</strong>
              </div>
            ))}
          </div>
          <div>
            <p className="field-label">Pendências recentes</p>
            {pending.slice(0, 5).map((item, index) => (
              <div key={`${String(item.site_id)}-${index}`} className="widget-list-row">
                <span>{String(item.site_id)} · {String(item.description)}</span>
                <Tag type="gray" size="sm">{String(item.kind_label)}</Tag>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export function DashboardPage() {
  const { user } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [data, setData] = useState<DashboardResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [dragId, setDragId] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerDragId, setDrawerDragId] = useState<string | null>(null);
  const [dropBeforeId, setDropBeforeId] = useState<string | null | undefined>(undefined);

  async function load() {
    try {
      const response = await api.get<{ data: DashboardResponse }>('/dashboard');
      setData(response.data);
    } catch (error) {
      toast.error('Não foi possível carregar o dashboard', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (loading || !data) return <Loading description="Carregando dashboard…" />;

  const hiddenSet = new Set(data.hidden);
  const visible = data.widgets.filter((widget) => !hiddenSet.has(widget.id));
  const hidden = data.widgets.filter((widget) => hiddenSet.has(widget.id));

  async function setVisibility(widget: DashboardWidget, hiddenValue: boolean) {
    setBusy(true);
    try {
      await api.put(`/dashboard/widgets/${encodeURIComponent(widget.id)}`, { hidden: hiddenValue });
      setData((current) => {
        if (!current) return current;
        const hiddenIds = new Set(current.hidden);
        if (hiddenValue) hiddenIds.add(widget.id);
        else hiddenIds.delete(widget.id);
        return { ...current, hidden: [...hiddenIds] };
      });
      toast.success(hiddenValue ? 'Widget removido do painel' : 'Widget adicionado ao painel');
    } catch (error) {
      toast.error('Não foi possível atualizar o widget', errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function setSpan(widget: DashboardWidget, span: number) {
    setBusy(true);
    try {
      await api.put(`/dashboard/widgets/${encodeURIComponent(widget.id)}/layout`, { span });
      setData((current) => {
        if (!current) return current;
        const spans = { ...current.spans };
        if (span === widget.default_span) delete spans[widget.id];
        else spans[widget.id] = span;
        return { ...current, spans };
      });
      toast.success('Largura do widget atualizada');
    } catch (error) {
      toast.error('Não foi possível redimensionar', errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function placeWidget(sourceId: string, beforeId: string | null, fromDrawer: boolean) {
    if (!data || beforeId === sourceId) return;
    setBusy(true);
    const nextHidden = data.hidden.filter((id) => id !== sourceId);
    const nextHiddenSet = new Set(nextHidden);
    const visibleIds = data.widgets
      .filter((widget) => widget.id === sourceId || !hiddenSet.has(widget.id))
      .map((widget) => widget.id)
      .filter((id) => id !== sourceId);
    const targetIndex = beforeId ? visibleIds.indexOf(beforeId) : visibleIds.length;
    visibleIds.splice(targetIndex < 0 ? visibleIds.length : targetIndex, 0, sourceId);
    const hiddenIds = data.widgets
      .filter((widget) => widget.id !== sourceId && nextHiddenSet.has(widget.id))
      .map((widget) => widget.id);
    const order = [...visibleIds, ...hiddenIds];

    try {
      if (fromDrawer) {
        await api.put(`/dashboard/widgets/${encodeURIComponent(sourceId)}`, { hidden: false });
      }
      await api.put('/dashboard/widgets/order', { order });
      const byId = new Map(data.widgets.map((widget) => [widget.id, widget]));
      setData({
        ...data,
        hidden: nextHidden,
        order,
        widgets: order.map((id) => byId.get(id)).filter((widget): widget is DashboardWidget => Boolean(widget)),
      });
      toast.success(fromDrawer ? 'Widget adicionado ao painel' : 'Ordem dos widgets atualizada');
    } catch (error) {
      toast.error('Não foi possível posicionar o widget', errorMessage(error));
    } finally {
      setBusy(false);
      setDragId(null);
      setDrawerDragId(null);
      setDropBeforeId(undefined);
    }
  }

  function handleDropAt(beforeId: string | null, event: React.DragEvent) {
    event.preventDefault();
    event.stopPropagation();
    const sourceId = event.dataTransfer.getData('text/plain') || drawerDragId || dragId;
    if (!sourceId) return;
    const fromDrawer = hiddenSet.has(sourceId);
    void placeWidget(sourceId, beforeId, fromDrawer);
  }

  return (
    <div className={`page ${editing && drawerOpen ? 'page--drawer-open' : ''}`}>
      <PageHeader
        eyebrow="Visão geral"
        title={data.greeting || `Bem-vindo, ${user?.name}!`}
        description={<>{greetingDate(data.date)} · Acompanhe os indicadores e personalize seu painel.</>}
        actions={
          <Button
            kind={editing ? 'primary' : 'tertiary'}
            size="md"
            renderIcon={editing ? Checkmark : Edit}
            onClick={() => {
              if (editing) {
                setEditing(false);
                setDrawerOpen(false);
              } else {
                setEditing(true);
                setDrawerOpen(true);
              }
            }}
          >
            {editing ? 'Concluir edição' : 'Editar widgets'}
          </Button>
        }
      />

      {editing && (
        <div className="edit-banner" role="status">
          <Draggable size={20} />
          <div>
            <strong>Modo edição</strong>
            <p>Arraste widgets da barra lateral para qualquer posição e use os controles para redimensionar ou remover.</p>
          </div>
          {busy && <InlineLoading description="Salvando…" />}
          {!drawerOpen && (
            <Button kind="ghost" size="sm" renderIcon={Add} onClick={() => setDrawerOpen(true)}>
              Widgets disponíveis
            </Button>
          )}
        </div>
      )}

      {visible.length === 0 && !editing ? (
        <EmptyState
          title="Nenhum widget visível"
          description="Ative o modo edição para adicionar widgets ao painel."
        />
      ) : (
        <div className={`widget-grid ${editing ? 'widget-grid--editing' : ''}`}>
          {editing && visible.length === 0 && (
            <div
              className={`widget-empty-drop ${dropBeforeId === null ? 'is-over' : ''}`}
              onDragOver={(event) => {
                event.preventDefault();
                setDropBeforeId(null);
              }}
              onDragLeave={() => setDropBeforeId(undefined)}
              onDrop={(event) => handleDropAt(null, event)}
            >
              <Draggable size={24} />
              <strong>Arraste widgets para cá</strong>
              <span>Solte um widget da barra lateral para começar seu painel.</span>
            </div>
          )}
          {visible.map((widget) => {
            const span = data.spans[widget.id] ?? widget.default_span;
            return (
              <Fragment key={widget.id}>
                {editing && (
                  <div
                    className={`widget-drop-zone ${dropBeforeId === widget.id ? 'is-over' : ''}`}
                    onDragOver={(event) => {
                      event.preventDefault();
                      setDropBeforeId(widget.id);
                    }}
                    onDragLeave={() => setDropBeforeId(undefined)}
                    onDrop={(event) => handleDropAt(widget.id, event)}
                  >
                    <span>Solte aqui</span>
                  </div>
                )}
                <section
                  className={`widget-card widget-card--span-${Math.max(1, Math.min(3, span))} ${editing ? 'widget-card--editing' : ''} ${dragId === widget.id ? 'widget-card--dragging' : ''}`}
                  style={{ ['--widget-span' as string]: Math.max(1, Math.min(3, span)) }}
                  draggable={editing}
                  onDragStart={(event) => {
                    event.dataTransfer.setData('text/plain', widget.id);
                    event.dataTransfer.effectAllowed = 'move';
                    setDragId(widget.id);
                  }}
                  onDragEnd={() => {
                    setDragId(null);
                    setDrawerDragId(null);
                    setDropBeforeId(undefined);
                  }}
                  onDragOver={(event) => {
                    if (editing) {
                      event.preventDefault();
                      setDropBeforeId(widget.id);
                    }
                  }}
                  onDrop={(event) => handleDropAt(widget.id, event)}
                >
                  {editing && (
                    <div className="widget-toolbar">
                      <div className="widget-toolbar__left">
                        <Draggable size={16} />
                        <span>Arraste para reordenar</span>
                      </div>
                      <div className="widget-toolbar__actions">
                        {[1, 2, 3].map((columns) => (
                          <Button
                            key={columns}
                            size="sm"
                            kind={columns === span ? 'primary' : 'ghost'}
                            onClick={() => void setSpan(widget, columns)}
                            disabled={busy}
                          >
                            {columns} col{columns > 1 ? 's' : ''}
                          </Button>
                        ))}
                        <Button
                          size="sm"
                          kind="danger--ghost"
                          renderIcon={TrashCan}
                          onClick={() => void setVisibility(widget, true)}
                          disabled={busy}
                        >
                          Remover
                        </Button>
                      </div>
                    </div>
                  )}
                  <div className="widget-card__content">
                    <WidgetBody widget={widget} onOpen={() => navigate('/users')} />
                  </div>
                </section>
              </Fragment>
            );
          })}
          {editing && (
            <div
              className={`widget-drop-zone widget-drop-zone--end ${dropBeforeId === null ? 'is-over' : ''}`}
              onDragOver={(event) => {
                event.preventDefault();
                setDropBeforeId(null);
              }}
              onDragLeave={() => setDropBeforeId(undefined)}
              onDrop={(event) => handleDropAt(null, event)}
            >
              <span>Solte aqui no final</span>
            </div>
          )}
        </div>
      )}

      {editing && drawerOpen && (
        <aside className="widget-drawer" aria-label="Widgets disponíveis">
          <header className="widget-drawer__header">
            <div>
              <p className="widget-drawer__eyebrow">Personalizar painel</p>
              <h2>Widgets disponíveis</h2>
            </div>
            <Button
              kind="ghost"
              size="sm"
              hasIconOnly
              renderIcon={Close}
              iconDescription="Fechar barra de widgets"
              onClick={() => setDrawerOpen(false)}
            />
          </header>
          <p className="widget-drawer__hint">
            Arraste um widget e solte entre os cards do painel. Você também pode usar o botão + para adicioná-lo ao final.
          </p>
          <div className="widget-drawer__list">
            {hidden.length === 0 ? (
              <div className="widget-drawer__empty">
                <Checkmark size={24} />
                <strong>Todos os widgets estão no painel</strong>
                <span>Remova um widget do painel para ele aparecer aqui novamente.</span>
              </div>
            ) : (
              hidden.map((widget) => (
                <div
                  key={widget.id}
                  className={`widget-drawer__item ${drawerDragId === widget.id ? 'is-dragging' : ''}`}
                  draggable
                  onDragStart={(event) => {
                    event.dataTransfer.setData('text/plain', widget.id);
                    event.dataTransfer.effectAllowed = 'copy';
                    setDrawerDragId(widget.id);
                  }}
                  onDragEnd={() => setDrawerDragId(null)}
                >
                  <div className="widget-drawer__icon"><WidgetIcon id={widget.id} /></div>
                  <div className="widget-drawer__text">
                    <strong>{widget.title}</strong>
                    <span>{widget.description}</span>
                  </div>
                  <Button
                    size="sm"
                    kind="ghost"
                    hasIconOnly
                    renderIcon={Add}
                    iconDescription={`Adicionar ${widget.title}`}
                    onClick={() => void placeWidget(widget.id, null, true)}
                    disabled={busy}
                  />
                </div>
              ))
            )}
          </div>
          <footer className="widget-drawer__footer">
            <Draggable size={16} />
            <span>{hidden.length} widget(s) disponível(is)</span>
          </footer>
        </aside>
      )}
    </div>
  );
}
