import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Add,
  Grid,
  List,
  OverflowMenuVertical,
  Table as TableIcon,
  TrashCan,
  UserFollow,
  UserMultiple,
} from '@carbon/icons-react';
import {
  Button,
  Checkbox,
  InlineNotification,
  Modal,
  OverflowMenu,
  OverflowMenuItem,
  Pagination,
  PasswordInput,
  Search,
  Select,
  SelectItem,
  Tag,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../components/PageHeader';
import { EmptyState, Loading, RoleTag, ActiveTag } from '../components/Feedback';
import { ConfirmModal } from '../components/ConfirmModal';
import { useToast } from '../contexts/ToastContext';
import { ApiError, api } from '../lib/api';
import { errorMessage, formatNumber, initials, relativeTime } from '../lib/format';
import type { User, UsersResponse } from '../lib/types';

type ViewMode = 'table' | 'cards' | 'compact';

interface UserForm {
  name: string;
  surname: string;
  email: string;
  password: string;
  password_confirmation: string;
  is_admin: boolean;
}

const emptyForm: UserForm = {
  name: '',
  surname: '',
  email: '',
  password: '',
  password_confirmation: '',
  is_admin: false,
};

function useIsMobile() {
  const [mobile, setMobile] = useState(() => window.matchMedia('(max-width: 671px)').matches);
  useEffect(() => {
    const media = window.matchMedia('(max-width: 671px)');
    const listener = (event: MediaQueryListEvent) => setMobile(event.matches);
    media.addEventListener('change', listener);
    return () => media.removeEventListener('change', listener);
  }, []);
  return mobile;
}

function statsCards(stats: UsersResponse['stats']) {
  return [
    { key: 'total', label: 'Total de usuários', value: stats.total, icon: UserMultiple, tone: 'blue' },
    { key: 'admins', label: 'Administradores', value: stats.admins, icon: UserFollow, tone: 'purple' },
    { key: 'active', label: 'Ativos', value: stats.active, icon: UserMultiple, tone: 'green' },
    { key: 'inactive', label: 'Inativos', value: stats.inactive, icon: UserMultiple, tone: 'red' },
  ];
}

export function UsersPage() {
  const toast = useToast();
  const isMobile = useIsMobile();
  const [data, setData] = useState<UsersResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [profileFilter, setProfileFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [sort, setSort] = useState('created_at');
  const [direction, setDirection] = useState('desc');
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(10);
  const [view, setView] = useState<ViewMode>(() => {
    const stored = localStorage.getItem('users-view') as ViewMode | null;
    if (stored === 'table' || stored === 'cards' || stored === 'compact') return stored;
    return window.matchMedia('(max-width: 671px)').matches ? 'cards' : 'table';
  });
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const [form, setForm] = useState<UserForm>(emptyForm);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState('');
  const [saving, setSaving] = useState(false);
  const [confirm, setConfirm] = useState<{ type: 'delete' | 'toggle'; user: User } | null>(null);
  const [confirmBusy, setConfirmBusy] = useState(false);
  const requestRef = useRef(0);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedSearch(search.trim());
      setPage(1);
    }, 400);
    return () => window.clearTimeout(timer);
  }, [search]);

  useEffect(() => {
    localStorage.setItem('users-view', view);
  }, [view]);

  const load = useCallback(async () => {
    const requestId = ++requestRef.current;
    setLoading(true);
    const params = new URLSearchParams({
      page: String(page),
      per_page: String(perPage),
      sort,
      direction,
    });
    if (debouncedSearch) params.set('search', debouncedSearch);
    if (profileFilter) params.set('profile', profileFilter);
    if (statusFilter) params.set('status', statusFilter);
    try {
      const response = await api.get<UsersResponse>(`/users?${params.toString()}`);
      if (requestId === requestRef.current) setData(response);
    } catch (error) {
      toast.error('Não foi possível carregar os usuários', errorMessage(error));
    } finally {
      if (requestId === requestRef.current) setLoading(false);
    }
  }, [page, perPage, sort, direction, debouncedSearch, profileFilter, statusFilter, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const stats = data?.stats;
  const users = data?.data ?? [];

  function toggleSort(column: string) {
    if (sort === column) {
      setDirection((current) => (current === 'asc' ? 'desc' : 'asc'));
    } else {
      setSort(column);
      setDirection(column === 'created_at' ? 'desc' : 'asc');
    }
    setPage(1);
  }

  function openCreate() {
    setEditing(null);
    setForm(emptyForm);
    setFieldErrors({});
    setFormError('');
    setModalOpen(true);
  }

  function openEdit(user: User) {
    setEditing(user);
    setForm({
      name: user.name,
      surname: user.surname ?? '',
      email: user.email,
      password: '',
      password_confirmation: '',
      is_admin: user.is_admin,
    });
    setFieldErrors({});
    setFormError('');
    setModalOpen(true);
  }

  async function submitForm() {
    setSaving(true);
    setFieldErrors({});
    setFormError('');
    const payload = {
      name: form.name,
      surname: form.surname,
      email: form.email,
      password: form.password,
      password_confirmation: form.password_confirmation,
      is_admin: form.is_admin,
    };
    try {
      if (editing) {
        await api.put(`/users/${editing.id}`, payload);
        toast.success('Usuário atualizado com sucesso');
      } else {
        await api.post('/users', payload);
        toast.success('Usuário criado com sucesso');
      }
      setModalOpen(false);
      await load();
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fields);
        if (!Object.keys(error.fields).length) setFormError(error.message);
      } else {
        setFormError('Não foi possível salvar o usuário.');
      }
    } finally {
      setSaving(false);
    }
  }

  async function runConfirm() {
    if (!confirm) return;
    setConfirmBusy(true);
    try {
      if (confirm.type === 'delete') {
        await api.delete(`/users/${confirm.user.id}`);
        toast.success('Usuário excluído com sucesso');
      } else {
        const response = await api.patch<{ message: string }>(`/users/${confirm.user.id}/toggle`);
        toast.success(response.message);
      }
      setConfirm(null);
      await load();
    } catch (error) {
      toast.error('Não foi possível concluir a ação', errorMessage(error));
    } finally {
      setConfirmBusy(false);
    }
  }

  const sortLabel = useMemo(() => {
    const labels: Record<string, string> = {
      name: 'Nome',
      email: 'E-mail',
      is_admin: 'Perfil',
      created_at: 'Criado em',
    };
    return labels[sort] ?? sort;
  }, [sort]);

  function rowActions(user: User) {
    return (
      <OverflowMenu
        aria-label={`Ações de ${user.full_name}`}
        iconDescription={`Ações de ${user.full_name}`}
        renderIcon={OverflowMenuVertical}
        size="sm"
        flipped
      >
        <OverflowMenuItem itemText="Editar" onClick={() => openEdit(user)} />
        <OverflowMenuItem
          itemText={user.is_active ? 'Desativar' : 'Ativar'}
          onClick={() => setConfirm({ type: 'toggle', user })}
        />
        <OverflowMenuItem
          itemText="Excluir"
          isDelete
          hasDivider
          onClick={() => setConfirm({ type: 'delete', user })}
        />
      </OverflowMenu>
    );
  }

  return (
    <div className="page">
      <PageHeader
        eyebrow="Administração"
        title="Usuários"
        description="Gerencie contas, permissões e status de acesso."
        actions={<Button renderIcon={Add} onClick={openCreate}>Novo usuário</Button>}
      />

      {stats && (
        <div className="stats-grid">
          {statsCards(stats).map((card) => {
            const Icon = card.icon;
            return (
              <Tile key={card.key} className={`stat-card stat-card--${card.tone}`}>
                <div className="stat-card__icon"><Icon size={20} /></div>
                <span className="stat-card__label">{card.label}</span>
                <strong className="stat-card__value">{formatNumber(card.value)}</strong>
              </Tile>
            );
          })}
        </div>
      )}

      <section className="toolbar">
        <div className="toolbar__search">
          <Search
            size="lg"
            labelText="Buscar usuários"
            placeholder="Buscar por nome, sobrenome ou e-mail"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            onClear={() => setSearch('')}
          />
        </div>
        <div className="toolbar__filters">
          <Select
            id="profile-filter"
            labelText="Perfil"
            value={profileFilter}
            size="sm"
            onChange={(event) => {
              setProfileFilter(event.target.value);
              setPage(1);
            }}
          >
            <SelectItem value="" text="Todos" />
            <SelectItem value="admin" text="Administrador" />
            <SelectItem value="user" text="Usuário" />
          </Select>
          <Select
            id="status-filter"
            labelText="Status"
            value={statusFilter}
            size="sm"
            onChange={(event) => {
              setStatusFilter(event.target.value);
              setPage(1);
            }}
          >
            <SelectItem value="" text="Todos" />
            <SelectItem value="active" text="Ativos" />
            <SelectItem value="inactive" text="Inativos" />
          </Select>
          <Select
            id="per-page"
            labelText="Por página"
            value={String(perPage)}
            size="sm"
            onChange={(event) => {
              setPerPage(Number(event.target.value));
              setPage(1);
            }}
          >
            <SelectItem value="10" text="10" />
            <SelectItem value="25" text="25" />
            <SelectItem value="50" text="50" />
          </Select>
        </div>
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
            iconDescription="Lista compacta"
            onClick={() => setView('compact')}
          />
        </div>
      </section>

      {loading ? (
        <Loading description="Carregando usuários…" />
      ) : users.length === 0 ? (
        <EmptyState
          icon={UserMultiple}
          title={debouncedSearch || profileFilter || statusFilter ? 'Nenhum usuário encontrado' : 'Nenhum usuário cadastrado'}
          description={debouncedSearch || profileFilter || statusFilter ? 'Ajuste os filtros ou limpe a busca.' : 'Crie o primeiro usuário para começar.'}
          action={<Button renderIcon={Add} onClick={openCreate}>Novo usuário</Button>}
        />
      ) : view === 'table' ? (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table" aria-label="Lista de usuários">
              <thead>
                <tr>
                  <th aria-sort={sort === 'name' ? (direction === 'asc' ? 'ascending' : 'descending') : 'none'}>
                    <button type="button" className="sort-button" onClick={() => toggleSort('name')}>
                      Nome {sort === 'name' && <span aria-hidden>{direction === 'asc' ? '↑' : '↓'}</span>}
                    </button>
                  </th>
                  <th className="hide-sm" aria-sort={sort === 'email' ? (direction === 'asc' ? 'ascending' : 'descending') : 'none'}>
                    <button type="button" className="sort-button" onClick={() => toggleSort('email')}>
                      E-mail {sort === 'email' && <span aria-hidden>{direction === 'asc' ? '↑' : '↓'}</span>}
                    </button>
                  </th>
                  <th aria-sort={sort === 'is_admin' ? (direction === 'asc' ? 'ascending' : 'descending') : 'none'}>
                    <button type="button" className="sort-button" onClick={() => toggleSort('is_admin')}>
                      Perfil {sort === 'is_admin' && <span aria-hidden>{direction === 'asc' ? '↑' : '↓'}</span>}
                    </button>
                  </th>
                  <th className="hide-md" aria-sort={sort === 'created_at' ? (direction === 'asc' ? 'ascending' : 'descending') : 'none'}>
                    <button type="button" className="sort-button" onClick={() => toggleSort('created_at')}>
                      Criado em {sort === 'created_at' && <span aria-hidden>{direction === 'asc' ? '↑' : '↓'}</span>}
                    </button>
                  </th>
                  <th className="cell-actions">Ações</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id}>
                    <td>
                      <div className="user-cell">
                        <span className="avatar">{initials(user.full_name)}</span>
                        <div>
                          <span className="cell-primary">{user.full_name}</span>
                          <span className="cell-secondary show-sm">{user.email}</span>
                        </div>
                      </div>
                    </td>
                    <td className="hide-sm">{user.email}</td>
                    <td>
                      <div className="tag-row">
                        <RoleTag isAdmin={user.is_admin} />
                        {!user.is_active && <Tag type="red" size="sm">Inativo</Tag>}
                      </div>
                    </td>
                    <td className="hide-md">{relativeTime(user.created_at)}</td>
                    <td className="cell-actions">{rowActions(user)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="table-footer">
            <span className="table-footer__order">Ordenado por {sortLabel} ({direction === 'asc' ? 'crescente' : 'decrescente'})</span>
          </div>
        </div>
      ) : (
        <div className={view === 'cards' ? 'user-cards' : 'user-compact'}>
          {users.map((user) => (
            <div key={user.id} className="user-card">
              <div className="user-card__main">
                <span className="avatar avatar--lg">{initials(user.full_name)}</span>
                <div className="user-card__text">
                  <strong>{user.full_name}</strong>
                  <span>{user.email}</span>
                  <div className="tag-row">
                    <RoleTag isAdmin={user.is_admin} />
                    <ActiveTag active={user.is_active} />
                  </div>
                </div>
              </div>
              <div className="user-card__meta">
                <span>Criado em {relativeTime(user.created_at)}</span>
              </div>
              <div className="user-card__actions">{rowActions(user)}</div>
            </div>
          ))}
        </div>
      )}

      {data && data.meta.total > 0 && (
        <Pagination
          page={page}
          pageSize={perPage}
          pageSizes={[10, 25, 50]}
          totalItems={data.meta.total}
          onChange={({ page: nextPage, pageSize: nextPerPage }) => {
            setPage(nextPage);
            if (nextPerPage !== perPage) {
              setPerPage(nextPerPage);
              setPage(1);
            }
          }}
          itemRangeText={(min, max, total) => `${min}–${max} de ${total} usuários`}
          itemsPerPageText="Itens por página"
          pageRangeText={(_current, total) => `de ${total} página(s)`}
          pageSelectLabelText={(total) => `Página de ${total}`}
          pageText={(current) => `Página ${current}`}
          backwardText="Página anterior"
          forwardText="Próxima página"
        />
      )}

      <Modal
        open={modalOpen}
        modalHeading={editing ? 'Editar usuário' : 'Novo usuário'}
        primaryButtonText={saving ? 'Salvando…' : editing ? 'Salvar alterações' : 'Criar usuário'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving}
        onRequestSubmit={() => void submitForm()}
        onRequestClose={() => setModalOpen(false)}
        onSecondarySubmit={() => setModalOpen(false)}
        size="sm"
      >
        <div className="modal-form">
          {formError && <InlineNotification kind="error" title="Não foi possível salvar" subtitle={formError} lowContrast hideCloseButton />}
          <TextInput
            id="user-name"
            labelText="Nome"
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
            invalid={Boolean(fieldErrors.name)}
            invalidText={fieldErrors.name}
            required
          />
          <TextInput
            id="user-surname"
            labelText="Sobrenome"
            value={form.surname}
            onChange={(event) => setForm({ ...form, surname: event.target.value })}
            invalid={Boolean(fieldErrors.surname)}
            invalidText={fieldErrors.surname}
          />
          <TextInput
            id="user-email"
            type="email"
            labelText="E-mail"
            value={form.email}
            onChange={(event) => setForm({ ...form, email: event.target.value })}
            invalid={Boolean(fieldErrors.email)}
            invalidText={fieldErrors.email}
            required
          />
          <PasswordInput
            id="user-password"
            labelText={editing ? 'Nova senha (opcional)' : 'Senha'}
            helperText={editing ? 'Deixe em branco para manter a senha atual.' : 'Mínimo de 8 caracteres.'}
            value={form.password}
            onChange={(event) => setForm({ ...form, password: event.target.value })}
            invalid={Boolean(fieldErrors.password)}
            invalidText={fieldErrors.password}
            required={!editing}
          />
          <PasswordInput
            id="user-password-confirmation"
            labelText="Confirmar senha"
            value={form.password_confirmation}
            onChange={(event) => setForm({ ...form, password_confirmation: event.target.value })}
            invalid={Boolean(fieldErrors.password_confirmation)}
            invalidText={fieldErrors.password_confirmation}
            required={!editing && Boolean(form.password)}
          />
          <Checkbox
            id="user-is-admin"
            labelText="Tornar administrador"
            checked={form.is_admin}
            onChange={(_event, { checked }) => setForm({ ...form, is_admin: checked })}
          />
        </div>
      </Modal>

      <ConfirmModal
        open={Boolean(confirm)}
        title={confirm?.type === 'delete' ? 'Excluir usuário' : confirm?.user?.is_active ? 'Desativar usuário' : 'Ativar usuário'}
        message={
          confirm?.type === 'delete'
            ? `Excluir a conta de ${confirm.user.full_name}? Esta ação não pode ser desfeita.`
            : confirm?.user?.is_active
              ? `Desativar ${confirm?.user?.full_name}? A conta perderá o acesso imediatamente.`
              : `Ativar ${confirm?.user?.full_name}? A conta voltará a poder acessar o sistema.`
        }
        confirmLabel={confirm?.type === 'delete' ? 'Excluir usuário' : confirm?.user?.is_active ? 'Desativar' : 'Ativar'}
        danger={confirm?.type === 'delete' || Boolean(confirm?.user?.is_active)}
        loading={confirmBusy}
        onCancel={() => setConfirm(null)}
        onConfirm={() => void runConfirm()}
      />
    </div>
  );
}
