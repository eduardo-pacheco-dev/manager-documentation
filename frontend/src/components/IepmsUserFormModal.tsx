import { useEffect, useState } from 'react';
import { Modal, ProgressIndicator, ProgressStep, TextArea, TextInput } from '@carbon/react';
import { useToast } from '../contexts/ToastContext';
import { ApiError, api } from '../lib/api';
import { errorMessage } from '../lib/format';
import type { IepmsUser } from '../lib/types';

const EMPTY = {
  resource_id: '',
  name: '',
  gender: '',
  nationality: '',
  us_green_card: '',
  subcontractor: '',
  certificate_type: '',
  id_number: '',
  birth_year: '',
  tel: '',
  personal_mailbox: '',
  remarks: '',
  restricted_party_screening_results: '',
  service_field: '',
  outsourcing_type: '',
  personnel_type: '',
  service_department_two_level: '',
  service_department_three_level: '',
  service_department_four_level: '',
  personnel_status: '',
  outsourcing_account: '',
  account_status: '',
  zte_mail: '',
  academic_qualifications: '',
  learning_mode: '',
  graduation_time: '',
  graduation_school: '',
  major: '',
  created_by: '',
  last_updated_by: '',
  password: '',
};

export type IepmsUserFormValues = typeof EMPTY;

const STEPS = [
  {
    key: 'identificacao',
    label: 'Identificação',
    description: 'Dados pessoais, documento, contato e triagem.',
  },
  {
    key: 'vinculo',
    label: 'Vínculo',
    description: 'Subcontratado, tipo de alocação e departamento de serviço.',
  },
  {
    key: 'conta',
    label: 'Conta e segurança',
    description: 'Conta de outsourcing, status e senha de acesso ao iEPMS.',
  },
  {
    key: 'formacao',
    label: 'Formação',
    description: 'Escolaridade, curso e observações adicionais.',
  },
];

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

  function validateRequired(): boolean {
    const errors: Record<string, string> = {};
    if (!form.resource_id.trim()) errors.resource_id = 'Informe o Resource ID.';
    if (!form.name.trim()) errors.name = 'Informe o nome.';
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  function goToStep(index: number) {
    if (index === step) return;
    if (index > 0 && !validateRequired()) return;
    setStep(Math.min(Math.max(index, 0), lastStep));
  }

  function advanceStep() {
    if (step === 0 && !validateRequired()) return;
    if (step < lastStep) {
      setStep(step + 1);
      return;
    }
    void save();
  }

  async function save() {
    if (!validateRequired()) {
      setStep(0);
      return;
    }

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
        if (error.fields.resource_id) setStep(0);
      }
      toast.error('Não foi possível salvar o usuário do iEPMS', errorMessage(error));
    } finally {
      setSaving(false);
    }
  }

  const field = (key: keyof IepmsUserFormValues, label: string, options: { type?: string; placeholder?: string } = {}) => (
    <TextInput
      id={`iepms-${key}`}
      labelText={label}
      value={form[key]}
      type={options.type}
      placeholder={options.placeholder}
      onChange={(event) => updateForm({ [key]: event.target.value })}
      invalid={Boolean(fieldErrors[key])}
      invalidText={fieldErrors[key]}
    />
  );

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
            {field('resource_id', 'Resource ID *', { placeholder: 'W0000000000' })}
            {field('name', 'Name *')}
            {field('gender', 'Gender', { placeholder: 'MALE / FEMALE' })}
            {field('nationality', 'Nationality')}
            {field('birth_year', 'Birth year', { placeholder: '1990' })}
            {field('certificate_type', 'Certificate Type')}
            {field('id_number', 'ID Number')}
            {field('tel', 'Tel', { placeholder: '0055|11999999999' })}
            {field('personal_mailbox', 'Personal mailbox', { type: 'email' })}
            {field('zte_mail', 'ZTE mail', { type: 'email' })}
            {field('us_green_card', 'Having US green cards or not')}
            {field('restricted_party_screening_results', 'Restricted party screening results')}
          </div>
        )}

        {step === 1 && (
          <div className="form-grid">
            {field('subcontractor', 'Subcontractor')}
            {field('service_field', 'Service Field')}
            {field('outsourcing_type', 'Outsourcing Type')}
            {field('personnel_type', 'Personnel Type')}
            {field('service_department_two_level', 'Service Department - two level')}
            {field('service_department_three_level', 'Service Department - three level')}
            {field('service_department_four_level', 'Service Department - four level')}
            {field('personnel_status', 'Personnel Status', { placeholder: 'Normal / NA' })}
          </div>
        )}

        {step === 2 && (
          <>
            <div className="form-grid">
              {field('outsourcing_account', 'Outsourcing account')}
              {field('account_status', 'Account Status', { placeholder: 'Normal' })}
              {field('created_by', 'Created by')}
              {field('last_updated_by', 'Last updated by')}
              {field('password', user ? 'Senha (em branco mantém a atual)' : 'Senha', { type: 'password' })}
            </div>
            <p className="field-help">
              A senha é armazenada criptografada (AES-256-GCM) e nunca é devolvida nas listagens ou exportações.
              {user && (user.has_password ? ' Este usuário já possui senha cadastrada.' : ' Este usuário ainda não possui senha.')}
            </p>
          </>
        )}

        {step === 3 && (
          <>
            <div className="form-grid">
              {field('academic_qualifications', 'Academic qualifications')}
              {field('learning_mode', 'Learning Mode')}
              {field('graduation_time', 'Graduation time')}
              {field('graduation_school', 'Graduation school')}
              {field('major', 'Major')}
            </div>
            <div className="form-grid">
              <TextArea
                id="iepms-remarks"
                labelText="Remarks"
                value={form.remarks}
                rows={4}
                onChange={(event) => updateForm({ remarks: event.target.value })}
              />
            </div>
          </>
        )}
      </div>
    </Modal>
  );
}
