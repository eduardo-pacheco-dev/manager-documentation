import { useCallback, useEffect, useState } from 'react';
import { Add, OverflowMenuVertical } from '@carbon/icons-react';
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
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState } from '../../components/Feedback';
import { ConfirmModal } from '../../components/ConfirmModal';
import { StationSearch } from '../../components/StationSearch';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage, relativeTime } from '../../lib/format';
import type { PageMeta } from '../../lib/types';

interface RadioLink {
  id: number;
  name: string;
  code: string | null;
  station_a_id: number;
  station_b_id: number;
  status: string;
  technology: string | null;
  frequency_mhz: number | null;
  capacity_mbps: number | null;
  distance_km: number | null;
  notes: string | null;
  station_a_site_id: string | null;
  station_a_station_id: string | null;
  station_a_municipio: string | null;
  station_a_estado: string | null;
  station_a_tipo_de_elemento: string | null;
  station_b_site_id: string | null;
  station_b_station_id: string | null;
  station_b_municipio: string | null;
  station_b_estado: string | null;
  station_b_tipo_de_elemento: string | null;
  created_at: string;
  updated_at: string;
}

interface ListResponse {
  data: RadioLink[];
  stats: { total: number; active: number; planned: number; inactive: number; stations: number };
  statuses: string[];
  technologies: string[];
  meta: PageMeta;
}

const STATUS_OPTIONS = [
  { value: 'planejado', label: 'Planejado' },
  { value: 'ativo', label: 'Ativo' },
  { value: 'inativo', label: 'Inativo' },
  { value: 'manutencao', label: 'Manutenção' },
];

const STATUS_LABELS: Record<string, string> = Object.fromEntries(STATUS_OPTIONS.map((item) => [item.value, item.label]));

function statusTagType(status: string): 'green' | 'blue' | 'purple' | 'gray' | 'red' {
  switch (status) {
    case 'ativo':
      return 'green';
    case 'planejado':
      return 'blue';
    case 'manutencao':
      return 'purple';
    case 'inativo':
      return 'gray';
    default:
      return 'gray';
  }
}

function stationLabel(site: string | null, code: string | null, city: string | null, state: string | null): string {
  const location = [city, state].filter(Boolean).join('/');
  return `${site ?? '—'}${location ? ` — ${location}` : ''}${code ? ` (${code})` : ''}`;
}

interface LinkForm {
  name: string;
  code: string;
  stationA: number | null;
  stationB: number | null;
  stationALabel: string;
  stationBLabel: string;
  status: string;
  technology: string;
  frequency: string;
  capacity: string;
  distance: string;
  notes: string;
}

const EMPTY_FORM: LinkForm = {
  name: '',
  code: '',
  stationA: null,
  stationB: null,
  stationALabel: '',
  stationBLabel: '',
  status: 'planejado',
  technology: '',
  frequency: '',
  capacity: '',
  distance: '',
  notes: '',
};

