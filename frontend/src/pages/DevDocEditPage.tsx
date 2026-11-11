import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, Save, Upload } from '@carbon/icons-react';
import {
  Button,
  InlineNotification,
  Select,
  SelectItem,
  TextArea,
  TextInput,
  Toggle,
} from '@carbon/react';
import { PageHeader } from '../components/PageHeader';
import { Loading } from '../components/Feedback';
import { useToast } from '../contexts/ToastContext';
import { ApiError, api } from '../lib/api';
import { DOC_CATEGORIES } from '../lib/constants';
import { errorMessage } from '../lib/format';

interface FormState {
  title: string;
  category: string;
  summary: string;
  content: string;
  is_published: boolean;
}

const emptyForm: FormState = {
  title: '',
  category: '',
  summary: '',
  content: '',
  is_published: true,
};

export function DevDocEditPage() {
  const { id } = useParams();
  const isEditing = Boolean(id);
  const navigate = useNavigate();
  const toast = useToast();
  const [form, setForm] = useState<FormState>(emptyForm);
  const [loading, setLoading] = useState(isEditing);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [generalError, setGeneralError] = useState('');
  const fileInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!id) return;
    let active = true;
    api
      .get<{ data: { document: FormState & { id: number } } }>(`/dev-docs/${id}`)
      .then((response) => {
        if (!active) return;
        const document = response.data.document;
        setForm({
          title: document.title,
          category: document.category,
          summary: document.summary ?? '',
          content: document.content,
          is_published: document.is_published,
        });
      })
      .catch((error) => {
        toast.error('Não foi possível carregar o documento', errorMessage(error));
        navigate('/dev-docs');
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [id, navigate, toast]);

  async function save() {
    setSaving(true);
    setErrors({});
    setGeneralError('');
    try {
      const payload = {
        title: form.title,
        category: form.category,
        summary: form.summary,
        content: form.content,
        is_published: form.is_published,
      };
      const response = isEditing
        ? await api.put<{ data: { document: { id: number } } }>(`/dev-docs/${id}`, payload)
        : await api.post<{ data: { document: { id: number } } }>('/dev-docs', payload);
      toast.success(isEditing ? 'Documento atualizado com sucesso' : 'Documento criado com sucesso');
      navigate(`/dev-docs/${response.data.document.id}`);
    } catch (error) {
      if (error instanceof ApiError) {
        setErrors(error.fields);
        if (!Object.keys(error.fields).length) setGeneralError(error.message);
      } else {
        setGeneralError('Não foi possível salvar o documento.');
      }
    } finally {
      setSaving(false);
    }
  }

  async function uploadImage(file: File) {
    setUploading(true);
    try {
      const data = new FormData();
      data.append('file', file);
      const response = await api.upload<{ url: string }>('/uploads', data);
      const textarea = document.getElementById('doc-content') as HTMLTextAreaElement | null;
      const snippet = `![${file.name}](${response.url})`;
      if (textarea) {
        const start = textarea.selectionStart ?? form.content.length;
        const end = textarea.selectionEnd ?? start;
        const next = `${form.content.slice(0, start)}${snippet}${form.content.slice(end)}`;
        setForm((current) => ({ ...current, content: next }));
        requestAnimationFrame(() => {
          textarea.focus();
          textarea.setSelectionRange(start + snippet.length, start + snippet.length);
        });
      } else {
        setForm((current) => ({ ...current, content: `${current.content}\n${snippet}\n` }));
      }
      toast.success('Imagem enviada', 'O link Markdown foi inserido no conteúdo.');
    } catch (error) {
      toast.error('Não foi possível enviar a imagem', errorMessage(error));
    } finally {
      setUploading(false);
      if (fileInput.current) fileInput.current.value = '';
    }
  }

  if (loading) return <Loading description="Carregando documento…" />;

  return (
    <div className="page page--narrow">
      <PageHeader
        eyebrow="Documentação Dev"
        title={isEditing ? 'Editar documento' : 'Novo documento'}
        description="Escreva em Markdown. O HTML é renderizado com sanitização no servidor."
        actions={
          <>
            <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate(isEditing ? `/dev-docs/${id}` : '/dev-docs')}>
              Cancelar
            </Button>
            <Button renderIcon={Save} onClick={() => void save()} disabled={saving}>
              {saving ? 'Salvando…' : isEditing ? 'Salvar alterações' : 'Criar documento'}
            </Button>
          </>
        }
      />

      {generalError && (
        <InlineNotification kind="error" title="Não foi possível salvar" subtitle={generalError} lowContrast hideCloseButton />
      )}

      <div className="form-card">
        <TextInput
          id="doc-title"
          labelText="Título"
          placeholder="Ex.: Como configurar o ambiente"
          value={form.title}
          onChange={(event) => setForm({ ...form, title: event.target.value })}
          invalid={Boolean(errors.title)}
          invalidText={errors.title}
          maxLength={180}
          required
        />
        <div className="form-row">
          <Select
            id="doc-category"
            labelText="Categoria"
            value={form.category}
            onChange={(event) => setForm({ ...form, category: event.target.value })}
            invalid={Boolean(errors.category)}
            invalidText={errors.category}
            required
          >
            <SelectItem value="" text="Selecione uma categoria" />
            {DOC_CATEGORIES.map((category) => (
              <SelectItem key={category.key} value={category.key} text={category.label} />
            ))}
          </Select>
          <Toggle
            id="doc-published"
            labelText="Publicação"
            labelA="Rascunho"
            labelB="Publicado"
            toggled={form.is_published}
            onToggle={(checked) => setForm({ ...form, is_published: checked })}
          />
        </div>
        <TextArea
          id="doc-summary"
          labelText="Resumo"
          helperText="Opcional. Aparece nos cards e na busca. Máximo de 300 caracteres."
          value={form.summary}
          onChange={(event) => setForm({ ...form, summary: event.target.value })}
          invalid={Boolean(errors.summary)}
          invalidText={errors.summary}
          maxLength={300}
          rows={2}
        />
        <div className="content-field">
          <div className="content-field__header">
            <label htmlFor="doc-content">Conteúdo (Markdown)</label>
            <div className="content-field__actions">
              <input
                ref={fileInput}
                type="file"
                accept="image/png,image/jpeg,image/gif,image/webp,image/bmp"
                hidden
                onChange={(event) => {
                  const file = event.target.files?.[0];
                  if (file) void uploadImage(file);
                }}
              />
              <Button
                kind="ghost"
                size="sm"
                renderIcon={Upload}
                onClick={() => fileInput.current?.click()}
                disabled={uploading}
              >
                {uploading ? 'Enviando…' : 'Inserir imagem'}
              </Button>
            </div>
          </div>
          <TextArea
            id="doc-content"
            labelText="Conteúdo"
            hideLabel
            value={form.content}
            onChange={(event) => setForm({ ...form, content: event.target.value })}
            invalid={Boolean(errors.content)}
            invalidText={errors.content}
            rows={24}
            className="markdown-editor"
            placeholder={'# Título\n\nEscreva o conteúdo em Markdown…'}
            required
          />
          <p className="field-help">
            Suporta títulos, listas, tabelas, links, blocos de código e imagens. HTML bruto é escapado.
          </p>
        </div>
      </div>

      <div className="mobile-form-actions">
        <Button kind="ghost" onClick={() => navigate(isEditing ? `/dev-docs/${id}` : '/dev-docs')}>Cancelar</Button>
        <Button renderIcon={Save} onClick={() => void save()} disabled={saving}>
          {saving ? 'Salvando…' : isEditing ? 'Salvar alterações' : 'Criar documento'}
        </Button>
      </div>

      <p className="back-link">
        <Link to="/dev-docs">← Voltar para a wiki</Link>
      </p>
    </div>
  );
}
