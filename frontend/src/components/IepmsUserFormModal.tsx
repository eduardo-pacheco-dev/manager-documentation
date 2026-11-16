import { useEffect, useState } from 'react';
import { Modal, TextArea, TextInput } from '@carbon/react';
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
};

export type IepmsUserFormValues = typeof EMPTY;

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
  const [saving, setSaving] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!open) return;
    setForm(user ? formFromUser(user) : { ...EMPTY });
    setFieldErrors({});
  }, [open, user]);

  function updateForm(patch: Partial<IepmsUserFormValues>) {
    setForm((current) => ({ ...current, ...patch }));
  }

  async function save() {
    const errors: Record<string, string> = {};
    if (!form.resource_id.trim()) errors.resource_id = 'Informe o Resource ID.';
    if (!form.name.trim()) errors.name = 'Informe o nome.';
    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) return;

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
      primaryButtonText={saving ? 'Salvando…' : 'Salvar'}
      secondaryButtonText="Cancelar"
      primaryButtonDisabled={saving}
      onRequestSubmit={() => void save()}
      onRequestClose={onClose}
      size="lg"
    >
      <h4>Identificação</h4>
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

      <h4>Vínculo</h4>
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

      <h4>Conta</h4>
      <div className="form-grid">
        {field('outsourcing_account', 'Outsourcing account')}
        {field('account_status', 'Account Status', { placeholder: 'Normal' })}
        {field('created_by', 'Created by')}
        {field('last_updated_by', 'Last updated by')}
      </div>

      <h4>Formação</h4>
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
          rows={3}
          onChange={(event) => updateForm({ remarks: event.target.value })}
        />
      </div>
    </Modal>
  );
}
