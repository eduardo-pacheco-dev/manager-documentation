import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, Download, Edit, TrashCan } from '@carbon/icons-react';
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
import { RelatedWork } from '../../components/RelatedWork';
import { useAuth } from '../../contexts/AuthContext';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';

interface Attachment {
  id: number; category: string; original_name: string; url: string; size: number;
  created_at: string; uploader_name: string | null;
}
interface Comment {
  id: number; body: string; user_id: number | null; author_name: string | null; created_at: string;
}
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

const UF_OPTIONS = [
  ['AC', 'Acre'], ['AL', 'Alagoas'], ['AP', 'Amapá'], ['AM', 'Amazonas'], ['BA', 'Bahia'],
  ['CE', 'Ceará'], ['DF', 'Distrito Federal'], ['ES', 'Espírito Santo'], ['GO', 'Goiás'],
  ['MA', 'Maranhão'], ['MT', 'Mato Grosso'], ['MS', 'Mato Grosso do Sul'], ['MG', 'Minas Gerais'],
  ['PA', 'Pará'], ['PB', 'Paraíba'], ['PR', 'Paraná'], ['PE', 'Pernambuco'], ['PI', 'Piauí'],
  ['RJ', 'Rio de Janeiro'], ['RN', 'Rio Grande do Norte'], ['RS', 'Rio Grande do Sul'],
  ['RO', 'Rondônia'], ['RR', 'Roraima'], ['SC', 'Santa Catarina'], ['SP', 'São Paulo'],
  ['SE', 'Sergipe'], ['TO', 'Tocantins'],
];

const FIELD_GROUPS = [
  {
    title: 'Identificação',
    fields: [
      ['site_id', 'ID do site'], ['endereco_id', 'ID do endereço'], ['tipo_de_elemento', 'Tipo de elemento'],
      ['tecnologia', 'Tecnologia'], ['tipo_de_conexao', 'Tipo de conexão'], ['classificacao', 'Classificação'],
      ['status', 'Status'],
    ],
  },
  {
    title: 'Infraestrutura',
    fields: [
      ['station_id', 'ID da estação'], ['detentor_area', 'Detentor da área'], ['tipo_de_infra', 'Tipo de infraestrutura'],
      ['tipo_de_ev', 'Tipo de EV'], ['tipo_da_torre', 'Tipo da torre'], ['regional', 'Regional'],
    ],
  },
  {
    title: 'Endereço',
    fields: [
      ['cep', 'CEP'], ['tipo_de_logradouro', 'Tipo de logradouro'], ['logradouro', 'Logradouro'],
      ['numero', 'Número'], ['complemento', 'Complemento'], ['bairro', 'Bairro'],
      ['municipio', 'Município'], ['estado', 'UF'],
    ],
  },
  {
    title: 'Localização',
    fields: [['latitude', 'Latitude'], ['longitude', 'Longitude']],
  },
] as const;

const CATEGORY_LABELS: Record<string, string> = { ppi_tssr: 'PPI/TSSR', attachment: 'Anexos' };
const CATEGORY_OPTIONS = [
  { value: 'ppi_tssr', label: 'PPI/TSSR' },
  { value: 'attachment', label: 'Anexos' },
];

function valueOrDash(value: string | null | undefined): string {
  return value && value.trim() !== '' ? value : '—';
}

function formatCEP(value: string | null | undefined): string {
  if (!value) return '—';
  const digits = value.replace(/\D/g, '');
  return digits.length === 8 ? `${digits.slice(0, 5)}-${digits.slice(5)}` : value;
}

function fullAddress(station: Station): string {
  const street = [station.tipo_de_logradouro, station.logradouro].filter(Boolean).join(' ');
  return [street, station.numero, station.complemento, station.bairro, station.municipio, station.estado]
    .filter((part) => part && String(part).trim() !== '')
    .join(' · ') || 'Não informado';
}

