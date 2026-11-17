import { useEffect, useState } from 'react';
import { Modal, Select, SelectItem, TextInput } from '@carbon/react';
import { useToast } from '../contexts/ToastContext';
import { api } from '../lib/api';
import { errorMessage } from '../lib/format';
import type { FormSettings, FormTemplateRecord } from '../lib/types';

interface ProjectOption {
  id: number;
  code: string;
  name: string;
}

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
  const [projects, setProjects] = useState<ProjectOption[]>([]);
  const [name, setName] = useState('');
  const [projectId, setProjectId] = useState('');
  const [type, setType] = useState(types[0] ?? 'Registro Fotográfico');
  const [status, setStatus] = useState(statuses[0] ?? 'Pendente');
  const [templateId, setTemplateId] = useState('');
  const [multiple, setMultiple] = useState('0');
  const [saving, setSaving] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!open) return;
    setName('');
    setProjectId('');
    setType(types[0] ?? 'Registro Fotográfico');
    setStatus(statuses[0] ?? 'Pendente');
    setMultiple('0');
    setTemplateId(settings?.default_template_id ? String(settings.default_template_id) : '');
    setFieldErrors({});
    void api
      .get<{ data: { projects: ProjectOption[] } }>('/projects')
      .then((response) => setProjects(response.data?.projects ?? []))
      .catch(() => setProjects([]));
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
        project_id: projectId ? Number(projectId) : null,
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
      <div className="form-grid">
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

        <Select
          id="form-project"
          labelText="Projeto (opcional)"
          value={projectId}
          onChange={(event) => setProjectId(event.target.value)}
        >
          <SelectItem value="" text="Sem projeto" />
          {projects.map((project) => (
            <SelectItem key={project.id} value={String(project.id)} text={`[${project.code}] ${project.name}`} />
          ))}
        </Select>

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

        <Select
          id="form-multiple"
          labelText="Respostas"
          value={multiple}
          onChange={(event) => setMultiple(event.target.value)}
        >
          <SelectItem value="0" text="Resposta única" />
          <SelectItem value="1" text="Múltiplas respostas" />
        </Select>
      </div>
      <p className="form-hint">
        Com resposta única, o mesmo registro é atualizado a cada novo envio. Com múltiplas respostas, cada envio cria um
        registro separado no histórico.
      </p>
    </Modal>
  );
}
