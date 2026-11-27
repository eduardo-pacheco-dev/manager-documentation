import { useEffect, useState } from 'react';
import { Modal, Select, SelectItem, TextArea, TextInput } from '@carbon/react';
import { StationSearch } from './StationSearch';
import { useToast } from '../contexts/ToastContext';
import { ApiError, api } from '../lib/api';
import { errorMessage } from '../lib/format';
import type { RadioLink } from '../lib/types';

export const RADIO_LINK_STATUS_OPTIONS = [
  { value: 'planejado', label: 'Planejado' },
  { value: 'ativo', label: 'Ativo' },
  { value: 'inativo', label: 'Inativo' },
  { value: 'manutencao', label: 'Manutenção' },
];

export const RADIO_LINK_STATUS_LABELS: Record<string, string> = Object.fromEntries(
  RADIO_LINK_STATUS_OPTIONS.map((item) => [item.value, item.label]),
);

export function radioLinkStatusLabel(status: string): string {
  return RADIO_LINK_STATUS_LABELS[status] ?? status;
}

export function radioLinkStatusTagType(status: string): 'green' | 'blue' | 'purple' | 'gray' | 'red' {
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

export function radioLinkStationLabel(
  site: string | null,
  code: string | null,
  city: string | null,
  state: string | null,
): string {
  const location = [city, state].filter(Boolean).join('/');
  return `${site ?? '—'}${location ? ` — ${location}` : ''}${code ? ` (${code})` : ''}`;
}

export function formatRadioDistance(value: number | null): string {
  return value != null ? `${Number(value).toLocaleString('pt-BR')} km` : '—';
}

interface FormState {
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

const EMPTY_FORM: FormState = {
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

function formFromLink(link: RadioLink): FormState {
  return {
    name: link.name,
    code: link.code ?? '',
    stationA: link.station_a_id,
    stationB: link.station_b_id,
    stationALabel: radioLinkStationLabel(link.station_a_site_id, link.station_a_station_id, link.station_a_municipio, link.station_a_estado),
    stationBLabel: radioLinkStationLabel(link.station_b_site_id, link.station_b_station_id, link.station_b_municipio, link.station_b_estado),
    status: link.status,
    technology: link.technology ?? '',
    frequency: link.frequency_mhz != null ? String(link.frequency_mhz) : '',
    capacity: link.capacity_mbps != null ? String(link.capacity_mbps) : '',
    distance: link.distance_km != null ? String(link.distance_km) : '',
    notes: link.notes ?? '',
  };
}

function numberOrNull(value: string): number | null {
  if (!value.trim()) return null;
  const parsed = Number(value.replace(',', '.'));
  return Number.isFinite(parsed) ? parsed : null;
}

interface Props {
  open: boolean;
  link: RadioLink | null;
  onClose: () => void;
  onSaved: () => void;
}

export function RadioLinkFormModal({ open, link, onClose, onSaved }: Props) {
  const toast = useToast();
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!open) return;
    setForm(link ? formFromLink(link) : EMPTY_FORM);
    setFieldErrors({});
  }, [open, link]);

  function updateForm(patch: Partial<FormState>) {
    setForm((current) => ({ ...current, ...patch }));
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
      if (link) {
        const response = await api.put<{ message: string }>(`/radio-links/${link.id}`, payload);
        toast.success('Rádio link atualizado', response.message);
      } else {
        const response = await api.post<{ message: string }>('/radio-links', payload);
        toast.success('Rádio link criado', response.message);
      }
      onSaved();
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fields);
      }
      toast.error('Não foi possível salvar o rádio link', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      open={open}
      modalHeading={link ? 'Editar rádio link' : 'Novo rádio link'}
      primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
      secondaryButtonText="Cancelar"
      primaryButtonDisabled={saving}
      onRequestSubmit={() => void save()}
      onRequestClose={onClose}
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
          onChange={(id) => updateForm({ stationA: id })}
        />
        <StationSearch
          id="radio-link-station-b"
          labelText="Ponta B (estação)"
          endpoint="/radio-links/stations"
          value={form.stationB}
          valueLabel={form.stationBLabel || null}
          onChange={(id) => updateForm({ stationB: id })}
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
          {RADIO_LINK_STATUS_OPTIONS.map((option) => (
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
  );
}
