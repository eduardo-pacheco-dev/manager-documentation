import { useEffect, useState } from 'react';
import {
  Button,
  Modal,
  ProgressIndicator,
  ProgressStep,
  Select,
  SelectItem,
  TextInput,
  Toggle,
} from '@carbon/react';
import { useToast } from '../contexts/ToastContext';
import { api } from '../lib/api';
import { errorMessage } from '../lib/format';

const EMPTY = {
  site_id: '', endereco_id: '', tipo_de_elemento: '', tecnologia: '', tipo_de_conexao: '',
  classificacao: '', detentor_area: '', tipo_de_infra: '', tipo_de_ev: '', tipo_de_logradouro: '',
  logradouro: '', numero: '', complemento: '', bairro: '', municipio: '', estado: '', cep: '',
  regional: '', latitude: '', longitude: '', status: '', tipo_da_torre: '', station_id: '',
};

export type StationFormValues = typeof EMPTY;

const STEPS = [
  { key: 'identificacao', label: 'Identificação', description: 'Identificadores do site e da estação.' },
  { key: 'classificacao', label: 'Classificação', description: 'Tecnologia, tipo de conexão e situação da estação.' },
  { key: 'endereco', label: 'Endereço', description: 'Endereço completo do elemento de rede.' },
  { key: 'coordenadas', label: 'Coordenadas', description: 'Latitude e longitude usadas no mapa.' },
];

const UF_OPTIONS = [
  ['AC', 'Acre'], ['AL', 'Alagoas'], ['AP', 'Amapá'], ['AM', 'Amazonas'], ['BA', 'Bahia'],
  ['CE', 'Ceará'], ['DF', 'Distrito Federal'], ['ES', 'Espírito Santo'], ['GO', 'Goiás'],
  ['MA', 'Maranhão'], ['MT', 'Mato Grosso'], ['MS', 'Mato Grosso do Sul'], ['MG', 'Minas Gerais'],
  ['PA', 'Pará'], ['PB', 'Paraíba'], ['PR', 'Paraná'], ['PE', 'Pernambuco'], ['PI', 'Piauí'],
  ['RJ', 'Rio de Janeiro'], ['RN', 'Rio Grande do Norte'], ['RS', 'Rio Grande do Sul'],
  ['RO', 'Rondônia'], ['RR', 'Roraima'], ['SC', 'Santa Catarina'], ['SP', 'São Paulo'],
  ['SE', 'Sergipe'], ['TO', 'Tocantins'],
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

export interface StationFormRecord extends Partial<Record<keyof StationFormValues, string | null>> {
  id: number;
}

interface Props {
  open: boolean;
  station: StationFormRecord | null;
  onClose: () => void;
  onSaved: () => void;
}

export function StationFormModal({ open, station, onClose, onSaved }: Props) {
  const toast = useToast();
  const [form, setForm] = useState<StationFormValues>({ ...EMPTY });
  const [step, setStep] = useState(0);
  const [coordFormat, setCoordFormat] = useState<CoordFormat>('decimal');
  const [dms, setDms] = useState<DmsValues>({ ...EMPTY_DMS });
  const [saving, setSaving] = useState(false);
  const [cepLoading, setCepLoading] = useState(false);

  useEffect(() => {
    if (!open) return;
    if (station) {
      setForm(Object.fromEntries(Object.keys(EMPTY).map((key) => [key, String(station[key as keyof StationFormValues] ?? '')])) as StationFormValues);
    } else {
      setForm({ ...EMPTY });
    }
    setStep(0);
    setCoordFormat('decimal');
    setDms({ ...EMPTY_DMS });
  }, [open, station]);

  const lastStep = STEPS.length - 1;
  const dmsFilled = Boolean(dms.latDeg || dms.latMin || dms.latSec || dms.lngDeg || dms.lngMin || dms.lngSec);
  const dmsInvalid = coordFormat === 'dms' && dmsFilled && (!form.latitude || !form.longitude);

  function goToStep(index: number) {
    if (index === step) return;
    if (index > 0 && !form.site_id.trim()) {
      toast.error('Informe o Site ID para continuar.');
      return;
    }
    setStep(Math.min(Math.max(index, 0), lastStep));
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

  async function lookupCEP() {
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
    if (!form.site_id.trim()) {
      setStep(0);
      toast.error('Informe o Site ID.');
      return;
    }
    setSaving(true);
    try {
      if (station) {
        await api.put(`/stations/${station.id}`, form);
        toast.success('Estação atualizada com sucesso.');
      } else {
        await api.post('/stations', form);
        toast.success('Estação cadastrada com sucesso.');
      }
      onSaved();
    } catch (error) {
      toast.error('Não foi possível salvar a estação', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  function advanceStep() {
    if (step === 0 && !form.site_id.trim()) {
      toast.error('Informe o Site ID para continuar.');
      return;
    }
    if (step < lastStep) {
      setStep(step + 1);
      return;
    }
    void save();
  }

  return (
    <Modal
      open={open}
      modalHeading={station ? 'Editar estação' : 'Nova estação'}
      primaryButtonText={saving ? 'Salvando…' : step === lastStep ? 'Salvar' : 'Próximo'}
      secondaryButtonText={step === 0 ? 'Cancelar' : 'Voltar'}
      primaryButtonDisabled={saving}
      onRequestSubmit={advanceStep}
      onSecondarySubmit={() => (step === 0 ? onClose() : setStep(step - 1))}
      onRequestClose={onClose}
      size="md"
    >
      <div className="modal-form">
        <ProgressIndicator className="modal-steps" currentIndex={step} onChange={goToStep} spaceEqually>
          {STEPS.map((item, index) => (
            <ProgressStep key={item.key} label={item.label} disabled={index > 0 && !form.site_id.trim()} />
          ))}
        </ProgressIndicator>
        <div className="wizard-step__header">
          <h4>{STEPS[step].label}</h4>
          <p className="muted small">{STEPS[step].description}</p>
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
              <Select id="st-estado" labelText="UF" value={form.estado} onChange={(event) => setForm({ ...form, estado: event.target.value })}>
                <SelectItem value="" text="Selecione" />
                {UF_OPTIONS.map(([uf, name]) => <SelectItem key={uf} value={uf} text={`${uf} — ${name}`} />)}
              </Select>
            </div>
            <div className="station-cep-field">
              <TextInput id="st-cep" labelText="CEP" value={form.cep} onChange={(event) => setForm({ ...form, cep: event.target.value })} />
              <Button kind="tertiary" size="sm" onClick={() => void lookupCEP()} disabled={cepLoading}>{cepLoading ? 'Buscando…' : 'Buscar CEP'}</Button>
            </div>
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
  );
}
