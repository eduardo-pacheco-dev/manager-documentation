import { useEffect, useState } from 'react';
import { ComboBox, Modal, ProgressIndicator, ProgressStep, Select, SelectItem, TextInput } from '@carbon/react';
import { useToast } from '../contexts/ToastContext';
import { ApiError, api } from '../lib/api';
import { errorMessage } from '../lib/format';
import type { IepmsUser } from '../lib/types';

const EMPTY = {
  resource_id: '',
  name: '',
  gender: '',
  nationality: '',
  certificate_type: '',
  id_number: '',
  birth_year: '',
  tel: '',
  personal_mailbox: '',
  outsourcing_account: '',
  last_updated_by: '',
  status: 'ativo',
  password: '',
};

export type IepmsUserFormValues = typeof EMPTY;

const STEPS = [
  {
    key: 'identificacao',
    label: 'Identificação',
    description: 'Dados pessoais, documento e contato.',
  },
  {
    key: 'conta',
    label: 'Conta e segurança',
    description: 'Conta de terceirização, status e senha de acesso.',
  },
];

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const RESOURCE_ID_PATTERN = /^W\d{6,10}$/;
const DOCUMENT_PATTERN = /^[A-Z0-9./-]{3,30}$/;
const ACCOUNT_PATTERN = /^[A-Za-z0-9._-]{1,30}$/;

const GENDER_OPTIONS = ['MALE', 'FEMALE', 'OTHER'];

const CERTIFICATE_TYPE_OPTIONS = [
  'ID Card',
  'Passport',
  'Work Permit',
  'Driver License',
  'Residence Permit',
  'Other',
];

const NATIONALITY_OPTIONS = [
  'Brazil',
  'Argentina',
  'Bolivia',
  'Chile',
  'Colombia',
  'Ecuador',
  'Mexico',
  'Paraguay',
  'Peru',
  'Uruguay',
  'Venezuela',
  'Angola',
  'China',
  'South Korea',
  'Spain',
  'United States',
  'France',
  'India',
  'Italy',
  'Japan',
  'Mozambique',
  'Portugal',
  'United Kingdom',
  'Germany',
  'Canada',
  'Australia',
  'South Africa',
  'Philippines',
  'Other',
];

function digitsOnly(value: string, max: number): string {
  return value.replace(/\D/g, '').slice(0, max);
}

function maskResourceID(value: string): string {
  const cleaned = value.toUpperCase().replace(/[^A-Z0-9]/g, '');
  if (!cleaned) return '';
  const prefixed = cleaned.startsWith('W') ? cleaned : `W${cleaned}`;
  return prefixed.slice(0, 11);
}

function maskPhone(value: string): string {
  return value
    .split(',')
    .map((part) => {
      if (!part.includes('|')) {
        return part.replace(/\D/g, '').slice(0, 4);
      }
      const [rawCountry, ...rest] = part.split('|');
      const country = rawCountry.replace(/\D/g, '').slice(0, 4);
      const local = rest.join('').replace(/\D/g, '').slice(0, 11);
      let formatted = local;
      if (local.length > 10) {
        formatted = `(${local.slice(0, 2)}) ${local.slice(2, 7)}-${local.slice(7)}`;
      } else if (local.length > 6) {
        formatted = `(${local.slice(0, 2)}) ${local.slice(2, 6)}-${local.slice(6)}`;
      } else if (local.length > 2) {
        formatted = `(${local.slice(0, 2)}) ${local.slice(2)}`;
      }
      return `${country}|${formatted}`;
    })
    .join(', ');
}

