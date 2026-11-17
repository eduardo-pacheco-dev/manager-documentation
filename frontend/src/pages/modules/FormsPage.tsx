import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Add,
  Checkmark,
  Copy,
  Edit,
  Grid,
  List,
  OverflowMenuVertical,
  Settings,
  Table as TableIcon,
  TrashCan,
} from '@carbon/icons-react';
import {
  Button,
  InlineLoading,
  OverflowMenu,
  OverflowMenuItem,
  Pagination,
  Search,
  Select,
  SelectItem,
  Tag,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState } from '../../components/Feedback';
import { ConfirmModal } from '../../components/ConfirmModal';
import { FormModal } from '../../components/FormModal';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import { formStatusTag, publicFormLink } from '../../lib/forms';
import type { FormRecord, FormsListResponse } from '../../lib/types';

type ViewMode = 'table' | 'cards' | 'compact';

export function FormsPage() {
  const toast = useToast();
  const navigate = useNavigate();
  const [forms, setForms] = useState<FormRecord[]>([]);
  const [response, setResponse] = useState<FormsListResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [view, setView] = useState<ViewMode>('table');
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(12);
  const [sort, setSort] = useState('updated_at');
  const [direction, setDirection] = useState<'asc' | 'desc'>('desc');
  const [createOpen, setCreateOpen] = useState(false);
  const [deleting, setDeleting] = useState<FormRecord | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);
  const preferencesApplied = useRef(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: String(page),
        per_page: String(perPage),
        sort,
        direction,
      });
      if (search.trim()) params.set('search', search.trim());
      if (statusFilter) params.set('status', statusFilter);
      const data = await api.get<FormsListResponse>(`/forms?${params.toString()}`);
      if (data.meta && page > data.meta.last_page) {
        setPage(data.meta.last_page);
        return;
      }
      setForms(data.data ?? []);
      setResponse(data);
      if (!preferencesApplied.current && data.settings) {
        preferencesApplied.current = true;
        setView(data.settings.default_view);
        if (data.settings.per_page && data.settings.per_page !== perPage) {
          setPerPage(data.settings.per_page);
        }
      }
    } catch (error) {
      toast.error('Não foi possível carregar os formulários', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [page, perPage, sort, direction, search, statusFilter, toast]);

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 200);
    return () => window.clearTimeout(timer);
  }, [load]);

  function toggleSort(column: string) {
    if (sort === column) {
      setDirection((current) => (current === 'asc' ? 'desc' : 'asc'));
    } else {
      setSort(column);
      setDirection(column === 'updated_at' ? 'desc' : 'asc');
    }
    setPage(1);
  }

  function sortLabel(column: string, label: string) {
    if (sort !== column) return label;
    return `${label} ${direction === 'asc' ? '↑' : '↓'}`;
  }

  async function remove() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await api.delete(`/forms/${deleting.id}`);
      toast.success('Formulário excluído');
      setDeleting(null);
      await load();
    } catch (error) {
      toast.error('Não foi possível excluir o formulário', errorMessage(error));
    } finally {
      setDeleteBusy(false);
    }
  }

  async function copyLink(form: FormRecord) {
    try {
      await navigator.clipboard.writeText(publicFormLink(form.public_token));
      toast.success('Link público copiado', 'Compartilhe o link com quem deve preencher o formulário.');
    } catch {
      toast.error('Não foi possível copiar o link', publicFormLink(form.public_token));
    }
  }

  const stats = response?.stats;
  const items = forms;

  return (
    <div className="page">
      <PageHeader
        eyebrow="Documentos"
        title="Formulários"
        description="Formulários dinâmicos com seções, perguntas, respostas, avaliações e preenchimento por link público."
        actions={
          <>
            <Button kind="ghost" renderIcon={Settings} onClick={() => navigate('/forms/settings')}>
              Configurações
            </Button>
            <Button renderIcon={Add} onClick={() => setCreateOpen(true)}>
              Novo formulário
            </Button>
          </>
        }
      />

      {stats && (
        <div className="stats-grid stats-grid--3">
          <Tile className="stat-card stat-card--blue">
            <span className="stat-card__label">Formulários</span>
            <strong className="stat-card__value">{stats.total}</strong>
          </Tile>
          <Tile className="stat-card stat-card--purple">
            <span className="stat-card__label">Pendentes / em preenchimento</span>
            <strong className="stat-card__value">{stats.pending + stats.in_progress}</strong>
          </Tile>
          <Tile className="stat-card stat-card--green">
            <span className="stat-card__label">Preenchidos / aprovados</span>
            <strong className="stat-card__value">{stats.filled + stats.approved}</strong>
          </Tile>
        </div>
      )}

      <Tile className="module-main__header">
        <div className="module-toolbar">
          <Search
            size="lg"
            labelText="Buscar formulários"
            placeholder="Buscar por código ou nome"
            value={search}
            onChange={(event) => {
              setSearch(event.target.value);
              setPage(1);
            }}
            onClear={() => {
              setSearch('');
              setPage(1);
            }}
          />
          <Select
            id="forms-status-filter"
            labelText="Status"
            size="sm"
            value={statusFilter}
            onChange={(event) => {
              setStatusFilter(event.target.value);
              setPage(1);
            }}
          >
            <SelectItem value="" text="Todos os status" />
            {(response?.statuses ?? []).map((status) => (
              <SelectItem key={status} value={status} text={status} />
            ))}
          </Select>
          <div className="toolbar__views" role="group" aria-label="Modo de visualização">
            <Button
              kind={view === 'table' ? 'primary' : 'ghost'}
              size="sm"
              hasIconOnly
              renderIcon={TableIcon}
              iconDescription="Tabela"
              onClick={() => setView('table')}
            />
            <Button
              kind={view === 'cards' ? 'primary' : 'ghost'}
              size="sm"
              hasIconOnly
              renderIcon={Grid}
              iconDescription="Cards"
              onClick={() => setView('cards')}
            />
            <Button
              kind={view === 'compact' ? 'primary' : 'ghost'}
              size="sm"
              hasIconOnly
              renderIcon={List}
              iconDescription="Lista"
              onClick={() => setView('compact')}
            />
          </div>
        </div>
      </Tile>

      {loading ? (
        <InlineLoading description="Carregando formulários…" />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Add}
          title={search || statusFilter ? 'Nenhum formulário encontrado' : 'Nenhum formulário cadastrado'}
          description={
            search || statusFilter
              ? 'Ajuste a busca ou os filtros para ver mais resultados.'
              : 'Crie o primeiro formulário para começar.'
          }
          action={
            <Button renderIcon={Add} onClick={() => setCreateOpen(true)}>
              Novo formulário
            </Button>
          }
        />
      ) : view === 'table' ? (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table" aria-label="Lista de formulários">
              <thead>
                <tr>
                  <th>
                    <button type="button" className="sort-button" onClick={() => toggleSort('code')}>
                      {sortLabel('code', 'Código')}
                    </button>
                  </th>
                  <th>
                    <button type="button" className="sort-button" onClick={() => toggleSort('name')}>
                      {sortLabel('name', 'Formulário')}
                    </button>
                  </th>
                  <th className="hide-md">Projeto</th>
                  <th className="hide-sm">
                    <button type="button" className="sort-button" onClick={() => toggleSort('type')}>
                      {sortLabel('type', 'Tipo')}
                    </button>
                  </th>
                  <th>
                    <button type="button" className="sort-button" onClick={() => toggleSort('status')}>
                      {sortLabel('status', 'Status')}
                    </button>
                  </th>
                  <th className="hide-sm">
                    <button type="button" className="sort-button" onClick={() => toggleSort('updated_at')}>
                      {sortLabel('updated_at', 'Atualizado')}
                    </button>
                  </th>
                  <th className="cell-actions">Ações</th>
                </tr>
              </thead>
              <tbody>
                {items.map((form) => (
                  <tr key={form.id}>
                    <td>
                      <button type="button" className="link-button" onClick={() => navigate(`/forms/${form.id}`)}>
                        {form.code}
                      </button>
                      <span className="cell-secondary">{form.response_count} resposta(s)</span>
                    </td>
                    <td>
                      <span className="cell-primary">{form.name}</span>
                      <span className="cell-secondary">{form.evaluation_count} avaliação(ões)</span>
                    </td>
                    <td className="hide-md">
                      {form.project_name ? `[${form.project_code}] ${form.project_name}` : '—'}
                    </td>
                    <td className="hide-sm">{form.type}</td>
                    <td>
                      <Tag type={formStatusTag(form.status)} size="sm">
                        {form.status}
                      </Tag>
                    </td>
                    <td className="hide-sm">{relativeTime(form.updated_at)}</td>
                    <td className="cell-actions">
                      <OverflowMenu
                        aria-label={`Ações de ${form.name}`}
                        iconDescription={`Ações de ${form.name}`}
                        renderIcon={OverflowMenuVertical}
                        size="sm"
                        flipped
                      >
                        <OverflowMenuItem itemText="Visualizar respostas" onClick={() => navigate(`/forms/${form.id}`)} />
                        <OverflowMenuItem itemText="Editar" onClick={() => navigate(`/forms/${form.id}/edit`)} />
                        <OverflowMenuItem itemText="Avaliar resposta" onClick={() => navigate(`/forms/${form.id}/evaluate`)} />
                        <OverflowMenuItem itemText="Copiar link público" onClick={() => void copyLink(form)} />
                        <OverflowMenuItem itemText="Excluir" isDelete hasDivider onClick={() => setDeleting(form)} />
                      </OverflowMenu>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : view === 'cards' ? (
        <div className="user-cards">
          {items.map((form) => (
            <article key={form.id} className="user-card">
              <div className="user-card__main">
                <div className="user-card__text">
                  <button type="button" className="link-button" onClick={() => navigate(`/forms/${form.id}`)}>
                    {form.name}
                  </button>
                  <span>{form.code} · {form.type}</span>
                  <div className="tag-row">
                    <Tag type={formStatusTag(form.status)} size="sm">{form.status}</Tag>
                    {form.allows_multiple_responses && <Tag type="blue" size="sm">Múltiplas respostas</Tag>}
                  </div>
                </div>
              </div>
              <p className="user-card__meta">
                {form.project_name ? `[${form.project_code}] ${form.project_name}` : 'Sem projeto'} ·{' '}
                {form.response_count} resposta(s) · {relativeTime(form.updated_at)}
              </p>
              <div className="user-card__actions">
                <Button kind="ghost" size="sm" renderIcon={Edit} onClick={() => navigate(`/forms/${form.id}/edit`)}>Editar</Button>
                <Button kind="ghost" size="sm" renderIcon={Checkmark} onClick={() => navigate(`/forms/${form.id}/evaluate`)}>Avaliar</Button>
                <Button kind="ghost" size="sm" hasIconOnly renderIcon={Copy} iconDescription="Copiar link público" onClick={() => void copyLink(form)} />
                <Button kind="danger--ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={() => setDeleting(form)} />
              </div>
            </article>
          ))}
        </div>
      ) : (
        <div className="user-compact">
          {items.map((form) => (
            <article key={form.id} className="user-card">
              <div className="user-card__text">
                <button type="button" className="link-button" onClick={() => navigate(`/forms/${form.id}`)}>
                  {form.code} · {form.name}
                </button>
                <span>{form.type} · {form.response_count} resposta(s)</span>
              </div>
              <Tag type={formStatusTag(form.status)} size="sm">{form.status}</Tag>
              <p className="user-card__meta">{relativeTime(form.updated_at)}</p>
              <div className="user-card__actions">
                <Button kind="ghost" size="sm" hasIconOnly renderIcon={Edit} iconDescription="Editar" onClick={() => navigate(`/forms/${form.id}/edit`)} />
                <Button kind="ghost" size="sm" hasIconOnly renderIcon={Copy} iconDescription="Copiar link público" onClick={() => void copyLink(form)} />
              </div>
            </article>
          ))}
        </div>
      )}

      {response && response.meta.total > 0 && (
        <Pagination
          page={page}
          pageSize={perPage}
          pageSizes={[12, 24, 36]}
          totalItems={response.meta.total}
          onChange={({ page: nextPage, pageSize }) => {
            if (pageSize && pageSize !== perPage) {
              setPerPage(pageSize);
              setPage(1);
            } else {
              setPage(nextPage);
            }
          }}
          itemRangeText={(min, max, total) => `${min}–${max} de ${total} formulários`}
          itemsPerPageText="Itens por página"
          pageRangeText={(_current, total) => `de ${total} página(s)`}
          pageSelectLabelText={(total) => `Página de ${total}`}
          pageText={(current) => `Página ${current}`}
          backwardText="Página anterior"
          forwardText="Próxima página"
        />
      )}

      <FormModal
        open={createOpen}
        types={response?.types ?? []}
        statuses={response?.statuses ?? []}
        templates={response?.templates ?? []}
        settings={response?.settings ?? null}
        onClose={() => setCreateOpen(false)}
        onSaved={(id) => {
          setCreateOpen(false);
          navigate(`/forms/${id}/edit`);
        }}
      />

      <ConfirmModal
        open={deleting !== null}
        title="Excluir formulário"
        message={deleting ? `Excluir o formulário ${deleting.code} (${deleting.name})? As respostas e avaliações também deixarão de aparecer.` : ''}
        confirmLabel="Excluir"
        loading={deleteBusy}
        onCancel={() => setDeleting(null)}
        onConfirm={() => void remove()}
      />
    </div>
  );
}
