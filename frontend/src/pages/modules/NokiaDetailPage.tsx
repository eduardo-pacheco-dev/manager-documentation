import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  Edit,
  TrashCan,
  Warning,
} from '@carbon/icons-react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  Checkbox,
  InlineLoading,
  Modal,
  ProgressBar,
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

interface ROM { id: number; first_name: string; last_name: string; email: string; phone: string }
interface Task { id: number; name: string; position: number; completed_at: string | null }
interface Rejection {
  id: number; checklist_field: string; reason: string; return_days: number;
  requires_field_return: boolean; started_at: string; resolved_at: string | null;
}
interface Comment { id: number; body: string; user_id: number | null; author_name: string | null; created_at: string }
interface Attachment { id: number; category: string; original_name: string; url: string; size: number; created_at: string }
interface NokiaSite {
  id: number; site_id: string; end_id: string | null; oc: string | null;
  os_fam_entrega: string | null; os_fam_instalacao: string | null;
  regional: string | null; subcon: string | null;
  interruption_reason: string | null; interruption_return_days: number | null; interruption_started_at: string | null;
  checklist_fam_entrega: boolean; checklist_fam_entrega_status: string;
  checklist_fam_instalacao: boolean; checklist_fam_instalacao_status: string;
  checklist_fam_panoramica: boolean; checklist_fam_panoramica_status: string;
  mos_plan_at: string | null; mos_real_at: string | null;
  instalacao_plan_at: string | null; instalacao_real_at: string | null;
  integracao_plan_at: string | null; integracao_real_at: string | null;
  rfa_plan_at: string | null; rfa_real_at: string | null;
  fam_entrega_approved_at: string | null; fam_instalacao_approved_at: string | null; fam_panoramica_approved_at: string | null;
  nokia_rom_id: number | null; rom_name: string | null;
  created_at: string; updated_at: string;
  tasks: Task[]; attachments: Attachment[]; comments: Comment[]; rejections: Rejection[];
}

const CHECKLISTS = [
  { field: 'checklist_fam_entrega', label: 'FAM Entrega', status: 'checklist_fam_entrega_status' },
  { field: 'checklist_fam_instalacao', label: 'FAM Instalação', status: 'checklist_fam_instalacao_status' },
  { field: 'checklist_fam_panoramica', label: 'FAM Panorâmica', status: 'checklist_fam_panoramica_status' },
];
const STATUS_LABEL: Record<string, string> = { pending: 'Pendente', analysis: 'Análise TIM', rejected: 'Rejeitado', sent: 'Aprovado' };
const FAM_LABEL: Record<string, string> = {
  checklist_fam_entrega: 'FAM Entrega',
  checklist_fam_instalacao: 'FAM Instalação',
  checklist_fam_panoramica: 'FAM Panorâmica',
};
const DATE_STAGES = [
  { label: 'MOS', plan: 'mos_plan_at', real: 'mos_real_at' },
  { label: 'Instalação', plan: 'instalacao_plan_at', real: 'instalacao_real_at' },
  { label: 'Integração', plan: 'integracao_plan_at', real: 'integracao_real_at' },
  { label: 'RFA', plan: 'rfa_plan_at', real: 'rfa_real_at' },
];
const APPROVAL_DATES = [
  ['fam_entrega_approved_at', 'FAM Entrega'],
  ['fam_instalacao_approved_at', 'FAM Instalação'],
  ['fam_panoramica_approved_at', 'FAM Panorâmica'],
];
const ALL_DATE_FIELDS = [
  ...DATE_STAGES.flatMap((stage) => [stage.plan, stage.real]),
  ...APPROVAL_DATES.map(([field]) => field),
];
const ATTACHMENT_CATEGORIES = [
  { value: 'tssr', label: 'TSSR' },
  { value: 'doc-d', label: 'DOC-D' },
  { value: 'dwg', label: 'DWG' },
];
const statusTag = (value: string) => value === 'sent' ? 'green' : value === 'rejected' ? 'red' : value === 'analysis' ? 'magenta' : 'gray';

const EMPTY_FORM = {
  site_id: '', end_id: '', oc: '', os_fam_entrega: '', os_fam_instalacao: '',
  regional: '', subcon: '', nokia_rom_id: '',
  checklist_fam_entrega: false, checklist_fam_instalacao: false, checklist_fam_panoramica: false,
};

