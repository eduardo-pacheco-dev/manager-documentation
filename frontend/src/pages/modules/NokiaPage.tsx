import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Add, ArrowDown, ArrowUp, Edit, OverflowMenuVertical, Settings, TrashCan, Warning } from '@carbon/icons-react';
import {
  Button,
  Checkbox,
  ComboBox,
  InlineLoading,
  Modal,
  OverflowMenu,
  OverflowMenuItem,
  Search,
  Select,
  SelectItem,
  Tag,
  TextArea,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState } from '../../components/Feedback';
import { useAuth } from '../../contexts/AuthContext';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';

interface ROM { id: number; first_name: string; last_name: string; email: string; phone: string }
interface StationSiteOption {
  site_id: string;
  endereco_id: string | null;
  regional: string | null;
  municipio: string | null;
  estado: string | null;
}
interface Task { id: number; name: string; position: number; completed_at: string | null }
interface Rejection { id: number; checklist_field: string; reason: string; return_days: number; started_at: string; resolved_at: string | null; requires_field_return: boolean }
interface Comment { id: number; body: string; user_id: number | null; author_name: string | null; created_at: string }
interface Attachment { id: number; category: string; original_name: string; url: string; created_at: string }
interface Site {
  id: number; site_id: string; end_id: string | null; oc: string | null; os_fam_entrega: string | null;
  os_fam_instalacao: string | null; regional: string | null; subcon: string | null;
  interruption_reason: string | null; interruption_return_days: number | null; interruption_started_at: string | null;
  checklist_fam_entrega_status: string; checklist_fam_instalacao_status: string; checklist_fam_panoramica_status: string;
  checklist_fam_entrega: boolean; checklist_fam_instalacao: boolean; checklist_fam_panoramica: boolean;
  mos_plan_at: string | null; mos_real_at: string | null; instalacao_plan_at: string | null; instalacao_real_at: string | null;
  integracao_plan_at: string | null; integracao_real_at: string | null; rfa_plan_at: string | null; rfa_real_at: string | null;
  fam_entrega_approved_at: string | null; fam_instalacao_approved_at: string | null; fam_panoramica_approved_at: string | null;
  nokia_rom_id: number | null; rom_name: string | null; tasks: Task[]; attachments: Attachment[]; comments: Comment[]; rejections: Rejection[];
  created_at: string; updated_at: string;
}
interface Data { data: Site[]; stats: { total: number; completed: number; pending: number }; tasks_total: number; meta: { total: number } }

const CHECKLISTS = [
  { field: 'checklist_fam_entrega', label: 'FAM Entrega', status: 'checklist_fam_entrega_status' },
  { field: 'checklist_fam_instalacao', label: 'FAM Instalação', status: 'checklist_fam_instalacao_status' },
  { field: 'checklist_fam_panoramica', label: 'FAM Panorâmica', status: 'checklist_fam_panoramica_status' },
];
const STATUS_LABEL: Record<string, string> = { pending: 'Pendente', analysis: 'Análise TIM', rejected: 'Rejeitado', sent: 'Aprovado' };
const statusTag = (value: string) => value === 'sent' ? 'green' : value === 'rejected' ? 'red' : value === 'analysis' ? 'magenta' : 'gray';
const DATE_FIELDS = [
  ['mos_plan_at', 'MOS Plan'], ['mos_real_at', 'MOS Real'], ['instalacao_plan_at', 'Instalação Plan'],
  ['instalacao_real_at', 'Instalação Real'], ['integracao_plan_at', 'Integração Plan'], ['integracao_real_at', 'Integração Real'],
  ['rfa_plan_at', 'RFA Plan'], ['rfa_real_at', 'RFA Real'], ['fam_entrega_approved_at', 'FAM Entrega Aprovado'],
  ['fam_instalacao_approved_at', 'FAM Instalação Aprovado'], ['fam_panoramica_approved_at', 'FAM Panorâmica Aprovado'],
];
const FAM_LABEL: Record<string, string> = {
  checklist_fam_entrega: 'FAM Entrega',
  checklist_fam_instalacao: 'FAM Instalação',
  checklist_fam_panoramica: 'FAM Panorâmica',
};
const ATTACHMENT_CATEGORIES = [
  { value: 'tssr', label: 'TSSR' },
  { value: 'doc-d', label: 'DOC-D' },
  { value: 'dwg', label: 'DWG' },
  { value: 'invoice', label: 'Nota Fiscal' },
  { value: 'other', label: 'Outros' },
];

