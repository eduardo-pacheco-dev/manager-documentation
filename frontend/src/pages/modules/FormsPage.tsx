import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Add, DocumentTasks, Settings } from '@carbon/icons-react';
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

  function actions(form: FormRecord) {
    return (
      <OverflowMenu
        aria-label={`Ações de ${form.name}`}
        iconDescription={`Ações de ${form.name}`}
        size="sm"
        flipped
      >
        <OverflowMenuItem itemText="Visualizar respostas" onClick={() => navigate(`/forms/${form.id}`)} />
        <OverflowMenuItem itemText="Editar" onClick={() => navigate(`/forms/${form.id}/edit`)} />
        <OverflowMenuItem itemText="Avaliar resposta" onClick={() => navigate(`/forms/${form.id}/evaluate`)} />
        <OverflowMenuItem itemText="Copiar link público" onClick={() => void copyLink(form)} />
        <OverflowMenuItem itemText="Excluir" isDelete hasDivider onClick={() => setDeleting(form)} />
      </OverflowMenu>
    );
  }

  return (
    <div className="page">
      <PageHeader
        eyebrow="Formulários"
        title="Formulários"
        description="Crie formulários dinâmicos com seções, perguntas, respostas, avaliações e link público."
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
        <div className="stats-grid stats-grid--3 forms-stats">
          <div className="stat-card stat-card--blue">
            <span className="stat-card__label">Total</span>
            <strong className="stat-card__value">{stats.total}</strong>
          </div>
          <div className="stat-card stat-card--purple">
            <span className="stat-card__label">Pendentes / em preenchimento</span>
            <strong className="stat-card__value">{stats.pending + stats.in_progress}</strong>
          </div>
          <div className="stat-card stat-card--green">
            <span className="stat-card__label">Preenchidos / aprovados</span>
            <strong className="stat-card__value">{stats.filled + stats.approved}</strong>
          </div>
        </div>
      )}

      <section className="module-toolbar forms-toolbar">
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
        <div className="form-view-toggle" role="group" aria-label="Modo de visualização">
          <button type="button" className={view === 'table' ? 'is-active' : ''} onClick={() => setView('table')}>
            Tabela
          </button>
          <button type="button" className={view === 'cards' ? 'is-active' : ''} onClick={() => setView('cards')}>
            Cards
          </button>
          <button type="button" className={view === 'compact' ? 'is-active' : ''} onClick={() => setView('compact')}>
            Lista
          </button>
        </div>
      </section>

      {loading ? (
        <InlineLoading description="Carregando formulários…" />
      ) : forms.length === 0 ? (
        <EmptyState
          title={search || statusFilter ? 'Nenhum formulário encontrado' : 'Nenhum formulário cadastrado'}
          description={
            search || statusFilter
              ? 'Ajuste a busca ou os filtros para ver mais resultados.'
              : 'Crie o primeiro formulário para começar.'
          }
          action={<Button renderIcon={Add} onClick={() => setCreateOpen(true)}>Novo formulário</Button>}
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
                      {sortLabel('name', 'Nome')}
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
                {forms.map((form) => (
                  <tr key={form.id}>
                    <td>
                      <Link className="cell-primary" to={`/forms/${form.id}`}>
                        {form.code}
                      </Link>
                    </td>
                    <td>
                      <span className="cell-primary">{form.name}</span>
                      <span className="cell-secondary">{form.response_count} resposta(s) · {form.evaluation_count} avaliação(ões)</span>
                    </td>
                    <td className="hide-md">{form.project_name ? `[${form.project_code}] ${form.project_name}` : '—'}</td>
                    <td className="hide-sm">{form.type}</td>
                    <td>
                      <Tag type={formStatusTag(form.status)} size="sm">
                        {form.status}
                      </Tag>
                    </td>
                    <td className="hide-sm">{relativeTime(form.updated_at)}</td>
                    <td className="cell-actions">{actions(form)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {response && response.meta.total > 0 && (
            <div className="table-footer">
              <span className="table-footer__order">{response.meta.total} formulário(s)</span>
              <Pagination
                page={response.meta.current_page}
                pageSize={response.meta.per_page}
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
                size="sm"
              />
            </div>
          )}
        </div>
      ) : view === 'cards' ? (
        <div className="forms-cards">
          {forms.map((form) => (
            <article className="forms-card" key={form.id}>
              <div className="forms-card__head">
                <span className="forms-card__icon">
                  <DocumentTasks size={18} />
                </span>
                <div className="forms-card__title">
                  <Link to={`/forms/${form.id}`}>{form.code}</Link>
                  <p>{form.name}</p>
                </div>
                {actions(form)}
              </div>
              <div className="forms-card__tags">
                <Tag type="gray" size="sm">{form.type}</Tag>
                <Tag type={formStatusTag(form.status)} size="sm">{form.status}</Tag>
              </div>
              {form.project_name && (
                <p className="forms-card__project">[{form.project_code}] {form.project_name}</p>
              )}
              <footer className="forms-card__footer">
                <span>{form.response_count} resposta(s)</span>
                <span>{relativeTime(form.updated_at)}</span>
              </footer>
            </article>
          ))}
        </div>
      ) : (
        <div className="forms-compact">
          {forms.map((form) => (
            <div className="forms-compact__row" key={form.id}>
              <span className="forms-card__icon">
                <DocumentTasks size={16} />
              </span>
              <div className="forms-compact__main">
                <Link to={`/forms/${form.id}`}>{form.code}</Link>
                <p>{form.name}{form.project_name ? ` · ${form.project_name}` : ''}</p>
              </div>
              <Tag type={formStatusTag(form.status)} size="sm">{form.status}</Tag>
              {actions(form)}
            </div>
          ))}
        </div>
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