function dateBR(value: string | null | undefined): string {
  if (!value) return '—';
  const date = new Date(value.length === 10 ? `${value}T00:00:00` : value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString('pt-BR');
}

function addDays(value: string | null | undefined, days: number | null | undefined): string {
  if (!value || days == null) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  date.setDate(date.getDate() + days);
  return date.toLocaleDateString('pt-BR');
}

export function NokiaDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const toast = useToast();
  const [site, setSite] = useState<NokiaSite | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [roms, setRoms] = useState<ROM[]>([]);
  const [editOpen, setEditOpen] = useState(false);
  const [form, setForm] = useState({ ...EMPTY_FORM });
  const [saving, setSaving] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [datesOpen, setDatesOpen] = useState(false);
  const [dates, setDates] = useState<Record<string, string>>({});
  const [interruptionOpen, setInterruptionOpen] = useState(false);
  const [interruption, setInterruption] = useState({ reason: '', return_days: 7 });
  const [rejectionOpen, setRejectionOpen] = useState(false);
  const [rejection, setRejection] = useState({ fams: [] as string[], reason: '', return_days: 7, requires_field_return: false });
  const [comment, setComment] = useState('');
  const [attachmentCategory, setAttachmentCategory] = useState('tssr');

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const response = await api.get<{ data: { site: NokiaSite } }>(`/nokia/sites/${id}`);
      const loaded = response.data.site;
      setSite({
        ...loaded,
        tasks: loaded.tasks ?? [],
        attachments: loaded.attachments ?? [],
        comments: loaded.comments ?? [],
        rejections: loaded.rejections ?? [],
      });
    } catch {
      setNotFound(true);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { void load(); }, [load]);
  useEffect(() => { api.get<{ data: ROM[] }>('/nokia/roms').then((response) => setRoms(response.data)).catch(() => undefined); }, []);

  function openEdit() {
    if (!site) return;
    setForm({
      site_id: site.site_id, end_id: site.end_id ?? '', oc: site.oc ?? '',
      os_fam_entrega: site.os_fam_entrega ?? '', os_fam_instalacao: site.os_fam_instalacao ?? '',
      regional: site.regional ?? '', subcon: site.subcon ?? '',
      nokia_rom_id: site.nokia_rom_id ? String(site.nokia_rom_id) : '',
      checklist_fam_entrega: site.checklist_fam_entrega,
      checklist_fam_instalacao: site.checklist_fam_instalacao,
      checklist_fam_panoramica: site.checklist_fam_panoramica,
    });
    setEditOpen(true);
  }

  async function saveEdit() {
    if (!site) return;
    setSaving(true);
    try {
      const payload = { ...form, nokia_rom_id: form.nokia_rom_id ? Number(form.nokia_rom_id) : null };
      await api.put(`/nokia/sites/${site.id}`, payload);
      toast.success('Site atualizado com sucesso.');
      setEditOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível atualizar o site', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function changeChecklist(field: string, status: string) {
    if (!site) return;
    try {
      await api.put(`/nokia/sites/${site.id}/checklist/${field}`, { status });
      toast.success(`${FAM_LABEL[field]} atualizado.`);
      await load();
    } catch (error) {
      toast.error('Não foi possível atualizar o checklist', errorMessage(error));
    }
  }

  async function toggleTask(task: Task) {
    if (!site) return;
    try {
      await api.post(`/nokia/sites/${site.id}/tasks/${task.id}`);
      await load();
    } catch (error) {
      toast.error('Não foi possível atualizar a tarefa', errorMessage(error));
    }
  }

  async function saveDates() {
    if (!site) return;
    try {
      await api.put(`/nokia/sites/${site.id}/dates`, dates);
      toast.success('Datas atualizadas com sucesso.');
      setDatesOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível atualizar as datas', errorMessage(error));
    }
  }

  async function saveInterruption() {
    if (!site) return;
    try {
      await api.put(`/nokia/sites/${site.id}/interruption`, interruption);
      toast.success('Interrupção registrada.');
      setInterruptionOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível registrar a interrupção', errorMessage(error));
    }
  }

  async function clearInterruption() {
    if (!site) return;
    try {
      await api.delete(`/nokia/sites/${site.id}/interruption`);
      toast.success('Interrupção encerrada.');
      setInterruptionOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível encerrar a interrupção', errorMessage(error));
    }
  }

  async function saveRejection() {
    if (!site) return;
    try {
      await api.post(`/nokia/sites/${site.id}/rejections`, rejection);
      toast.success('Rejeição registrada.');
      setRejectionOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível registrar a rejeição', errorMessage(error));
    }
  }

  async function resolveRejection(rejectionID: number) {
    if (!site) return;
    try {
      await api.patch(`/nokia/sites/${site.id}/rejections/${rejectionID}`);
      toast.success('Rejeição encerrada.');
      await load();
    } catch (error) {
      toast.error('Não foi possível encerrar a rejeição', errorMessage(error));
    }
  }

  async function uploadAttachments(files: FileList | null) {
    if (!site || !files || files.length === 0) return;
    const payload = new FormData();
    payload.append('category', attachmentCategory);
    Array.from(files).forEach((file) => payload.append('files', file));
    try {
      const response = await api.upload<{ message: string }>(`/nokia/sites/${site.id}/attachments`, payload);
      toast.success(response.message);
      await load();
    } catch (error) {
      toast.error('Não foi possível enviar o anexo', errorMessage(error));
    }
  }

  async function deleteAttachment(attachmentID: number) {
    if (!site) return;
    await api.delete(`/nokia/sites/${site.id}/attachments/${attachmentID}`);
    await load();
  }

  async function addComment() {
    if (!site || !comment.trim()) return;
    try {
      await api.post(`/nokia/sites/${site.id}/comments`, { body: comment });
      setComment('');
      await load();
    } catch (error) {
      toast.error('Não foi possível publicar o comentário', errorMessage(error));
    }
  }

  async function deleteComment(commentID: number) {
    if (!site) return;
    await api.delete(`/nokia/sites/${site.id}/comments/${commentID}`);
    await load();
  }

  async function deleteSite() {
    if (!site) return;
    setDeleting(true);
    try {
      await api.delete(`/nokia/sites/${site.id}`);
      toast.success('Site excluído com sucesso.');
      navigate('/nokia');
    } catch (error) {
      toast.error('Não foi possível excluir o site', errorMessage(error));
      setDeleting(false);
    }
  }

  if (loading) return <InlineLoading description="Carregando site…" />;

  if (notFound || !site) {
    return (
      <div className="status-page">
        <p className="status-page__code">404</p>
        <h1>Site não encontrado</h1>
        <p>O site pode ter sido excluído ou o endereço está incorreto.</p>
        <Link to="/nokia"><Button>Voltar para Projetos Nokia</Button></Link>
      </div>
    );
  }

  const checklistDone = CHECKLISTS.filter((item) => (site as unknown as Record<string, string>)[item.status] === 'sent').length;
  const taskDone = site.tasks.filter((task) => task.completed_at).length;
  const totalProgress = 3 + site.tasks.length;
  const doneProgress = checklistDone + taskDone;
  const progressPercent = totalProgress > 0 ? (doneProgress / totalProgress) * 100 : 0;

  return (
    <div className="page nokia-detail-page">
      <Breadcrumb noTrailingSlash className="doc-breadcrumb">
        <BreadcrumbItem><Link to="/nokia">Projetos Nokia</Link></BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>{site.site_id}</BreadcrumbItem>
      </Breadcrumb>

      <PageHeader
        eyebrow="Projetos Nokia"
        title={site.site_id}
        description={
          <span className="tag-row">
            {site.oc && <Tag type="blue" size="sm">OC {site.oc}</Tag>}
            {site.rom_name && <Tag type="purple" size="sm">{site.rom_name}</Tag>}
            {site.interruption_reason && <Tag type="red" size="sm">Obra interrompida</Tag>}
            <Tag type="gray" size="sm">{doneProgress}/{totalProgress} concluído</Tag>
          </span>
        }
        actions={
          <>
            <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/nokia')}>Voltar</Button>
            <Button renderIcon={Edit} onClick={openEdit}>Editar site</Button>
            <Button kind="danger--ghost" renderIcon={TrashCan} onClick={() => setDeleteOpen(true)}>Excluir</Button>
          </>
        }
      />

      <Tile className="nokia-progress-hero">
        <div>
          <p className="field-label">Progresso do site</p>
          <strong>{doneProgress} de {totalProgress} itens</strong>
        </div>
        <ProgressBar value={progressPercent} max={100} label="Progresso" helperText={`${progressPercent.toFixed(0)}% concluído`} hideLabel />
        <div className="nokia-progress-hero__meta">
          <div><span>Regional</span><strong>{site.regional ?? '—'}</strong></div>
          <div><span>Subcon</span><strong>{site.subcon ?? '—'}</strong></div>
          <div><span>OS Entrega</span><strong>{site.os_fam_entrega ?? '—'}</strong></div>
          <div><span>OS Instalação</span><strong>{site.os_fam_instalacao ?? '—'}</strong></div>
          <div><span>Criado em</span><strong>{dateBR(site.created_at)}</strong></div>
          <div><span>Atualizado</span><strong>{relativeTime(site.updated_at)}</strong></div>
        </div>
      </Tile>

      {site.interruption_reason && (
        <Tile className="nokia-interruption">
          <div>
            <Tag type="red" size="sm">Obra interrompida</Tag>
            <strong>{site.interruption_reason}</strong>
            <span>
              Início: {dateBR(site.interruption_started_at)} · Prazo: {site.interruption_return_days ?? '—'} dias ·
              Retorno previsto: {addDays(site.interruption_started_at, site.interruption_return_days)}
            </span>
          </div>
          <Button kind="ghost" size="sm" onClick={clearInterruption}>Encerrar interrupção</Button>
        </Tile>
      )}

      <div className="nokia-detail-layout">
        <div className="nokia-detail-main">
          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div>
                <h2>Checklist FAM</h2>
                <p>Status de entrega, instalação e panorâmica.</p>
              </div>
            </div>
            <div className="nokia-checklist-list">
              {CHECKLISTS.map((item) => {
                const status = (site as unknown as Record<string, string>)[item.status];
                return (
                  <div key={item.field} className="nokia-checklist-row">
                    <div>
                      <strong>{item.label}</strong>
                      <Tag type={statusTag(status)} size="sm">{STATUS_LABEL[status] ?? status}</Tag>
                    </div>
                    <Select
                      id={`detail-checklist-${item.field}`}
                      labelText="Status"
                      hideLabel
                      size="sm"
                      value={status}
                      onChange={(event) => void changeChecklist(item.field, event.target.value)}
                    >
                      {Object.entries(STATUS_LABEL).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
                    </Select>
                  </div>
                );
              })}
            </div>
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div>
                <h2>Datas</h2>
                <p>Marcos planejados, realizados e aprovações FAM.</p>
              </div>
              <Button
                kind="tertiary"
                size="sm"
                renderIcon={Edit}
                onClick={() => {
                  setDates(Object.fromEntries(ALL_DATE_FIELDS.map((field) => [field, String((site as unknown as Record<string, string>)[field] ?? '')])));
                  setDatesOpen(true);
                }}
              >
                Editar datas
              </Button>
            </div>
            <div className="table-scroll">
              <table className="data-table data-table--compact">
                <thead><tr><th>Etapa</th><th>Data Plan</th><th>Data Real</th></tr></thead>
                <tbody>
                  {DATE_STAGES.map((stage) => (
                    <tr key={stage.label}>
                      <td>{stage.label}</td>
                      <td>{dateBR((site as unknown as Record<string, string>)[stage.plan])}</td>
                      <td>{dateBR((site as unknown as Record<string, string>)[stage.real])}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="field-label nokia-approvals-label">Aprovações FAM</p>
            <div className="table-scroll">
              <table className="data-table data-table--compact">
                <thead><tr><th>FAM</th><th>Data de aprovação</th></tr></thead>
                <tbody>
                  {APPROVAL_DATES.map(([field, label]) => (
                    <tr key={field}><td>{label}</td><td>{dateBR((site as unknown as Record<string, string>)[field])}</td></tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div>
                <h2>Tarefas</h2>
                <p>{taskDone} de {site.tasks.length} tarefas concluídas.</p>
              </div>
            </div>
            {site.tasks.length === 0 ? (
              <p className="muted">Nenhuma tarefa cadastrada.</p>
            ) : (
              <div className="nokia-task-list">
                {site.tasks.map((task) => (
                  <Checkbox
                    key={task.id}
                    id={`detail-task-${task.id}`}
                    labelText={task.name}
                    checked={Boolean(task.completed_at)}
                    onChange={() => void toggleTask(task)}
                  />
                ))}
              </div>
            )}
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div>
                <h2>Rejeições</h2>
                <p>Histórico de rejeições por FAM e previsão de retorno.</p>
              </div>
              <Button
                kind="tertiary"
                size="sm"
                renderIcon={Warning}
                onClick={() => {
                  setRejection({ fams: [], reason: '', return_days: 7, requires_field_return: false });
                  setRejectionOpen(true);
                }}
              >
                Nova rejeição
              </Button>
            </div>
            {site.rejections.length === 0 ? (
              <p className="muted">Nenhuma rejeição registrada.</p>
            ) : (
              <div className="nokia-rejection-list">
                {site.rejections.map((rejectionItem) => (
                  <div key={rejectionItem.id} className="nokia-rejection-row">
                    <div>
                      <div className="tag-row">
                        <Tag type={rejectionItem.resolved_at ? 'gray' : 'red'} size="sm">
                          {rejectionItem.resolved_at ? 'Encerrada' : 'Ativa'}
                        </Tag>
                        <strong>{FAM_LABEL[rejectionItem.checklist_field] ?? rejectionItem.checklist_field}</strong>
                      </div>
                      <p>{rejectionItem.reason}</p>
                      <span>
                        {rejectionItem.return_days} dias · retorno previsto {addDays(rejectionItem.started_at, rejectionItem.return_days)}
                        {rejectionItem.requires_field_return ? ' · requer retorno a campo' : ''}
                      </span>
                    </div>
                    {!rejectionItem.resolved_at && (
                      <Button kind="ghost" size="sm" onClick={() => void resolveRejection(rejectionItem.id)}>Encerrar</Button>
                    )}
                  </div>
                ))}
              </div>
            )}
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div>
                <h2>Anexos</h2>
                <p>{site.attachments.length} arquivo(s) enviado(s).</p>
              </div>
            </div>
            <div className="inline-form nokia-upload-row">
              <Select id="detail-attachment-category" labelText="Categoria" value={attachmentCategory} onChange={(event) => setAttachmentCategory(event.target.value)}>
                {ATTACHMENT_CATEGORIES.map((category) => <SelectItem key={category.value} value={category.value} text={category.label} />)}
              </Select>
              <TextInput
                id="detail-attachment-files"
                type="file"
                labelText="Arquivos (até 20 MB cada)"
                multiple
                onChange={(event) => void uploadAttachments(event.target.files)}
              />
            </div>
            {ATTACHMENT_CATEGORIES.map((category) => {
              const files = site.attachments.filter((attachment) => attachment.category === category.value);
              if (files.length === 0) return null;
              return (
                <div key={category.value} className="nokia-attachment-group">
                  <p className="field-label">{category.label} ({files.length})</p>
                  {files.map((attachment) => (
                    <div key={attachment.id} className="attachment-row">
                      <a href={attachment.url} target="_blank" rel="noreferrer">{attachment.original_name}</a>
                      <span>{relativeTime(attachment.created_at)}</span>
                      <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover anexo" onClick={() => void deleteAttachment(attachment.id)} />
                    </div>
                  ))}
                </div>
              );
            })}
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div>
                <h2>Comentários</h2>
                <p>{site.comments.length} comentário(s).</p>
              </div>
            </div>
            <div className="inline-form">
              <TextArea id="detail-comment" labelText="Novo comentário" value={comment} onChange={(event) => setComment(event.target.value)} rows={2} />
              <Button kind="tertiary" onClick={() => void addComment()}>Comentar</Button>
            </div>
            {site.comments.length === 0 ? (
              <p className="muted">Nenhum comentário ainda.</p>
            ) : (
              <div className="nokia-comment-list">
                {site.comments.map((commentItem) => (
                  <div key={commentItem.id} className="comment-row">
                    <div>
                      <strong>{commentItem.author_name ?? 'Usuário removido'}</strong>
                      <span>{relativeTime(commentItem.created_at)}</span>
                    </div>
                    <p>{commentItem.body}</p>
                    {(user?.is_admin || commentItem.user_id === user?.id) && (
                      <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover comentário" onClick={() => void deleteComment(commentItem.id)} />
                    )}
                  </div>
                ))}
              </div>
            )}
          </Tile>
        </div>

        <aside className="nokia-detail-aside">
          <Tile className="nokia-detail-aside__card">
            <h2>Resumo</h2>
            <dl>
              <div><dt>Site ID</dt><dd>{site.site_id}</dd></div>
              <div><dt>End ID</dt><dd>{site.end_id ?? '—'}</dd></div>
              <div><dt>OC</dt><dd>{site.oc ?? '—'}</dd></div>
              <div><dt>ROM</dt><dd>{site.rom_name ?? 'Sem ROM'}</dd></div>
              <div><dt>Regional</dt><dd>{site.regional ?? '—'}</dd></div>
              <div><dt>Subcon</dt><dd>{site.subcon ?? '—'}</dd></div>
            </dl>
          </Tile>
          <Tile className="nokia-detail-aside__card">
            <h2>Ações</h2>
            <div className="nokia-aside-actions">
              <Button kind="ghost" size="sm" renderIcon={Warning} onClick={() => { setInterruption({ reason: site.interruption_reason ?? '', return_days: site.interruption_return_days ?? 7 }); setInterruptionOpen(true); }}>Interrupção</Button>
              <Button kind="ghost" size="sm" renderIcon={Edit} onClick={openEdit}>Editar site</Button>
              <Button kind="danger--ghost" size="sm" renderIcon={TrashCan} onClick={() => setDeleteOpen(true)}>Excluir site</Button>
            </div>
          </Tile>
        </aside>
      </div>

      <Modal open={editOpen} modalHeading="Editar site" primaryButtonText={saving ? 'Salvando…' : 'Salvar'} secondaryButtonText="Cancelar" primaryButtonDisabled={saving} onRequestSubmit={() => void saveEdit()} onRequestClose={() => setEditOpen(false)} size="md">
        <div className="modal-form">
          <div className="form-row">
            <TextInput id="detail-edit-site" labelText="Site ID *" value={form.site_id} onChange={(event) => setForm({ ...form, site_id: event.target.value })} required />
            <TextInput id="detail-edit-end" labelText="End ID" value={form.end_id} onChange={(event) => setForm({ ...form, end_id: event.target.value })} />
          </div>
          <div className="form-row">
            <TextInput id="detail-edit-oc" labelText="OC" value={form.oc} onChange={(event) => setForm({ ...form, oc: event.target.value })} />
            <Select id="detail-edit-rom" labelText="ROM" value={form.nokia_rom_id} onChange={(event) => setForm({ ...form, nokia_rom_id: event.target.value })}>
              <SelectItem value="" text="Sem ROM" />
              {roms.map((rom) => <SelectItem key={rom.id} value={String(rom.id)} text={`${rom.first_name} ${rom.last_name}`} />)}
            </Select>
          </div>
          <div className="form-row">
            <TextInput id="detail-edit-os-e" labelText="OS FAM Entrega" value={form.os_fam_entrega} onChange={(event) => setForm({ ...form, os_fam_entrega: event.target.value })} />
            <TextInput id="detail-edit-os-i" labelText="OS FAM Instalação" value={form.os_fam_instalacao} onChange={(event) => setForm({ ...form, os_fam_instalacao: event.target.value })} />
          </div>
          <div className="form-row">
            <TextInput id="detail-edit-regional" labelText="Regional" value={form.regional} onChange={(event) => setForm({ ...form, regional: event.target.value })} />
            <TextInput id="detail-edit-subcon" labelText="Subcon" value={form.subcon} onChange={(event) => setForm({ ...form, subcon: event.target.value })} />
          </div>
          <Checkbox id="detail-check-e" labelText="FAM Entrega enviada" checked={form.checklist_fam_entrega} onChange={(_event, { checked }) => setForm({ ...form, checklist_fam_entrega: checked })} />
          <Checkbox id="detail-check-i" labelText="FAM Instalação enviada" checked={form.checklist_fam_instalacao} onChange={(_event, { checked }) => setForm({ ...form, checklist_fam_instalacao: checked })} />
          <Checkbox id="detail-check-p" labelText="FAM Panorâmica enviada" checked={form.checklist_fam_panoramica} onChange={(_event, { checked }) => setForm({ ...form, checklist_fam_panoramica: checked })} />
        </div>
      </Modal>

      <Modal open={datesOpen} modalHeading="Datas do site" primaryButtonText="Salvar" secondaryButtonText="Cancelar" onRequestSubmit={() => void saveDates()} onRequestClose={() => setDatesOpen(false)} size="md">
        <div className="nokia-date-editor">
          {DATE_STAGES.map((stage) => (
            <div key={stage.label} className="nokia-date-stage">
              <p className="field-label">{stage.label}</p>
              <div className="form-row">
                <TextInput id={`detail-date-${stage.plan}`} type="date" labelText="Data Plan" value={dates[stage.plan] ?? ''} onChange={(event) => setDates({ ...dates, [stage.plan]: event.target.value })} />
                <TextInput id={`detail-date-${stage.real}`} type="date" labelText="Data Real" value={dates[stage.real] ?? ''} onChange={(event) => setDates({ ...dates, [stage.real]: event.target.value })} />
              </div>
            </div>
          ))}
          <div className="nokia-date-stage">
            <p className="field-label">Aprovações FAM</p>
            <div className="form-grid">
              {APPROVAL_DATES.map(([field, label]) => (
                <TextInput key={field} id={`detail-date-${field}`} type="date" labelText={label} value={dates[field] ?? ''} onChange={(event) => setDates({ ...dates, [field]: event.target.value })} />
              ))}
            </div>
          </div>
        </div>
      </Modal>

      <Modal open={interruptionOpen} modalHeading="Interrupção de obra" primaryButtonText="Salvar" secondaryButtonText="Cancelar" onRequestSubmit={() => void saveInterruption()} onRequestClose={() => setInterruptionOpen(false)} size="sm">
        <div className="modal-form">
          <TextArea id="detail-int-reason" labelText="Motivo" value={interruption.reason} onChange={(event) => setInterruption({ ...interruption, reason: event.target.value })} rows={3} />
          <Select id="detail-int-days" labelText="Prazo de retorno" value={String(interruption.return_days)} onChange={(event) => setInterruption({ ...interruption, return_days: Number(event.target.value) })}>
            {[7, 14, 21, 28].map((days) => <SelectItem key={days} value={String(days)} text={`${days} dias`} />)}
          </Select>
          {site.interruption_reason && <Button kind="danger--ghost" onClick={() => void clearInterruption()}>Encerrar interrupção</Button>}
        </div>
      </Modal>

      <Modal open={rejectionOpen} modalHeading="Registrar rejeição" primaryButtonText="Registrar" secondaryButtonText="Cancelar" onRequestSubmit={() => void saveRejection()} onRequestClose={() => setRejectionOpen(false)} size="sm">
        <div className="modal-form">
          {CHECKLISTS.map((item) => (
            <Checkbox
              key={item.field}
              id={`detail-rej-${item.field}`}
              labelText={item.label}
              checked={rejection.fams.includes(item.field)}
              onChange={(_event, { checked }) => setRejection({ ...rejection, fams: checked ? [...rejection.fams, item.field] : rejection.fams.filter((field) => field !== item.field) })}
            />
          ))}
          <TextArea id="detail-rej-reason" labelText="Motivo" value={rejection.reason} onChange={(event) => setRejection({ ...rejection, reason: event.target.value })} rows={3} />
          <Select id="detail-rej-days" labelText="Prazo de retorno" value={String(rejection.return_days)} onChange={(event) => setRejection({ ...rejection, return_days: Number(event.target.value) })}>
            {[7, 14, 21, 28].map((days) => <SelectItem key={days} value={String(days)} text={`${days} dias`} />)}
          </Select>
          <Checkbox id="detail-rej-field" labelText="Necessário retorno a campo" checked={rejection.requires_field_return} onChange={(_event, { checked }) => setRejection({ ...rejection, requires_field_return: checked })} />
        </div>
      </Modal>

      <ConfirmModal
        open={deleteOpen}
        title="Excluir site"
        message={`Excluir o site ${site.site_id}? Essa ação não pode ser desfeita.`}
        confirmLabel="Excluir site"
        loading={deleting}
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => void deleteSite()}
      />
    </div>
  );
}
