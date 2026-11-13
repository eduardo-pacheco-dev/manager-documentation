import { useCallback, useEffect, useState } from 'react';
import { Add, Edit, Power, TrashCan, UserMultiple, View } from '@carbon/icons-react';
import {
  Button,
  InlineLoading,
  Modal,
  Search,
  Select,
  SelectItem,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Tag,
  TextArea,
  TextInput,
  Tile,
} from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState } from '../../components/Feedback';
import { useToast } from '../../contexts/ToastContext';
import { api } from '../../lib/api';
import { errorMessage } from '../../lib/format';
import type { PageMeta } from '../../lib/types';

interface Collaborator {
  id: number;
  name: string;
  social_name: string | null;
  email: string | null;
  personal_email: string | null;
  phone: string | null;
  mobile: string | null;
  role: string | null;
  department: string | null;
  manager_name: string | null;
  document: string | null;
  cpf: string | null;
  rg: string | null;
  rg_issuer: string | null;
  ctps: string | null;
  ctps_series: string | null;
  pis: string | null;
  voter_registration: string | null;
  military_registration: string | null;
  cnh: string | null;
  cnh_category: string | null;
  cnh_expiry: string | null;
  birth_date: string | null;
  gender: string | null;
  marital_status: string | null;
  nationality: string | null;
  birthplace: string | null;
  mother_name: string | null;
  father_name: string | null;
  zip_code: string | null;
  street: string | null;
  address_number: string | null;
  complement: string | null;
  neighborhood: string | null;
  city: string | null;
  state: string | null;
  country: string | null;
  emergency_contact_name: string | null;
  emergency_contact_phone: string | null;
  contract_type: string | null;
  work_regime: string | null;
  work_location: string | null;
  hire_date: string | null;
  termination_date: string | null;
  weekly_hours: number;
  salary: number;
  bank_name: string | null;
  bank_agency: string | null;
  bank_account: string | null;
  bank_account_type: string | null;
  pix_key: string | null;
  cost_hour: number;
  max_units: number;
  color: string;
  is_active: boolean;
  notes: string | null;
  project_count: number;
  created_at: string;
  updated_at: string;
}

interface Stats {
  total: number;
  active: number;
  inactive: number;
  allocated: number;
}

interface ListResponse {
  data: Collaborator[];
  stats: Stats;
  meta: PageMeta;
}

const COLORS: Record<string, string> = {
  blue: 'Azul', green: 'Verde', purple: 'Roxo', orange: 'Laranja', teal: 'Turquesa',
  gray: 'Cinza', magenta: 'Magenta', cyan: 'Ciano', red: 'Vermelho',
};

const CONTRACT_TYPES: Record<string, string> = {
  clt: 'CLT', pj: 'PJ', estagio: 'Estágio', temporario: 'Temporário', aprendiz: 'Aprendiz',
};

const WORK_REGIMES: Record<string, string> = {
  presencial: 'Presencial', hibrido: 'Híbrido', remoto: 'Remoto',
};

const GENDERS: Record<string, string> = {
  feminino: 'Feminino', masculino: 'Masculino', nao_binario: 'Não binário',
  prefiro_nao_informar: 'Prefiro não informar', outro: 'Outro',
};

const MARITAL_STATUSES: Record<string, string> = {
  solteiro: 'Solteiro(a)', casado: 'Casado(a)', divorciado: 'Divorciado(a)',
  viuvo: 'Viúvo(a)', uniao_estavel: 'União estável',
};

const BANK_ACCOUNT_TYPES: Record<string, string> = {
  corrente: 'Conta corrente', poupanca: 'Poupança', salario: 'Conta salário',
};