export function RadioLinksPage() {
  const toast = useToast();
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
  const [form, setForm] = useState<LinkForm>(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
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
    setForm(EMPTY_FORM);
    setFieldErrors({});
    setFormOpen(true);
  }

  function openEdit(link: RadioLink) {
    setEditing(link);
    setForm({
      name: link.name,
      code: link.code ?? '',
      stationA: link.station_a_id,
      stationB: link.station_b_id,
      stationALabel: stationLabel(link.station_a_site_id, link.station_a_station_id, link.station_a_municipio, link.station_a_estado),
      stationBLabel: stationLabel(link.station_b_site_id, link.station_b_station_id, link.station_b_municipio, link.station_b_estado),
      status: link.status,
      technology: link.technology ?? '',
      frequency: link.frequency_mhz != null ? String(link.frequency_mhz) : '',
      capacity: link.capacity_mbps != null ? String(link.capacity_mbps) : '',
      distance: link.distance_km != null ? String(link.distance_km) : '',
      notes: link.notes ?? '',
    });
    setFieldErrors({});
    setFormOpen(true);
  }

  function updateForm(patch: Partial<LinkForm>) {
    setForm((current) => ({ ...current, ...patch }));
  }

  function numberOrNull(value: string): number | null {
    if (!value.trim()) return null;
    const parsed = Number(value.replace(',', '.'));
    return Number.isFinite(parsed) ? parsed : null;
  }

  async function save() {
    const errors: Record<string, string> = {};
    if (!form.name.trim()) errors.name = 'Informe um nome.';
    if (!form.stationA) errors.station_a_id = 'Selecione a ponta A.';
    if (!form.stationB) errors.station_b_id = 'Selecione a ponta B.';
    if (form.stationA && form.stationA === form.stationB) errors.station_b_id = 'As pontas precisam ser diferentes.';
    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) return;

    const payload = {
      name: form.name.trim(),
      code: form.code.trim(),
      station_a_id: form.stationA,
      station_b_id: form.stationB,
      status: form.status,
      technology: form.technology.trim(),
      frequency_mhz: numberOrNull(form.frequency),
      capacity_mbps: numberOrNull(form.capacity),
      distance_km: numberOrNull(form.distance),
      notes: form.notes.trim(),
    };

    setSaving(true);
    try {
      if (editing) {
        const response = await api.put<{ message: string }>(`/radio-links/${editing.id}`, payload);
        toast.success('Rádio link atualizado', response.message);
      } else {
        const response = await api.post<{ message: string }>('/radio-links', payload);
        toast.success('Rádio link criado', response.message);
      }
      setFormOpen(false);
      await load();
    } catch (error) {
      if (error && typeof error === 'object' && 'fields' in error) {
        setFieldErrors((error as { fields?: Record<string, string> }).fields ?? {});
      }
      toast.error('Não foi possível salvar o rádio link', errorMessage(error));
    } finally {
      setSaving(false);
    }
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
            {(statuses.length > 0 ? statuses : STATUS_OPTIONS.map((item) => item.value)).map((status) => (
              <SelectItem key={status} value={status} text={STATUS_LABELS[status] ?? status} />
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
                      <div className="user-cell">
                        <span className="cell-primary">{link.name}</span>
                        <span className="cell-secondary">{link.code || 'Sem código'}</span>
                      </div>
                    </td>
                    <td>{stationLabel(link.station_a_site_id, link.station_a_station_id, link.station_a_municipio, link.station_a_estado)}</td>
                    <td>{stationLabel(link.station_b_site_id, link.station_b_station_id, link.station_b_municipio, link.station_b_estado)}</td>
                    <td className="hide-sm">
                      <Tag type={statusTagType(link.status)} size="sm">{STATUS_LABELS[link.status] ?? link.status}</Tag>
                    </td>
                    <td className="hide-md">{link.technology || '—'}</td>
                    <td className="hide-md">{link.distance_km != null ? `${Number(link.distance_km).toLocaleString('pt-BR')} km` : '—'}</td>
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

      <Modal
        open={formOpen}
        modalHeading={editing ? 'Editar rádio link' : 'Novo rádio link'}
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving}
        onRequestSubmit={() => void save()}
        onRequestClose={() => setFormOpen(false)}
        size="lg"
      >
        <div className="form-grid">
          <TextInput
            id="radio-link-name"
            labelText="Nome"
            value={form.name}
            onChange={(event) => updateForm({ name: event.target.value })}
            invalid={Boolean(fieldErrors.name)}
            invalidText={fieldErrors.name}
          />
          <TextInput
            id="radio-link-code"
            labelText="Código"
            value={form.code}
            onChange={(event) => updateForm({ code: event.target.value })}
            invalid={Boolean(fieldErrors.code)}
            invalidText={fieldErrors.code}
          />
        </div>

        <div className="form-grid">
          <StationSearch
            id="radio-link-station-a"
            labelText="Ponta A (estação)"
            endpoint="/radio-links/stations"
            value={form.stationA}
            valueLabel={form.stationALabel || null}
            onChange={(id, station) => updateForm({ stationA: id, stationALabel: station ? station.site_id : '' })}
          />
          <StationSearch
            id="radio-link-station-b"
            labelText="Ponta B (estação)"
            endpoint="/radio-links/stations"
            value={form.stationB}
            valueLabel={form.stationBLabel || null}
            onChange={(id, station) => updateForm({ stationB: id, stationBLabel: station ? station.site_id : '' })}
          />
        </div>
        {(fieldErrors.station_a_id || fieldErrors.station_b_id) && (
          <p className="form-error">{fieldErrors.station_a_id ?? fieldErrors.station_b_id}</p>
        )}

        <div className="form-grid">
          <Select
            id="radio-link-status-field"
            labelText="Status"
            value={form.status}
            onChange={(event) => updateForm({ status: event.target.value })}
            invalid={Boolean(fieldErrors.status)}
            invalidText={fieldErrors.status}
          >
            {STATUS_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value} text={option.label} />
            ))}
          </Select>
          <TextInput
            id="radio-link-technology-field"
            labelText="Tecnologia"
            placeholder="Ex.: MW, UHF, 5 GHz"
            value={form.technology}
            onChange={(event) => updateForm({ technology: event.target.value })}
          />
        </div>

        <div className="form-grid">
          <TextInput
            id="radio-link-frequency"
            labelText="Frequência (MHz)"
            type="number"
            value={form.frequency}
            onChange={(event) => updateForm({ frequency: event.target.value })}
            invalid={Boolean(fieldErrors.frequency_mhz)}
            invalidText={fieldErrors.frequency_mhz}
          />
          <TextInput
            id="radio-link-capacity"
            labelText="Capacidade (Mbps)"
            type="number"
            value={form.capacity}
            onChange={(event) => updateForm({ capacity: event.target.value })}
            invalid={Boolean(fieldErrors.capacity_mbps)}
            invalidText={fieldErrors.capacity_mbps}
          />
          <TextInput
            id="radio-link-distance"
            labelText="Distância (km)"
            type="number"
            value={form.distance}
            onChange={(event) => updateForm({ distance: event.target.value })}
            invalid={Boolean(fieldErrors.distance_km)}
            invalidText={fieldErrors.distance_km}
          />
        </div>

        <TextArea
          id="radio-link-notes"
          labelText="Observações"
          value={form.notes}
          onChange={(event) => updateForm({ notes: event.target.value })}
          rows={3}
        />
      </Modal>

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
