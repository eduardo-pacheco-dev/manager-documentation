import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, Copy, OverflowMenuVertical, TrashCan } from '@carbon/icons-react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  InlineLoading,
  OverflowMenu,
  OverflowMenuItem,
  Tag,
  TextArea,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { ConfirmModal } from '../../components/ConfirmModal';
import { RadioLinkMap } from '../../components/RadioLinkMap';
import { useAuth } from '../../contexts/AuthContext';
import {
  RadioLinkFormModal,
  formatRadioDistance,
  radioLinkStatusLabel,
  radioLinkStatusTagType,
} from '../../components/RadioLinkFormModal';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import type { FormRecord, FormsListResponse, RadioLink } from '../../lib/types';

interface StationDetails {
  id: number;
  site_id: string;
  station_id: string | null;
  endereco_id: string | null;
  logradouro: string | null;
  numero: string | null;
  complemento: string | null;
  bairro: string | null;
  municipio: string | null;
  estado: string | null;
  cep: string | null;
  regional: string | null;
  latitude: string | null;
  longitude: string | null;
  status: string | null;
  tecnologia: string | null;
  tipo_de_elemento: string | null;
}

function addressOf(station: StationDetails | null): string {
  if (!station) return '—';
  const line = [station.logradouro, station.numero, station.complemento].filter(Boolean).join(', ');
  return [line, station.bairro, station.municipio, station.estado].filter(Boolean).join(' — ') || '—';
}

function coordinatesOf(station: StationDetails | null): string {
  if (!station?.latitude || !station.longitude) return '';
  return `${station.latitude}, ${station.longitude}`;
}

function valueOrDash(value: string | null | undefined): string {
  return value && value.trim() ? value : '—';
}