const FORM_FIELDS = [
  'name', 'social_name', 'cpf', 'birth_date', 'gender', 'marital_status', 'nationality', 'birthplace',
  'mother_name', 'father_name', 'rg', 'rg_issuer', 'ctps', 'ctps_series', 'pis', 'voter_registration',
  'military_registration', 'document', 'cnh', 'cnh_category', 'cnh_expiry', 'email', 'personal_email',
  'phone', 'mobile', 'zip_code', 'street', 'address_number', 'complement', 'neighborhood', 'city', 'state',
  'country', 'emergency_contact_name', 'emergency_contact_phone', 'role', 'department', 'manager_name',
  'contract_type', 'work_regime', 'work_location', 'hire_date', 'termination_date', 'weekly_hours',
  'salary', 'cost_hour', 'max_units', 'color', 'is_active', 'bank_name', 'bank_agency', 'bank_account',
  'bank_account_type', 'pix_key', 'notes',
] as const;

type FormField = (typeof FORM_FIELDS)[number];
type FormState = Record<FormField, string>;
const NUMERIC_FIELDS = new Set<FormField>(['weekly_hours', 'salary', 'cost_hour', 'max_units']);

function blankForm(): FormState {
  const form = Object.fromEntries(FORM_FIELDS.map((field) => [field, ''])) as FormState;
  return {
    ...form,
    color: 'blue',
    is_active: 'true',
    weekly_hours: '44',
    salary: '0',
    cost_hour: '0',
    max_units: '100',
    contract_type: 'clt',
    work_regime: 'presencial',
    country: 'Brasil',
  };
}

function formFrom(item: Collaborator): FormState {
  const form = blankForm();
  for (const field of FORM_FIELDS) {
    const value = (item as unknown as Record<string, unknown>)[field];
    if (value !== null && value !== undefined) form[field] = String(value);
  }
  form.is_active = item.is_active ? 'true' : 'false';
  return form;
}

function fmtDate(value: string | null | undefined) {
  if (!value) return '—';
  const [year, month, day] = value.slice(0, 10).split('-');
  return year && month && day ? `${day}/${month}/${year}` : value;
}

function money(value: number) {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(value || 0);
}

function label(map: Record<string, string>, value: string | null | undefined) {
  if (!value) return '—';
  return map[value] ?? value;
}

function formatCPF(value: string | null | undefined) {
  const digits = (value ?? '').replace(/\D/g, '');
  if (digits.length !== 11) return value || '—';
  return `${digits.slice(0, 3)}.${digits.slice(3, 6)}.${digits.slice(6, 9)}-${digits.slice(9)}`;
}

