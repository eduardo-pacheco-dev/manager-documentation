import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Add, ArrowLeft, Download, TrashCan, Upload } from '@carbon/icons-react';
import { Button, InlineLoading, Select, SelectItem, TextInput } from '@carbon/react';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorDetails, errorMessage } from '../../lib/format';
import type { FormSettings, FormSettingsResponse, FormTemplateRecord } from '../../lib/types';

function ListEditor({
  label,
  hint,
  addLabel,
  placeholder,
  items,
  onChange,
}: {
  label: string;
  hint: string;
  addLabel: string;
  placeholder: string;
  items: string[];
  onChange: (items: string[]) => void;
}) {
  return (
    <div className="form-list-editor">
      <span className="form-eval-label">{label}</span>
      <div className="form-list-editor__rows">
        {items.map((item, index) => (
          <div className="form-list-editor__row" key={index}>
            <span>{index + 1}</span>
            <TextInput
              id={`settings-list-${label}-${index}`}
              labelText=""
              hideLabel
              placeholder={placeholder}
              value={item}
              onChange={(event) =>
                onChange(items.map((current, position) => (position === index ? event.target.value : current)))
              }
            />
            <button
              type="button"
              className="form-icon-button form-icon-button--danger"
              title="Remover item"
              disabled={items.length <= 1}
              onClick={() => onChange(items.filter((_, position) => position !== index))}
            >
              <TrashCan size={16} />
            </button>
          </div>
        ))}
      </div>
      <Button kind="ghost" size="sm" renderIcon={Add} onClick={() => onChange([...items, ''])}>
        {addLabel}
      </Button>
      <p className="form-panel__hint">{hint}</p>
    </div>
  );
}

export function FormsSettingsPage() {
  const navigate = useNavigate();
  const toast = useToast();
  const [settings, setSettings] = useState<FormSettings | null>(null);
  const [templates, setTemplates] = useState<FormTemplateRecord[]>([]);
  const [views, setViews] = useState<Record<string, string>>({});
  const [perPageOptions, setPerPageOptions] = useState<number[]>([12, 24, 36]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [importing, setImporting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.get<FormSettingsResponse>('/forms/settings');
      setSettings(response.data.settings);
      setTemplates(response.data.templates ?? []);
      setViews(response.data.views ?? {});
      setPerPageOptions(response.data.per_page_options ?? [12, 24, 36]);
    } catch (error) {
      toast.error('Não foi possível carregar as configurações', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

  async function save() {
    if (!settings) return;
    setSaving(true);
    try {
      const response = await api.put<{ message: string }>('/forms/settings', settings);
      toast.success('Configurações salvas', response.message);
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar as configurações', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  function importTemplate(file: File) {
    setImporting(true);
    const formData = new FormData();
    formData.append('file', file);
    api
      .upload<{ message: string }>('/forms/settings/import', formData)
      .then((response) => {
        toast.success('Template importado', response.message);
        void load();
      })
      .catch((error) => toast.error('Não foi possível importar o template', errorDetails(error)))
      .finally(() => setImporting(false));
  }

  if (loading || !settings) {
    return <InlineLoading description="Carregando configurações…" />;
  }

  return (
    <div className="page">
      <div className="form-detail-head">
        <div>
          <nav className="form-breadcrumb">
            <button type="button" onClick={() => navigate('/forms')}>Formulários</button>
            <span>/</span>
            <span>Configurações</span>
          </nav>
          <h1>Configurações de Formulários</h1>
          <p className="form-detail-subtitle">
            Ajuste os tipos, status e preferências do módulo. As configurações são individuais para a sua conta.
          </p>
        </div>
        <div className="page-header__actions">
          <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/forms')}>
            Voltar para formulários
          </Button>
        </div>
      </div>

      <div className="form-settings-grid">
        <section className="form-panel">
          <header className="form-panel__head"><h2>Tipos de formulário</h2></header>
          <ListEditor
            label="Lista de tipos"
            hint="Os tipos atuais dos formulários existentes continuam disponíveis."
            addLabel="Adicionar tipo"
            placeholder="Ex.: Registro Fotográfico"
            items={settings.types}
            onChange={(types) => setSettings((current) => (current ? { ...current, types } : current))}
          />
        </section>

        <section className="form-panel">
          <header className="form-panel__head"><h2>Status de formulário</h2></header>
          <ListEditor
            label="Lista de status"
            hint="Os status atuais dos formulários existentes continuam disponíveis."
            addLabel="Adicionar status"
            placeholder="Ex.: Pendente"
            items={settings.statuses}
            onChange={(statuses) => setSettings((current) => (current ? { ...current, statuses } : current))}
          />
        </section>
      </div>

      <section className="form-panel">
        <header className="form-panel__head"><h2>Preferências do módulo</h2></header>
        <div className="form-grid form-grid--three">
          <Select
            id="settings-default-template"
            labelText="Template padrão"
            value={settings.default_template_id ? String(settings.default_template_id) : ''}
            onChange={(event) =>
              setSettings((current) =>
                current ? { ...current, default_template_id: event.target.value ? Number(event.target.value) : null } : current,
              )
            }
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

          <Select
            id="settings-default-view"
            labelText="Visualização padrão"
            value={settings.default_view}
            onChange={(event) =>
              setSettings((current) =>
                current ? { ...current, default_view: event.target.value as FormSettings['default_view'] } : current,
              )
            }
          >
            {Object.entries(views).map(([value, label]) => (
              <SelectItem key={value} value={value} text={label} />
            ))}
          </Select>

          <Select
            id="settings-per-page"
            labelText="Itens por página"
            value={String(settings.per_page)}
            onChange={(event) =>
              setSettings((current) => (current ? { ...current, per_page: Number(event.target.value) } : current))
            }
          >
            {perPageOptions.map((option) => (
              <SelectItem key={option} value={String(option)} text={`${option} itens`} />
            ))}
          </Select>
        </div>
        <div className="form-panel__actions">
          <Button kind="ghost" onClick={() => navigate('/forms')}>Cancelar</Button>
          <Button onClick={() => void save()} disabled={saving}>
            {saving ? 'Salvando…' : 'Salvar configurações'}
          </Button>
        </div>
      </section>

      <section className="form-panel">
        <header className="form-panel__head"><h2>Template de configuração</h2></header>
        <p className="form-panel__hint">
          Exporte as configurações atuais em Excel para reutilizá-las em outra conta ou ambiente.
        </p>
        <div className="form-settings-grid">
          <div className="form-side-card">
            <h3>Exportar template</h3>
            <p className="form-side-hint">Baixe uma planilha com os tipos, status, visualização, itens por página e template padrão atuais.</p>
            <Button
              kind="tertiary"
              renderIcon={Download}
              onClick={() => window.open('/api/v1/forms/settings/export', '_blank')}
            >
              Exportar template
            </Button>
          </div>
          <div className="form-side-card">
            <h3>Importar template</h3>
            <p className="form-side-hint">Envie uma planilha no mesmo formato da exportação para substituir as configurações atuais.</p>
            <label className="form-upload">
              <Upload size={16} />
              <span>{importing ? 'Importando…' : 'Selecionar planilha (.xlsx)'}</span>
              <input
                type="file"
                accept=".xlsx,.xls"
                disabled={importing}
                onChange={(event) => {
                  const file = event.target.files?.[0];
                  if (file) importTemplate(file);
                  event.target.value = '';
                }}
              />
            </label>
          </div>
        </div>
      </section>
    </div>
  );
}
