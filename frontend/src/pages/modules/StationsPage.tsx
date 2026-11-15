import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Add, Edit, Grid, OverflowMenuVertical, Table as TableIcon, TrashCan } from '@carbon/icons-react';
import {
  Button,
  InlineLoading,
  Modal,
  OverflowMenu,
  OverflowMenuItem,
  Pagination,
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
import { StationFormModal } from '../../components/StationFormModal';
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
interface Data { data: Station[]; stats: { total: number; sites: number; municipios: number }; statuses: string[]; tecnologias: string[]; classificacoes: string[]; meta: { current_page: number; last_page: number; per_page: number; total: number } }
type ViewMode = 'table' | 'cards';

function statusTagType(value: string): 'green' | 'gray' {
  return ['ativo', 'ativa', 'aquisitado'].includes(value.toLowerCase()) ? 'green' : 'gray';
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
  const [classificacao, setClassificacao] = useState('');
  const [view, setView] = useState<ViewMode>('table');
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(15);
  const [modal, setModal] = useState(false);
  const [editing, setEditing] = useState<Station | null>(null);
  const [detail, setDetail] = useState<Station | null>(null);
  const [comment, setComment] = useState('');
  const [category, setCategory] = useState('attachment');

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
    if (search) params.set('search', search);
    if (status) params.set('status', status);
    if (tech) params.set('tecnologia', tech);
    if (classificacao) params.set('classificacao', classificacao);
    try {
      const response = await api.get<Data>(`/stations?${params.toString()}`);
      if (response.meta && page > response.meta.last_page) {
        setPage(response.meta.last_page);
        return;
      }
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
  }, [search, status, tech, classificacao, page, perPage, toast, detail]);

  useEffect(() => { void load(); /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, [search, status, tech, classificacao, page, perPage]);

  function openCreate() {
    setEditing(null);
    setModal(true);
  }

  function openEdit(item: Station) {
    setEditing(item);
    setModal(true);
  }

  async function openDetail(item: Station) {
    try {
      const response = await api.get<{ data: { station: Station } }>(`/stations/${item.id}`);
      setDetail(response.data.station);
    } catch (error) {
      toast.error('Não foi possível abrir a estação', errorMessage(error));
    }
  }

  async function deleteStation(item: Station) {
    if (!window.confirm(`Excluir a estação ${item.site_id}?`)) return;
    try {
      await api.delete(`/stations/${item.id}`);
      toast.success('Estação excluída.');
      await load();
    } catch (error) {
      toast.error('Não foi possível excluir a estação', errorMessage(error));
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
          <Search size="lg" labelText="Buscar estações" placeholder="Buscar por ID, endereço, município..." value={search} onChange={(event) => { setSearch(event.target.value); setPage(1); }} onClear={() => { setSearch(''); setPage(1); }} />
          <Select id="station-status" labelText="Status" size="sm" value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todos" />
            {(data?.statuses ?? []).map((value) => <SelectItem key={value} value={value} text={value} />)}
          </Select>
          <Select id="station-tech" labelText="Tecnologia" size="sm" value={tech} onChange={(event) => { setTech(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todas" />
            {(data?.tecnologias ?? []).map((value) => <SelectItem key={value} value={value} text={value} />)}
          </Select>
          <Select id="station-classificacao" labelText="Classificação" size="sm" value={classificacao} onChange={(event) => { setClassificacao(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todas" />
            {(data?.classificacoes ?? []).map((value) => <SelectItem key={value} value={value} text={value} />)}
          </Select>
          <div className="toolbar__views" role="group" aria-label="Modo de visualização">
            <Button kind={view === 'table' ? 'primary' : 'ghost'} size="sm" hasIconOnly renderIcon={TableIcon} iconDescription="Tabela" onClick={() => setView('table')} />
            <Button kind={view === 'cards' ? 'primary' : 'ghost'} size="sm" hasIconOnly renderIcon={Grid} iconDescription="Cards" onClick={() => setView('cards')} />
          </div>
        </div>
      </Tile>

      {loading ? <InlineLoading description="Carregando estações…" /> : items.length === 0 ? (
        <EmptyState icon={Add} title="Nenhuma estação encontrada" description="Cadastre a primeira estação ou ajuste os filtros." action={<Button renderIcon={Add} onClick={openCreate}>Nova estação</Button>} />
      ) : view === 'table' ? (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table" aria-label="Lista de estações">
              <thead><tr><th>Site</th><th>Tecnologia</th><th>Classificação</th><th className="hide-sm">Endereço</th><th>Status</th><th className="cell-actions">Ações</th></tr></thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id}>
                    <td><button type="button" className="link-button" onClick={() => navigate(`/stations/${item.id}`)}>{item.site_id}</button><span className="cell-secondary">{item.station_id ? `ESTAÇÃO ${item.station_id}` : '—'}</span></td>
                    <td>{item.tecnologia ?? 'Sem tecnologia'}<span className="cell-secondary">{item.tipo_de_elemento ?? '—'}</span></td>
                    <td>{item.classificacao ?? 'Sem classificação'}<span className="cell-secondary">{item.tipo_de_conexao ?? '—'}</span></td>
                    <td className="hide-sm">{[item.logradouro, item.numero, item.bairro, item.municipio].filter(Boolean).join(', ') || '—'}<span className="cell-secondary">{item.regional ?? 'Sem regional'}</span></td>
                    <td>{item.status ? <Tag type={statusTagType(item.status)} size="sm">{item.status}</Tag> : '—'}</td>
                    <td className="cell-actions">
                      <Button kind="ghost" size="sm" hasIconOnly renderIcon={Edit} iconDescription="Editar" onClick={() => openEdit(item)} />
                      <Button kind="danger--ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={() => void deleteStation(item)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="user-cards">
          {items.map((item) => (
            <article key={item.id} className="user-card">
              <div className="user-card__main">
                <div className="user-card__text">
                  <button type="button" className="link-button" onClick={() => navigate(`/stations/${item.id}`)}>{item.site_id}</button>
                  <span>{item.station_id ? `Estação ${item.station_id}` : 'Sem ID detentora'}</span>
                  <div className="tag-row">
                    {item.status && <Tag type={statusTagType(item.status)} size="sm">{item.status}</Tag>}
                    {item.tecnologia && <Tag type="blue" size="sm">{item.tecnologia}</Tag>}
                    {item.classificacao && <Tag type="gray" size="sm">{item.classificacao}</Tag>}
                  </div>
                </div>
              </div>
              <p className="user-card__meta">{[item.logradouro, item.numero, item.bairro, item.municipio].filter(Boolean).join(', ') || 'Endereço não informado'}</p>
              <div className="user-card__actions">
                <Button kind="ghost" size="sm" renderIcon={Edit} onClick={() => openEdit(item)}>Editar</Button>
                <Button kind="danger--ghost" size="sm" renderIcon={TrashCan} onClick={() => void deleteStation(item)}>Excluir</Button>
              </div>
            </article>
          ))}
        </div>
      )}

      {data && data.meta.total > 0 && (
        <Pagination
          page={page}
          pageSize={perPage}
          pageSizes={[10, 15, 25, 50]}
          totalItems={data.meta.total}
          onChange={({ page: nextPage, pageSize: nextPerPage }) => {
            setPage(nextPage);
            if (nextPerPage !== perPage) {
              setPerPage(nextPerPage);
              setPage(1);
            }
          }}
          itemRangeText={(min, max, total) => `${min}–${max} de ${total} estações`}
          itemsPerPageText="Itens por página"
          pageRangeText={(_current, total) => `de ${total} página(s)`}
          pageSelectLabelText={(total) => `Página de ${total}`}
          pageText={(current) => `Página ${current}`}
          backwardText="Página anterior"
          forwardText="Próxima página"
        />
      )}

      <StationFormModal
        open={modal}
        station={editing}
        onClose={() => setModal(false)}
        onSaved={() => { setModal(false); void load(); }}
      />

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
