import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, OverflowMenuVertical, Save } from '@carbon/icons-react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  Checkbox,
  InlineLoading,
  InlineNotification,
  OverflowMenu,
  OverflowMenuItem,
  Tag,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState, Loading } from '../../components/Feedback';
import { ConfirmModal } from '../../components/ConfirmModal';
import { IepmsUserFormModal } from '../../components/IepmsUserFormModal';
import { IepmsPasswordModal } from '../../components/IepmsPasswordModal';
import { useToast } from '../../contexts/ToastContext';
import { ApiError, api } from '../../lib/api';
import { errorMessage, fullDate, relativeTime } from '../../lib/format';
import type { IepmsProject, IepmsUser, IepmsUserProject } from '../../lib/types';

function statusLabel(status: string): string {
  if (status === 'ativo') return 'Ativo';
  if (status === 'inativado') return 'Inativado';
  return status || '—';
}

function valueOrDash(value: string | null | undefined): string {
  return value && value.trim() ? value : '—';
}

function isExpired(value: string): boolean {
  if (!value) return false;
  const today = new Date();
  const todayKey = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;
  return value < todayKey;
}

export function IepmsUserDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const toast = useToast();
  const [user, setUser] = useState<IepmsUser | null>(null);
  const [loading, setLoading] = useState(true);
  const [editOpen, setEditOpen] = useState(false);
  const [revealing, setRevealing] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [projects, setProjects] = useState<IepmsProject[]>([]);
  const [checklist, setChecklist] = useState<Record<number, string>>({});
  const [projectsLoading, setProjectsLoading] = useState(true);
  const [projectsSaving, setProjectsSaving] = useState(false);
  const [projectsError, setProjectsError] = useState('');

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const response = await api.get<{ data: { user: IepmsUser } }>(`/iepms-users/${id}`);
      setUser(response.data.user);
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        setUser(null);
      } else {
        toast.error('Não foi possível carregar o usuário do iEPMS', errorMessage(error));
      }
    } finally {
      setLoading(false);
    }
  }, [id, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const loadProjects = useCallback(async () => {
    if (!id) return;
    setProjectsLoading(true);
    try {
      const [catalogResponse, userProjectsResponse] = await Promise.all([
        api.get<{ data: IepmsProject[] }>('/iepms-projects'),
        api.get<{ data: { projects: IepmsUserProject[] } }>(`/iepms-users/${id}/projects`),
      ]);
      setProjects(catalogResponse.data ?? []);
      const selected: Record<number, string> = {};
      for (const item of userProjectsResponse.data.projects ?? []) {
        selected[item.project_id] = item.expires_at ?? '';
      }
      setChecklist(selected);
    } catch (error) {
      toast.error('Não foi possível carregar o checklist de projetos', errorMessage(error));
    } finally {
      setProjectsLoading(false);
    }
  }, [id, toast]);

  useEffect(() => {
    void loadProjects();
  }, [loadProjects]);

  function toggleProject(projectID: number, checked: boolean) {
    setChecklist((current) => {
      const next = { ...current };
      if (checked) {
        next[projectID] = current[projectID] ?? '';
      } else {
        delete next[projectID];
      }
      return next;
    });
    setProjectsError('');
  }

  async function saveChecklist() {
    if (!user) return;
    setProjectsSaving(true);
    setProjectsError('');
    try {
      const payload = {
        projects: Object.entries(checklist).map(([projectID, expiresAt]) => ({
          project_id: Number(projectID),
          expires_at: expiresAt || null,
        })),
      };
      const response = await api.put<{ data: { projects: IepmsUserProject[] } }>(
        `/iepms-users/${user.id}/projects`,
        payload,
      );
      const selected: Record<number, string> = {};
      for (const item of response.data.projects ?? []) {
        selected[item.project_id] = item.expires_at ?? '';
      }
      setChecklist(selected);
      toast.success('Checklist de projetos atualizado');
    } catch (error) {
      if (error instanceof ApiError && error.fields.expires_at) {
        setProjectsError(error.fields.expires_at);
      } else if (error instanceof ApiError && error.fields.projects) {
        setProjectsError(error.fields.projects);
      }
      toast.error('Não foi possível salvar o checklist', errorMessage(error));
    } finally {
      setProjectsSaving(false);
    }
  }

  async function remove() {
    if (!user) return;
    setDeleting(true);
    try {
      await api.delete(`/iepms-users/${user.id}`);
      toast.success('Usuário do iEPMS excluído');
      navigate('/iepms-users');
    } catch (error) {
      toast.error('Não foi possível excluir o usuário do iEPMS', errorMessage(error));
    } finally {
      setDeleting(false);
    }
  }

  if (loading) {
    return (
      <div className="page">
        <Loading description="Carregando usuário do iEPMS…" />
      </div>
    );
  }

  if (!user) {
    return (
      <div className="page">
        <Breadcrumb noTrailingSlash className="doc-breadcrumb">
          <BreadcrumbItem><Link to="/iepms-users">Usuários iEPMS</Link></BreadcrumbItem>
          <BreadcrumbItem isCurrentPage>Usuário não encontrado</BreadcrumbItem>
        </Breadcrumb>
        <EmptyState
          title="Usuário do iEPMS não encontrado"
          description="O usuário pode ter sido excluído ou o endereço está incorreto."
          action={<Button onClick={() => navigate('/iepms-users')}>Voltar para usuários iEPMS</Button>}
        />
      </div>
    );
  }

  return (
    <div className="page">
      <Breadcrumb noTrailingSlash className="doc-breadcrumb">
        <BreadcrumbItem><Link to="/iepms-users">Usuários iEPMS</Link></BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>{user.name}</BreadcrumbItem>
      </Breadcrumb>

      <PageHeader
        eyebrow="iEPMS"
        title={user.name}
        description={
          <span className="tag-row">
            <Tag type={user.status === 'ativo' ? 'green' : 'red'} size="sm">{statusLabel(user.status)}</Tag>
            <Tag type="gray" size="sm">{user.resource_id}</Tag>
            <Tag type={user.has_password ? 'blue' : 'cool-gray'} size="sm">
              {user.has_password ? 'Senha cadastrada' : 'Sem senha'}
            </Tag>
          </span>
        }
        actions={
          <>
            <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/iepms-users')}>
              Voltar
            </Button>
            <OverflowMenu
              aria-label="Ações do usuário do iEPMS"
              iconDescription="Ações do usuário do iEPMS"
              renderIcon={OverflowMenuVertical}
              size="md"
              flipped
            >
              {user.has_password && (
                <OverflowMenuItem itemText="Ver senha" onClick={() => setRevealing(true)} />
              )}
              <OverflowMenuItem itemText="Editar" onClick={() => setEditOpen(true)} />
              <OverflowMenuItem
                itemText="Excluir"
                isDelete
                hasDivider
                onClick={() => setDeleteOpen(true)}
              />
            </OverflowMenu>
          </>
        }
      />

      <Tile className="nokia-detail-section">
        <div className="nokia-section-header">
          <div>
            <h2>Identificação</h2>
            <p>Dados pessoais, documento e contato.</p>
          </div>
        </div>
        <dl className="demand-info-grid">
          <div><dt>Resource ID</dt><dd>{valueOrDash(user.resource_id)}</dd></div>
          <div><dt>Nome</dt><dd>{valueOrDash(user.name)}</dd></div>
          <div><dt>Gênero</dt><dd>{valueOrDash(user.gender)}</dd></div>
          <div><dt>Nacionalidade</dt><dd>{valueOrDash(user.nationality)}</dd></div>
          <div><dt>Ano de nascimento</dt><dd>{valueOrDash(user.birth_year)}</dd></div>
          <div><dt>Tipo de certificado</dt><dd>{valueOrDash(user.certificate_type)}</dd></div>
          <div><dt>Número do documento</dt><dd>{valueOrDash(user.id_number)}</dd></div>
          <div><dt>Telefone</dt><dd>{valueOrDash(user.tel)}</dd></div>
          <div><dt>E-mail pessoal</dt><dd>{valueOrDash(user.personal_mailbox)}</dd></div>
        </dl>
      </Tile>

      <Tile className="nokia-detail-section">
        <div className="nokia-section-header">
          <div>
            <h2>Conta e segurança</h2>
            <p>Conta de terceirização, status e metadados do cadastro.</p>
          </div>
        </div>
        <dl className="demand-info-grid">
          <div><dt>Conta de terceirização</dt><dd>{valueOrDash(user.outsourcing_account)}</dd></div>
          <div><dt>Status</dt><dd>{statusLabel(user.status)}</dd></div>
          <div><dt>Senha</dt><dd>{user.has_password ? 'Cadastrada' : 'Não cadastrada'}</dd></div>
          <div><dt>Última atualização por</dt><dd>{valueOrDash(user.last_updated_by)}</dd></div>
          <div><dt>Criado em</dt><dd>{fullDate(user.created_at)}</dd></div>
          <div>
            <dt>Atualizado em</dt>
            <dd title={fullDate(user.updated_at)}>{relativeTime(user.updated_at)}</dd>
          </div>
        </dl>
      </Tile>

      <Tile className="nokia-detail-section">
        <div className="nokia-section-header">
          <div>
            <h2>Checklist de projetos</h2>
            <p>Marque os projetos em que o usuário está cadastrado e informe a validade.</p>
          </div>
          <Button
            renderIcon={Save}
            onClick={() => void saveChecklist()}
            disabled={projectsLoading || projectsSaving || projects.length === 0}
          >
            {projectsSaving ? 'Salvando…' : 'Salvar checklist'}
          </Button>
        </div>

        {projectsLoading ? (
          <InlineLoading description="Carregando projetos…" />
        ) : projects.length === 0 ? (
          <p className="muted">
            Nenhum projeto cadastrado. <Link to="/iepms-users/settings">Cadastre os projetos nas configurações</Link>.
          </p>
        ) : (
          <div className="iepms-project-list">
            {projects.map((project) => {
              const checked = checklist[project.id] !== undefined;
              const expiresAt = checklist[project.id] ?? '';
              const expired = checked && isExpired(expiresAt);
              return (
                <div key={project.id} className={`iepms-project-row${checked ? ' is-checked' : ''}`}>
                  <Checkbox
                    id={`iepms-project-${project.id}`}
                    labelText={project.name}
                    checked={checked}
                    onChange={(_event, { checked: isChecked }) => toggleProject(project.id, isChecked)}
                  />
                  <div className="iepms-project-row__expiry">
                    <TextInput
                      id={`iepms-project-expiry-${project.id}`}
                      type="date"
                      labelText="Validade"
                      value={expiresAt}
                      disabled={!checked}
                      onChange={(event) =>
                        setChecklist((current) => ({ ...current, [project.id]: event.target.value }))
                      }
                    />
                    {checked && (
                      expiresAt ? (
                        <Tag type={expired ? 'red' : 'green'} size="sm">
                          {expired ? 'Vencido' : 'Válido'}
                        </Tag>
                      ) : (
                        <Tag type="gray" size="sm">Sem validade</Tag>
                      )
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {projectsError && (
          <InlineNotification
            kind="error"
            title="Revise o checklist"
            subtitle={projectsError}
            lowContrast
            hideCloseButton
          />
        )}
      </Tile>

      <IepmsUserFormModal
        open={editOpen}
        user={user}
        onClose={() => setEditOpen(false)}
        onSaved={() => {
          setEditOpen(false);
          void load();
        }}
      />

      <IepmsPasswordModal
        open={revealing}
        user={user}
        onClose={() => setRevealing(false)}
      />

      <ConfirmModal
        open={deleteOpen}
        title="Excluir usuário do iEPMS"
        message={`Excluir o usuário ${user.name} (${user.resource_id})? Essa ação não pode ser desfeita.`}
        confirmLabel="Excluir"
        loading={deleting}
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => void remove()}
      />
    </div>
  );
}
