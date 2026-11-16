import { useCallback, useEffect, useState } from 'react';
import { Add, Download, OverflowMenuVertical, Upload } from '@carbon/icons-react';
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
import { IepmsUserFormModal } from '../../components/IepmsUserFormModal';
import { IepmsPasswordModal } from '../../components/IepmsPasswordModal';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorDetails, errorMessage, relativeTime } from '../../lib/format';
import type { IepmsUser, IepmsUsersResponse } from '../../lib/types';

export const IEPMS_STATUS_OPTIONS = [
  { value: 'ativo', label: 'Ativo' },
  { value: 'inativado', label: 'Inativado' },
];

function statusLabel(status: string): string {
  return IEPMS_STATUS_OPTIONS.find((item) => item.value === status)?.label ?? status;
}

export function IepmsUsersPage() {
  const toast = useToast();
  const [users, setUsers] = useState<IepmsUser[]>([]);
  const [data, setData] = useState<IepmsUsersResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [page, setPage] = useState(1);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<IepmsUser | null>(null);
  const [revealing, setRevealing] = useState<IepmsUser | null>(null);
  const [deleting, setDeleting] = useState<IepmsUser | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);
  const [importing, setImporting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: String(page), per_page: '20' });
      if (search.trim()) params.set('search', search.trim());
      if (statusFilter) params.set('status', statusFilter);
      const response = await api.get<IepmsUsersResponse>(`/iepms-users?${params.toString()}`);
      if (response.meta && page > response.meta.last_page) {
        setPage(response.meta.last_page);
        return;
      }
      setUsers(response.data ?? []);
      setData(response);
    } catch (error) {
      toast.error('Não foi possível carregar os usuários do iEPMS', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [page, search, statusFilter, toast]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void load();
    }, 250);
    return () => window.clearTimeout(timer);
  }, [load]);

  function openCreate() {
    setEditing(null);
    setFormOpen(true);
  }

  function openEdit(user: IepmsUser) {
    setEditing(user);
    setFormOpen(true);
  }

  async function remove() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await api.delete(`/iepms-users/${deleting.id}`);
      toast.success('Usuário do iEPMS excluído');
      setDeleting(null);
      await load();
    } catch (error) {
      toast.error('Não foi possível excluir o usuário do iEPMS', errorMessage(error));
    } finally {
      setDeleteBusy(false);
    }
  }

  function importUsers() {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.csv,.xlsx';
    input.onchange = async () => {
      if (!input.files?.[0]) return;
      setImporting(true);
      try {
        const formData = new FormData();
        formData.append('file', input.files[0]);
        const response = await api.upload<{ message: string }>('/iepms-users/import', formData);
        toast.success('Importação concluída', response.message);
        await load();
      } catch (error) {
        toast.error('Falha na importação', errorDetails(error));
      } finally {
        setImporting(false);
      }
    };
    input.click();
  }

  function rowActions(user: IepmsUser) {
    return (
      <OverflowMenu
        aria-label={`Ações de ${user.name}`}
        iconDescription={`Ações de ${user.name}`}
        renderIcon={OverflowMenuVertical}
        size="sm"
        flipped
      >
        {user.has_password && (
          <OverflowMenuItem itemText="Ver senha" onClick={() => setRevealing(user)} />
        )}
        <OverflowMenuItem itemText="Editar" onClick={() => openEdit(user)} />
        <OverflowMenuItem itemText="Excluir" isDelete hasDivider onClick={() => setDeleting(user)} />
      </OverflowMenu>
    );
  }

  return (
    <div className="page">
      <PageHeader
        eyebrow="iEPMS"
        title="Usuários iEPMS"
        description="Gerencie os usuários do aplicativo iEPMS e importe a base em Excel."
        actions={
          <>
            <Button renderIcon={Add} onClick={openCreate} disabled={importing}>Novo usuário</Button>
            <OverflowMenu
              aria-label="Ações de importação e exportação"
              iconDescription="Mais ações"
              renderIcon={OverflowMenuVertical}
              size="md"
              flipped
            >
              <OverflowMenuItem itemText="Exportar XLSX" onClick={() => window.open('/api/v1/iepms-users/export', '_blank')} />
              <OverflowMenuItem itemText="Modelo XLSX" onClick={() => window.open('/api/v1/iepms-users/template', '_blank')} />
              <OverflowMenuItem itemText={importing ? 'Importando…' : 'Importar Excel'} hasDivider onClick={importUsers} />
            </OverflowMenu>
          </>
        }
      />

      {data && (
        <div className="stats-grid stats-grid--3">
          <Tile className="stat-card stat-card--blue">
            <span className="stat-card__label">Usuários</span>
            <strong className="stat-card__value">{data.stats.total}</strong>
          </Tile>
          <Tile className="stat-card stat-card--green">
            <span className="stat-card__label">Ativos</span>
            <strong className="stat-card__value">{data.stats.active}</strong>
          </Tile>
          <Tile className="stat-card stat-card--red">
            <span className="stat-card__label">Inativados</span>
            <strong className="stat-card__value">{data.stats.inactive}</strong>
          </Tile>
        </div>
      )}

      <Tile className="module-main__header">
        <div className="module-toolbar">
          <Search
            size="lg"
            labelText="Buscar usuários do iEPMS"
            placeholder="Buscar por Resource ID, nome, documento, e-mail ou conta"
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
            id="iepms-filter-status"
            labelText="Status"
            size="sm"
            value={statusFilter}
            onChange={(event) => {
              setStatusFilter(event.target.value);
              setPage(1);
            }}
          >
            <SelectItem value="" text="Todos" />
            {IEPMS_STATUS_OPTIONS.map((status) => (
              <SelectItem key={status.value} value={status.value} text={status.label} />
            ))}
          </Select>
        </div>
      </Tile>

      {loading ? (
        <InlineLoading description="Carregando usuários do iEPMS…" />
      ) : users.length === 0 ? (
        <EmptyState
          title="Nenhum usuário do iEPMS encontrado"
          description="Cadastre o primeiro usuário ou importe a base em Excel."
          action={<Button renderIcon={Upload} onClick={importUsers}>Importar Excel</Button>}
        />
      ) : (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table" aria-label="Lista de usuários do iEPMS">
              <thead>
                <tr>
                  <th>Usuário</th>
                  <th className="hide-sm">E-mail</th>
                  <th>Status</th>
                  <th className="hide-md">Conta</th>
                  <th className="hide-sm">Atualizado</th>
                  <th className="cell-actions">Ações</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id}>
                    <td>
                      <button type="button" className="link-button" onClick={() => openEdit(user)}>
                        {user.name}
                      </button>
                      <span className="cell-secondary">{user.resource_id}</span>
                    </td>
                    <td className="hide-sm">{user.personal_mailbox || '—'}</td>
                    <td>
                      {user.status ? (
                        <Tag type={user.status === 'ativo' ? 'green' : 'red'} size="sm">
                          {statusLabel(user.status)}
                        </Tag>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="hide-md">{user.outsourcing_account || '—'}</td>
                    <td className="hide-sm">{relativeTime(user.updated_at)}</td>
                    <td className="cell-actions">{rowActions(user)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {data && data.meta.total > 0 && (
            <div className="table-footer">
              <span className="table-footer__order">
                <Download size={16} /> {data.meta.total.toLocaleString('pt-BR')} usuário(s)
              </span>
              <Pagination
                page={data.meta.current_page}
                pageSize={data.meta.per_page}
                pageSizes={[20]}
                totalItems={data.meta.total}
                onChange={({ page: nextPage }) => setPage(nextPage)}
                size="sm"
              />
            </div>
          )}
        </div>
      )}

      <IepmsUserFormModal
        open={formOpen}
        user={editing}
        onClose={() => setFormOpen(false)}
        onSaved={() => {
          setFormOpen(false);
          void load();
        }}
      />

      <IepmsPasswordModal
        open={revealing !== null}
        user={revealing}
        onClose={() => setRevealing(null)}
      />

      <ConfirmModal
        open={deleting !== null}
        title="Excluir usuário do iEPMS"
        message={deleting ? `Excluir o usuário ${deleting.name} (${deleting.resource_id})? Essa ação não pode ser desfeita.` : ''}
        confirmLabel="Excluir"
        loading={deleteBusy}
        onCancel={() => setDeleting(null)}
        onConfirm={() => void remove()}
      />
    </div>
  );
}
