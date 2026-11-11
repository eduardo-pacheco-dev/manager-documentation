import { useCallback, useEffect, useRef, useState } from 'react';
import { ComboBox } from '@carbon/react';
import { api } from '../lib/api';

export interface StationOption {
  id: number;
  site_id: string;
  station_id: string | null;
  endereco_id: string | null;
  logradouro: string | null;
  municipio: string | null;
  estado: string | null;
}

function stationLabel(station: StationOption): string {
  const location = [station.municipio, station.estado].filter(Boolean).join('/');
  return `${station.site_id}${location ? ` — ${location}` : ''}${station.station_id ? ` (${station.station_id})` : ''}`;
}

interface Props {
  id: string;
  value: number | null;
  valueLabel?: string | null;
  onChange: (id: number | null, station?: StationOption) => void;
  disabled?: boolean;
}

export function StationSearch({ id, value, valueLabel, onChange, disabled }: Props) {
  const [options, setOptions] = useState<StationOption[]>([]);
  const [selectedStation, setSelectedStation] = useState<StationOption | null>(null);
  const [loading, setLoading] = useState(false);
  const timer = useRef<number | undefined>(undefined);

  const load = useCallback(async (search: string) => {
    setLoading(true);
    try {
      const response = await api.get<{ data: StationOption[] }>(`/demands/stations?limit=30&search=${encodeURIComponent(search)}`);
      setOptions(response.data ?? []);
    } catch {
      setOptions([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load('');
    return () => { if (timer.current) window.clearTimeout(timer.current); };
  }, [load]);

  useEffect(() => {
    if (!value) {
      setSelectedStation(null);
      return;
    }
    setSelectedStation((current) => {
      if (current?.id === value) return current;
      const found = options.find((station) => station.id === value);
      if (found) return found;
      if (valueLabel) {
        return {
          id: value, site_id: valueLabel, station_id: null, endereco_id: null,
          logradouro: null, municipio: null, estado: null,
        };
      }
      return current;
    });
  }, [value, valueLabel, options]);

  return (
    <ComboBox
      id={id}
      titleText="Estação"
      placeholder={loading ? 'Buscando estações…' : 'Buscar por Site ID, estação, endereço ou município'}
      items={options}
      itemToString={(item) => (item ? stationLabel(item) : '')}
      selectedItem={selectedStation}
      onInputChange={(input) => {
        if (selectedStation && input !== stationLabel(selectedStation)) {
          setSelectedStation(null);
          onChange(null);
        }
        if (timer.current) window.clearTimeout(timer.current);
        timer.current = window.setTimeout(() => { void load(input); }, 300);
      }}
      onChange={({ selectedItem }) => {
        const station = selectedItem ?? null;
        setSelectedStation(station);
        onChange(station?.id ?? null, station ?? undefined);
      }}
      disabled={disabled}
      helperText="Digite para buscar entre as estações cadastradas."
    />
  );
}
