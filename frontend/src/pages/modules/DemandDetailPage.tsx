import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  Checkmark,
  Copy,
  Launch,
  OverflowMenuVertical,
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
  OverflowMenu,
  OverflowMenuItem,
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
import { StationSearch } from '../../components/StationSearch';
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
  created_at: string; uploader_name: string | null; uploaded_by_name: string | null; download_url: string;
}
interface Demand {
  id: number; requester_id: number; assignee_id: number; title: string;
  description: string | null; status: string; priority: string; due_at: string | null;
  completed_at: string | null; created_at: string; updated_at: string;
  client_name: string | null; client_email: string | null;
  station_id: number | null; station_site_id: string | null; station_municipio: string | null; station_estado: string | null;
  station_endereco_id: string | null; station_tecnologia: string | null; station_tipo_elemento: string | null; station_regional: string | null;
  operadora: string | null; demand_type: string | null; oc_tim: string | null;
  client_os: string | null; key_number: string | null; activity_scope: string | null;
  requester_name: string | null; assignee_name: string | null; assignee_email: string | null;
  overdue: boolean; update_count: number; updates: Update[]; attachments: AttachmentFile[];
  upload_url?: string; download_url?: string;
}

const EMPTY_FORM = {
  title: '', description: '', assignee_id: '', priority: 'media', due_at: '', status: 'aberta',
  client_name: '', client_email: '', station_id: '', operadora: '', demand_type: '',
  oc_tim: '', client_os: '', key_number: '', activity_scope: '',
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
      station_id: demand.station_id ? String(demand.station_id) : '',
      operadora: demand.operadora ?? '', demand_type: demand.demand_type ?? '',
      oc_tim: demand.oc_tim ?? '', client_os: demand.client_os ?? '',
      key_number: demand.key_number ?? '', activity_scope: demand.activity_scope ?? '',
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
        station_id: form.station_id ? Number(form.station_id) : null,
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
    const project = (demand.demand_type || 'DEMANDA').toUpperCase();
    const operator = demand.operadora || demand.client_name || 'ATIVIDADE';
    const site = demand.station_site_id || '—';
    const technology = demand.station_tecnologia ? ` ${demand.station_tecnologia}` : '';
    const endID = demand.station_endereco_id ? ` / ${demand.station_endereco_id}` : '';
    const subject = `DISPARO DE ATIVIDADE_${project} ${demand.oc_tim || '—'} ${site}${technology}${endID} :: ${operator.toUpperCase()}`;
    const hour = new Date().getHours();
    const greeting = hour < 12 ? 'Bom dia!' : hour < 18 ? 'Boa tarde!' : 'Boa noite!';
    const dueDate = demand.due_at
      ? new Date(`${demand.due_at}T12:00:00`).toLocaleDateString('pt-BR')
      : 'a combinar';
    const columns = ['OC', 'Status_OC', 'Tipo_Site', 'Site_ID', 'End_ID', 'Classe_Casa', 'Regional', 'Tecnologia', 'Escopo', 'Vendor_HW'];
    const row = [
      demand.oc_tim || '—',
      DEMAND_STATUS[demand.status] ?? demand.status,
      demand.station_tipo_elemento || '—',
      demand.station_site_id || '—',
      demand.station_endereco_id || '—',
      '—',
      demand.station_regional || '—',
      demand.station_tecnologia || '—',
      demand.demand_type || '—',
      '—',
    ];
    const lines = [
      `Para: ${demand.assignee_email ?? '—'}`,
      `Assunto: ${subject}`,
      '',
      greeting,
      '',
      `@${demand.assignee_name ?? 'equipe'} | ${demand.client_name || demand.operadora || 'equipe'} favor atender a demanda "${demand.title}" referente ao projeto ${project}.`,
      '',
    ];
    if (demand.description?.trim()) {
      lines.push('Descrição:', demand.description.trim(), '');
    }
    if (demand.activity_scope?.trim()) {
      lines.push('Escopo da atividade:', demand.activity_scope.trim(), '');
    }
    lines.push(
      `Seguir documentação padrão do projeto ${project}.`,
      '',
      'Prazo: ' + dueDate,
      '',
      columns.join('\t'),
      row.join('\t'),
      '',
      'Acesse a demanda:',
      `${window.location.origin}/demands/${demand.id}`,
      '',
      'Atenciosamente,',
      'Documentation Center',
    );
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

  async function copyText(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      toast.success('Link copiado.');
    } catch {
      toast.error('Não foi possível copiar o link', value);
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
            <div className="demand-actions-menu">
              <OverflowMenu
                aria-label="Ações da demanda"
                iconDescription="Ações da demanda"
                renderIcon={OverflowMenuVertical}
                direction="bottom"
                autoAlign
              >
                <OverflowMenuItem itemText="Copiar e-mail" onClick={() => void copyDispatchEmail()} />
                <OverflowMenuItem itemText="Copiar e-mail cliente" onClick={() => void copyClientEmail()} />
                {canManage && <OverflowMenuItem itemText="Editar" onClick={openEdit} />}
                {canManage && <OverflowMenuItem itemText="Excluir" isDelete onClick={() => setDeleteOpen(true)} />}
              </OverflowMenu>
            </div>
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
              <div><h2>Dados operacionais</h2><p>Estação, operadora e referências do acionamento.</p></div>
            </div>
            <dl className="demand-info-grid">
              <div>
                <dt>Estação</dt>
                <dd>
                  {demand.station_id && demand.station_site_id ? (
                    <Link to={`/stations/${demand.station_id}`}>{demand.station_site_id}</Link>
                  ) : '—'}
                </dd>
              </div>
              <div><dt>Operadora</dt><dd>{demand.operadora ?? '—'}</dd></div>
              <div><dt>Tipo de demanda</dt><dd>{demand.demand_type ?? '—'}</dd></div>
              <div><dt>OC TIM</dt><dd>{demand.oc_tim ?? '—'}</dd></div>
              <div><dt>OS do cliente</dt><dd>{demand.client_os ?? '—'}</dd></div>
              <div><dt>Número da chave</dt><dd>{demand.key_number ?? '—'}</dd></div>
            </dl>
          </Tile>

          {demand.activity_scope && (
            <Tile className="nokia-detail-section">
              <div className="nokia-section-header">
                <div><h2>Escopo da atividade</h2><p>Detalhamento operacional solicitado.</p></div>
              </div>
              <p className="demand-detail__description">{demand.activity_scope}</p>
            </Tile>
          )}

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
                    <span>{formatBytes(file.size)} · {file.uploader_name ?? file.uploaded_by_name ?? 'Link público'} · {relativeTime(file.created_at)}</span>
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
              <div><dt>Estação</dt><dd>{demand.station_site_id ?? '—'}</dd></div>
              <div><dt>Operadora</dt><dd>{demand.operadora ?? '—'}</dd></div>
              <div><dt>Tipo</dt><dd>{demand.demand_type ?? '—'}</dd></div>
              <div><dt>OC TIM</dt><dd>{demand.oc_tim ?? '—'}</dd></div>
              <div><dt>Prazo</dt><dd>{demand.due_at ?? '—'}</dd></div>
              <div><dt>Criada em</dt><dd>{relativeTime(demand.created_at)}</dd></div>
              <div><dt>Atualizada em</dt><dd>{relativeTime(demand.updated_at)}</dd></div>
              {demand.completed_at && <div><dt>Concluída em</dt><dd>{relativeTime(demand.completed_at)}</dd></div>}
            </dl>
          </Tile>

          {canManage && (
            <Tile className="nokia-detail-aside__card">
              <h2>Links externos</h2>
              <p className="muted">Compartilhe o link de envio com o colaborador e o de download com o cliente.</p>
              {demand.upload_url && (
                <div className="demand-public-link">
                  <span>Upload do colaborador</span>
                  <code>{demand.upload_url}</code>
                  <div>
                    <Button kind="ghost" size="xs" hasIconOnly renderIcon={Copy} iconDescription="Copiar link de upload" onClick={() => void copyText(demand.upload_url!)} />
                    <Button kind="ghost" size="xs" hasIconOnly renderIcon={Launch} iconDescription="Abrir link de upload" onClick={() => window.open(demand.upload_url, '_blank')} />
                  </div>
                </div>
              )}
              {demand.download_url && (
                <div className="demand-public-link">
                  <span>Download do cliente</span>
                  <code>{demand.download_url}</code>
                  <div>
                    <Button kind="ghost" size="xs" hasIconOnly renderIcon={Copy} iconDescription="Copiar link de download" onClick={() => void copyText(demand.download_url!)} />
                    <Button kind="ghost" size="xs" hasIconOnly renderIcon={Launch} iconDescription="Abrir link de download" onClick={() => window.open(demand.download_url, '_blank')} />
                  </div>
                </div>
              )}
            </Tile>
          )}

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
          <StationSearch
            id="demand-detail-station"
            value={form.station_id ? Number(form.station_id) : null}
            valueLabel={demand.station_site_id}
            onChange={(stationId) => setForm({ ...form, station_id: stationId ? String(stationId) : '' })}
          />
          <div className="form-row">
            <TextInput id="demand-detail-operadora" labelText="Operadora" value={form.operadora} onChange={(event) => setForm({ ...form, operadora: event.target.value })} />
            <TextInput id="demand-detail-type" labelText="Tipo de demanda" value={form.demand_type} onChange={(event) => setForm({ ...form, demand_type: event.target.value })} />
          </div>
          <div className="form-row">
            <TextInput id="demand-detail-oc-tim" labelText="Número OC TIM" value={form.oc_tim} onChange={(event) => setForm({ ...form, oc_tim: event.target.value })} />
            <TextInput id="demand-detail-client-os" labelText="Número OS do Cliente" value={form.client_os} onChange={(event) => setForm({ ...form, client_os: event.target.value })} />
          </div>
          <TextInput id="demand-detail-key" labelText="Número da chave" value={form.key_number} onChange={(event) => setForm({ ...form, key_number: event.target.value })} />
          <TextArea id="demand-detail-scope" labelText="Escopo da atividade" rows={3} value={form.activity_scope} onChange={(event) => setForm({ ...form, activity_scope: event.target.value })} />
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
