import { useCallback, useEffect, useState } from 'react';
import {
  Add,
  Copy,
  Download,
  Edit,
  Launch,
  Link as LinkIcon,
  Power,
  QrCode,
  TrashCan,
  View,
} from '@carbon/icons-react';
import {
  Button,
  Checkbox,
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
import { ConfirmModal } from '../../components/ConfirmModal';
import { useAuth } from '../../contexts/AuthContext';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';

interface ShortLink {
  id: number; user_id: number; code: string; target_url: string; title: string | null;
  description: string | null; is_active: boolean; expires_at: string | null;
  click_count: number; unique_clicks: number; last_click_at: string | null;
  created_at: string; updated_at: string; creator_name: string | null;
  expired: boolean; short_url: string;
}
interface ShortLinkStats {
  total: number; active: number; inactive: number; expired: number; clicks: number; unique_clicks: number;
}
interface ShortLinkClick { id: number; clicked_at: string; referrer: string | null; user_agent: string | null }
interface Daily { day: string; count: number; unique_clicks: number }
interface Source { source: string; count: number }
interface ListData {
  data: ShortLink[]; stats: ShortLinkStats;
  meta: { current_page: number; last_page: number; per_page: number; total: number };
  filters: { search: string; status: string };
}
interface DetailData {
  link: ShortLink; short_url: string; clicks: ShortLinkClick[]; daily: Daily[]; sources: Source[];
}

const EMPTY_FORM = {
  title: '', target_url: '', code: '', description: '', expires_at: '', is_active: true,
};

function shortURL(link: ShortLink): string {
  return link.short_url || `${window.location.origin}/s/${link.code}`;
}

function localDateTime(value: string | null): string {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value.slice(0, 16);
  const pad = (part: number) => String(part).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function statusTag(link: ShortLink) {
  if (link.expired) return <Tag type="red" size="sm">Expirado</Tag>;
  if (!link.is_active) return <Tag type="gray" size="sm">Inativo</Tag>;
  return <Tag type="green" size="sm">Ativo</Tag>;
}

export function LinksPage() {
  const toast = useToast();
  const { user } = useAuth();
  const [data, setData] = useState<ListData | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<ShortLink | null>(null);
  const [form, setForm] = useState({ ...EMPTY_FORM });
  const [saving, setSaving] = useState(false);
  const [detail, setDetail] = useState<DetailData | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [qrLink, setQrLink] = useState<ShortLink | null>(null);
  const [deleteLink, setDeleteLink] = useState<ShortLink | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({ page: String(page) });
    if (search) params.set('search', search);
    if (status) params.set('status', status);
    try {
      const response = await api.get<ListData>(`/links?${params.toString()}`);
      setData(response);
    } catch (error) {
      toast.error('Não foi possível carregar os links', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [page, search, status, toast]);

  useEffect(() => { void load(); }, [load]);

  function openCreate() {
    setEditing(null);
    setForm({ ...EMPTY_FORM });
    setCreateOpen(true);
  }

  function openEdit(link: ShortLink) {
    setEditing(link);
    setForm({
      title: link.title ?? '', target_url: link.target_url, code: link.code,
      description: link.description ?? '',
      expires_at: localDateTime(link.expires_at),
      is_active: link.is_active,
    });
    setCreateOpen(true);
  }

  async function save() {
    setSaving(true);
    try {
      const payload = { ...form, expires_at: form.expires_at || '' };
      if (editing) {
        await api.put(`/links/${editing.id}`, payload);
        toast.success('Link atualizado com sucesso.');
      } else {
        await api.post('/links', payload);
        toast.success('Link curto criado com sucesso.');
      }
      setCreateOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar o link', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function openDetail(link: ShortLink) {
    setDetailLoading(true);
    setDetail({ link, short_url: shortURL(link), clicks: [], daily: [], sources: [] });
    try {
      const response = await api.get<{ data: DetailData }>(`/links/${link.id}`);
      setDetail(response.data);
    } catch (error) {
      toast.error('Não foi possível carregar as estatísticas', errorMessage(error));
      setDetail(null);
    } finally {
      setDetailLoading(false);
    }
  }

  async function copyLink(link: ShortLink) {
    try {
      await navigator.clipboard.writeText(shortURL(link));
      toast.success('Link copiado para a área de transferência.');
    } catch {
      toast.error('Não foi possível copiar o link', 'Copie manualmente: ' + shortURL(link));
    }
  }

  async function toggleLink(link: ShortLink) {
    try {
      const response = await api.patch<{ message: string }>(`/links/${link.id}/toggle`);
      toast.success(response.message);
      await load();
      if (detail?.link.id === link.id) await openDetail({ ...link, is_active: !link.is_active });
    } catch (error) {
      toast.error('Não foi possível alterar o link', errorMessage(error));
    }
  }

  async function removeLink() {
    if (!deleteLink) return;
    setDeleting(true);
    try {
      await api.delete(`/links/${deleteLink.id}`);
      toast.success('Link excluído com sucesso.');
      setDeleteLink(null);
      await load();
    } catch (error) {
      toast.error('Não foi possível excluir o link', errorMessage(error));
    } finally {
      setDeleting(false);
    }
  }

  function exportLinks() {
    const params = new URLSearchParams();
    if (search) params.set('search', search);
    if (status) params.set('status', status);
    window.open(`/api/v1/links/export?${params.toString()}`, '_blank');
  }

  const items = data?.data ?? [];
  const stats = data?.stats;
  const canManage = (link: ShortLink) => Boolean(user?.is_admin || user?.id === link.user_id);
  const maxDaily = Math.max(1, ...(detail?.daily ?? []).map((item) => item.count));

  return (
    <div className="page">
      <PageHeader
        eyebrow="Utilidades"
        title="Links curtos"
        description="Crie links curtos com QR Code, expiração e estatísticas de acesso."
        actions={
          <>
            <Button kind="ghost" renderIcon={Download} onClick={exportLinks}>Exportar</Button>
            <Button renderIcon={Add} onClick={openCreate}>Novo link</Button>
          </>
        }
      />

      {stats && (
        <div className="stats-grid">
          <Tile className="stat-card stat-card--blue"><span className="stat-card__label">Links</span><strong className="stat-card__value">{stats.total}</strong></Tile>
          <Tile className="stat-card stat-card--green"><span className="stat-card__label">Ativos</span><strong className="stat-card__value">{stats.active}</strong></Tile>
          <Tile className="stat-card stat-card--purple"><span className="stat-card__label">Cliques</span><strong className="stat-card__value">{stats.clicks}</strong></Tile>
          <Tile className="stat-card stat-card--blue"><span className="stat-card__label">Visitantes únicos</span><strong className="stat-card__value">{stats.unique_clicks}</strong></Tile>
        </div>
      )}

      <Tile className="module-main__header">
        <div className="module-toolbar">
          <Search
            size="lg"
            labelText="Buscar links"
            placeholder="Buscar por código, destino, título ou criador..."
            value={search}
            onChange={(event) => { setSearch(event.target.value); setPage(1); }}
            onClear={() => setSearch('')}
          />
          <Select id="links-status" labelText="Status" size="sm" value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}>
            <SelectItem value="" text="Todos" />
            <SelectItem value="active" text="Ativos" />
            <SelectItem value="inactive" text="Inativos" />
            <SelectItem value="expired" text="Expirados" />
          </Select>
        </div>
      </Tile>

      {loading ? <InlineLoading description="Carregando links…" /> : items.length === 0 ? (
        <EmptyState
          icon={LinkIcon}
          title="Nenhum link curto encontrado"
          description="Crie um link curto para compartilhar documentos e páginas internas."
          action={<Button renderIcon={Add} onClick={openCreate}>Novo link</Button>}
        />
      ) : (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Link curto</th>
                  <th className="hide-md">Destino</th>
                  <th>Cliques</th>
                  <th className="hide-sm">Criador</th>
                  <th>Status</th>
                  <th className="hide-sm">Criado em</th>
                  <th>Ações</th>
                </tr>
              </thead>
              <tbody>
                {items.map((link) => (
                  <tr key={link.id}>
                    <td>
                      <button type="button" className="link-button" onClick={() => void openDetail(link)}>
                        /s/{link.code}
                      </button>
                      <span className="cell-secondary">{link.title || 'Sem título'}</span>
                    </td>
                    <td className="hide-md link-target">
                      <a href={link.target_url} target="_blank" rel="noreferrer">{link.target_url}</a>
                    </td>
                    <td>
                      <strong>{link.click_count}</strong>
                      <span className="cell-secondary">{link.unique_clicks} único(s)</span>
                    </td>
                    <td className="hide-sm">{link.creator_name ?? 'Usuário removido'}</td>
                    <td>{statusTag(link)}</td>
                    <td className="hide-sm">{relativeTime(link.created_at)}</td>
                    <td className="cell-actions">
                      <Button kind="ghost" size="sm" hasIconOnly renderIcon={Copy} iconDescription="Copiar link" onClick={() => void copyLink(link)} />
                      <Button kind="ghost" size="sm" hasIconOnly renderIcon={Launch} iconDescription="Abrir link" onClick={() => window.open(shortURL(link), '_blank')} />
                      <Button kind="ghost" size="sm" hasIconOnly renderIcon={QrCode} iconDescription="QR Code" onClick={() => setQrLink(link)} />
                      {canManage(link) && (
                        <>
                          <Button kind="ghost" size="sm" hasIconOnly renderIcon={Edit} iconDescription="Editar" onClick={() => openEdit(link)} />
                          <Button kind="ghost" size="sm" hasIconOnly renderIcon={Power} iconDescription={link.is_active ? 'Desativar' : 'Ativar'} onClick={() => void toggleLink(link)} />
                          <Button kind="danger--ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={() => setDeleteLink(link)} />
                        </>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="table-footer"><span className="table-footer__order">{data ? `${data.meta.total} link(s)` : ''}</span></div>
        </div>
      )}

      {data && data.meta.last_page > 1 && (
        <div className="pager">
          <Button kind="ghost" size="sm" disabled={page <= 1} onClick={() => setPage((current) => Math.max(1, current - 1))}>Anterior</Button>
          <span>Página {data.meta.current_page} de {data.meta.last_page}</span>
          <Button kind="ghost" size="sm" disabled={page >= data.meta.last_page} onClick={() => setPage((current) => current + 1)}>Próxima</Button>
        </div>
      )}

      <Modal
        open={createOpen}
        modalHeading={editing ? 'Editar link curto' : 'Novo link curto'}
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving || !form.target_url.trim()}
        onRequestSubmit={() => void save()}
        onRequestClose={() => setCreateOpen(false)}
        size="md"
      >
        <div className="modal-form">
          <TextInput
            id="link-target"
            labelText="URL de destino *"
            placeholder="https://exemplo.com/pagina"
            value={form.target_url}
            onChange={(event) => setForm({ ...form, target_url: event.target.value })}
            required
          />
          <div className="form-row">
            <TextInput
              id="link-title"
              labelText="Título"
              value={form.title}
              onChange={(event) => setForm({ ...form, title: event.target.value })}
            />
            <TextInput
              id="link-code"
              labelText="Código personalizado"
              placeholder="gerado automaticamente"
              value={form.code}
              onChange={(event) => setForm({ ...form, code: event.target.value })}
              helperText="3 a 32 caracteres: letras minúsculas, números, hífen ou sublinhado."
            />
          </div>
          <TextArea
            id="link-description"
            labelText="Descrição"
            rows={2}
            value={form.description}
            onChange={(event) => setForm({ ...form, description: event.target.value })}
          />
          <div className="form-row">
            <TextInput
              id="link-expires"
              type="datetime-local"
              labelText="Expira em"
              value={form.expires_at}
              onChange={(event) => setForm({ ...form, expires_at: event.target.value })}
            />
            <div className="link-form-check">
              <Checkbox
                id="link-active"
                labelText="Link ativo"
                checked={form.is_active}
                onChange={(_, { checked }) => setForm({ ...form, is_active: checked })}
              />
            </div>
          </div>
          {form.code && (
            <p className="muted">Prévia: <strong>{window.location.origin}/s/{form.code.toLowerCase()}</strong></p>
          )}
        </div>
      </Modal>

      <Modal
        open={qrLink !== null}
        passiveModal
        modalHeading={qrLink ? `QR Code — /s/${qrLink.code}` : 'QR Code'}
        onRequestClose={() => setQrLink(null)}
        size="sm"
      >
        {qrLink && (
          <div className="link-qr">
            <img src={`/api/v1/links/${qrLink.id}/qr`} alt={`QR Code do link ${qrLink.code}`} />
            <code>{shortURL(qrLink)}</code>
            <div className="link-qr__actions">
              <Button size="sm" renderIcon={Copy} onClick={() => void copyLink(qrLink)}>Copiar link</Button>
              <a className="cds--btn cds--btn--tertiary cds--btn--sm" href={`/api/v1/links/${qrLink.id}/qr`} download={`${qrLink.code}.png`}>Baixar PNG</a>
            </div>
          </div>
        )}
      </Modal>

      <Modal
        open={detail !== null}
        modalHeading={detail ? `Estatísticas — /s/${detail.link.code}` : 'Estatísticas'}
        primaryButtonText="Fechar"
        onRequestSubmit={() => setDetail(null)}
        onRequestClose={() => setDetail(null)}
        size="lg"
      >
        {detail && (
          <div className="link-detail">
            <div className="link-detail__summary">
              <div><span>URL curta</span><strong>{detail.short_url}</strong></div>
              <div><span>Destino</span><a href={detail.link.target_url} target="_blank" rel="noreferrer">{detail.link.target_url}</a></div>
              <div><span>Status</span>{statusTag(detail.link)}</div>
            </div>
            <div className="mini-stats">
              <div><strong>{detail.link.click_count}</strong><span>Cliques</span></div>
              <div><strong>{detail.link.unique_clicks}</strong><span>Únicos</span></div>
              <div><strong>{detail.link.last_click_at ? relativeTime(detail.link.last_click_at) : '—'}</strong><span>Último acesso</span></div>
              <div><strong>{detail.link.creator_name ?? '—'}</strong><span>Criador</span></div>
            </div>
            <div className="link-detail__actions">
              <Button size="sm" renderIcon={Copy} onClick={() => void copyLink(detail.link)}>Copiar</Button>
              <Button size="sm" kind="tertiary" renderIcon={QrCode} onClick={() => setQrLink(detail.link)}>QR Code</Button>
              <Button size="sm" kind="tertiary" renderIcon={View} onClick={() => window.open(detail.short_url, '_blank')}>Abrir</Button>
            </div>

            {detailLoading ? <InlineLoading description="Carregando estatísticas…" /> : (
              <>
                <p className="field-label">Cliques nos últimos 14 dias</p>
                {detail.daily.length === 0 ? <p className="muted">Nenhum clique registrado.</p> : (
                  <div className="link-chart">
                    {detail.daily.map((item) => (
                      <div key={item.day} className="link-chart__bar" title={`${item.day}: ${item.count} clique(s), ${item.unique_clicks} único(s)`}>
                        <i style={{ height: `${Math.max(4, (item.count / maxDaily) * 100)}%` }} />
                        <span>{item.day.slice(8)}/{item.day.slice(5, 7)}</span>
                      </div>
                    ))}
                  </div>
                )}

                <div className="link-detail__grid">
                  <div>
                    <p className="field-label">Origens</p>
                    {detail.sources.length === 0 ? <p className="muted">Sem dados.</p> : detail.sources.map((source) => (
                      <div key={source.source} className="link-source-row">
                        <span>{source.source}</span><strong>{source.count}</strong>
                      </div>
                    ))}
                  </div>
                  <div>
                    <p className="field-label">Acessos recentes</p>
                    {detail.clicks.length === 0 ? <p className="muted">Sem acessos.</p> : detail.clicks.slice(0, 8).map((click) => (
                      <div key={click.id} className="link-click-row">
                        <span>{relativeTime(click.clicked_at)}</span>
                        <em>{click.referrer || 'Direto'}</em>
                      </div>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>
        )}
      </Modal>

      <ConfirmModal
        open={deleteLink !== null}
        title="Excluir link curto"
        message={deleteLink ? `Excluir /s/${deleteLink.code}? As estatísticas de acesso também serão removidas.` : ''}
        confirmLabel="Excluir link"
        loading={deleting}
        onCancel={() => setDeleteLink(null)}
        onConfirm={() => void removeLink()}
      />
    </div>
  );
}
