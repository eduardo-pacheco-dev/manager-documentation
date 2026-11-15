import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Add, Edit, OverflowMenuVertical, TrashCan } from '@carbon/icons-react';
import {
  Button,
  InlineLoading,
  Modal,
  OverflowMenu,
  OverflowMenuItem,
  ProgressIndicator,
  ProgressStep,
  Search,
  Select,
  SelectItem,
  Tag,
  TextArea,
  TextInput,
  Tile,
  Toggle,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState } from '../../components/Feedback';
import { useAuth } from '../../contexts/AuthContext';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';

interface Attachment { id: number; category: string; original_name: string; url: string; size: number; created_at: string; uploader_name: string | null }
interface Comment { id: number; body: string; user_id: number | null; author_name: string | null; created_at: string }
interface Station {
  id: number; site_id: string; endereco_id: string | null; tipo_de_elemento: string | null;
  tecnologia: string | null; tipo_de_conexao: string | null; classificacao: string | null;
  detentor_area: string | null; tipo_de_infra: string | null; tipo_de_ev: string | null;
  tipo_de_logradouro: string | null; logradouro: string | null; numero: string | null;
  complemento: string | null; bairro: string | null; municipio: string | null; estado: string | null;
  cep: string | null; regional: string | null; latitude: string | null; longitude: string | null;
  status: string | null; tipo_da_torre: string | null; station_id: string | null;
  created_at: string; updated_at: string; attachments: Attachment[]; comments: Comment[];
}
interface Data { data: Station[]; stats: { total: number; sites: number; municipios: number }; statuses: string[]; tecnologias: string[]; meta: { total: number } }

const EMPTY = {
  site_id: '', endereco_id: '', tipo_de_elemento: '', tecnologia: '', tipo_de_conexao: '',
  classificacao: '', detentor_area: '', tipo_de_infra: '', tipo_de_ev: '', tipo_de_logradouro: '',
  logradouro: '', numero: '', complemento: '', bairro: '', municipio: '', estado: '', cep: '',
  regional: '', latitude: '', longitude: '', status: '', tipo_da_torre: '', station_id: '',
};

const STATION_STEPS = [
  { key: 'identificacao', label: 'Identificação', description: 'Identificadores do site e da estação.' },
  { key: 'classificacao', label: 'Classificação', description: 'Tecnologia, tipo de conexão e situação da estação.' },
  { key: 'endereco', label: 'Endereço', description: 'Endereço completo do elemento de rede.' },
  { key: 'coordenadas', label: 'Coordenadas', description: 'Latitude e longitude usadas no mapa.' },
];

type CoordFormat = 'decimal' | 'dms';
interface DmsValues {
  latDeg: string; latMin: string; latSec: string; latHemi: 'N' | 'S';
  lngDeg: string; lngMin: string; lngSec: string; lngHemi: 'E' | 'W';
}
const EMPTY_DMS: DmsValues = { latDeg: '', latMin: '', latSec: '', latHemi: 'S', lngDeg: '', lngMin: '', lngSec: '', lngHemi: 'W' };

function decimalToDms(value: string, positive: 'N' | 'E', negative: 'S' | 'W') {
  const raw = Number(value.trim().replace(',', '.'));
  if (!value.trim() || !Number.isFinite(raw)) return null;
  const abs = Math.abs(raw);
  let degrees = Math.floor(abs);
  const minutesFloat = (abs - degrees) * 60;
  let minutes = Math.floor(minutesFloat);
  let seconds = Math.round((minutesFloat - minutes) * 60 * 100) / 100;
  if (seconds >= 60) { seconds = 0; minutes += 1; }
  if (minutes >= 60) { minutes = 0; degrees += 1; }
  return { degrees: String(degrees), minutes: String(minutes), seconds: String(seconds), hemisphere: raw < 0 ? negative : positive };
}

function dmsToDecimal(degrees: string, minutes: string, seconds: string, hemisphere: string, negative: string, limit: number): string {
  if (!degrees.trim() && !minutes.trim() && !seconds.trim()) return '';
  const d = Number(degrees || 0);
  const m = Number(minutes || 0);
  const s = Number(seconds || 0);
  if (![d, m, s].every(Number.isFinite)) return '';
  if (d < 0 || m < 0 || s < 0 || d > limit || m >= 60 || s >= 60) return '';
  const total = d + m / 60 + s / 3600;
  return (hemisphere === negative ? -total : total).toFixed(6);
}

