import { useCallback, useEffect, useState } from 'react';
import { Add, Download, Edit, TrashCan, Upload } from '@carbon/icons-react';
import {
  Button,
  InlineLoading,
  Modal,
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

export function StationsPage() {
  const { user } = useAuth();
  const toast = useToast();
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
    setModal(true);
  }

  function openEdit(item: Station) {
    setEditing(item);
    setForm(Object.fromEntries(Object.keys(EMPTY).map((key) => [key, String((item as unknown as Record<string, unknown>)[key] ?? '')])) as typeof EMPTY);
    setModal(true);
  }

  async function save() {
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

  const items = data?.data ?? [];

  return (
    <div className="page">
      <PageHeader
        eyebrow="Inventário"
        title="Estações"
        description="Cadastro de estações e elementos de rede, anexos e comentários."
        actions={
          <>
            <Button kind="ghost" renderIcon={Download} onClick={() => window.open('/api/v1/stations/export', '_blank')}>Exportar</Button>
            <Button kind="ghost" renderIcon={Upload} onClick={() => { const input = document.createElement('input'); input.type = 'file'; input.accept = '.csv'; input.onchange = async () => { if (!input.files?.[0]) return; const formData = new FormData(); formData.append('file', input.files[0]); try { const response = await api.upload<{ message: string }>('/stations/import', formData); toast.success(response.message); await load(); } catch (error) { toast.error('Falha na importação', errorMessage(error)); } }; input.click(); }}>Importar</Button>
            <Button renderIcon={Add} onClick={openCreate}>Nova estação</Button>
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
                    <td><button type="button" className="link-button" onClick={() => void openDetail(item)}>{item.site_id}</button><span className="cell-secondary">{item.station_id ? `ESTAÇÃO ${item.station_id}` : '—'}</span></td>
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

      <Modal open={modal} modalHeading={editing ? 'Editar estação' : 'Nova estação'} primaryButtonText={saving ? 'Salvando…' : 'Salvar'} secondaryButtonText="Cancelar" primaryButtonDisabled={saving} onRequestSubmit={() => void save()} onRequestClose={() => setModal(false)} size="md">
        <div className="modal-form">
          <div className="form-row">
            <TextInput id="st-site" labelText="ID do site *" value={form.site_id} onChange={(event) => setForm({ ...form, site_id: event.target.value })} required />
            <TextInput id="st-station" labelText="ID da estação" value={form.station_id} onChange={(event) => setForm({ ...form, station_id: event.target.value })} />
          </div>
          <div className="form-row">
            <TextInput id="st-endereco" labelText="ID do endereço" value={form.endereco_id} onChange={(event) => setForm({ ...form, endereco_id: event.target.value })} />
            <TextInput id="st-tipo" labelText="Tipo de elemento" value={form.tipo_de_elemento} onChange={(event) => setForm({ ...form, tipo_de_elemento: event.target.value })} />
          </div>
          <div className="form-row">
            <TextInput id="st-tech" labelText="Tecnologia" value={form.tecnologia} onChange={(event) => setForm({ ...form, tecnologia: event.target.value })} />
            <TextInput id="st-status" labelText="Status" value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })} />
          </div>
          <div className="form-row">
            <TextInput id="st-logradouro" labelText="Logradouro" value={form.logradouro} onChange={(event) => setForm({ ...form, logradouro: event.target.value })} />
            <TextInput id="st-numero" labelText="Número" value={form.numero} onChange={(event) => setForm({ ...form, numero: event.target.value })} />
          </div>
          <div className="form-row">
            <TextInput id="st-bairro" labelText="Bairro" value={form.bairro} onChange={(event) => setForm({ ...form, bairro: event.target.value })} />
            <TextInput id="st-municipio" labelText="Município" value={form.municipio} onChange={(event) => setForm({ ...form, municipio: event.target.value })} />
          </div>
          <div className="form-row">
            <TextInput id="st-estado" labelText="UF" value={form.estado} onChange={(event) => setForm({ ...form, estado: event.target.value })} maxLength={2} />
            <TextInput id="st-cep" labelText="CEP" value={form.cep} onChange={(event) => setForm({ ...form, cep: event.target.value })} />
          </div>
          <div className="form-row">
            <TextInput id="st-lat" labelText="Latitude" value={form.latitude} onChange={(event) => setForm({ ...form, latitude: event.target.value })} />
            <TextInput id="st-lng" labelText="Longitude" value={form.longitude} onChange={(event) => setForm({ ...form, longitude: event.target.value })} />
          </div>
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
