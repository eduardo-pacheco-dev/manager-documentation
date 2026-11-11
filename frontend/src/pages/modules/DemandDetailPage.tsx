import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  Checkmark,
  Edit,
  Email,
  Pause,
  Play,
  Restart,
  Send,
  TrashCan,
} from '@carbon/icons-react';
import {
  Breadcrumb,
  BreadcrumbItem,
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
import { PageHeader } from '../../components/PageHeader';
import { ConfirmModal } from '../../components/ConfirmModal';
import { useAuth } from '../../contexts/AuthContext';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import {
  DEMAND_PRIORITY,
  DEMAND_STATUS,
  demandPriorityType,
  demandStatusType,
} from './DemandsPage';

interface Collaborator { id: number; name: string; email: string }
interface Update {
  id: number; body: string; created_at: string; author_name: string | null;
  status_from: string | null; status_to: string | null; user_id: number | null;
}
interface AttachmentFile {
  id: number; original_name: string; mime_type: string | null; size: number;
  created_at: string; uploader_name: string | null; download_url: string;
}
interface Demand {
  id: number; requester_id: number; assignee_id: number; title: string;
  description: string | null; status: string; priority: string; due_at: string | null;
  completed_at: string | null; created_at: string; updated_at: string;
  client_name: string | null; client_email: string | null;
  requester_name: string | null; assignee_name: string | null; assignee_email: string | null;
  overdue: boolean; update_count: number; updates: Update[]; attachments: AttachmentFile[];
}

const EMPTY_FORM = {
  title: '', description: '', assignee_id: '', priority: 'media', due_at: '', status: 'aberta',
  client_name: '', client_email: '',
};

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

export function DemandDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const toast = useToast();
  const [demand, setDemand] = useState<Demand | null>(null);
  const [collaborators, setCollaborators] = useState<Collaborator[]>([]);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [form, setForm] = useState({ ...EMPTY_FORM });
  const [saving, setSaving] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [comment, setComment] = useState('');
  const [uploading, setUploading] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const response = await api.get<{ data: { demand: Demand } }>(`/demands/${id}`);
      setDemand(response.data.demand);
    } catch {
      setNotFound(true);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { void load(); }, [load]);

  useEffect(() => {
    api.get<{ data: Collaborator[] }>('/demands/collaborators')
      .then((response) => setCollaborators(response.data ?? []))
      .catch(() => setCollaborators([]));
  }, []);

  function openEdit() {
    if (!demand) return;
    setForm({
      title: demand.title, description: demand.description ?? '', assignee_id: String(demand.assignee_id),
      priority: demand.priority, due_at: demand.due_at ?? '', status: demand.status,
      client_name: demand.client_name ?? '', client_email: demand.client_email ?? '',
    });
    setEditOpen(true);
  }

  async function save() {
    if (!demand) return;
    setSaving(true);
    try {
      await api.put(`/demands/${demand.id}`, {
        ...form,
        assignee_id: form.assignee_id ? Number(form.assignee_id) : 0,
        due_at: form.due_at || '',
      });
      toast.success('Demanda atualizada com sucesso.');
      setEditOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar a demanda', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function changeStatus(next: string) {
    if (!demand) return;
    try {
      const response = await api.patch<{ data: { demand: Demand } }>(`/demands/${demand.id}/status`, { status: next });
      setDemand(response.data.demand);
      toast.success('Status atualizado.');
    } catch (error) {
      toast.error('Não foi possível alterar o status', errorMessage(error));
    }
  }

  async function addComment() {
    if (!demand || !comment.trim()) return;
    try {
      await api.post(`/demands/${demand.id}/updates`, { body: comment });
      setComment('');
      await load();
    } catch (error) {
      toast.error('Não foi possível publicar o comentário', errorMessage(error));
    }
  }

  async function uploadAttachments(files: FileList | null) {
    if (!demand || !files || files.length === 0) return;
    setUploading(true);
    try {
      const payload = new FormData();
      Array.from(files).forEach((file) => payload.append('files', file));
      const response = await api.upload<{ message: string }>(`/demands/${demand.id}/attachments`, payload);
      toast.success(response.message);
      await load();
    } catch (error) {
      toast.error('Não foi possível enviar o arquivo', errorMessage(error));
    } finally {
      setUploading(false);
    }
  }

  async function deleteAttachment(file: AttachmentFile) {
    if (!demand) return;
    try {
      await api.delete(`/demands/${demand.id}/attachments/${file.id}`);
      toast.success('Arquivo removido.');
      await load();
    } catch (error) {
      toast.error('Não foi possível remover o arquivo', errorMessage(error));
    }
  }

  async function removeDemand() {
    if (!demand) return;
    setDeleting(true);
    try {
      await api.delete(`/demands/${demand.id}`);
      toast.success('Demanda excluída.');
      navigate('/demands');
    } catch (error) {
      toast.error('Não foi possível excluir a demanda', errorMessage(error));
      setDeleting(false);
    }
  }

  async function copyDispatchEmail() {
    if (!demand) return;
    const dueDate = demand.due_at
      ? new Date(`${demand.due_at}T12:00:00`).toLocaleDateString('pt-BR')
      : 'não informado';
    const status = DEMAND_STATUS[demand.status] ?? demand.status;
    const priority = DEMAND_PRIORITY[demand.priority] ?? demand.priority;
    const link = `${window.location.origin}/demands/${demand.id}`;
    const lines = [
      `Para: ${demand.assignee_email ?? '—'}`,
      `Assunto: [Demanda] ${demand.title}`,
      '',
      `Olá, ${demand.assignee_name ?? 'colaborador(a)'},`,
      '',
      'Você foi acionado(a) para atender a seguinte demanda:',
      '',
      `Demanda: ${demand.title}`,
      `Solicitante: ${demand.requester_name ?? '—'}`,
      `Responsável: ${demand.assignee_name ?? '—'}`,
      `Prioridade: ${priority}`,
      `Status: ${status}`,
      `Prazo: ${dueDate}`,
    ];
    if (demand.description?.trim()) {
      lines.push('', 'Descrição:', demand.description.trim());
    }
    lines.push('', 'Acesse a demanda:', link, '', 'Atenciosamente,', 'Documentation Center');
    try {
      await navigator.clipboard.writeText(lines.join('\n'));
      toast.success('E-mail de acionamento copiado.');
    } catch {
      toast.error('Não foi possível copiar o e-mail', 'Copie manualmente o conteúdo da demanda.');
    }
  }

  async function copyClientEmail() {
    if (!demand) return;
    if (!demand.client_email && !demand.client_name) {
      toast.warning('Cliente não informado', 'Edite a demanda e preencha o cliente/e-mail.');
      return;
    }
    const dueDate = demand.due_at
      ? new Date(`${demand.due_at}T12:00:00`).toLocaleDateString('pt-BR')
      : 'a combinar';
    const status = DEMAND_STATUS[demand.status] ?? demand.status;
    const link = `${window.location.origin}/demands/${demand.id}`;
    const lines = [
      `Para: ${demand.client_email ?? '—'}`,
      `Assunto: [Demanda] ${demand.title}`,
      '',
      `Olá, ${demand.client_name ?? 'cliente'},`,
      '',
      'Segue o andamento da demanda registrada junto à nossa equipe:',
      '',
      `Demanda: ${demand.title}`,
      `Responsável: ${demand.assignee_name ?? '—'}`,
      `Status: ${status}`,
      `Prazo: ${dueDate}`,
    ];
    if (demand.description?.trim()) {
      lines.push('', 'Descrição:', demand.description.trim());
    }
    lines.push('', 'Para acompanhar, acesse:', link, '', 'Atenciosamente,', 'Documentation Center');
    try {
      await navigator.clipboard.writeText(lines.join('\n'));
      toast.success('E-mail para o cliente copiado.');
    } catch {
      toast.error('Não foi possível copiar o e-mail', 'Copie manualmente o conteúdo da demanda.');
    }
  }

  if (loading) return <InlineLoading description="Carregando demanda…" />;
  if (notFound || !demand) {
    return (
      <div className="status-page">
        <p className="status-page__code">404</p>
        <h1>Demanda não encontrada</h1>
        <p>A demanda pode ter sido excluída ou o endereço está incorreto.</p>
        <Link to="/demands"><Button>Voltar para Demandas</Button></Link>
      </div>
    );
  }

  const isAdmin = Boolean(user?.is_admin);
  const canManage = isAdmin || demand.requester_id === user?.id;
  const canParticipate = canManage || demand.assignee_id === user?.id;

  return (
    <div className="page">
      <Breadcrumb noTrailingSlash className="doc-breadcrumb">
        <BreadcrumbItem><Link to="/demands">Demandas</Link></BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>{demand.title}</BreadcrumbItem>
      </Breadcrumb>

      <PageHeader
        eyebrow="Colaboração"
        title={demand.title}
        description={
          <span className="tag-row">
            <Tag type={demandStatusType(demand.status)} size="sm">{DEMAND_STATUS[demand.status] ?? demand.status}</Tag>
            <Tag type={demandPriorityType(demand.priority)} size="sm">{DEMAND_PRIORITY[demand.priority] ?? demand.priority}</Tag>
            {demand.overdue && <Tag type="red" size="sm">Atrasada</Tag>}
            {demand.due_at && <Tag type="gray" size="sm">Prazo {demand.due_at}</Tag>}
          </span>
        }
        actions={
          <>
            <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/demands')}>Voltar</Button>
            <Button kind="ghost" renderIcon={Email} onClick={() => void copyDispatchEmail()}>Copiar e-mail</Button>
            <Button kind="ghost" renderIcon={Email} onClick={() => void copyClientEmail()}>Copiar e-mail cliente</Button>
            {canManage && <Button renderIcon={Edit} onClick={openEdit}>Editar</Button>}
            {canManage && <Button kind="danger--ghost" renderIcon={TrashCan} onClick={() => setDeleteOpen(true)}>Excluir</Button>}
          </>
        }
      />

      <div className="nokia-detail-layout">
        <div className="nokia-detail-main">
          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Descrição</h2><p>Contexto e orientações da demanda.</p></div>
            </div>
            {demand.description ? (
              <p className="demand-detail__description">{demand.description}</p>
            ) : (
              <p className="muted">Nenhuma descrição informada.</p>
            )}
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Anexos</h2><p>{demand.attachments.length} arquivo(s) anexado(s).</p></div>
            </div>
            {canParticipate && (
              <div className="inline-form">
                <TextInput
                  id="demand-detail-files"
                  type="file"
                  multiple
                  labelText="Adicionar arquivos (até 50 MB cada)"
                  disabled={uploading}
                  onChange={(event) => void uploadAttachments(event.target.files)}
                />
              </div>
            )}
            {uploading && <InlineLoading description="Enviando arquivos…" />}
            {demand.attachments.length === 0 ? (
              <p className="muted">Nenhum arquivo anexado.</p>
            ) : (
              <div className="demand-attachments">
                {demand.attachments.map((file) => (
                  <div key={file.id} className="attachment-row">
                    <a href={file.download_url} target="_blank" rel="noreferrer">{file.original_name}</a>
                    <span>{formatBytes(file.size)} · {file.uploader_name ?? 'Usuário removido'} · {relativeTime(file.created_at)}</span>
                    {canParticipate && (
                      <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover arquivo" onClick={() => void deleteAttachment(file)} />
                    )}
                  </div>
                ))}
              </div>
            )}
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Histórico</h2><p>{demand.updates.length} atualização(ões).</p></div>
            </div>
            {canParticipate && (
              <div className="inline-form">
                <TextArea id="demand-detail-comment" labelText="Novo comentário" rows={2} value={comment} onChange={(event) => setComment(event.target.value)} />
                <Button kind="tertiary" renderIcon={Send} onClick={() => void addComment()}>Publicar</Button>
              </div>
            )}
            <div className="nokia-comment-list">
              {demand.updates.map((update) => (
                <div key={update.id} className="comment-row">
                  <div>
                    <strong>{update.author_name ?? 'Sistema'}</strong>
                    <span>{relativeTime(update.created_at)}</span>
                  </div>
                  <p>{update.body}</p>
                </div>
              ))}
            </div>
          </Tile>
        </div>

        <aside className="nokia-detail-aside">
          <Tile className="nokia-detail-aside__card">
            <h2>Resumo</h2>
            <dl>
              <div><dt>Solicitante</dt><dd>{demand.requester_name ?? '—'}</dd></div>
              <div><dt>Responsável</dt><dd>{demand.assignee_name ?? '—'}</dd></div>
              <div><dt>E-mail</dt><dd>{demand.assignee_email ?? '—'}</dd></div>
              <div><dt>Cliente</dt><dd>{demand.client_name ?? '—'}</dd></div>
              <div><dt>E-mail do cliente</dt><dd>{demand.client_email ?? '—'}</dd></div>
              <div><dt>Prazo</dt><dd>{demand.due_at ?? '—'}</dd></div>
              <div><dt>Criada em</dt><dd>{relativeTime(demand.created_at)}</dd></div>
              <div><dt>Atualizada em</dt><dd>{relativeTime(demand.updated_at)}</dd></div>
              {demand.completed_at && <div><dt>Concluída em</dt><dd>{relativeTime(demand.completed_at)}</dd></div>}
            </dl>
          </Tile>

          {canParticipate && (
            <Tile className="nokia-detail-aside__card">
              <h2>Ações</h2>
              <div className="demand-status-actions demand-status-actions--aside">
                {demand.status !== 'concluida' && demand.status !== 'cancelada' && (
                  <>
                    {demand.status !== 'em_andamento' && <Button kind="ghost" size="sm" renderIcon={Play} onClick={() => void changeStatus('em_andamento')}>Iniciar</Button>}
                    {demand.status !== 'aguardando' && <Button kind="ghost" size="sm" renderIcon={Pause} onClick={() => void changeStatus('aguardando')}>Aguardar</Button>}
                    <Button kind="ghost" size="sm" renderIcon={Checkmark} onClick={() => void changeStatus('concluida')}>Concluir</Button>
                  </>
                )}
                {(demand.status === 'concluida' || demand.status === 'cancelada') && (
                  <Button kind="ghost" size="sm" renderIcon={Restart} onClick={() => void changeStatus('aberta')}>Reabrir</Button>
                )}
                {canManage && demand.status !== 'cancelada' && demand.status !== 'concluida' && (
                  <Button kind="danger--ghost" size="sm" renderIcon={TrashCan} onClick={() => void changeStatus('cancelada')}>Cancelar</Button>
                )}
              </div>
            </Tile>
          )}
        </aside>
      </div>

      <Modal
        open={editOpen}
        modalHeading="Editar demanda"
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving || !form.title.trim() || !form.assignee_id}
        onRequestSubmit={() => void save()}
        onRequestClose={() => setEditOpen(false)}
        size="md"
      >
        <div className="modal-form">
          <TextInput id="demand-detail-title" labelText="Título *" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} required />
          <TextArea id="demand-detail-description" labelText="Descrição" rows={4} value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} />
          <div className="form-row">
            <TextInput id="demand-detail-client-name" labelText="Cliente" value={form.client_name} onChange={(event) => setForm({ ...form, client_name: event.target.value })} />
            <TextInput id="demand-detail-client-email" type="email" labelText="E-mail do cliente" value={form.client_email} onChange={(event) => setForm({ ...form, client_email: event.target.value })} />
          </div>
          <div className="form-row">
            <Select id="demand-detail-assignee" labelText="Responsável *" value={form.assignee_id} onChange={(event) => setForm({ ...form, assignee_id: event.target.value })}>
              <SelectItem value="" text="Selecione" />
              {collaborators.map((collaborator) => <SelectItem key={collaborator.id} value={String(collaborator.id)} text={`${collaborator.name} — ${collaborator.email}`} />)}
            </Select>
            <Select id="demand-detail-priority" labelText="Prioridade" value={form.priority} onChange={(event) => setForm({ ...form, priority: event.target.value })}>
              {Object.entries(DEMAND_PRIORITY).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
            </Select>
          </div>
          <div className="form-row">
            <TextInput id="demand-detail-due" type="date" labelText="Prazo" value={form.due_at} onChange={(event) => setForm({ ...form, due_at: event.target.value })} />
            <Select id="demand-detail-status" labelText="Status" value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })}>
              {Object.entries(DEMAND_STATUS).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
            </Select>
          </div>
        </div>
      </Modal>

      <ConfirmModal
        open={deleteOpen}
        title="Excluir demanda"
        message={`Excluir a demanda "${demand.title}"? O histórico e os anexos também serão removidos.`}
        confirmLabel="Excluir demanda"
        loading={deleting}
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => void removeDemand()}
      />
    </div>
  );
}