export function CollaboratorsPage() {
  const toast = useToast();
  const [items, setItems] = useState<Collaborator[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [meta, setMeta] = useState<PageMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [formTab, setFormTab] = useState(0);
  const [editing, setEditing] = useState<Collaborator | null>(null);
  const [detail, setDetail] = useState<Collaborator | null>(null);
  const [form, setForm] = useState<FormState>(blankForm);

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({ page: String(page), per_page: '15' });
    if (search.trim()) params.set('search', search.trim());
    if (status) params.set('status', status);
    try {
      const response = await api.get<ListResponse>(`/collaborators?${params.toString()}`);
      setItems(response.data);
      setStats(response.stats);
      setMeta(response.meta);
    } catch (error) {
      toast.error('Não foi possível carregar os colaboradores', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [page, search, status, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  function openCreate() {
    setEditing(null);
    setForm(blankForm());
    setFormTab(0);
    setModalOpen(true);
  }

  function openEdit(item: Collaborator) {
    setEditing(item);
    setForm(formFrom(item));
    setFormTab(0);
    setModalOpen(true);
  }

  async function save() {
    if (!form.name.trim()) {
      toast.error('Informe o nome do colaborador');
      setFormTab(0);
      return;
    }
    setSaving(true);
    try {
      const payload: Record<string, unknown> = { is_active: form.is_active === 'true' };
      for (const field of FORM_FIELDS) {
        if (field === 'is_active') continue;
        payload[field] = NUMERIC_FIELDS.has(field) ? Number(form[field]) || 0 : form[field];
      }
      if (editing) {
        await api.put(`/collaborators/${editing.id}`, payload);
        toast.success('Colaborador atualizado');
      } else {
        await api.post('/collaborators', payload);
        toast.success('Colaborador criado');
      }
      setModalOpen(false);
      await load();
    } catch (error) {
      toast.error('Não foi possível salvar o colaborador', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function toggle(item: Collaborator) {
    try {
      const response = await api.patch<{ message: string }>(`/collaborators/${item.id}/toggle`);
      toast.success(response.message);
      await load();
    } catch (error) {
      toast.error('Não foi possível alterar o status', errorMessage(error));
    }
  }

  async function remove(item: Collaborator) {
    if (!window.confirm(`Excluir o colaborador "${item.name}"? Os recursos já vinculados aos projetos serão preservados.`)) return;
    try {
      await api.delete(`/collaborators/${item.id}`);
      toast.success('Colaborador excluído');
      if (items.length === 1 && page > 1) {
        setPage((current) => current - 1);
      } else {
        await load();
      }
    } catch (error) {
      toast.error('Não foi possível excluir o colaborador', errorMessage(error));
    }
  }

  function update(field: FormField, value: string) {
    setForm((current) => ({ ...current, [field]: value }));
  }

  return (
    <div className="page">
      <PageHeader
        eyebrow="Pessoas"
        title="Colaboradores"
        description="Ficha completa de RH: dados pessoais, documentos, endereço, contrato, remuneração e dados bancários. Use os colaboradores como recursos nos projetos."
        actions={<Button renderIcon={Add} onClick={openCreate}>Novo colaborador</Button>}
      />

      {stats && (
        <div className="stats-grid">
          <Tile className="stat-card stat-card--blue">
            <span className="stat-card__label">Total</span>
            <strong className="stat-card__value">{stats.total}</strong>
          </Tile>
          <Tile className="stat-card stat-card--green">
            <span className="stat-card__label">Ativos</span>
            <strong className="stat-card__value">{stats.active}</strong>
          </Tile>
          <Tile className="stat-card stat-card--purple">
            <span className="stat-card__label">Inativos</span>
            <strong className="stat-card__value">{stats.inactive}</strong>
          </Tile>
          <Tile className="stat-card stat-card--red">
            <span className="stat-card__label">Alocados em projetos</span>
            <strong className="stat-card__value">{stats.allocated}</strong>
          </Tile>
        </div>
      )}

      <Tile className="module-main__header">
        <div className="module-toolbar">
          <Search
            size="lg"
            labelText="Buscar colaboradores"
            placeholder="Buscar por nome, CPF, e-mail, cargo, cidade…"
            value={search}
            onChange={(event) => { setSearch(event.target.value); setPage(1); }}
            onClear={() => { setSearch(''); setPage(1); }}
          />
          <Select
            id="collaborator-status"
            labelText="Status"
            size="sm"
            value={status}
            onChange={(event) => { setStatus(event.target.value); setPage(1); }}
          >
            <SelectItem value="" text="Todos" />
            <SelectItem value="active" text="Ativos" />
            <SelectItem value="inactive" text="Inativos" />
          </Select>
        </div>
      </Tile>

      {loading ? (
        <InlineLoading description="Carregando colaboradores…" />
      ) : items.length === 0 ? (
        <EmptyState
          icon={UserMultiple}
          title="Nenhum colaborador encontrado"
          description="Cadastre pessoas para reutilizá-las como recursos nos projetos."
          action={<Button renderIcon={Add} onClick={openCreate}>Novo colaborador</Button>}
        />
      ) : (
        <div className="table-wrapper">
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Colaborador</th>
                  <th className="hide-sm">Contato</th>
                  <th className="hide-sm">Departamento</th>
                  <th>Contrato</th>
                  <th className="hide-sm">Admissão</th>
                  <th>Status</th>
                  <th>Ações</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id}>
                    <td>
                      <span className="user-cell">
                        <span className="color-dot" style={{ background: item.color }} />
                        <span>
                          <strong>{item.social_name || item.name}</strong>
                          {item.role && <span className="cell-secondary">{item.role}</span>}
                        </span>
                      </span>
                    </td>
                    <td className="hide-sm">
                      {item.email ?? '—'}
                      {item.mobile && <span className="cell-secondary">{item.mobile}</span>}
                    </td>
                    <td className="hide-sm">{item.department ?? '—'}</td>
                    <td>
                      {label(CONTRACT_TYPES, item.contract_type)}
                      {item.work_regime && <span className="cell-secondary">{label(WORK_REGIMES, item.work_regime)}</span>}
                    </td>
                    <td className="hide-sm">{fmtDate(item.hire_date)}</td>
                    <td>
                      <Tag type={item.is_active ? 'green' : 'gray'} size="sm">
                        {item.is_active ? 'Ativo' : 'Inativo'}
                      </Tag>
                    </td>
                    <td className="cell-actions">
                      <Button kind="ghost" size="sm" hasIconOnly renderIcon={View} iconDescription="Ver ficha" onClick={() => setDetail(item)} />
                      <Button kind="ghost" size="sm" hasIconOnly renderIcon={Edit} iconDescription="Editar" onClick={() => openEdit(item)} />
                      <Button
                        kind="ghost"
                        size="sm"
                        hasIconOnly
                        renderIcon={Power}
                        iconDescription={item.is_active ? 'Desativar' : 'Ativar'}
                        onClick={() => void toggle(item)}
                      />
                      <Button kind="danger--ghost" size="sm" hasIconOnly renderIcon={TrashCan} iconDescription="Excluir" onClick={() => void remove(item)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="table-footer">
            <span className="table-footer__order">{meta ? `${meta.total} colaborador(es)` : ''}</span>
          </div>
        </div>
      )}

      {meta && meta.last_page > 1 && (
        <div className="pager">
          <Button kind="ghost" size="sm" disabled={page <= 1} onClick={() => setPage((current) => Math.max(1, current - 1))}>Anterior</Button>
          <span>Página {meta.current_page} de {meta.last_page}</span>
          <Button kind="ghost" size="sm" disabled={page >= meta.last_page} onClick={() => setPage((current) => current + 1)}>Próxima</Button>
        </div>
      )}

      <Modal
        open={modalOpen}
        modalHeading={editing ? `Editar ${editing.name}` : 'Novo colaborador'}
        primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
        secondaryButtonText="Cancelar"
        primaryButtonDisabled={saving}
        onRequestSubmit={() => void save()}
        onRequestClose={() => setModalOpen(false)}
        size="lg"
      >
        <Tabs selectedIndex={formTab} onChange={({ selectedIndex }) => setFormTab(selectedIndex)}>
          <TabList aria-label="Seções da ficha do colaborador" contained>
            <Tab>Dados pessoais</Tab>
            <Tab>Documentos</Tab>
            <Tab>Contato e endereço</Tab>
            <Tab>Contrato</Tab>
            <Tab>Dados bancários</Tab>
          </TabList>
          <TabPanels>
            <TabPanel>
              <div className="modal-form">
                <div className="form-row">
                  <TextInput id="collaborator-name" labelText="Nome completo" value={form.name} onChange={(event) => update('name', event.target.value)} required />
                  <TextInput id="collaborator-social-name" labelText="Nome social" value={form.social_name} onChange={(event) => update('social_name', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-cpf" labelText="CPF" value={form.cpf} onChange={(event) => update('cpf', event.target.value)} placeholder="000.000.000-00" />
                  <TextInput id="collaborator-birth" type="date" labelText="Data de nascimento" value={form.birth_date} onChange={(event) => update('birth_date', event.target.value)} />
                </div>
                <div className="form-row">
                  <Select id="collaborator-gender" labelText="Gênero" value={form.gender} onChange={(event) => update('gender', event.target.value)}>
                    <SelectItem value="" text="Não informado" />
                    {Object.entries(GENDERS).map(([value, text]) => <SelectItem key={value} value={value} text={text} />)}
                  </Select>
                  <Select id="collaborator-marital" labelText="Estado civil" value={form.marital_status} onChange={(event) => update('marital_status', event.target.value)}>
                    <SelectItem value="" text="Não informado" />
                    {Object.entries(MARITAL_STATUSES).map(([value, text]) => <SelectItem key={value} value={value} text={text} />)}
                  </Select>
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-nationality" labelText="Nacionalidade" value={form.nationality} onChange={(event) => update('nationality', event.target.value)} />
                  <TextInput id="collaborator-birthplace" labelText="Naturalidade" value={form.birthplace} onChange={(event) => update('birthplace', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-mother" labelText="Nome da mãe" value={form.mother_name} onChange={(event) => update('mother_name', event.target.value)} />
                  <TextInput id="collaborator-father" labelText="Nome do pai" value={form.father_name} onChange={(event) => update('father_name', event.target.value)} />
                </div>
              </div>
            </TabPanel>

            <TabPanel>
              <div className="modal-form">
                <div className="form-row">
                  <TextInput id="collaborator-rg" labelText="RG" value={form.rg} onChange={(event) => update('rg', event.target.value)} />
                  <TextInput id="collaborator-rg-issuer" labelText="Órgão emissor" value={form.rg_issuer} onChange={(event) => update('rg_issuer', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-ctps" labelText="CTPS" value={form.ctps} onChange={(event) => update('ctps', event.target.value)} />
                  <TextInput id="collaborator-ctps-series" labelText="Série da CTPS" value={form.ctps_series} onChange={(event) => update('ctps_series', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-pis" labelText="PIS/PASEP" value={form.pis} onChange={(event) => update('pis', event.target.value)} />
                  <TextInput id="collaborator-voter" labelText="Título de eleitor" value={form.voter_registration} onChange={(event) => update('voter_registration', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-military" labelText="Certificado de reservista" value={form.military_registration} onChange={(event) => update('military_registration', event.target.value)} />
                  <TextInput id="collaborator-document" labelText="Registro profissional (CREA/CRM/OAB)" value={form.document} onChange={(event) => update('document', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-cnh" labelText="CNH" value={form.cnh} onChange={(event) => update('cnh', event.target.value)} />
                  <TextInput id="collaborator-cnh-category" labelText="Categoria da CNH" value={form.cnh_category} onChange={(event) => update('cnh_category', event.target.value)} />
                </div>
                <TextInput id="collaborator-cnh-expiry" type="date" labelText="Validade da CNH" value={form.cnh_expiry} onChange={(event) => update('cnh_expiry', event.target.value)} />
              </div>
            </TabPanel>

            <TabPanel>
              <div className="modal-form">
                <div className="form-row">
                  <TextInput id="collaborator-email" type="email" labelText="E-mail corporativo" value={form.email} onChange={(event) => update('email', event.target.value)} />
                  <TextInput id="collaborator-personal-email" type="email" labelText="E-mail pessoal" value={form.personal_email} onChange={(event) => update('personal_email', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-phone" labelText="Telefone" value={form.phone} onChange={(event) => update('phone', event.target.value)} />
                  <TextInput id="collaborator-mobile" labelText="Celular" value={form.mobile} onChange={(event) => update('mobile', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-zip" labelText="CEP" value={form.zip_code} onChange={(event) => update('zip_code', event.target.value)} />
                  <TextInput id="collaborator-street" labelText="Logradouro" value={form.street} onChange={(event) => update('street', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-number" labelText="Número" value={form.address_number} onChange={(event) => update('address_number', event.target.value)} />
                  <TextInput id="collaborator-complement" labelText="Complemento" value={form.complement} onChange={(event) => update('complement', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-neighborhood" labelText="Bairro" value={form.neighborhood} onChange={(event) => update('neighborhood', event.target.value)} />
                  <TextInput id="collaborator-city" labelText="Cidade" value={form.city} onChange={(event) => update('city', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-state" labelText="UF" value={form.state} onChange={(event) => update('state', event.target.value)} />
                  <TextInput id="collaborator-country" labelText="País" value={form.country} onChange={(event) => update('country', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-emergency" labelText="Contato de emergência" value={form.emergency_contact_name} onChange={(event) => update('emergency_contact_name', event.target.value)} />
                  <TextInput id="collaborator-emergency-phone" labelText="Telefone de emergência" value={form.emergency_contact_phone} onChange={(event) => update('emergency_contact_phone', event.target.value)} />
                </div>
              </div>
            </TabPanel>

            <TabPanel>
              <div className="modal-form">
                <div className="form-row">
                  <TextInput id="collaborator-role" labelText="Cargo" value={form.role} onChange={(event) => update('role', event.target.value)} />
                  <TextInput id="collaborator-department" labelText="Departamento" value={form.department} onChange={(event) => update('department', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-manager" labelText="Gestor imediato" value={form.manager_name} onChange={(event) => update('manager_name', event.target.value)} />
                  <Select id="collaborator-contract" labelText="Tipo de contrato" value={form.contract_type} onChange={(event) => update('contract_type', event.target.value)}>
                    <SelectItem value="" text="Não informado" />
                    {Object.entries(CONTRACT_TYPES).map(([value, text]) => <SelectItem key={value} value={value} text={text} />)}
                  </Select>
                </div>
                <div className="form-row">
                  <Select id="collaborator-regime" labelText="Regime de trabalho" value={form.work_regime} onChange={(event) => update('work_regime', event.target.value)}>
                    <SelectItem value="" text="Não informado" />
                    {Object.entries(WORK_REGIMES).map(([value, text]) => <SelectItem key={value} value={value} text={text} />)}
                  </Select>
                  <TextInput id="collaborator-work-location" labelText="Local de trabalho" value={form.work_location} onChange={(event) => update('work_location', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-hire" type="date" labelText="Data de admissão" value={form.hire_date} onChange={(event) => update('hire_date', event.target.value)} />
                  <TextInput id="collaborator-termination" type="date" labelText="Data de desligamento" value={form.termination_date} onChange={(event) => update('termination_date', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-weekly" type="number" labelText="Jornada semanal (h)" value={form.weekly_hours} onChange={(event) => update('weekly_hours', event.target.value)} />
                  <TextInput id="collaborator-salary" type="number" labelText="Salário mensal" value={form.salary} onChange={(event) => update('salary', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-cost" type="number" labelText="Custo por hora (projetos)" value={form.cost_hour} onChange={(event) => update('cost_hour', event.target.value)} />
                  <TextInput id="collaborator-units" type="number" labelText="Capacidade (%)" value={form.max_units} onChange={(event) => update('max_units', event.target.value)} />
                </div>
                <div className="form-row">
                  <Select id="collaborator-color" labelText="Cor" value={form.color} onChange={(event) => update('color', event.target.value)}>
                    {Object.entries(COLORS).map(([value, text]) => <SelectItem key={value} value={value} text={text} />)}
                  </Select>
                  <Select id="collaborator-active" labelText="Status" value={form.is_active} onChange={(event) => update('is_active', event.target.value)}>
                    <SelectItem value="true" text="Ativo" />
                    <SelectItem value="false" text="Inativo" />
                  </Select>
                </div>
              </div>
            </TabPanel>

            <TabPanel>
              <div className="modal-form">
                <div className="form-row">
                  <TextInput id="collaborator-bank" labelText="Banco" value={form.bank_name} onChange={(event) => update('bank_name', event.target.value)} />
                  <TextInput id="collaborator-agency" labelText="Agência" value={form.bank_agency} onChange={(event) => update('bank_agency', event.target.value)} />
                </div>
                <div className="form-row">
                  <TextInput id="collaborator-account" labelText="Conta" value={form.bank_account} onChange={(event) => update('bank_account', event.target.value)} />
                  <Select id="collaborator-account-type" labelText="Tipo de conta" value={form.bank_account_type} onChange={(event) => update('bank_account_type', event.target.value)}>
                    <SelectItem value="" text="Não informado" />
                    {Object.entries(BANK_ACCOUNT_TYPES).map(([value, text]) => <SelectItem key={value} value={value} text={text} />)}
                  </Select>
                </div>
                <TextInput id="collaborator-pix" labelText="Chave Pix" value={form.pix_key} onChange={(event) => update('pix_key', event.target.value)} />
                <TextArea id="collaborator-notes" labelText="Observações" value={form.notes} onChange={(event) => update('notes', event.target.value)} rows={4} />
              </div>
            </TabPanel>
          </TabPanels>
        </Tabs>
      </Modal>

      <Modal
        open={detail !== null}
        modalHeading={detail ? `Ficha de ${detail.social_name || detail.name}` : ''}
        primaryButtonText="Fechar"
        onRequestSubmit={() => setDetail(null)}
        onRequestClose={() => setDetail(null)}
        size="lg"
      >
        {detail && (
          <div className="modal-form">
            <div className="tag-row">
              <Tag type={detail.is_active ? 'green' : 'gray'} size="sm">{detail.is_active ? 'Ativo' : 'Inativo'}</Tag>
              <Tag type="blue" size="sm">{detail.role ?? 'Sem cargo'}</Tag>
              <Tag type="gray" size="sm">{detail.department ?? 'Sem departamento'}</Tag>
              <Tag type="purple" size="sm">{detail.project_count} projeto(s)</Tag>
            </div>
            <DetailSection title="Dados pessoais" rows={[
              ['Nome completo', detail.name], ['Nome social', detail.social_name], ['CPF', formatCPF(detail.cpf)],
              ['Nascimento', fmtDate(detail.birth_date)], ['Gênero', label(GENDERS, detail.gender)],
              ['Estado civil', label(MARITAL_STATUSES, detail.marital_status)], ['Nacionalidade', detail.nationality],
              ['Naturalidade', detail.birthplace], ['Nome da mãe', detail.mother_name], ['Nome do pai', detail.father_name],
            ]} />
            <DetailSection title="Documentos" rows={[
              ['RG', detail.rg], ['Órgão emissor', detail.rg_issuer], ['CTPS', detail.ctps],
              ['Série CTPS', detail.ctps_series], ['PIS/PASEP', detail.pis], ['Título de eleitor', detail.voter_registration],
              ['Reservista', detail.military_registration], ['Registro profissional', detail.document],
              ['CNH', detail.cnh], ['Categoria CNH', detail.cnh_category], ['Validade CNH', fmtDate(detail.cnh_expiry)],
            ]} />
            <DetailSection title="Contato" rows={[
              ['E-mail corporativo', detail.email], ['E-mail pessoal', detail.personal_email],
              ['Telefone', detail.phone], ['Celular', detail.mobile],
              ['Contato de emergência', detail.emergency_contact_name], ['Telefone de emergência', detail.emergency_contact_phone],
            ]} />
            <DetailSection title="Endereço" rows={[
              ['CEP', detail.zip_code], ['Logradouro', detail.street], ['Número', detail.address_number],
              ['Complemento', detail.complement], ['Bairro', detail.neighborhood], ['Cidade', detail.city],
              ['UF', detail.state], ['País', detail.country],
            ]} />
            <DetailSection title="Contrato e remuneração" rows={[
              ['Cargo', detail.role], ['Departamento', detail.department], ['Gestor', detail.manager_name],
              ['Contrato', label(CONTRACT_TYPES, detail.contract_type)], ['Regime', label(WORK_REGIMES, detail.work_regime)],
              ['Local', detail.work_location], ['Admissão', fmtDate(detail.hire_date)], ['Desligamento', fmtDate(detail.termination_date)],
              ['Jornada semanal', `${detail.weekly_hours || 0}h`], ['Salário', money(detail.salary)],
              ['Custo/hora', money(detail.cost_hour)], ['Capacidade', `${detail.max_units}%`],
            ]} />
            <DetailSection title="Dados bancários" rows={[
              ['Banco', detail.bank_name], ['Agência', detail.bank_agency], ['Conta', detail.bank_account],
              ['Tipo de conta', label(BANK_ACCOUNT_TYPES, detail.bank_account_type)], ['Chave Pix', detail.pix_key],
            ]} />
            {detail.notes && (
              <>
                <h4>Observações</h4>
                <p>{detail.notes}</p>
              </>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}

function DetailSection({ title, rows }: { title: string; rows: [string, string | null | undefined][] }) {
  return (
    <>
      <h4>{title}</h4>
      <div className="detail-grid">
        {rows.map(([name, value]) => (
          <div key={name}>
            <span>{name}</span>
            <strong>{value || '—'}</strong>
          </div>
        ))}
      </div>
    </>
  );
}