export function StationDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const toast = useToast();
  const [station, setStation] = useState<Station | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [form, setForm] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [comment, setComment] = useState('');
  const [attachmentCategory, setAttachmentCategory] = useState('attachment');
  const [cepLoading, setCepLoading] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const response = await api.get<{ data: { station: Station } }>(`/stations/${id}`);
      const loaded = response.data.station;
      setStation({ ...loaded, attachments: loaded.attachments ?? [], comments: loaded.comments ?? [] });
    } catch {
      setNotFound(true);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { void load(); }, [load]);

  function openEdit() {
    if (!station) return;
    const values: Record<string, string> = {};
    FIELD_GROUPS.forEach((group) => group.fields.forEach(([field]) => {
      values[field] = String((station as unknown as Record<string, unknown>)[field] ?? '');
    }));
    setForm(values);
    setEditOpen(true);
  }

  async function lookupCEP() {
    if (!form.cep) return;
    const digits = form.cep.replace(/\D/g, '');
    if (digits.length !== 8) {
      toast.warning('CEP inválido', 'Informe um CEP com 8 dígitos.');
      return;
    }
    setCepLoading(true);
    try {
      const response = await fetch(`https://viacep.com.br/ws/${digits}/json/`);
      const data = await response.json();
      if (data.erro) {
        toast.error('CEP não encontrado', 'Verifique o número informado.');
        return;
      }
      setForm((current) => ({
        ...current,
        logradouro: data.logradouro ?? current.logradouro,
        bairro: data.bairro ?? current.bairro,
        municipio: data.localidade ?? current.municipio,
        estado: data.uf ?? current.estado,
        complemento: current.complemento || data.complemento || '',
      }));
      toast.success('Endereço preenchido pelo ViaCEP');
    } catch {
      toast.error('Não foi possível consultar o CEP', 'Verifique sua conexão e tente novamente.');
    } finally {
      setCepLoading(false);
    }
  }

  async function save() {
    if (!station) return;
    setSaving(true);
    try {
      await api.put(`/stations/${station.id}`, form);
      toast.success('Estação atualizada com sucesso.');
      setEditOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar a estação', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function uploadAttachments(files: FileList | null) {
    if (!station || !files || files.length === 0) return;
    const payload = new FormData();
    payload.append('category', attachmentCategory);
    Array.from(files).forEach((file) => payload.append('files', file));
    try {
      const response = await api.upload<{ message: string }>(`/stations/${station.id}/attachments`, payload);
      toast.success(response.message);
      await load();
    } catch (error) {
      toast.error('Não foi possível enviar o anexo', errorMessage(error));
    }
  }

  async function deleteAttachment(attachmentID: number) {
    if (!station) return;
    await api.delete(`/stations/${station.id}/attachments/${attachmentID}`);
    await load();
  }

  async function addComment() {
    if (!station || !comment.trim()) return;
    try {
      await api.post(`/stations/${station.id}/comments`, { body: comment });
      setComment('');
      await load();
    } catch (error) {
      toast.error('Não foi possível publicar o comentário', errorMessage(error));
    }
  }

  async function deleteComment(commentID: number) {
    if (!station) return;
    await api.delete(`/stations/${station.id}/comments/${commentID}`);
    await load();
  }

  async function deleteStation() {
    if (!station) return;
    setDeleting(true);
    try {
      await api.delete(`/stations/${station.id}`);
      toast.success('Estação excluída com sucesso.');
      navigate('/stations');
    } catch (error) {
      toast.error('Não foi possível excluir a estação', errorMessage(error));
      setDeleting(false);
    }
  }

  if (loading) return <InlineLoading description="Carregando estação…" />;
  if (notFound || !station) {
    return (
      <div className="status-page">
        <p className="status-page__code">404</p>
        <h1>Estação não encontrada</h1>
        <p>A estação pode ter sido excluída ou o endereço está incorreto.</p>
        <Link to="/stations"><Button>Voltar para Estações</Button></Link>
      </div>
    );
  }

  const title = station.station_id || station.site_id;
  const lat = Number(station.latitude);
  const lng = Number(station.longitude);
  const hasCoordinates = Boolean(station.latitude && station.longitude && Number.isFinite(lat) && Number.isFinite(lng));
  const delta = 0.01;
  const mapUrl = hasCoordinates
    ? `https://www.openstreetmap.org/export/embed.html?bbox=${lng - delta}%2C${lat - delta}%2C${lng + delta}%2C${lat + delta}&layer=mapnik&marker=${lat}%2C${lng}`
    : '';
  const mapsUrl = hasCoordinates
    ? `https://www.google.com/maps/search/?api=1&query=${lat},${lng}`
    : `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(fullAddress(station))}`;

  return (
    <div className="page">
      <Breadcrumb noTrailingSlash className="doc-breadcrumb">
        <BreadcrumbItem><Link to="/stations">Estações</Link></BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>{title}</BreadcrumbItem>
      </Breadcrumb>

      <PageHeader
        eyebrow="Inventário"
        title={title}
        description={
          <span className="tag-row">
            {station.status && <Tag type="green" size="sm">{station.status}</Tag>}
            {station.tecnologia && <Tag type="blue" size="sm">{station.tecnologia}</Tag>}
            {station.regional && <Tag type="gray" size="sm">Regional {station.regional}</Tag>}
            <Tag type="cool-gray" size="sm">SITE {station.site_id}</Tag>
          </span>
        }
        actions={
          <>
            <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/stations')}>Voltar</Button>
            <Button kind="ghost" renderIcon={Download} onClick={() => window.open(`/api/v1/stations/${station.id}/export`, '_blank')}>Exportar</Button>
            <Button renderIcon={Edit} onClick={openEdit}>Editar</Button>
            <Button kind="danger--ghost" renderIcon={TrashCan} onClick={() => setDeleteOpen(true)}>Excluir</Button>
          </>
        }
      />

      <div className="station-summary-grid">
        <Tile><span>Endereço</span><strong>{fullAddress(station)}</strong></Tile>
        <Tile><span>Coordenadas</span><strong>{hasCoordinates ? `${station.latitude}, ${station.longitude}` : 'Não informadas'}</strong></Tile>
        <Tile><span>Atualizado</span><strong>{relativeTime(station.updated_at)}</strong></Tile>
      </div>

      <div className="nokia-detail-layout">
        <div className="nokia-detail-main">
          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Informações</h2><p>Identificação e infraestrutura da estação.</p></div>
            </div>
            <div className="station-field-groups">
              {FIELD_GROUPS.slice(0, 2).map((group) => (
                <div key={group.title} className="station-field-group">
                  <p className="field-label">{group.title}</p>
                  <dl>
                    {group.fields.map(([field, label]) => (
                      <div key={field}><dt>{label}</dt><dd>{valueOrDash(String((station as unknown as Record<string, unknown>)[field] ?? ''))}</dd></div>
                    ))}
                  </dl>
                </div>
              ))}
            </div>
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Localização</h2><p>Endereço e coordenadas geográficas.</p></div>
              <Button kind="ghost" size="sm" onClick={() => window.open(mapsUrl, '_blank')}>Abrir no Google Maps</Button>
            </div>
            <div className="station-address-block">
              <div><span>Endereço</span><strong>{fullAddress(station)}</strong></div>
              <div><span>CEP</span><strong>{formatCEP(station.cep)}</strong></div>
              <div><span>Município</span><strong>{valueOrDash(station.municipio)}</strong></div>
              <div><span>UF</span><strong>{valueOrDash(station.estado)}</strong></div>
              <div><span>Regional</span><strong>{valueOrDash(station.regional)}</strong></div>
            </div>
            {hasCoordinates ? (
              <div className="station-map">
                <iframe title={`Mapa de ${title}`} src={mapUrl} loading="lazy" />
              </div>
            ) : (
              <p className="muted">Informe latitude e longitude para visualizar o mapa.</p>
            )}
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Arquivos</h2><p>PPI/TSSR e anexos da estação.</p></div>
            </div>
            <div className="inline-form nokia-upload-row">
              <Select id="station-detail-category" labelText="Categoria" value={attachmentCategory} onChange={(event) => setAttachmentCategory(event.target.value)}>
                {CATEGORY_OPTIONS.map((category) => <SelectItem key={category.value} value={category.value} text={category.label} />)}
              </Select>
              <TextInput
                id="station-detail-files"
                type="file"
                labelText="Arquivos (até 20 MB cada)"
                multiple
                onChange={(event) => void uploadAttachments(event.target.files)}
              />
            </div>
            {Object.entries(CATEGORY_LABELS).map(([category, label]) => {
              const files = station.attachments.filter((attachment) => attachment.category === category);
              if (files.length === 0) return null;
              return (
                <div key={category} className="nokia-attachment-group">
                  <p className="field-label">{label} ({files.length})</p>
                  {files.map((attachment) => (
                    <div key={attachment.id} className="attachment-row">
                      <a href={attachment.url} target="_blank" rel="noreferrer">{attachment.original_name}</a>
                      <span>{attachment.uploader_name ?? 'Usuário removido'} · {relativeTime(attachment.created_at)}</span>
                      <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover anexo" onClick={() => void deleteAttachment(attachment.id)} />
                    </div>
                  ))}
                </div>
              );
            })}
          </Tile>

          <RelatedWork resourceType="station" resourceId={station.id} />

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Comentários</h2><p>{station.comments.length} comentário(s).</p></div>
            </div>
            <div className="inline-form">
              <TextArea id="station-detail-comment" labelText="Novo comentário" value={comment} onChange={(event) => setComment(event.target.value)} rows={2} />
              <Button kind="tertiary" onClick={() => void addComment()}>Comentar</Button>
            </div>
            {station.comments.length === 0 ? (
              <p className="muted">Nenhum comentário ainda.</p>
            ) : (
              <div className="nokia-comment-list">
                {station.comments.map((item) => (
                  <div key={item.id} className="comment-row">
                    <div><strong>{item.author_name ?? 'Usuário removido'}</strong><span>{relativeTime(item.created_at)}</span></div>
                    <p>{item.body}</p>
                    {(user?.is_admin || item.user_id === user?.id) && (
                      <Button kind="danger--ghost" size="xs" hasIconOnly renderIcon={TrashCan} iconDescription="Remover comentário" onClick={() => void deleteComment(item.id)} />
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
              <div><dt>ID do site</dt><dd>{station.site_id}</dd></div>
              <div><dt>ID da estação</dt><dd>{valueOrDash(station.station_id)}</dd></div>
              <div><dt>ID do endereço</dt><dd>{valueOrDash(station.endereco_id)}</dd></div>
              <div><dt>Tipo</dt><dd>{valueOrDash(station.tipo_de_elemento)}</dd></div>
              <div><dt>Tecnologia</dt><dd>{valueOrDash(station.tecnologia)}</dd></div>
              <div><dt>Regional</dt><dd>{valueOrDash(station.regional)}</dd></div>
            </dl>
          </Tile>
          <Tile className="nokia-detail-aside__card">
            <h2>Registro</h2>
            <dl>
              <div><dt>Criado em</dt><dd>{relativeTime(station.created_at)}</dd></div>
              <div><dt>Atualizado em</dt><dd>{relativeTime(station.updated_at)}</dd></div>
            </dl>
          </Tile>
        </aside>
      </div>

      <Modal open={editOpen} modalHeading="Editar estação" primaryButtonText={saving ? 'Salvando…' : 'Salvar'} secondaryButtonText="Cancelar" primaryButtonDisabled={saving} onRequestSubmit={() => void save()} onRequestClose={() => setEditOpen(false)} size="lg">
        <div className="modal-form">
          {FIELD_GROUPS.map((group) => (
            <div key={group.title} className="nokia-date-stage">
              <p className="field-label">{group.title}</p>
              <div className="form-grid">
                {group.fields.map(([field, label]) => {
                  if (field === 'estado') {
                    return (
                      <Select key={field} id={`station-edit-${field}`} labelText={label} value={form[field] ?? ''} onChange={(event) => setForm({ ...form, [field]: event.target.value })}>
                        <SelectItem value="" text="Selecione" />
                        {UF_OPTIONS.map(([uf, name]) => <SelectItem key={uf} value={uf} text={`${uf} — ${name}`} />)}
                      </Select>
                    );
                  }
                  if (field === 'cep') {
                    return (
                      <div key={field} className="station-cep-field">
                        <TextInput id={`station-edit-${field}`} labelText={label} value={form[field] ?? ''} onChange={(event) => setForm({ ...form, [field]: event.target.value })} />
                        <Button kind="tertiary" size="sm" onClick={() => void lookupCEP()} disabled={cepLoading}>{cepLoading ? 'Buscando…' : 'Buscar CEP'}</Button>
                      </div>
                    );
                  }
                  return <TextInput key={field} id={`station-edit-${field}`} labelText={label} value={form[field] ?? ''} onChange={(event) => setForm({ ...form, [field]: event.target.value })} required={field === 'site_id'} />;
                })}
              </div>
            </div>
          ))}
        </div>
      </Modal>

      <ConfirmModal
        open={deleteOpen}
        title="Excluir estação"
        message={`Excluir a estação ${title}? Essa ação não pode ser desfeita.`}
        confirmLabel="Excluir estação"
        loading={deleting}
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => void deleteStation()}
      />
    </div>
  );
}
