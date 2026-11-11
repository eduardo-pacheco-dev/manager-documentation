import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Add,
  ArrowDown,
  ArrowUp,
  Checkmark,
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
    case 'core.users':
      return <UserCountWidget widget={widget} onOpen={onOpen} />;
    case 'core.users.recent':
      return <RecentUsersWidget widget={widget} onOpen={onOpen} />;
    default:
      return <Tile className="widget-generic">{widget.description}</Tile>;
  }
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
  const visible = editing ? data.widgets : data.widgets.filter((widget) => !hiddenSet.has(widget.id));
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

  async function persistOrder(order: string[]) {
    setBusy(true);
    try {
      await api.put('/dashboard/widgets/order', { order });
      setData((current) => (current ? { ...current, order } : current));
      toast.success('Ordem dos widgets atualizada');
    } catch (error) {
      toast.error('Não foi possível salvar a ordem', errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  function handleDrop(targetId: string) {
    if (!dragId || dragId === targetId) return;
    const ids = data!.widgets.map((widget) => widget.id);
    const from = ids.indexOf(dragId);
    const to = ids.indexOf(targetId);
    if (from < 0 || to < 0) return;
    ids.splice(to, 0, ids.splice(from, 1)[0]);
    const reordered = ids
      .map((id) => data!.widgets.find((widget) => widget.id === id))
      .filter((widget): widget is DashboardWidget => Boolean(widget));
    setData({ ...data!, widgets: reordered, order: ids });
    setDragId(null);
    void persistOrder(ids);
  }

  return (
    <div className="page">
      <PageHeader
        eyebrow="Visão geral"
        title={data.greeting || `Bem-vindo, ${user?.name}!`}
        description={<>{greetingDate(data.date)} · Acompanhe os indicadores e personalize seu painel.</>}
        actions={
          <Button
            kind={editing ? 'primary' : 'tertiary'}
            size="md"
            renderIcon={editing ? Checkmark : Edit}
            onClick={() => setEditing((value) => !value)}
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
            <p>Arraste para reordenar, redimensione, remova widgets ou adicione de volta os que estão ocultos.</p>
          </div>
          {busy && <InlineLoading description="Salvando…" />}
        </div>
      )}

      {visible.length === 0 ? (
        <EmptyState
          title="Nenhum widget visível"
          description={editing ? 'Adicione widgets ocultos abaixo.' : 'Ative o modo edição para adicionar widgets ao painel.'}
        />
      ) : (
        <div className="widget-grid">
          {visible.map((widget) => {
            const span = data.spans[widget.id] ?? widget.default_span;
            return (
              <section
                key={widget.id}
                className={`widget-card widget-card--span-${Math.max(1, Math.min(3, span))} ${editing ? 'widget-card--editing' : ''} ${dragId === widget.id ? 'widget-card--dragging' : ''}`}
                style={{ ['--widget-span' as string]: Math.max(1, Math.min(3, span)) }}
                draggable={editing}
                onDragStart={() => setDragId(widget.id)}
                onDragEnd={() => setDragId(null)}
                onDragOver={(event) => {
                  if (editing) event.preventDefault();
                }}
                onDrop={() => handleDrop(widget.id)}
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
            );
          })}
        </div>
      )}

      {editing && (
        <section className="hidden-widgets">
          <div className="hidden-widgets__header">
            <div>
              <h2>Widgets ocultos</h2>
              <p>{hidden.length} widget(s) fora do painel.</p>
            </div>
          </div>
          {hidden.length === 0 ? (
            <p className="muted">Todos os widgets disponíveis já estão visíveis.</p>
          ) : (
            <div className="hidden-widgets__list">
              {hidden.map((widget) => (
                <div key={widget.id} className="hidden-widget">
                  <div className="hidden-widget__icon"><WidgetIcon id={widget.id} /></div>
                  <div className="hidden-widget__text">
                    <strong>{widget.title}</strong>
                    <span>{widget.description}</span>
                  </div>
                  <Button
                    size="sm"
                    kind="ghost"
                    renderIcon={Add}
                    onClick={() => void setVisibility(widget, false)}
                    disabled={busy}
                  >
                    Adicionar
                  </Button>
                </div>
              ))}
            </div>
          )}
        </section>
      )}
    </div>
  );
}