function dateBR(value: string | null | undefined): string {
  if (!value) return '—';
  const date = new Date(value.length === 10 ? `${value}T00:00:00` : value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString('pt-BR');
}

function addDays(value: string | null | undefined, days: number | null | undefined): string {
  if (!value || days == null) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  date.setDate(date.getDate() + days);
  return date.toLocaleDateString('pt-BR');
}

const EMPTY_FORM = {
  site_id: '', end_id: '', oc: '', os_fam_entrega: '', os_fam_instalacao: '', regional: '', subcon: '',
  nokia_rom_id: '', checklist_fam_entrega: false, checklist_fam_instalacao: false, checklist_fam_panoramica: false,
};

export function NokiaPage() {
  const { user } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [data, setData] = useState<Data | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [roms, setRoms] = useState<ROM[]>([]);
  const [stationSites, setStationSites] = useState<StationSiteOption[]>([]);
  const [modal, setModal] = useState(false);
  const [editing, setEditing] = useState<Site | null>(null);
  const [detail, setDetail] = useState<Site | null>(null);
  const [form, setForm] = useState({ ...EMPTY_FORM });
  const [saving, setSaving] = useState(false);
  const [datesOpen, setDatesOpen] = useState(false);
  const [dates, setDates] = useState<Record<string, string>>({});
  const [interruptionOpen, setInterruptionOpen] = useState(false);
  const [interruption, setInterruption] = useState({ reason: '', return_days: 7 });
  const [rejectionOpen, setRejectionOpen] = useState(false);
  const [rejection, setRejection] = useState({ fams: [] as string[], reason: '', return_days: 7, requires_field_return: false });
  const [comment, setComment] = useState('');
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settings, setSettings] = useState<{ roms: ROM[]; tasks: Task[] }>({ roms: [], tasks: [] });
  const [romForm, setRomForm] = useState({ first_name: '', last_name: '', email: '', phone: '' });
  const [romEditingId, setRomEditingId] = useState<number | null>(null);
  const [taskName, setTaskName] = useState('');
  const [attachmentCategory, setAttachmentCategory] = useState('tssr');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.get<Data>(`/nokia/sites?search=${encodeURIComponent(search)}`);
      setData(response);
    } catch (error) {
      toast.error('Não foi possível carregar os sites', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [search, toast]);

  useEffect(() => { void load(); }, [load]);
  useEffect(() => {
    api.get<{ data: ROM[] }>('/nokia/roms').then((response) => setRoms(response.data)).catch(() => undefined);
    api.get<{ data: StationSiteOption[] }>('/nokia/station-sites').then((response) => setStationSites(response.data)).catch(() => undefined);
  }, []);

  function openCreate() {
    setEditing(null);
    setForm({ ...EMPTY_FORM });
    setModal(true);
  }

  function openEdit(site: Site) {
    setEditing(site);
    setForm({
      site_id: site.site_id, end_id: site.end_id ?? '', oc: site.oc ?? '', os_fam_entrega: site.os_fam_entrega ?? '',
      os_fam_instalacao: site.os_fam_instalacao ?? '', regional: site.regional ?? '', subcon: site.subcon ?? '',
      nokia_rom_id: site.nokia_rom_id ? String(site.nokia_rom_id) : '',
      checklist_fam_entrega: site.checklist_fam_entrega, checklist_fam_instalacao: site.checklist_fam_instalacao,
      checklist_fam_panoramica: site.checklist_fam_panoramica,
    });
    setModal(true);
  }

  async function save() {
    setSaving(true);
    try {
      const payload = { ...form, nokia_rom_id: form.nokia_rom_id ? Number(form.nokia_rom_id) : null };
      if (editing) await api.put(`/nokia/sites/${editing.id}`, payload);
      else await api.post('/nokia/sites', payload);
      toast.success(editing ? 'Site atualizado.' : 'Site cadastrado.');
      setModal(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function openDetail(site: Site) {
    try {
      const response = await api.get<{ data: { site: Site } }>(`/nokia/sites/${site.id}`);
      const loaded = response.data.site;
      setDetail({
        ...loaded,
        tasks: loaded.tasks ?? [],
        attachments: loaded.attachments ?? [],
        comments: loaded.comments ?? [],
        rejections: loaded.rejections ?? [],
      });
    } catch (error) {
      toast.error('Não foi possível abrir o site', errorMessage(error));
    }
  }

  async function changeChecklist(field: string, status: string) {
    if (!detail) return;
    await api.put(`/nokia/sites/${detail.id}/checklist/${field}`, { status });
    await openDetail(detail);
    await load();
  }

  async function toggleTask(task: Task) {
    if (!detail) return;
    await api.post(`/nokia/sites/${detail.id}/tasks/${task.id}`);
    await openDetail(detail);
    await load();
  }

  async function saveDates() {
    if (!detail) return;
    await api.put(`/nokia/sites/${detail.id}/dates`, dates);
    setDatesOpen(false);
    await openDetail(detail);
    await load();
  }

  async function saveInterruption() {
    if (!detail) return;
    await api.put(`/nokia/sites/${detail.id}/interruption`, interruption);
    setInterruptionOpen(false);
    await openDetail(detail);
    await load();
  }

  async function saveRejection() {
    if (!detail) return;
    await api.post(`/nokia/sites/${detail.id}/rejections`, rejection);
    setRejectionOpen(false);
    await openDetail(detail);
  }

  async function openSettings() {
    const response = await api.get<{ data: { roms: ROM[]; tasks: Task[] } }>('/nokia/settings');
    setSettings(response.data);
    setSettingsOpen(true);
  }

  function importSites() {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.csv,.xlsx';
    input.onchange = async () => {
      if (!input.files?.[0]) return;
      const formData = new FormData();
      formData.append('file', input.files[0]);
      try {
        const response = await api.upload<{ message: string }>('/nokia/sites/import', formData);
        toast.success(response.message);
        await load();
      } catch (error) {
        toast.error('Falha na importação', errorMessage(error));
      }
    };
    input.click();
  }

  async function reloadNokiaSettings() {
    const response = await api.get<{ data: { roms: ROM[]; tasks: Task[] } }>('/nokia/settings');
    setSettings(response.data);
    const publicRoms = await api.get<{ data: ROM[] }>('/nokia/roms');
    setRoms(publicRoms.data);
  }

  async function saveROM() {
    if (romEditingId) {
      await api.put(`/nokia/settings/roms/${romEditingId}`, romForm);
      toast.success('ROM atualizado com sucesso.');
    } else {
      await api.post('/nokia/settings/roms', romForm);
      toast.success('ROM cadastrado com sucesso.');
    }
    setRomForm({ first_name: '', last_name: '', email: '', phone: '' });
    setRomEditingId(null);
    await reloadNokiaSettings();
  }

  async function saveTask() {
    await api.post('/nokia/settings/tasks', { name: taskName });
    setTaskName('');
    await reloadNokiaSettings();
  }

  async function renameTask(task: Task) {
    const name = window.prompt('Nome da tarefa', task.name);
    if (!name || name.trim() === '' || name === task.name) return;
    await api.put(`/nokia/settings/tasks/${task.id}`, { name });
    await reloadNokiaSettings();
  }

  async function moveTask(task: Task, direction: 'up' | 'down') {
    await api.patch(`/nokia/settings/tasks/${task.id}/move`, { direction });
    await reloadNokiaSettings();
  }

  async function addComment() {
    if (!detail || !comment.trim()) return;
    await api.post(`/nokia/sites/${detail.id}/comments`, { body: comment });
    setComment('');
    await openDetail(detail);
  }

  const items = data?.data ?? [];

  return (
    <div className="page">
      <PageHeader
        eyebrow="Projetos"
        title="Projetos Nokia"
        description="Sites, checklist de FAMs, tarefas, datas, rejeições e interrupções."
        titleAccessory={user?.is_admin ? (
          <Button
            kind="ghost"
            size="md"
            hasIconOnly
            renderIcon={Settings}
            iconDescription="Configurações Nokia"
            onClick={() => void openSettings()}
          />
        ) : undefined}
        actions={
          <>
            <Button renderIcon={Add} onClick={openCreate}>Novo site</Button>
            <OverflowMenu
              aria-label="Ações de importação e exportação"
              iconDescription="Mais ações"
              renderIcon={OverflowMenuVertical}
              size="md"
              flipped
            >
              <OverflowMenuItem
                itemText="Exportar"
                onClick={() => window.open(`/api/v1/nokia/sites/export?search=${encodeURIComponent(search)}`, '_blank')}
              />
              <OverflowMenuItem
                itemText="Pendências"
                onClick={() => window.open('/api/v1/nokia/sites/export/pending', '_blank')}
              />
              <OverflowMenuItem
                itemText="Modelo XLSX"
                onClick={() => window.open('/api/v1/nokia/sites/template', '_blank')}
              />
              <OverflowMenuItem
                itemText="Importar"
                hasDivider
                onClick={importSites}
              />
            </OverflowMenu>
          </>
        }
      />

      {data && (
        <div className="stats-grid stats-grid--3">
          <Tile className="stat-card stat-card--blue"><span className="stat-card__label">Sites</span><strong className="stat-card__value">{data.stats.total}</strong></Tile>
          <Tile className="stat-card stat-card--green"><span className="stat-card__label">Concluídos</span><strong className="stat-card__value">{data.stats.completed}</strong></Tile>
          <Tile className="stat-card stat-card--red"><span className="stat-card__label">Pendentes</span><strong className="stat-card__value">{data.stats.pending}</strong></Tile>
        </div>
      )}

      <Tile className="module-main__header">
        <div className="module-toolbar">
          <Search size="lg" labelText="Buscar sites" placeholder="Buscar por Site ID, End ID, OC, OS..." value={search} onChange={(event) => setSearch(event.target.value)} onClear={() => setSearch('')} />
        </div>
      </Tile>

      {loading ? <InlineLoading description="Carregando sites…" /> : items.length === 0 ? (
        <EmptyState icon={Add} title="Nenhum site cadastrado" description="Cadastre o primeiro site Nokia." action={<Button renderIcon={Add} onClick={openCreate}>Novo site</Button>} />
      ) : (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table">
              <thead><tr><th>Site</th><th>OC</th><th className="hide-sm">ROM</th><th className="hide-sm">Regional</th><th>FAM Instalação</th><th>Progresso</th><th>Ações</th></tr></thead>
              <tbody>
                {items.map((site) => {
                  const done = CHECKLISTS.filter((item) => (site as unknown as Record<string, string>)[item.status] === 'sent').length + site.tasks.filter((task) => task.completed_at).length;
                  const total = 3 + site.tasks.length;
                  return (
                    <tr key={site.id}>
                      <td><button type="button" className="link-button" onClick={() => navigate(`/nokia/${site.id}`)}>{site.site_id}</button><span className="cell-secondary">{site.end_id ?? '—'}</span></td>
                      <td>{site.oc ?? '—'}</td>
                      <td className="hide-sm">{site.rom_name ?? 'Sem ROM'}</td>
                      <td className="hide-sm">{site.regional ?? '—'}</td>
                      <td><Tag type={statusTag(site.checklist_fam_instalacao_status)} size="sm">{STATUS_LABEL[site.checklist_fam_instalacao_status]}</Tag></td>
                      <td>{done}/{total}</td>
                      <td className="cell-actions">
                        <Button kind="ghost" size="sm" hasIconOnly renderIcon={Edit} iconDescription="Editar" onClick={() => openEdit(site)} />
                        <Button kind="danger--ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={async () => { if (!window.confirm(`Excluir o site ${site.site_id}?`)) return; await api.delete(`/nokia/sites/${site.id}`); toast.success('Site excluído.'); await load(); }} />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <Modal open={modal} modalHeading={editing ? 'Editar site' : 'Novo site'} primaryButtonText={saving ? 'Salvando…' : 'Salvar'} secondaryButtonText="Cancelar" primaryButtonDisabled={saving} onRequestSubmit={() => void save()} onRequestClose={() => setModal(false)} size="md">
        <div className="modal-form">
          <div className="form-row">
            {editing ? (
              <>
                <TextInput id="nk-site" labelText="Site ID *" value={form.site_id} onChange={(event) => setForm({ ...form, site_id: event.target.value })} required />
                <TextInput id="nk-end" labelText="End ID" value={form.end_id} onChange={(event) => setForm({ ...form, end_id: event.target.value })} />
              </>
            ) : (
              <div className="nokia-site-combobox">
                <ComboBox
                  id="nk-site"
                  titleText="Site ID *"
                  placeholder="Busque um site nas Estações ou digite um novo"
                  items={stationSites}
                  itemToString={(item) => item?.site_id ?? ''}
                  selectedItem={stationSites.find((item) => item.site_id === form.site_id) ?? null}
                  allowCustomValue
                  onInputChange={(value) => {
                    setForm((current) => {
                      const existing = stationSites.some((item) => item.site_id === value);
                      return existing
                        ? { ...current, site_id: value }
                        : { ...current, site_id: value, end_id: '', regional: '' };
                    });
                  }}
                  onChange={({ selectedItem, inputValue }) => {
                    if (selectedItem) {
                      setForm((current) => ({
                        ...current,
                        site_id: selectedItem.site_id,
                        end_id: selectedItem.endereco_id ?? '',
                        regional: selectedItem.regional ?? current.regional,
                      }));
                    } else if (inputValue) {
                      setForm((current) => ({ ...current, site_id: inputValue }));
                    }
                  }}
                />
                <p className="field-help">
                  Selecione um site já cadastrado em Estações ou digite um novo Site ID para cadastrá-lo automaticamente.
                </p>
              </div>
            )}
          </div>
          <div className="form-row">
            <TextInput id="nk-oc" labelText="OC" value={form.oc} onChange={(event) => setForm({ ...form, oc: event.target.value })} />
            <Select id="nk-rom" labelText="ROM" value={form.nokia_rom_id} onChange={(event) => setForm({ ...form, nokia_rom_id: event.target.value })}>
              <SelectItem value="" text="Sem ROM" />
              {roms.map((rom) => <SelectItem key={rom.id} value={String(rom.id)} text={`${rom.first_name} ${rom.last_name}`} />)}
            </Select>
          </div>
          <div className="form-row">
            <TextInput id="nk-os-e" labelText="OS FAM Entrega" value={form.os_fam_entrega} onChange={(event) => setForm({ ...form, os_fam_entrega: event.target.value })} />
            <TextInput id="nk-os-i" labelText="OS FAM Instalação" value={form.os_fam_instalacao} onChange={(event) => setForm({ ...form, os_fam_instalacao: event.target.value })} />
          </div>
          <div className="form-row">
            <TextInput id="nk-regional" labelText="Regional" value={form.regional} onChange={(event) => setForm({ ...form, regional: event.target.value })} />
            <TextInput id="nk-subcon" labelText="Subcon" value={form.subcon} onChange={(event) => setForm({ ...form, subcon: event.target.value })} />
          </div>
          <Checkbox id="nk-check-e" labelText="FAM Entrega enviada" checked={form.checklist_fam_entrega} onChange={(_e, { checked }) => setForm({ ...form, checklist_fam_entrega: checked })} />
          <Checkbox id="nk-check-i" labelText="FAM Instalação enviada" checked={form.checklist_fam_instalacao} onChange={(_e, { checked }) => setForm({ ...form, checklist_fam_instalacao: checked })} />
          <Checkbox id="nk-check-p" labelText="FAM Panorâmica enviada" checked={form.checklist_fam_panoramica} onChange={(_e, { checked }) => setForm({ ...form, checklist_fam_panoramica: checked })} />
        </div>
      </Modal>

      <Modal open={Boolean(detail)} modalHeading={detail ? `${detail.site_id}` : ''} primaryButtonText="Fechar" onRequestClose={() => setDetail(null)} onRequestSubmit={() => setDetail(null)} size="lg">
        {detail && (
          <div className="modal-form">
            <div className="tag-row">
              {detail.oc && <Tag type="blue" size="sm">OC {detail.oc}</Tag>}
              {detail.rom_name && <Tag type="purple" size="sm">{detail.rom_name}</Tag>}
              {detail.interruption_reason && <Tag type="red" size="sm">Obra interrompida</Tag>}
            </div>
            <div className="detail-grid">
              <div><span>Regional</span><strong>{detail.regional ?? '—'}</strong></div>
              <div><span>Subcon</span><strong>{detail.subcon ?? '—'}</strong></div>
              <div><span>OS Entrega</span><strong>{detail.os_fam_entrega ?? '—'}</strong></div>
              <div><span>OS Instalação</span><strong>{detail.os_fam_instalacao ?? '—'}</strong></div>
              <div><span>Progresso</span><strong>{CHECKLISTS.filter((item) => (detail as unknown as Record<string, string>)[item.status] === 'sent').length + detail.tasks.filter((task) => task.completed_at).length}/{3 + detail.tasks.length}</strong></div>
              <div><span>Criado em</span><strong>{dateBR(detail.created_at)}</strong></div>
            </div>

            {detail.interruption_reason && (
              <div className="nokia-interruption">
                <Tag type="red" size="sm">Obra interrompida</Tag>
                <strong>{detail.interruption_reason}</strong>
                <span>Início: {dateBR(detail.interruption_started_at)} · Retorno previsto: {addDays(detail.interruption_started_at, detail.interruption_return_days)}</span>
              </div>
            )}

            <h4>Datas</h4>
            <div className="table-scroll">
              <table className="data-table data-table--compact">
                <thead><tr><th>Etapa</th><th>Data</th></tr></thead>
                <tbody>
                  {DATE_FIELDS.map(([field, label]) => (
                    <tr key={field}><td>{label}</td><td>{dateBR(String((detail as unknown as Record<string, string>)[field] ?? ''))}</td></tr>
                  ))}
                </tbody>
              </table>
            </div>

            <h4>Checklist FAM</h4>
            {CHECKLISTS.map((item) => (
              <div key={item.field} className="checklist-row">
                <span>{item.label}</span>
                <Tag type={statusTag((detail as unknown as Record<string, string>)[item.status])} size="sm">{STATUS_LABEL[(detail as unknown as Record<string, string>)[item.status]]}</Tag>
                <Select id={`check-${item.field}`} labelText="Status" hideLabel size="sm" value={(detail as unknown as Record<string, string>)[item.status]} onChange={(event) => void changeChecklist(item.field, event.target.value)}>
                  {Object.entries(STATUS_LABEL).map(([value, label]) => <SelectItem key={value} value={value} text={label} />)}
                </Select>
              </div>
            ))}

            <div className="section-actions">
              <Button kind="tertiary" renderIcon={Edit} onClick={() => { setDates(Object.fromEntries(DATE_FIELDS.map(([field]) => [field, String((detail as unknown as Record<string, string>)[field] ?? '')]))); setDatesOpen(true); }}>Editar datas</Button>
              <Button kind="tertiary" renderIcon={Warning} onClick={() => { setInterruption({ reason: detail.interruption_reason ?? '', return_days: detail.interruption_return_days ?? 7 }); setInterruptionOpen(true); }}>Interrupção</Button>
              <Button kind="tertiary" renderIcon={Warning} onClick={() => { setRejection({ fams: [], reason: '', return_days: 7, requires_field_return: false }); setRejectionOpen(true); }}>Rejeição</Button>
            </div>

            <h4>Tarefas</h4>
            {detail.tasks.length === 0 && <p className="muted">Nenhuma tarefa cadastrada.</p>}
            {detail.tasks.map((task) => (
              <Checkbox key={task.id} id={`nk-task-${task.id}`} labelText={task.name} checked={Boolean(task.completed_at)} onChange={() => void toggleTask(task)} />
            ))}

            <h4>Rejeições</h4>
            {detail.rejections.length === 0 && <p className="muted">Nenhuma rejeição registrada.</p>}
            {detail.rejections.map((item) => (
              <div key={item.id} className="comment-row">
                <div><strong>{FAM_LABEL[item.checklist_field] ?? item.checklist_field}</strong><span>{relativeTime(item.started_at)}</span></div>
                <p>
                  {item.reason} · {item.return_days} dias · retorno previsto {addDays(item.started_at, item.return_days)}
                  {item.requires_field_return ? ' · requer retorno a campo' : ''}
                  {item.resolved_at ? ' · Encerrada' : ' · Ativa'}
                </p>
                {!item.resolved_at && <Button kind="ghost" size="xs" onClick={async () => { await api.patch(`/nokia/sites/${detail.id}/rejections/${item.id}`); await openDetail(detail); }}>Encerrar</Button>}
              </div>
            ))}

            <h4>Anexos ({detail.attachments.length})</h4>
            <div className="inline-form">
              <Select id="nk-category" labelText="Categoria" value={attachmentCategory} onChange={(event) => setAttachmentCategory(event.target.value)}>
                {ATTACHMENT_CATEGORIES.map((category) => <SelectItem key={category.value} value={category.value} text={category.label} />)}
              </Select>
              <TextInput id="nk-files" type="file" labelText="Arquivos (até 40 MB cada)" multiple onChange={async (event) => {
                const files = event.target.files;
                if (!files || !detail) return;
                const payload = new FormData();
                payload.append('category', attachmentCategory);
                Array.from(files).forEach((file) => payload.append('files', file));
                try {
                  const response = await api.upload<{ message: string }>(`/nokia/sites/${detail.id}/attachments`, payload);
                  toast.success(response.message);
                  await openDetail(detail);
                  event.target.value = '';
                } catch (error) {
                  toast.error('Falha no upload', errorMessage(error));
                }
              }} />
            </div>
            {ATTACHMENT_CATEGORIES.map((category) => {
              const files = detail.attachments.filter((item) => item.category === category.value);
              if (files.length === 0) return null;
              return (
                <div key={category.value}>
                  <p className="field-label">{category.label} ({files.length})</p>
                  {files.map((item) => (
                    <div key={item.id} className="attachment-row">
                      <a href={item.url} target="_blank" rel="noreferrer">{item.original_name}</a>
                      <span>{relativeTime(item.created_at)}</span>
                      <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover anexo" onClick={async () => { await api.delete(`/nokia/sites/${detail.id}/attachments/${item.id}`); await openDetail(detail); }} />
                    </div>
                  ))}
                </div>
              );
            })}

            <h4>Comentários ({detail.comments.length})</h4>
            <div className="inline-form">
              <TextArea id="nk-comment" labelText="Comentário" value={comment} onChange={(event) => setComment(event.target.value)} rows={2} />
              <Button kind="tertiary" onClick={() => void addComment()}>Comentar</Button>
            </div>
            {detail.comments.map((item) => (
              <div key={item.id} className="comment-row">
                <div><strong>{item.author_name ?? 'Usuário removido'}</strong><span>{relativeTime(item.created_at)}</span></div>
                <p>{item.body}</p>
                {(user?.is_admin || item.user_id === user?.id) && (
                  <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover comentário" onClick={async () => { await api.delete(`/nokia/sites/${detail.id}/comments/${item.id}`); await openDetail(detail); }} />
                )}
              </div>
            ))}
          </div>
        )}
      </Modal>

      <Modal open={datesOpen} modalHeading="Datas do site" primaryButtonText="Salvar" secondaryButtonText="Cancelar" onRequestSubmit={() => void saveDates()} onRequestClose={() => setDatesOpen(false)} size="md">
        <div className="form-grid">
          {DATE_FIELDS.map(([field, label]) => (
            <TextInput key={field} id={`nk-date-${field}`} type="date" labelText={label} value={dates[field] ?? ''} onChange={(event) => setDates({ ...dates, [field]: event.target.value })} />
          ))}
        </div>
      </Modal>

      <Modal open={interruptionOpen} modalHeading="Interrupção de obra" primaryButtonText="Salvar" secondaryButtonText="Cancelar" onRequestSubmit={() => void saveInterruption()} onRequestClose={() => setInterruptionOpen(false)} size="sm">
        <div className="modal-form">
          <TextArea id="nk-int-reason" labelText="Motivo" value={interruption.reason} onChange={(event) => setInterruption({ ...interruption, reason: event.target.value })} rows={3} />
          <Select id="nk-int-days" labelText="Prazo de retorno" value={String(interruption.return_days)} onChange={(event) => setInterruption({ ...interruption, return_days: Number(event.target.value) })}>
            {[7, 14, 21, 28].map((days) => <SelectItem key={days} value={String(days)} text={`${days} dias`} />)}
          </Select>
          <Button kind="danger--ghost" onClick={async () => { if (!detail) return; await api.delete(`/nokia/sites/${detail.id}/interruption`); setInterruptionOpen(false); await openDetail(detail); }}>Encerrar interrupção</Button>
        </div>
      </Modal>

      <Modal open={rejectionOpen} modalHeading="Registrar rejeição" primaryButtonText="Registrar" secondaryButtonText="Cancelar" onRequestSubmit={() => void saveRejection()} onRequestClose={() => setRejectionOpen(false)} size="sm">
        <div className="modal-form">
          {CHECKLISTS.map((item) => <Checkbox key={item.field} id={`rej-${item.field}`} labelText={item.label} checked={rejection.fams.includes(item.field)} onChange={(_e, { checked }) => setRejection({ ...rejection, fams: checked ? [...rejection.fams, item.field] : rejection.fams.filter((field) => field !== item.field) })} />)}
          <TextArea id="nk-rej-reason" labelText="Motivo" value={rejection.reason} onChange={(event) => setRejection({ ...rejection, reason: event.target.value })} rows={3} />
          <Select id="nk-rej-days" labelText="Prazo de retorno" value={String(rejection.return_days)} onChange={(event) => setRejection({ ...rejection, return_days: Number(event.target.value) })}>
            {[7, 14, 21, 28].map((days) => <SelectItem key={days} value={String(days)} text={`${days} dias`} />)}
          </Select>
          <Checkbox id="nk-rej-field" labelText="Necessário retorno a campo" checked={rejection.requires_field_return} onChange={(_e, { checked }) => setRejection({ ...rejection, requires_field_return: checked })} />
        </div>
      </Modal>

      <Modal open={settingsOpen} modalHeading="Configurações Nokia" primaryButtonText="Fechar" onRequestClose={() => setSettingsOpen(false)} onRequestSubmit={() => setSettingsOpen(false)} size="md">
        <div className="modal-form">
          <h4>ROMs</h4>
          {settings.roms.map((rom) => (
            <div key={rom.id} className="attachment-row">
              <span>{rom.first_name} {rom.last_name}</span>
              <span>{rom.email} · {rom.phone}</span>
              <Button kind="ghost" size="xs" hasIconOnly renderIcon={Edit} iconDescription="Editar ROM" onClick={() => { setRomEditingId(rom.id); setRomForm({ first_name: rom.first_name, last_name: rom.last_name, email: rom.email, phone: rom.phone }); }} />
              <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={async () => { await api.delete(`/nokia/settings/roms/${rom.id}`); await reloadNokiaSettings(); }} />
            </div>
          ))}
          <div className="form-grid">
            <TextInput id="nk-rom-first" labelText="Nome" value={romForm.first_name} onChange={(event) => setRomForm({ ...romForm, first_name: event.target.value })} />
            <TextInput id="nk-rom-last" labelText="Sobrenome" value={romForm.last_name} onChange={(event) => setRomForm({ ...romForm, last_name: event.target.value })} />
            <TextInput id="nk-rom-email" labelText="E-mail" value={romForm.email} onChange={(event) => setRomForm({ ...romForm, email: event.target.value })} />
            <TextInput id="nk-rom-phone" labelText="Telefone" value={romForm.phone} onChange={(event) => setRomForm({ ...romForm, phone: event.target.value })} />
          </div>
          <div className="section-actions">
            {romEditingId && <Button kind="ghost" onClick={() => { setRomEditingId(null); setRomForm({ first_name: '', last_name: '', email: '', phone: '' }); }}>Cancelar edição</Button>}
            <Button kind="tertiary" onClick={() => void saveROM()}>{romEditingId ? 'Salvar ROM' : 'Adicionar ROM'}</Button>
          </div>
          <h4>Tarefas</h4>
          {settings.tasks.map((task, index) => (
            <div key={task.id} className="attachment-row">
              <span>{task.name}</span>
              <Button kind="ghost" size="xs" hasIconOnly renderIcon={ArrowUp} iconDescription="Mover para cima" disabled={index === 0} onClick={() => void moveTask(task, 'up')} />
              <Button kind="ghost" size="xs" hasIconOnly renderIcon={ArrowDown} iconDescription="Mover para baixo" disabled={index === settings.tasks.length - 1} onClick={() => void moveTask(task, 'down')} />
              <Button kind="ghost" size="xs" hasIconOnly renderIcon={Edit} iconDescription="Renomear tarefa" onClick={() => void renameTask(task)} />
              <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={async () => { await api.delete(`/nokia/settings/tasks/${task.id}`); await reloadNokiaSettings(); }} />
            </div>
          ))}
          <div className="inline-form">
            <TextInput id="nk-task-name" labelText="Nova tarefa" value={taskName} onChange={(event) => setTaskName(event.target.value)} />
            <Button kind="tertiary" onClick={() => void saveTask()}>Adicionar</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
