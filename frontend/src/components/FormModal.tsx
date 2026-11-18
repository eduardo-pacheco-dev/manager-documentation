import { useEffect, useState } from 'react';
import { Modal, Select, SelectItem, TextInput } from '@carbon/react';
import { useToast } from '../contexts/ToastContext';
import { api } from '../lib/api';
import { errorMessage } from '../lib/format';
import { StationSearch } from './StationSearch';
import { RadioLinkSearch } from './RadioLinkSearch';
import type { FormSettings, FormTemplateRecord } from '../lib/types';

interface Props {
  open: boolean;
  types: string[];
  statuses: string[];
  templates: FormTemplateRecord[];
  settings: FormSettings | null;
  onClose: () => void;
  onSaved: (id: number) => void;
}

export function FormModal({ open, types, statuses, templates, settings, onClose, onSaved }: Props) {
  const toast = useToast();
  const [name, setName] = useState('');
  const [type, setType] = useState(types[0] ?? 'Registro Fotográfico');
  const [status, setStatus] = useState(statuses[0] ?? 'Pendente');
  const [templateId, setTemplateId] = useState('');
  const [multiple, setMultiple] = useState('0');
  const [targetKind, setTargetKind] = useState<'none' | 'station' | 'radio'>('none');
  const [stationId, setStationId] = useState<number | null>(null);
  const [stationLabel, setStationLabel] = useState<string | null>(null);
  const [radioLinkId, setRadioLinkId] = useState<number | null>(null);
  const [radioLinkLabel, setRadioLinkLabel] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!open) return;
    setName('');
    setType(types[0] ?? 'Registro Fotográfico');
    setStatus(statuses[0] ?? 'Pendente');
    setMultiple('0');
    setTargetKind('none');
    setStationId(null);
    setStationLabel(null);
    setRadioLinkId(null);
    setRadioLinkLabel(null);
    setTemplateId(settings?.default_template_id ? String(settings.default_template_id) : '');
    setFieldErrors({});
  }, [open, types, statuses, settings]);

  async function save() {
    if (!name.trim()) {
      setFieldErrors({ name: 'Informe o nome do formulário.' });
      return;
    }
    setSaving(true);
    try {
      const response = await api.post<{ data: { form: { id: number } } }>('/forms', {
        name: name.trim(),
        project_id: null,
        station_id: targetKind === 'station' ? stationId : null,
        radio_link_id: targetKind === 'radio' ? radioLinkId : null,
        type,
        status,
        allows_multiple_responses: multiple === '1',
        template_id: templateId ? Number(templateId) : null,
      });
      toast.success('Formulário criado', 'Continue configurando as seções e perguntas.');
      onSaved(response.data.form.id);
    } catch (error) {
      toast.error('Não foi possível criar o formulário', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      open={open}
      modalHeading="Novo formulário"
      primaryButtonText={saving ? 'Criando…' : 'Criar formulário'}
      secondaryButtonText="Cancelar"
      primaryButtonDisabled={saving || !name.trim()}
      onRequestSubmit={() => void save()}
      onRequestClose={onClose}
      size="lg"
    >
      <div className="modal-form">
        {templates.length > 0 && (
          <Select
            id="form-template"
            labelText="Template (opcional)"
            value={templateId}
            onChange={(event) => {
              const value = event.target.value;
              setTemplateId(value);
              const template = templates.find((item) => String(item.id) === value);
              if (template?.type) setType(template.type);
            }}
          >
            <SelectItem value="" text="Começar do zero" />
            {templates.map((template) => (
              <SelectItem
                key={template.id}
                value={String(template.id)}
                text={`${template.name}${template.type ? ` — ${template.type}` : ''}`}
              />
            ))}
          </Select>
        )}

        <TextInput
          id="form-name"
          labelText="Nome do formulário"
          placeholder="Ex.: Registro Fotográfico Completo - Estrutura"
          value={name}
          invalid={Boolean(fieldErrors.name)}
          invalidText={fieldErrors.name}
          maxLength={255}
          onChange={(event) => {
            setName(event.target.value);
            setFieldErrors({});
          }}
        />

        <div className="form-row">
          <Select id="form-type" labelText="Tipo" value={type} onChange={(event) => setType(event.target.value)}>
            {[type, ...types.filter((item) => item !== type)].map((item) => (
              <SelectItem key={item} value={item} text={item} />
            ))}
          </Select>

          <Select id="form-status" labelText="Status" value={status} onChange={(event) => setStatus(event.target.value)}>
            {[status, ...statuses.filter((item) => item !== status)].map((item) => (
              <SelectItem key={item} value={item} text={item} />
            ))}
          </Select>
        </div>

        <Select
          id="form-target"
          labelText="Vínculo (opcional)"
          value={targetKind}
          onChange={(event) => {
            const value = event.target.value as 'none' | 'station' | 'radio';
            setTargetKind(value);
            if (value !== 'station') {
              setStationId(null);
              setStationLabel(null);
            }
            if (value !== 'radio') {
              setRadioLinkId(null);
              setRadioLinkLabel(null);
            }
          }}
        >
          <SelectItem value="none" text="Sem vínculo" />
          <SelectItem value="station" text="Estação" />
          <SelectItem value="radio" text="Rádio link" />
        </Select>

        {targetKind === 'station' && (
          <StationSearch
            id="form-station"
            value={stationId}
            valueLabel={stationLabel}
            onChange={(id, station) => {
              setStationId(id);
              setStationLabel(station?.site_id ?? null);
            }}
          />
        )}

        {targetKind === 'radio' && (
          <RadioLinkSearch
            id="form-radio-link"
            value={radioLinkId}
            valueLabel={radioLinkLabel}
            onChange={(id, link) => {
              setRadioLinkId(id);
              setRadioLinkLabel(link?.name ?? null);
            }}
          />
        )}

        <Select
          id="form-multiple"
          labelText="Respostas"
          value={multiple}
          onChange={(event) => setMultiple(event.target.value)}
        >
          <SelectItem value="0" text="Resposta única" />
          <SelectItem value="1" text="Múltiplas respostas" />
        </Select>
        <p className="field-help">
          Com resposta única, o mesmo registro é atualizado a cada novo envio. Com múltiplas respostas, cada envio cria um
          registro separado no histórico.
        </p>
      </div>
    </Modal>
  );
}