const MASKS: Partial<Record<keyof IepmsUserFormValues, (value: string) => string>> = {
  resource_id: maskResourceID,
  name: (value) => value.replace(/[^\p{L}\s'.-]/gu, '').slice(0, 255),
  gender: (value) => value.toUpperCase().replace(/[^A-Z\s-]/g, '').slice(0, 32),
  nationality: (value) => value.replace(/[^\p{L}\s'.-]/gu, '').slice(0, 128),
  certificate_type: (value) => value.replace(/[^\p{L}\s'./-]/gu, '').slice(0, 128),
  id_number: (value) => value.toUpperCase().replace(/[^A-Z0-9./-]/g, '').slice(0, 30),
  birth_year: (value) => digitsOnly(value, 4),
  tel: maskPhone,
  personal_mailbox: (value) => value.slice(0, 255),
  outsourcing_account: (value) => value.replace(/[^A-Za-z0-9._-]/g, '').slice(0, 30),
  last_updated_by: (value) => value.slice(0, 255),
  password: (value) => value.slice(0, 128),
};

const STEP_FIELDS: (keyof IepmsUserFormValues)[][] = [
  ['resource_id', 'name', 'gender', 'nationality', 'birth_year', 'certificate_type', 'id_number', 'tel', 'personal_mailbox'],
  ['outsourcing_account', 'last_updated_by', 'status', 'password'],
];

function validateValues(values: IepmsUserFormValues): Record<string, string> {
  const errors: Record<string, string> = {};
  const resourceID = values.resource_id.trim();
  if (!resourceID) {
    errors.resource_id = 'Informe o ID do recurso.';
  } else if (!RESOURCE_ID_PATTERN.test(resourceID)) {
    errors.resource_id = 'Use o formato W seguido de 6 a 10 números (ex.: W0000326975).';
  }
  const name = values.name.trim();
  if (!name) {
    errors.name = 'Informe o nome.';
  } else if (name.length < 2) {
    errors.name = 'O nome precisa ter pelo menos 2 caracteres.';
  }
  const birthYear = values.birth_year.trim();
  if (birthYear) {
    const year = Number(birthYear);
    if (!/^\d{4}$/.test(birthYear) || year < 1900 || year > new Date().getFullYear()) {
      errors.birth_year = 'Informe um ano entre 1900 e o ano atual.';
    }
  }
  const email = values.personal_mailbox.trim();
  if (email && !EMAIL_PATTERN.test(email)) {
    errors.personal_mailbox = 'Informe um e-mail válido.';
  }
  const tel = values.tel.trim();
  if (tel) {
    const digits = tel.replace(/\D/g, '');
    if (digits.length < 8 || digits.length > 20) {
      errors.tel = 'Informe um telefone com 8 a 20 dígitos.';
    }
  }
  const document = values.id_number.trim();
  if (document && !DOCUMENT_PATTERN.test(document)) {
    errors.id_number = 'Use de 3 a 30 caracteres (letras, números, ponto, barra ou hífen).';
  }
  const account = values.outsourcing_account.trim();
  if (account && !ACCOUNT_PATTERN.test(account)) {
    errors.outsourcing_account = 'Use até 30 caracteres (letras, números, ponto, hífen ou sublinhado).';
  }
  if (values.last_updated_by.length > 255) {
    errors.last_updated_by = 'Use no máximo 255 caracteres.';
  }
  if (values.password.length > 128) {
    errors.password = 'A senha deve ter no máximo 128 caracteres.';
  }
  return errors;
}

function formFromUser(user: IepmsUser): IepmsUserFormValues {
  return Object.fromEntries(
    Object.keys(EMPTY).map((key) => [key, String(user[key as keyof IepmsUser] ?? '')]),
  ) as IepmsUserFormValues;
}

interface Props {
  open: boolean;
  user: IepmsUser | null;
  onClose: () => void;
  onSaved: () => void;
}

export function IepmsUserFormModal({ open, user, onClose, onSaved }: Props) {
  const toast = useToast();
  const [form, setForm] = useState<IepmsUserFormValues>({ ...EMPTY });
  const [step, setStep] = useState(0);
  const [saving, setSaving] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!open) return;
    setForm(user ? formFromUser(user) : { ...EMPTY });
    setStep(0);
    setFieldErrors({});
  }, [open, user]);

  const lastStep = STEPS.length - 1;
  const requiredFilled = Boolean(form.resource_id.trim() && form.name.trim());

  function updateForm(patch: Partial<IepmsUserFormValues>) {
    setForm((current) => ({ ...current, ...patch }));
  }

  function validateStep(index: number): boolean {
    const errors = validateValues(form);
    const scoped: Record<string, string> = {};
    for (const key of STEP_FIELDS[index]) {
      if (errors[key]) scoped[key] = errors[key];
    }
    setFieldErrors(scoped);
    return Object.keys(scoped).length === 0;
  }

  function validateAll(): boolean {
    const errors = validateValues(form);
    setFieldErrors(errors);
    if (Object.keys(errors).length === 0) return true;
    const firstStepHasErrors = STEP_FIELDS[0].some((key) => Boolean(errors[key]));
    setStep(firstStepHasErrors ? 0 : 1);
    return false;
  }

  function goToStep(index: number) {
    if (index === step) return;
    if (index > step && !validateStep(step)) return;
    setStep(Math.min(Math.max(index, 0), lastStep));
  }

  function advanceStep() {
    if (step < lastStep) {
      if (!validateStep(step)) return;
      setStep(step + 1);
      return;
    }
    void save();
  }

  async function save() {
    if (!validateAll()) return;

    setSaving(true);
    try {
      if (user) {
        const response = await api.put<{ message: string }>(`/iepms-users/${user.id}`, form);
        toast.success('Usuário do iEPMS atualizado', response.message);
      } else {
        const response = await api.post<{ message: string }>('/iepms-users', form);
        toast.success('Usuário do iEPMS cadastrado', response.message);
      }
      onSaved();
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fields);
        if (error.fields.resource_id || error.fields.name) setStep(0);
      }
      toast.error('Não foi possível salvar o usuário do iEPMS', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  function clearFieldError(key: keyof IepmsUserFormValues) {
    setFieldErrors((current) => {
      if (!current[key]) return current;
      const next = { ...current };
      delete next[key];
      return next;
    });
  }

  const field = (
    key: keyof IepmsUserFormValues,
    label: string,
    options: { type?: string; placeholder?: string; helperText?: string } = {},
  ) => (
    <TextInput
      id={`iepms-${key}`}
      labelText={label}
      value={form[key]}
      type={options.type}
      placeholder={options.placeholder}
      helperText={options.helperText}
      onChange={(event) => {
        const mask = MASKS[key];
        const value = mask ? mask(event.target.value) : event.target.value;
        updateForm({ [key]: value });
        clearFieldError(key);
      }}
      invalid={Boolean(fieldErrors[key])}
      invalidText={fieldErrors[key]}
    />
  );

  const combo = (
    key: keyof IepmsUserFormValues,
    label: string,
    items: string[],
    helperText?: string,
  ) => {
    const value = form[key];
    const options = value && !items.includes(value) ? [value, ...items] : items;
    return (
      <ComboBox
        id={`iepms-${key}`}
        titleText={label}
        placeholder="Selecione ou digite"
        items={options}
        itemToString={(item) => item ?? ''}
        selectedItem={value || null}
        allowCustomValue
        helperText={helperText}
        onInputChange={(inputValue) => {
          updateForm({ [key]: inputValue ?? '' });
          clearFieldError(key);
        }}
        onChange={({ selectedItem, inputValue }) => {
          updateForm({ [key]: selectedItem ?? inputValue ?? '' });
          clearFieldError(key);
        }}
        invalid={Boolean(fieldErrors[key])}
        invalidText={fieldErrors[key]}
      />
    );
  };

  return (
    <Modal
      open={open}
      modalHeading={user ? 'Editar usuário do iEPMS' : 'Novo usuário do iEPMS'}
      primaryButtonText={saving ? 'Salvando…' : step === lastStep ? 'Salvar' : 'Próximo'}
      secondaryButtonText={step === 0 ? 'Cancelar' : 'Voltar'}
      primaryButtonDisabled={saving}
      onRequestSubmit={advanceStep}
      onSecondarySubmit={() => (step === 0 ? onClose() : setStep(step - 1))}
      onRequestClose={onClose}
      size="lg"
    >
      <div className="modal-form">
        <ProgressIndicator className="modal-steps" currentIndex={step} onChange={goToStep} spaceEqually>
          {STEPS.map((item, index) => (
            <ProgressStep key={item.key} label={item.label} disabled={index > 0 && !requiredFilled} />
          ))}
        </ProgressIndicator>

        <div className="wizard-step__header">
          <h4>{STEPS[step].label}</h4>
          <p className="muted small">{STEPS[step].description}</p>
        </div>

        {step === 0 && (
          <div className="form-grid">
            {field('resource_id', 'ID do recurso *', { placeholder: 'W0000000000', helperText: 'Formato: W + 6 a 10 números.' })}
            {field('name', 'Nome *')}
            {combo('gender', 'Gênero', GENDER_OPTIONS, 'Selecione ou digite outro valor.')}
            {combo('nationality', 'Nacionalidade', NATIONALITY_OPTIONS, 'Selecione ou digite o país.')}
            {field('birth_year', 'Ano de nascimento', { placeholder: '1990', helperText: 'Informe 4 dígitos.' })}
            {combo('certificate_type', 'Tipo de certificado', CERTIFICATE_TYPE_OPTIONS, 'Selecione ou digite o tipo do documento.')}
            {field('id_number', 'Número do documento')}
            {field('tel', 'Telefone', { placeholder: '0055|11999999999', helperText: 'Use país|DDD+número. Ex.: 0055|(91) 98163-1459.' })}
            {field('personal_mailbox', 'E-mail pessoal', { type: 'email' })}
          </div>
        )}

        {step === 1 && (
          <>
            <div className="form-grid">
              <Select
                id="iepms-user-status"
                labelText="Status"
                value={form.status}
                onChange={(event) => {
                  updateForm({ status: event.target.value });
                  clearFieldError('status');
                }}
                invalid={Boolean(fieldErrors.status)}
                invalidText={fieldErrors.status}
              >
                <SelectItem value="ativo" text="Ativo" />
                <SelectItem value="inativado" text="Inativado" />
              </Select>
              {field('outsourcing_account', 'Conta de terceirização', { helperText: 'Somente letras, números, ponto, hífen ou sublinhado.' })}
              {field('last_updated_by', 'Última atualização por')}
              {field('password', user ? 'Senha (em branco mantém a atual)' : 'Senha', { type: 'password' })}
            </div>
            <p className="field-help">
              A senha é armazenada criptografada (AES-256-GCM) e nunca é devolvida nas listagens ou exportações.
              {user && (user.has_password ? ' Este usuário já possui senha cadastrada.' : ' Este usuário ainda não possui senha.')}
            </p>
          </>
        )}
      </div>
    </Modal>
  );
}
