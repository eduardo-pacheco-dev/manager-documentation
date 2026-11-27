import { useCallback, useEffect, useMemo, useState } from 'react';
import { Add, Building, Edit } from '@carbon/icons-react';
import {
  Button,
  Checkbox,
  InlineNotification,
  Modal,
  PasswordInput,
  Search,
  TextInput,
} from '@carbon/react';
import { PageHeader } from '../components/PageHeader';
import { ActiveTag, EmptyState, Loading } from '../components/Feedback';
import { useAuth } from '../contexts/AuthContext';
import { useToast } from '../contexts/ToastContext';
import { ApiError, api } from '../lib/api';
import { errorMessage, relativeTime } from '../lib/format';
import type { Company } from '../lib/types';

interface CompanyForm {
  name: string;
  slug: string;
  is_active: boolean;
  admin_name: string;
  admin_email: string;
  admin_password: string;
}

const emptyForm: CompanyForm = {
  name: '',
  slug: '',
  is_active: true,
  admin_name: '',
  admin_email: '',
  admin_password: '',
};

export function CompaniesPage() {
  const toast = useToast();
  const { user, activeCompanyID, setActiveCompany } = useAuth();
  const [companies, setCompanies] = useState<Company[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Company | null>(null);
  const [form, setForm] = useState<CompanyForm>(emptyForm);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState('');
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.get<{ data: Company[] }>('/companies');
      setCompanies(response.data ?? []);
    } catch (error) {
      toast.error('Não foi possível carregar as empresas', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase();
    if (!term) return companies;
    return companies.filter(
      (company) =>
        company.name.toLowerCase().includes(term) || company.slug.toLowerCase().includes(term),
    );
  }, [companies, search]);

  function openCreate() {
    setEditing(null);
    setForm(emptyForm);
    setFieldErrors({});
    setFormError('');
    setModalOpen(true);
  }

  function openEdit(company: Company) {
    setEditing(company);
    setForm({
      name: company.name,
      slug: company.slug,
      is_active: company.is_active,
      admin_name: '',
      admin_email: '',
      admin_password: '',
    });
    setFieldErrors({});
    setFormError('');
    setModalOpen(true);
  }

  async function submitForm() {
    setSaving(true);
    setFieldErrors({});
    setFormError('');
    try {
      if (editing) {
        await api.put(`/companies/${editing.id}`, {
          name: form.name,
          slug: form.slug,
          is_active: form.is_active,
        });
        toast.success('Empresa atualizada com sucesso');
      } else {
        await api.post('/companies', {
          name: form.name,
          slug: form.slug,
          is_active: form.is_active,
          ...(form.admin_email.trim()
            ? {
                admin_name: form.admin_name,
                admin_email: form.admin_email,
                admin_password: form.admin_password,
              }
            : {}),
        });
        toast.success('Empresa criada com sucesso');
      }
      setModalOpen(false);
      if (
        editing &&
        editing.id === activeCompanyID &&
        (editing.slug !== form.slug || (editing.is_active && !form.is_active))
      ) {
        // O header X-Company-ID usa o slug; mantê-lo atualizado evita perder a sessão.
        try {
          await setActiveCompany(form.is_active ? form.slug : user?.company_id ?? editing.id);
        } catch {
          toast.warning('Empresa salva, mas não foi possível manter o contexto ativo.');
        }
      }
      await load();
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fields);
        if (!Object.keys(error.fields).length) setFormError(error.message);
      } else {
        setFormError('Não foi possível salvar a empresa.');
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page">
      <PageHeader
        eyebrow="Super-administração"
        title="Empresas"
        description="Cadastre e mantenha as empresas (tenants) que usam o Documentation Center."
        actions={<Button renderIcon={Add} onClick={openCreate}>Nova empresa</Button>}
      />

      <section className="toolbar">
        <div className="toolbar__search">
          <Search
            size="lg"
            labelText="Buscar empresas"
            placeholder="Buscar por nome ou identificador"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            onClear={() => setSearch('')}
          />
        </div>
      </section>

      {loading ? (
        <Loading description="Carregando empresas…" />
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={Building}
          title={search.trim() ? 'Nenhuma empresa encontrada' : 'Nenhuma empresa cadastrada'}
          description={search.trim() ? 'Ajuste a busca ou limpe o filtro.' : 'Crie a primeira empresa para começar.'}
          action={<Button renderIcon={Add} onClick={openCreate}>Nova empresa</Button>}
        />
      ) : (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table" aria-label="Lista de empresas">
              <thead>
                <tr>
                  <th>Nome</th>
                  <th>Identificador</th>
                  <th>Status</th>
                  <th className="hide-md">Criada em</th>
                  <th className="cell-actions">Ações</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((company) => (
                  <tr key={company.id}>
                    <td>
                      <div className="user-cell">
                        <span className="avatar"><Building size={16} /></span>
                        <span className="cell-primary">{company.name}</span>
                      </div>
                    </td>
                    <td><code>{company.slug}</code></td>
                    <td>
                      <div className="tag-row">
                        <ActiveTag active={company.is_active} />
                      </div>
                    </td>
                    <td className="hide-md">{relativeTime(company.created_at)}</td>
                    <td className="cell-actions">
                      <Button
                        kind="ghost"
                        size="sm"
                        hasIconOnly
                        renderIcon={Edit}
                        iconDescription={`Editar ${company.name}`}
                        onClick={() => openEdit(company)}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="table-footer">
            <span className="table-footer__order">{filtered.length} empresa(s)</span>
          </div>
        </div>
      )}

      <Modal
        open={modalOpen}
        modalHeading={editing ? 'Editar empresa' : 'Nova empresa'}
        primaryButtonText={saving ? 'Salvando…' : editing ? 'Salvar alterações' : 'Criar empresa'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving}
        onRequestSubmit={() => void submitForm()}
        onRequestClose={() => setModalOpen(false)}
        onSecondarySubmit={() => setModalOpen(false)}
        size="sm"
      >
        <div className="modal-form">
          {formError && (
            <InlineNotification kind="error" title="Não foi possível salvar" subtitle={formError} lowContrast hideCloseButton />
          )}
          <TextInput
            id="company-name"
            labelText="Nome"
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
            invalid={Boolean(fieldErrors.name)}
            invalidText={fieldErrors.name}
            required
          />
          <TextInput
            id="company-slug"
            labelText="Identificador (slug)"
            helperText="Letras minúsculas, números e hífens. Usado no campo Empresa do login."
            value={form.slug}
            onChange={(event) => setForm({ ...form, slug: event.target.value })}
            invalid={Boolean(fieldErrors.slug)}
            invalidText={fieldErrors.slug}
            required
          />
          <Checkbox
            id="company-active"
            labelText="Empresa ativa"
            checked={form.is_active}
            onChange={(_event, { checked }) => setForm({ ...form, is_active: checked })}
          />
          {!editing && (
            <div className="company-admin-fields">
              <p className="company-admin-fields__title">Administrador da empresa (opcional)</p>
              <TextInput
                id="company-admin-name"
                labelText="Nome do administrador"
                value={form.admin_name}
                onChange={(event) => setForm({ ...form, admin_name: event.target.value })}
                invalid={Boolean(fieldErrors.admin_name)}
                invalidText={fieldErrors.admin_name}
              />
              <TextInput
                id="company-admin-email"
                type="email"
                labelText="E-mail do administrador"
                value={form.admin_email}
                onChange={(event) => setForm({ ...form, admin_email: event.target.value })}
                invalid={Boolean(fieldErrors.admin_email)}
                invalidText={fieldErrors.admin_email}
              />
              <PasswordInput
                id="company-admin-password"
                labelText="Senha do administrador"
                helperText="Mínimo de 8 caracteres. Preencha junto com o e-mail."
                value={form.admin_password}
                onChange={(event) => setForm({ ...form, admin_password: event.target.value })}
                invalid={Boolean(fieldErrors.admin_password)}
                invalidText={fieldErrors.admin_password}
              />
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
}
