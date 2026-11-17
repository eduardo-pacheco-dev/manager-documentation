import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ArrowLeft, Download, TrashCan, Upload } from '@carbon/icons-react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  InlineLoading,
  Select,
  SelectItem,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
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
    <div>
      <div className="module-sidebar__heading">
        <p className="field-label">{label}</p>
        <Button kind="ghost" size="sm" onClick={() => onChange([...items, ''])}>
          {addLabel}
        </Button>
      </div>
      <div className="projects-list">
        {items.map((item, index) => (
          <div className="projects-list-row" key={index}>
            <TextInput
              id={`settings-list-${label}-${index}`}
              labelText={`Item ${index + 1}`}
              hideLabel
              placeholder={placeholder}
              value={item}
              onChange={(event) =>
                onChange(items.map((current, position) => (position === index ? event.target.value : current)))
              }
            />
            <Button
              kind="danger--ghost"
              size="sm"
              hasIconOnly
              renderIcon={TrashCan}
              iconDescription="Remover item"
              disabled={items.length <= 1}
              onClick={() => onChange(items.filter((_, position) => position !== index))}
            />
          </div>
        ))}
      </div>
      <p className="field-help">{hint}</p>
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

  if (loading || !settings) return <InlineLoading description="Carregando configurações…" />;

  return (
    <div className="page">
      <Breadcrumb noTrailingSlash className="doc-breadcrumb">
        <BreadcrumbItem><Link to="/forms">Formulários</Link></BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>Configurações</BreadcrumbItem>
      </Breadcrumb>

      <PageHeader
        eyebrow="Documentos"
        title="Configurações de Formulários"
        description="Tipos, status e preferências do módulo. As configurações são individuais para a sua conta."
        actions={
          <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/forms')}>
            Voltar para formulários
          </Button>
        }
      />

      <div className="projects-two-columns">
        <Tile className="nokia-detail-section">
          <div className="nokia-section-header">
            <div>
              <h2>Tipos de formulário</h2>
              <p>Opções disponíveis ao criar ou editar um formulário.</p>
            </div>
          </div>
          <ListEditor
            label="Tipos"
            hint="Os tipos atuais dos formulários existentes continuam disponíveis."
            addLabel="Adicionar tipo"
            placeholder="Ex.: Registro Fotográfico"
            items={settings.types}
            onChange={(types) => setSettings((current) => (current ? { ...current, types } : current))}
          />
        </Tile>

        <Tile className="nokia-detail-section">
          <div className="nokia-section-header">
            <div>
              <h2>Status de formulário</h2>
              <p>Situações disponíveis para acompanhar os formulários.</p>
            </div>
          </div>
          <ListEditor
            label="Status"
            hint="Os status atuais dos formulários existentes continuam disponíveis."
            addLabel="Adicionar status"
            placeholder="Ex.: Pendente"
            items={settings.statuses}
            onChange={(statuses) => setSettings((current) => (current ? { ...current, statuses } : current))}
          />
        </Tile>
      </div>

      <Tile className="nokia-detail-section">
        <div className="nokia-section-header">
          <div>
            <h2>Preferências do módulo</h2>
            <p>Valores usados como padrão ao abrir a lista e criar novos formulários.</p>
          </div>
        </div>

        <div className="projects-form-grid">
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

        <div className="section-actions">
          <Button kind="ghost" onClick={() => navigate('/forms')}>Cancelar</Button>
          <Button onClick={() => void save()} disabled={saving}>
            {saving ? 'Salvando…' : 'Salvar configurações'}
          </Button>
        </div>
      </Tile>

      <Tile className="nokia-detail-section">
        <div className="nokia-section-header">
          <div>
            <h2>Template de configuração</h2>
            <p>Exporte as configurações atuais em Excel para reutilizá-las em outra conta ou ambiente.</p>
          </div>
        </div>

        <div className="projects-two-columns">
          <div className="module-subcard">
            <h3>Exportar template</h3>
            <p className="muted">
              Baixe uma planilha com os tipos, status, visualização, itens por página e template padrão atuais.
            </p>
            <div className="section-actions section-actions--start">
              <Button
                kind="tertiary"
                renderIcon={Download}
                onClick={() => window.open('/api/v1/forms/settings/export', '_blank')}
              >
                Exportar template
              </Button>
            </div>
          </div>

          <div className="module-subcard">
            <h3>Importar template</h3>
            <p className="muted">
              Envie uma planilha no mesmo formato da exportação para substituir as configurações atuais.
            </p>
            <TextInput
              id="settings-template-file"
              type="file"
              labelText="Planilha (.xlsx)"
              accept=".xlsx,.xls"
              disabled={importing}
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (file) importTemplate(file);
                event.target.value = '';
              }}
            />
            {importing && (
              <div className="section-actions section-actions--start">
                <Button kind="tertiary" renderIcon={Upload} disabled>Importando…</Button>
              </div>
            )}
          </div>
        </div>
      </Tile>
    </div>
  );
}
