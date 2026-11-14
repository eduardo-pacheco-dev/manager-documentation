import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Add, OverflowMenuVertical } from '@carbon/icons-react';
import {
  Button,
  InlineLoading,
  OverflowMenu,
  OverflowMenuItem,
  Pagination,
  Search,
  Select,
  SelectItem,
  Tag,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState } from '../../components/Feedback';
import { ConfirmModal } from '../../components/ConfirmModal';
import {
  RADIO_LINK_STATUS_OPTIONS,
  RadioLinkFormModal,
  formatRadioDistance,
  radioLinkStationLabel,
  radioLinkStatusLabel,
  radioLinkStatusTagType,
} from '../../components/RadioLinkFormModal';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import type { PageMeta, RadioLink } from '../../lib/types';

interface ListResponse {
  data: RadioLink[];
  stats: { total: number; active: number; planned: number; inactive: number; stations: number };
  statuses: string[];
  technologies: string[];
  meta: PageMeta;
}

export function RadioLinksPage() {
  const toast = useToast();
  const navigate = useNavigate();
  const [links, setLinks] = useState<RadioLink[]>([]);
  const [meta, setMeta] = useState<PageMeta | null>(null);
  const [statuses, setStatuses] = useState<string[]>([]);
  const [technologies, setTechnologies] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [technologyFilter, setTechnologyFilter] = useState('');
  const [page, setPage] = useState(1);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<RadioLink | null>(null);
  const [deleting, setDeleting] = useState<RadioLink | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: String(page), per_page: '20' });
      if (search.trim()) params.set('search', search.trim());
      if (statusFilter) params.set('status', statusFilter);
      if (technologyFilter) params.set('technology', technologyFilter);
      const response = await api.get<ListResponse>(`/radio-links?${params.toString()}`);
      setLinks(response.data ?? []);
      setMeta(response.meta);
      setStatuses(response.statuses ?? []);
      setTechnologies(response.technologies ?? []);
    } catch (error) {
      toast.error('Não foi possível carregar os rádio links', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [page, search, statusFilter, technologyFilter, toast]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void load();
    }, 250);
    return () => window.clearTimeout(timer);
  }, [load]);

  function openCreate() {
    setEditing(null);
    setFormOpen(true);
  }

  function openEdit(link: RadioLink) {
    setEditing(link);
    setFormOpen(true);
  }

  async function remove() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await api.delete(`/radio-links/${deleting.id}`);
      toast.success('Rádio link excluído');
      setDeleting(null);
      await load();
    } catch (error) {
      toast.error('Não foi possível excluir o rádio link', errorMessage(error));
    } finally {
      setDeleteBusy(false);
    }
  }

  function rowActions(link: RadioLink) {
    return (
      <OverflowMenu
        aria-label={`Ações de ${link.name}`}
        iconDescription={`Ações de ${link.name}`}
        renderIcon={OverflowMenuVertical}
        size="sm"
        flipped
      >
        <OverflowMenuItem itemText="Detalhes" onClick={() => navigate(`/radio-links/${link.id}`)} />
        <OverflowMenuItem itemText="Editar" onClick={() => openEdit(link)} />
        <OverflowMenuItem itemText="Excluir" isDelete hasDivider onClick={() => setDeleting(link)} />
      </OverflowMenu>
    );
  }

  return (
    <div className="page">
      <PageHeader
        eyebrow="Rede"
        title="Rádio Links"
        description="Gerencie enlaces ponto a ponto entre duas estações (ponta A e ponta B)."
        actions={<Button renderIcon={Add} onClick={openCreate}>Novo rádio link</Button>}
      />

      <section className="toolbar">
        <div className="toolbar__search">
          <Search
            size="lg"
            labelText="Buscar rádio links"
            placeholder="Buscar por nome, código, estação ou município"
            value={search}
            onChange={(event) => {
              setSearch(event.target.value);
              setPage(1);
            }}
            onClear={() => {
              setSearch('');
              setPage(1);
            }}
          />
        </div>
        <div className="toolbar__filters">
          <Select
            id="radio-link-status"
            labelText="Status"
            size="sm"
            value={statusFilter}
            onChange={(event) => {
              setStatusFilter(event.target.value);
              setPage(1);
            }}
          >
            <SelectItem value="" text="Todos" />
            {(statuses.length > 0 ? statuses : RADIO_LINK_STATUS_OPTIONS.map((item) => item.value)).map((status) => (
              <SelectItem key={status} value={status} text={radioLinkStatusLabel(status)} />
            ))}
          </Select>
          <Select
            id="radio-link-technology"
            labelText="Tecnologia"
            size="sm"
            value={technologyFilter}
            onChange={(event) => {
              setTechnologyFilter(event.target.value);
              setPage(1);
            }}
          >
            <SelectItem value="" text="Todas" />
            {technologies.map((technology) => (
              <SelectItem key={technology} value={technology} text={technology} />
            ))}
          </Select>
        </div>
      </section>

      {loading ? (
        <InlineLoading description="Carregando rádio links…" />
      ) : links.length === 0 ? (
        <EmptyState
          title="Nenhum rádio link encontrado"
          description="Cadastre o primeiro enlace entre duas estações."
          action={<Button renderIcon={Add} onClick={openCreate}>Novo rádio link</Button>}
        />
      ) : (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table" aria-label="Lista de rádio links">
              <thead>
                <tr>
                  <th>Rádio link</th>
                  <th>Ponta A</th>
                  <th>Ponta B</th>
                  <th className="hide-sm">Status</th>
                  <th className="hide-md">Tecnologia</th>
                  <th className="hide-md">Distância</th>
                  <th className="hide-sm">Atualizado</th>
                  <th className="cell-actions">Ações</th>
                </tr>
              </thead>
              <tbody>
                {links.map((link) => (
                  <tr key={link.id}>
                    <td>
                      <Link to={`/radio-links/${link.id}`} className="cell-primary">
                        {link.name}
                      </Link>
                      <span className="cell-secondary">{link.code || 'Sem código'}</span>
                    </td>
                    <td>{radioLinkStationLabel(link.station_a_site_id, link.station_a_station_id, link.station_a_municipio, link.station_a_estado)}</td>
                    <td>{radioLinkStationLabel(link.station_b_site_id, link.station_b_station_id, link.station_b_municipio, link.station_b_estado)}</td>
                    <td className="hide-sm">
                      <Tag type={radioLinkStatusTagType(link.status)} size="sm">{radioLinkStatusLabel(link.status)}</Tag>
                    </td>
                    <td className="hide-md">{link.technology || '—'}</td>
                    <td className="hide-md">{formatRadioDistance(link.distance_km)}</td>
                    <td className="hide-sm">{relativeTime(link.updated_at)}</td>
                    <td className="cell-actions">{rowActions(link)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {meta && meta.total > 0 && (
            <div className="table-footer">
              <span className="table-footer__order">
                {meta.total.toLocaleString('pt-BR')} rádio link(s)
              </span>
              <Pagination
                page={meta.current_page}
                pageSize={meta.per_page}
                pageSizes={[20]}
                totalItems={meta.total}
                onChange={({ page: nextPage }) => setPage(nextPage)}
                size="sm"
              />
            </div>
          )}
        </div>
      )}

      <RadioLinkFormModal
        open={formOpen}
        link={editing}
        onClose={() => setFormOpen(false)}
        onSaved={() => {
          setFormOpen(false);
          void load();
        }}
      />

      <ConfirmModal
        open={deleting !== null}
        title="Excluir rádio link"
        message={deleting ? `Excluir o rádio link ${deleting.name}? Essa ação não pode ser desfeita.` : ''}
        confirmLabel="Excluir"
        loading={deleteBusy}
        onCancel={() => setDeleting(null)}
        onConfirm={() => void remove()}
      />
    </div>
  );
}