export function StationsPage() {
  const { user } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [data, setData] = useState<Data | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [tech, setTech] = useState('');
  const [modal, setModal] = useState(false);
  const [editing, setEditing] = useState<Station | null>(null);
  const [detail, setDetail] = useState<Station | null>(null);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({ ...EMPTY });
  const [step, setStep] = useState(0);
  const [coordFormat, setCoordFormat] = useState<CoordFormat>('decimal');
  const [dms, setDms] = useState<DmsValues>({ ...EMPTY_DMS });
  const [comment, setComment] = useState('');
  const [category, setCategory] = useState('attachment');

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams();
    if (search) params.set('search', search);
    if (status) params.set('status', status);
    if (tech) params.set('tecnologia', tech);
    try {
      const response = await api.get<Data>(`/stations?${params.toString()}`);
      setData(response);
      if (detail) {
        const refreshed = response.data.find((item) => item.id === detail.id);
        if (refreshed) setDetail(refreshed);
      }
    } catch (error) {
      toast.error('Não foi possível carregar as estações', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [search, status, tech, toast, detail]);

  useEffect(() => { void load(); /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, [search, status, tech]);

  function openCreate() {
    setEditing(null);
    setForm({ ...EMPTY });
    setStep(0);
    setCoordFormat('decimal');
    setDms({ ...EMPTY_DMS });
    setModal(true);
  }

  function openEdit(item: Station) {
    setEditing(item);
    setForm(Object.fromEntries(Object.keys(EMPTY).map((key) => [key, String((item as unknown as Record<string, unknown>)[key] ?? '')])) as typeof EMPTY);
    setStep(0);
    setCoordFormat('decimal');
    setDms({ ...EMPTY_DMS });
    setModal(true);
  }

  function changeCoordFormat(next: CoordFormat) {
    if (next === 'dms') {
      const lat = decimalToDms(form.latitude, 'N', 'S');
      const lng = decimalToDms(form.longitude, 'E', 'W');
      setDms({
        latDeg: lat?.degrees ?? '', latMin: lat?.minutes ?? '', latSec: lat?.seconds ?? '', latHemi: lat?.hemisphere === 'N' ? 'N' : 'S',
        lngDeg: lng?.degrees ?? '', lngMin: lng?.minutes ?? '', lngSec: lng?.seconds ?? '', lngHemi: lng?.hemisphere === 'E' ? 'E' : 'W',
      });
    }
    setCoordFormat(next);
  }

  function updateDms(patch: Partial<DmsValues>) {
    const next = { ...dms, ...patch };
    setDms(next);
    setForm({
      ...form,
      latitude: dmsToDecimal(next.latDeg, next.latMin, next.latSec, next.latHemi, 'S', 90),
      longitude: dmsToDecimal(next.lngDeg, next.lngMin, next.lngSec, next.lngHemi, 'W', 180),
    });
  }

  const lastStep = STATION_STEPS.length - 1;

  function goToStep(index: number) {
    if (index === step) return;
    if (index > 0 && !form.site_id.trim()) {
      toast.error('Informe o ID do site para continuar.');
      return;
    }
    setStep(Math.min(Math.max(index, 0), lastStep));
  }

  function advanceStep() {
    if (step === 0 && !form.site_id.trim()) {
      toast.error('Informe o ID do site para continuar.');
      return;
    }
    if (step < lastStep) {
      setStep(step + 1);
      return;
    }
    void save();
  }

  async function save() {
    if (!form.site_id.trim()) {
      setStep(0);
      toast.error('Informe o ID do site.');
      return;
    }
    setSaving(true);
    try {
      if (editing) await api.put(`/stations/${editing.id}`, form);
      else await api.post('/stations', form);
      toast.success(editing ? 'Estação atualizada.' : 'Estação cadastrada.');
      setModal(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function openDetail(item: Station) {
    try {
      const response = await api.get<{ data: { station: Station } }>(`/stations/${item.id}`);
      setDetail(response.data.station);
    } catch (error) {
      toast.error('Não foi possível abrir a estação', errorMessage(error));
    }
  }

  async function uploadFiles(files: FileList | null) {
    if (!detail || !files || files.length === 0) return;
    const payload = new FormData();
    payload.append('category', category);
    Array.from(files).forEach((file) => payload.append('files', file));
    try {
      const response = await api.upload<{ message: string }>(`/stations/${detail.id}/attachments`, payload);
      toast.success(response.message);
      await openDetail(detail);
    } catch (error) {
      toast.error('Não foi possível enviar o anexo', errorMessage(error));
    }
  }

  async function addComment() {
    if (!detail || !comment.trim()) return;
    try {
      await api.post(`/stations/${detail.id}/comments`, { body: comment });
      setComment('');
      await openDetail(detail);
    } catch (error) {
      toast.error('Não foi possível comentar', errorMessage(error));
    }
  }

  function importStations() {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.csv,.xlsx';
    input.onchange = async () => {
      if (!input.files?.[0]) return;
      const formData = new FormData();
      formData.append('file', input.files[0]);
      try {
        const response = await api.upload<{ message: string }>('/stations/import', formData);
        toast.success(response.message);
        await load();
      } catch (error) {
        toast.error('Falha na importação', errorMessage(error));
      }
    };
    input.click();
  }

  const items = data?.data ?? [];
  const dmsFilled = Boolean(dms.latDeg || dms.latMin || dms.latSec || dms.lngDeg || dms.lngMin || dms.lngSec);
  const dmsInvalid = coordFormat === 'dms' && dmsFilled && (!form.latitude || !form.longitude);

  return (
    <div className="page">
      <PageHeader
        eyebrow="Inventário"
        title="Estações"
        description="Cadastro de estações e elementos de rede, anexos e comentários."
        actions={
          <>
            <Button renderIcon={Add} onClick={openCreate}>Nova estação</Button>
            <OverflowMenu
              aria-label="Ações de importação e exportação"
              iconDescription="Mais ações"
              renderIcon={OverflowMenuVertical}
              size="md"
              flipped
            >
              <OverflowMenuItem itemText="Exportar" onClick={() => window.open('/api/v1/stations/export', '_blank')} />
              <OverflowMenuItem itemText="Modelo XLSX" onClick={() => window.open('/api/v1/stations/template', '_blank')} />
              <OverflowMenuItem itemText="Importar" hasDivider onClick={importStations} />
            </OverflowMenu>
          </>
        }
      />

      {data && (
        <div className="stats-grid stats-grid--3">
          <Tile className="stat-card stat-card--blue"><span className="stat-card__label">Estações</span><strong className="stat-card__value">{data.stats.total}</strong></Tile>
          <Tile className="stat-card stat-card--purple"><span className="stat-card__label">Sites distintos</span><strong className="stat-card__value">{data.stats.sites}</strong></Tile>
          <Tile className="stat-card stat-card--green"><span className="stat-card__label">Municípios</span><strong className="stat-card__value">{data.stats.municipios}</strong></Tile>
        </div>
      )}

      <Tile className="module-main__header">
        <div className="module-toolbar">
          <Search size="lg" labelText="Buscar estações" placeholder="Buscar por ID, endereço, município..." value={search} onChange={(event) => setSearch(event.target.value)} onClear={() => setSearch('')} />
          <Select id="station-status" labelText="Status" size="sm" value={status} onChange={(event) => setStatus(event.target.value)}>
            <SelectItem value="" text="Todos" />
            {(data?.statuses ?? []).map((value) => <SelectItem key={value} value={value} text={value} />)}
          </Select>
          <Select id="station-tech" labelText="Tecnologia" size="sm" value={tech} onChange={(event) => setTech(event.target.value)}>
            <SelectItem value="" text="Todas" />
            {(data?.tecnologias ?? []).map((value) => <SelectItem key={value} value={value} text={value} />)}
          </Select>
        </div>
      </Tile>

      {loading ? <InlineLoading description="Carregando estações…" /> : items.length === 0 ? (
        <EmptyState icon={Add} title="Nenhuma estação encontrada" description="Cadastre a primeira estação ou ajuste os filtros." action={<Button renderIcon={Add} onClick={openCreate}>Nova estação</Button>} />
      ) : (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table">
              <thead><tr><th>Site</th><th>Tecnologia</th><th className="hide-sm">Endereço</th><th>Status</th><th>Ações</th></tr></thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id}>
                    <td><button type="button" className="link-button" onClick={() => navigate(`/stations/${item.id}`)}>{item.site_id}</button><span className="cell-secondary">{item.station_id ? `ESTAÇÃO ${item.station_id}` : '—'}</span></td>
                    <td>{item.tecnologia ?? 'Sem tecnologia'}<span className="cell-secondary">{item.tipo_de_elemento ?? '—'}</span></td>
                    <td className="hide-sm">{[item.logradouro, item.numero, item.bairro, item.municipio].filter(Boolean).join(', ') || '—'}<span className="cell-secondary">{item.regional ?? 'Sem regional'}</span></td>
                    <td>{item.status ? <Tag type={['ativo', 'ativa', 'aquisitado'].includes(item.status.toLowerCase()) ? 'green' : 'gray'} size="sm">{item.status}</Tag> : '—'}</td>
                    <td className="cell-actions">
                      <Button kind="ghost" size="sm" hasIconOnly renderIcon={Edit} iconDescription="Editar" onClick={() => openEdit(item)} />
                      <Button kind="danger--ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={async () => { if (!window.confirm(`Excluir a estação ${item.site_id}?`)) return; await api.delete(`/stations/${item.id}`); toast.success('Estação excluída.'); await load(); }} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <Modal
        open={modal}
        modalHeading={editing ? 'Editar estação' : 'Nova estação'}
        primaryButtonText={saving ? 'Salvando…' : step === lastStep ? 'Salvar' : 'Próximo'}
        secondaryButtonText={step === 0 ? 'Cancelar' : 'Voltar'}
        primaryButtonDisabled={saving}
        onRequestSubmit={advanceStep}
        onSecondarySubmit={() => (step === 0 ? setModal(false) : setStep(step - 1))}
        onRequestClose={() => setModal(false)}
        size="md"
      >
        <div className="modal-form">
          <ProgressIndicator
            className="modal-steps"
            currentIndex={step}
            onChange={goToStep}
            spaceEqually
          >
            {STATION_STEPS.map((item, index) => (
              <ProgressStep key={item.key} label={item.label} disabled={index > 0 && !form.site_id.trim()} />
            ))}
          </ProgressIndicator>
          <div className="wizard-step__header">
            <h4>{STATION_STEPS[step].label}</h4>
            <p className="muted small">{STATION_STEPS[step].description}</p>
          </div>

          {step === 0 && (
            <>
              <div className="form-row">
                <TextInput id="st-site" labelText="Site ID *" value={form.site_id} onChange={(event) => setForm({ ...form, site_id: event.target.value })} required />
                <TextInput id="st-station" labelText="ID Detentora" value={form.station_id} onChange={(event) => setForm({ ...form, station_id: event.target.value })} />
              </div>
              <div className="form-row">
                <TextInput id="st-endereco" labelText="End ID" value={form.endereco_id} onChange={(event) => setForm({ ...form, endereco_id: event.target.value })} />
                <TextInput id="st-tipo" labelText="Tipo de elemento" value={form.tipo_de_elemento} onChange={(event) => setForm({ ...form, tipo_de_elemento: event.target.value })} />
              </div>
            </>
          )}

          {step === 1 && (
            <>
              <div className="form-row">
                <TextInput id="st-tech" labelText="Tecnologia" value={form.tecnologia} onChange={(event) => setForm({ ...form, tecnologia: event.target.value })} />
                <TextInput id="st-conexao" labelText="Tipo de conexão" value={form.tipo_de_conexao} onChange={(event) => setForm({ ...form, tipo_de_conexao: event.target.value })} />
              </div>
              <div className="form-row">
                <TextInput id="st-classificacao" labelText="Classificação" value={form.classificacao} onChange={(event) => setForm({ ...form, classificacao: event.target.value })} />
                <TextInput id="st-status" labelText="Status" value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })} />
              </div>
              <div className="form-row">
                <TextInput id="st-regional" labelText="Regional" value={form.regional} onChange={(event) => setForm({ ...form, regional: event.target.value })} />
                <TextInput id="st-torre" labelText="Tipo da torre" value={form.tipo_da_torre} onChange={(event) => setForm({ ...form, tipo_da_torre: event.target.value })} />
              </div>
              <div className="form-row">
                <TextInput id="st-infra" labelText="Tipo de infra" value={form.tipo_de_infra} onChange={(event) => setForm({ ...form, tipo_de_infra: event.target.value })} />
                <TextInput id="st-ev" labelText="Tipo de EV" value={form.tipo_de_ev} onChange={(event) => setForm({ ...form, tipo_de_ev: event.target.value })} />
              </div>
              <TextInput id="st-detentor" labelText="Detentor da área" value={form.detentor_area} onChange={(event) => setForm({ ...form, detentor_area: event.target.value })} />
            </>
          )}

          {step === 2 && (
            <>
              <div className="form-row">
                <TextInput id="st-logradouro" labelText="Logradouro" value={form.logradouro} onChange={(event) => setForm({ ...form, logradouro: event.target.value })} />
                <TextInput id="st-numero" labelText="Número" value={form.numero} onChange={(event) => setForm({ ...form, numero: event.target.value })} />
              </div>
              <div className="form-row">
                <TextInput id="st-complemento" labelText="Complemento" value={form.complemento} onChange={(event) => setForm({ ...form, complemento: event.target.value })} />
                <TextInput id="st-bairro" labelText="Bairro" value={form.bairro} onChange={(event) => setForm({ ...form, bairro: event.target.value })} />
              </div>
              <div className="form-row">
                <TextInput id="st-municipio" labelText="Município" value={form.municipio} onChange={(event) => setForm({ ...form, municipio: event.target.value })} />
                <TextInput id="st-estado" labelText="UF" value={form.estado} onChange={(event) => setForm({ ...form, estado: event.target.value })} maxLength={2} />
              </div>
              <TextInput id="st-cep" labelText="CEP" value={form.cep} onChange={(event) => setForm({ ...form, cep: event.target.value })} />
            </>
          )}

          {step === 3 && (
            <>
              <Toggle
                id="st-coord-format"
                labelText="Formato das coordenadas"
                labelA="Decimal"
                labelB="Graus, minutos e segundos"
                toggled={coordFormat === 'dms'}
                onToggle={(checked) => changeCoordFormat(checked ? 'dms' : 'decimal')}
              />

              {coordFormat === 'decimal' ? (
                <>
                  <div className="form-row">
                    <TextInput id="st-lat" labelText="Latitude" value={form.latitude} onChange={(event) => setForm({ ...form, latitude: event.target.value })} />
                    <TextInput id="st-lng" labelText="Longitude" value={form.longitude} onChange={(event) => setForm({ ...form, longitude: event.target.value })} />
                  </div>
                  <p className="field-help">Use ponto ou vírgula como separador decimal; valores negativos indicam sul/oeste.</p>
                </>
              ) : (
                <>
                  <div>
                    <p className="field-label">Latitude</p>
                    <div className="coord-dms">
                      <TextInput id="st-lat-deg" type="number" min={0} max={90} labelText="Graus" value={dms.latDeg} onChange={(event) => updateDms({ latDeg: event.target.value })} />
                      <TextInput id="st-lat-min" type="number" min={0} max={59} labelText="Minutos" value={dms.latMin} onChange={(event) => updateDms({ latMin: event.target.value })} />
                      <TextInput id="st-lat-sec" type="number" min={0} max={59} step="0.01" labelText="Segundos" value={dms.latSec} onChange={(event) => updateDms({ latSec: event.target.value })} />
                      <Select id="st-lat-hemi" labelText="Hemisfério" value={dms.latHemi} onChange={(event) => updateDms({ latHemi: event.target.value as 'N' | 'S' })}>
                        <SelectItem value="N" text="Norte (N)" />
                        <SelectItem value="S" text="Sul (S)" />
                      </Select>
                    </div>
                  </div>
                  <div>
                    <p className="field-label">Longitude</p>
                    <div className="coord-dms">
                      <TextInput id="st-lng-deg" type="number" min={0} max={180} labelText="Graus" value={dms.lngDeg} onChange={(event) => updateDms({ lngDeg: event.target.value })} />
                      <TextInput id="st-lng-min" type="number" min={0} max={59} labelText="Minutos" value={dms.lngMin} onChange={(event) => updateDms({ lngMin: event.target.value })} />
                      <TextInput id="st-lng-sec" type="number" min={0} max={59} step="0.01" labelText="Segundos" value={dms.lngSec} onChange={(event) => updateDms({ lngSec: event.target.value })} />
                      <Select id="st-lng-hemi" labelText="Hemisfério" value={dms.lngHemi} onChange={(event) => updateDms({ lngHemi: event.target.value as 'E' | 'W' })}>
                        <SelectItem value="E" text="Leste (E)" />
                        <SelectItem value="W" text="Oeste (W)" />
                      </Select>
                    </div>
                  </div>
                  <p className={dmsInvalid ? 'field-help field-help--danger' : 'field-help'}>
                    {dmsInvalid
                      ? 'Preencha graus, minutos, segundos e hemisfério de latitude e longitude (graus até 90/180 e minutos/segundos abaixo de 60).'
                      : `Convertido para decimal: ${form.latitude || '—'}, ${form.longitude || '—'}`}
                  </p>
                </>
              )}
            </>
          )}
        </div>
      </Modal>

      <Modal open={Boolean(detail)} modalHeading={detail ? `${detail.site_id}` : ''} primaryButtonText="Fechar" onRequestClose={() => setDetail(null)} onRequestSubmit={() => setDetail(null)} size="md">
        {detail && (
          <div className="modal-form">
            <div className="tag-row">
              {detail.status && <Tag type="green" size="sm">{detail.status}</Tag>}
              {detail.tecnologia && <Tag type="blue" size="sm">{detail.tecnologia}</Tag>}
            </div>
            <p>{[detail.logradouro, detail.numero, detail.bairro, detail.municipio, detail.estado].filter(Boolean).join(' · ') || 'Endereço não informado'}</p>
            <div className="detail-grid">
              <div><span>Tipo</span><strong>{detail.tipo_de_elemento ?? '—'}</strong></div>
              <div><span>Regional</span><strong>{detail.regional ?? '—'}</strong></div>
              <div><span>Coordenadas</span><strong>{detail.latitude && detail.longitude ? `${detail.latitude}, ${detail.longitude}` : '—'}</strong></div>
              <div><span>Atualizado</span><strong>{relativeTime(detail.updated_at)}</strong></div>
            </div>
            <div className="section-actions">
              <Button kind="tertiary" renderIcon={Edit} onClick={() => { setDetail(null); openEdit(detail); }}>Editar</Button>
              <Button kind="danger--ghost" renderIcon={TrashCan} onClick={async () => { if (!window.confirm('Excluir esta estação?')) return; await api.delete(`/stations/${detail.id}`); setDetail(null); await load(); }}>Excluir</Button>
            </div>
            <h4>Anexos ({detail.attachments.length})</h4>
            <div className="inline-form">
              <Select id="st-category" labelText="Categoria" value={category} onChange={(event) => setCategory(event.target.value)}>
                <SelectItem value="attachment" text="Anexo" />
                <SelectItem value="ppi_tssr" text="PPI/TSSR" />
              </Select>
              <TextInput id="st-files" type="file" labelText="Arquivos" multiple onChange={(event) => void uploadFiles(event.target.files)} />
            </div>
            {detail.attachments.map((attachment) => (
              <div key={attachment.id} className="attachment-row">
                <a href={attachment.url} target="_blank" rel="noreferrer">{attachment.original_name}</a>
                <span>{attachment.uploader_name ?? 'Usuário removido'} · {relativeTime(attachment.created_at)}</span>
                <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover" onClick={async () => { await api.delete(`/stations/${detail.id}/attachments/${attachment.id}`); await openDetail(detail); }} />
              </div>
            ))}
            <h4>Comentários ({detail.comments.length})</h4>
            <div className="inline-form">
              <TextArea id="st-comment" labelText="Comentário" value={comment} onChange={(event) => setComment(event.target.value)} rows={2} />
              <Button kind="tertiary" onClick={() => void addComment()}>Comentar</Button>
            </div>
            {detail.comments.map((item) => (
              <div key={item.id} className="comment-row">
                <div><strong>{item.author_name ?? 'Usuário removido'}</strong><span>{relativeTime(item.created_at)}</span></div>
                <p>{item.body}</p>
                {(user?.is_admin || item.user_id === user?.id) && <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover" onClick={async () => { await api.delete(`/stations/${detail.id}/comments/${item.id}`); await openDetail(detail); }} />}
              </div>
            ))}
          </div>
        )}
      </Modal>
    </div>
  );
}