export function RadioLinkDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const toast = useToast();
  const { user } = useAuth();
  const [link, setLink] = useState<RadioLink | null>(null);
  const [stationA, setStationA] = useState<StationDetails | null>(null);
  const [stationB, setStationB] = useState<StationDetails | null>(null);
  const [forms, setForms] = useState<FormRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [comment, setComment] = useState('');
  const [commentBusy, setCommentBusy] = useState(false);
  const [uploading, setUploading] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    try {
      const response = await api.get<{ data: { radio_link: RadioLink } }>(`/radio-links/${id}`);
      const current = response.data.radio_link;
      setLink(current);
      const [a, b] = await Promise.all([
        api.get<{ data: { station: StationDetails } }>(`/stations/${current.station_a_id}`).catch(() => null),
        api.get<{ data: { station: StationDetails } }>(`/stations/${current.station_b_id}`).catch(() => null),
      ]);
      setStationA(a?.data.station ?? null);
      setStationB(b?.data.station ?? null);
      const formsResponse = await api
        .get<FormsListResponse>(`/forms?radio_link_id=${id}&per_page=100`)
        .catch(() => null);
      setForms(formsResponse?.data ?? []);
    } catch (error) {
      setNotFound(true);
      toast.error('Não foi possível carregar o rádio link', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [id, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  async function remove() {
    if (!link) return;
    setDeleting(true);
    try {
      await api.delete(`/radio-links/${link.id}`);
      toast.success('Rádio link excluído');
      navigate('/radio-links');
    } catch (error) {
      toast.error('Não foi possível excluir o rádio link', errorMessage(error));
      setDeleting(false);
    }
  }

  async function copyCoordinates(station: StationDetails | null) {
    const value = coordinatesOf(station);
    if (!value) return;
    try {
      await navigator.clipboard.writeText(value);
      toast.success('Coordenadas copiadas', value);
    } catch {
      toast.error('Não foi possível copiar as coordenadas', value);
    }
  }

  async function uploadAttachments(files: FileList | null) {
    const linkID = link?.id;
    if (!linkID || !files || files.length === 0) return;
    setUploading(true);
    try {
      const form = new FormData();
      Array.from(files).forEach((file) => form.append('files', file));
      const response = await api.upload<{ message: string }>(`/radio-links/${linkID}/attachments`, form);
      toast.success('Arquivos enviados', response.message);
      await load();
    } catch (error) {
      toast.error('Não foi possível enviar os arquivos', errorMessage(error));
    } finally {
      setUploading(false);
    }
  }

  async function deleteAttachment(attachmentId: number) {
    const linkID = link?.id;
    if (!linkID) return;
    try {
      await api.delete(`/radio-links/${linkID}/attachments/${attachmentId}`);
      toast.success('Anexo removido');
      await load();
    } catch (error) {
      toast.error('Não foi possível remover o anexo', errorMessage(error));
    }
  }

  async function addComment() {
    const linkID = link?.id;
    if (!linkID) return;
    if (comment.trim().length < 2) {
      toast.error('O comentário precisa ter pelo menos 2 caracteres');
      return;
    }
    setCommentBusy(true);
    try {
      await api.post(`/radio-links/${linkID}/comments`, { body: comment.trim() });
      setComment('');
      toast.success('Comentário publicado');
      await load();
    } catch (error) {
      toast.error('Não foi possível publicar o comentário', errorMessage(error));
    } finally {
      setCommentBusy(false);
    }
  }

  async function deleteComment(commentId: number) {
    const linkID = link?.id;
    if (!linkID) return;
    try {
      await api.delete(`/radio-links/${linkID}/comments/${commentId}`);
      toast.success('Comentário removido');
      await load();
    } catch (error) {
      toast.error('Não foi possível remover o comentário', errorMessage(error));
    }
  }

  if (loading) return <InlineLoading description="Carregando rádio link…" />;
  if (notFound || !link) {
    return (
      <div className="status-page">
        <p className="status-page__code">404</p>
        <h1>Rádio link não encontrado</h1>
        <p>O enlace pode ter sido excluído ou o endereço está incorreto.</p>
        <Link to="/radio-links"><Button>Voltar para Rádio Links</Button></Link>
      </div>
    );
  }

  const hasBothCoordinates = Boolean(
    stationA?.latitude && stationA?.longitude && stationB?.latitude && stationB?.longitude,
  );

  function endpointFields(station: StationDetails | null) {
    return [
      ['Site ID', station?.site_id ?? '—'],
      ['Estação', station?.station_id ?? '—'],
      ['Endereço ID', station?.endereco_id ?? '—'],
      ['Tipo de elemento', station?.tipo_de_elemento ?? '—'],
      ['Tecnologia', station?.tecnologia ?? '—'],
      ['Status', station?.status ?? '—'],
      ['Regional', station?.regional ?? '—'],
      ['Município/UF', [station?.municipio, station?.estado].filter(Boolean).join('/') || '—'],
    ];
  }

  function endpointSection(title: string, subtitle: string, station: StationDetails | null, stationId: number) {    return (
      <Tile className="nokia-detail-section radio-link-endpoint">
        <div className="nokia-section-header">
          <div>
            <h2>{title}</h2>
            <p>{subtitle} — {station?.site_id ?? '—'}</p>
          </div>
          <Button kind="ghost" size="sm" onClick={() => navigate(`/stations/${stationId}`)}>
            Abrir estação
          </Button>
        </div>
        <div className="station-address-block">
          <div><span>Endereço</span><strong>{addressOf(station)}</strong></div>
          <div>
            <span>Coordenadas</span>
            <div className="station-summary-value">
              <strong>{coordinatesOf(station) || 'Não informadas'}</strong>
              {coordinatesOf(station) && (
                <Button
                  kind="ghost"
                  size="xs"
                  hasIconOnly
                  renderIcon={Copy}
                  iconDescription="Copiar coordenadas"
                  onClick={() => void copyCoordinates(station)}
                />
              )}
            </div>
          </div>
        </div>
        <div className="station-field-groups">
          <div className="station-field-group">
            <dl>
              {endpointFields(station).map(([label, value]) => (
                <div key={label}><dt>{label}</dt><dd>{valueOrDash(value)}</dd></div>
              ))}
            </dl>
          </div>
        </div>
      </Tile>
    );
  }

  return (
    <div className="page">
      <Breadcrumb noTrailingSlash className="doc-breadcrumb">
        <BreadcrumbItem><Link to="/radio-links">Rádio Links</Link></BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>{link.name}</BreadcrumbItem>
      </Breadcrumb>

      <PageHeader
        eyebrow="Rede"
        title={link.name}
        description={
          <span className="tag-row">
            <Tag type={radioLinkStatusTagType(link.status)} size="sm">{radioLinkStatusLabel(link.status)}</Tag>
            {link.technology && <Tag type="blue" size="sm">{link.technology}</Tag>}
            {link.code && <Tag type="cool-gray" size="sm">{link.code}</Tag>}
          </span>
        }
        actions={
          <>
            <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/radio-links')}>Voltar</Button>
            <OverflowMenu
              aria-label="Ações do rádio link"
              iconDescription="Ações do rádio link"
              renderIcon={OverflowMenuVertical}
              flipped
            >
              <OverflowMenuItem itemText="Editar" onClick={() => setEditOpen(true)} />
              <OverflowMenuItem itemText="Excluir" isDelete hasDivider onClick={() => setDeleteOpen(true)} />
            </OverflowMenu>
          </>
        }
      />

      <div className="station-summary-grid">
        <Tile>
          <span>Ponta A</span>
          <strong>{link.station_a_site_id ?? '—'}</strong>
        </Tile>
        <Tile>
          <span>Ponta B</span>
          <strong>{link.station_b_site_id ?? '—'}</strong>
        </Tile>
        <Tile>
          <span>Distância</span>
          <strong>{formatRadioDistance(link.distance_km)}</strong>
        </Tile>
      </div>

      <div className="nokia-detail-layout radio-link-detail-layout">
        <div className="nokia-detail-main">
          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Informações do enlace</h2><p>Configuração técnica e identificação.</p></div>
            </div>
            <div className="station-field-groups">
              <div className="station-field-group">
                <p className="field-label">Identificação</p>
                <dl>
                  <div><dt>Código</dt><dd>{valueOrDash(link.code)}</dd></div>
                  <div><dt>Status</dt><dd>{radioLinkStatusLabel(link.status)}</dd></div>
                  <div><dt>Tecnologia</dt><dd>{valueOrDash(link.technology)}</dd></div>
                  <div><dt>Frequência</dt><dd>{link.frequency_mhz != null ? `${Number(link.frequency_mhz).toLocaleString('pt-BR')} MHz` : '—'}</dd></div>
                  <div><dt>Capacidade</dt><dd>{link.capacity_mbps != null ? `${Number(link.capacity_mbps).toLocaleString('pt-BR')} Mbps` : '—'}</dd></div>
                  <div><dt>Distância</dt><dd>{formatRadioDistance(link.distance_km)}</dd></div>
                </dl>
              </div>
              <div className="station-field-group">
                <p className="field-label">Auditoria</p>
                <dl>
                  <div><dt>Criado em</dt><dd>{relativeTime(link.created_at)}</dd></div>
                  <div><dt>Atualizado em</dt><dd>{relativeTime(link.updated_at)}</dd></div>
                </dl>
              </div>
            </div>
          </Tile>

          {link.notes && (
            <Tile className="nokia-detail-section">
              <div className="nokia-section-header">
                <div><h2>Observações</h2><p>Anotações do enlace.</p></div>
              </div>
              <p className="preserve-lines">{link.notes}</p>
            </Tile>
          )}

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Mapa do enlace</h2><p>Distribuição geográfica das pontas A e B, com a linha tracejada representando o link.</p></div>
              {hasBothCoordinates && (
                <Button
                  kind="ghost"
                  size="sm"
                  onClick={() => window.open(
                    `https://www.google.com/maps/dir/?api=1&origin=${stationA?.latitude},${stationA?.longitude}&destination=${stationB?.latitude},${stationB?.longitude}`,
                    '_blank',
                  )}
                >
                  Abrir no Google Maps
                </Button>
              )}
            </div>
            {hasBothCoordinates && stationA && stationB ? (
              <div className="radio-link-map">
                <RadioLinkMap
                  key={`${link.station_a_id}-${link.station_b_id}`}
                  pointA={{
                    siteId: stationA.site_id,
                    latitude: stationA.latitude ?? '',
                    longitude: stationA.longitude ?? '',
                    municipio: stationA.municipio,
                    estado: stationA.estado,
                  }}
                  pointB={{
                    siteId: stationB.site_id,
                    latitude: stationB.latitude ?? '',
                    longitude: stationB.longitude ?? '',
                    municipio: stationB.municipio,
                    estado: stationB.estado,
                  }}
                  distanceKm={link.distance_km}
                />
              </div>
            ) : (
              <p className="muted">Informe latitude e longitude nas duas estações para visualizar o mapa do enlace.</p>
            )}
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Arquivos</h2><p>Documentos e anexos do enlace.</p></div>
            </div>
            <div className="inline-form nokia-upload-row">
              <TextInput
                id="radio-link-files"
                type="file"
                labelText="Arquivos (até 20 MB cada)"
                multiple
                disabled={uploading}
                onChange={(event) => {
                  void uploadAttachments(event.target.files);
                  event.target.value = '';
                }}
              />
            </div>
            {uploading && <InlineLoading description="Enviando arquivos…" />}
            {link.attachments.length === 0 ? (
              <p className="muted">Nenhum arquivo anexado.</p>
            ) : (
              <div className="nokia-attachment-group">
                {link.attachments.map((attachment) => (
                  <div key={attachment.id} className="attachment-row">
                    <a href={attachment.url} target="_blank" rel="noreferrer">{attachment.original_name}</a>
                    <span>{attachment.uploader_name ?? 'Usuário removido'} · {relativeTime(attachment.created_at)}</span>
                    <Button
                      kind="danger--ghost"
                      size="xs"
                      hasIconOnly
                      renderIcon={TrashCan}
                      iconDescription="Remover anexo"
                      onClick={() => void deleteAttachment(attachment.id)}
                    />
                  </div>
                ))}
              </div>
            )}
          </Tile>

          <Tile className="nokia-detail-section">
            <div className="nokia-section-header">
              <div><h2>Comentários</h2><p>{link.comments.length} comentário(s).</p></div>
            </div>
            <div className="inline-form">
              <TextArea
                id="radio-link-comment"
                labelText="Novo comentário"
                value={comment}
                onChange={(event) => setComment(event.target.value)}
                rows={2}
              />
              <Button kind="tertiary" onClick={() => void addComment()} disabled={commentBusy}>
                {commentBusy ? 'Publicando…' : 'Comentar'}
              </Button>
            </div>
            {link.comments.length === 0 ? (
              <p className="muted">Nenhum comentário publicado.</p>
            ) : (
              <div className="nokia-comment-list">
                {link.comments.map((item) => (
                  <div key={item.id} className="comment-row">
                    <div>
                      <strong>{item.author_name ?? 'Usuário removido'}</strong>
                      <span>{relativeTime(item.created_at)}</span>
                      <p className="preserve-lines">{item.body}</p>
                    </div>
                    {(user?.is_admin || (item.user_id != null && item.user_id === user?.id)) && (
                      <Button
                        kind="danger--ghost"
                        size="xs"
                        hasIconOnly
                        renderIcon={TrashCan}
                        iconDescription="Remover comentário"
                        onClick={() => void deleteComment(item.id)}
                      />
                    )}
                  </div>
                ))}
              </div>
            )}
          </Tile>
        </div>

        <aside className="nokia-detail-aside">
          {endpointSection('Ponta A', 'Estação de origem', stationA, link.station_a_id)}
          {endpointSection('Ponta B', 'Estação de destino', stationB, link.station_b_id)}
          <Tile className="nokia-detail-aside__card">
            <h2>Formulários</h2>
            {forms.length === 0 ? (
              <p className="muted">Nenhum formulário vinculado a este rádio link.</p>
            ) : (
              <div className="module-main">
                {forms.map((form) => (
                  <div className="module-nav-row" key={form.id}>
                    <button type="button" onClick={() => navigate(`/forms/${form.id}`)}>
                      <span>{form.code} · {form.name}</span>
                      <span className="module-nav-item__count">{form.status}</span>
                    </button>
                  </div>
                ))}
              </div>
            )}
          </Tile>
        </aside>
      </div>

      <RadioLinkFormModal
        open={editOpen}
        link={link}
        onClose={() => setEditOpen(false)}
        onSaved={() => {
          setEditOpen(false);
          void load();
        }}
      />

      <ConfirmModal
        open={deleteOpen}
        title="Excluir rádio link"
        message={`Excluir o rádio link ${link.name}? Essa ação não pode ser desfeita.`}
        confirmLabel="Excluir"
        loading={deleting}
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => void remove()}
      />
    </div>
  );
}
