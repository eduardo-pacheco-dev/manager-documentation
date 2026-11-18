import { useCallback, useEffect, useRef, useState } from 'react';
import { ComboBox } from '@carbon/react';
import { api } from '../lib/api';

export interface RadioLinkOption {
  id: number;
  name: string;
  code: string | null;
}

function radioLinkLabel(link: RadioLinkOption): string {
  return `${link.name}${link.code ? ` (${link.code})` : ''}`;
}

interface Props {
  id: string;
  value: number | null;
  valueLabel?: string | null;
  onChange: (id: number | null, link?: RadioLinkOption) => void;
  disabled?: boolean;
  labelText?: string;
}

export function RadioLinkSearch({ id, value, valueLabel, onChange, disabled, labelText = 'Rádio link' }: Props) {
  const [options, setOptions] = useState<RadioLinkOption[]>([]);
  const [selectedLink, setSelectedLink] = useState<RadioLinkOption | null>(null);
  const [loading, setLoading] = useState(false);
  const timer = useRef<number | undefined>(undefined);

  const load = useCallback(async (search: string) => {
    setLoading(true);
    try {
      const response = await api.get<{ data: RadioLinkOption[] }>(
        `/radio-links/options?limit=30&search=${encodeURIComponent(search)}`,
      );
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
      setSelectedLink(null);
      return;
    }
    setSelectedLink((current) => {
      if (current?.id === value) return current;
      const found = options.find((link) => link.id === value);
      if (found) return found;
      if (valueLabel) {
        return { id: value, name: valueLabel, code: null };
      }
      return current;
    });
  }, [value, valueLabel, options]);

  return (
    <ComboBox
      id={id}
      titleText={labelText}
      placeholder={loading ? 'Buscando rádio links…' : 'Buscar por nome, código ou tecnologia'}
      items={options}
      itemToString={(item) => (item ? radioLinkLabel(item) : '')}
      selectedItem={selectedLink}
      onInputChange={(input) => {
        if (selectedLink && input !== radioLinkLabel(selectedLink)) {
          setSelectedLink(null);
          onChange(null);
        }
        if (timer.current) window.clearTimeout(timer.current);
        timer.current = window.setTimeout(() => { void load(input); }, 300);
      }}
      onChange={({ selectedItem }) => {
        const link = selectedItem ?? null;
        setSelectedLink(link);
        onChange(link?.id ?? null, link ?? undefined);
      }}
      disabled={disabled}
      helperText="Digite para buscar entre os rádio links cadastrados."
    />
  );
}
