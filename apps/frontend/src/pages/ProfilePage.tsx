import { useState } from 'react';
import { Save, UserProfile } from '@carbon/icons-react';
import { Button, InlineNotification, TextInput, Tile } from '@carbon/react';
import { PageHeader } from '../components/PageHeader';
import { RoleTag } from '../components/Feedback';
import { useAuth } from '../contexts/AuthContext';
import { useToast } from '../contexts/ToastContext';
import { ApiError, api } from '../lib/api';
import { errorMessage, initials, longDate } from '../lib/format';
import type { User } from '../lib/types';

export function ProfilePage() {
  const { user, patchUser } = useAuth();
  const toast = useToast();
  const [name, setName] = useState(user?.name ?? '');
  const [email, setEmail] = useState(user?.email ?? '');
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [generalError, setGeneralError] = useState('');
  const [saving, setSaving] = useState(false);

  if (!user) return null;

  async function save() {
    setSaving(true);
    setErrors({});
    setGeneralError('');
    try {
      const response = await api.put<{ data: { user: User } }>('/profile', { name, email });
      patchUser(response.data.user);
      toast.success('Perfil atualizado com sucesso');
    } catch (error) {
      if (error instanceof ApiError) {
        setErrors(error.fields);
        if (!Object.keys(error.fields).length) setGeneralError(error.message);
      } else {
        setGeneralError(errorMessage(error));
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page page--narrow">
      <PageHeader
        eyebrow="Conta"
        title="Perfil"
        description="Atualize os dados básicos da sua conta."
      />

      <div className="profile-layout">
        <Tile className="profile-card">
          <div className="profile-avatar">{initials(user.full_name)}</div>
          <h2>{user.full_name}</h2>
          <p>{user.email}</p>
          <RoleTag isAdmin={user.is_admin} />
          <dl>
            <div>
              <dt>Membro desde</dt>
              <dd>{longDate(user.created_at)}</dd>
            </div>
            <div>
              <dt>Status</dt>
              <dd>{user.is_active ? 'Ativo' : 'Inativo'}</dd>
            </div>
          </dl>
        </Tile>

        <Tile className="profile-form-card">
          <div className="profile-form-card__header">
            <UserProfile size={20} />
            <div>
              <h2>Dados pessoais</h2>
              <p>O e-mail é usado para login e notificações.</p>
            </div>
          </div>
          {generalError && (
            <InlineNotification kind="error" title="Não foi possível salvar" subtitle={generalError} lowContrast hideCloseButton />
          )}
          <TextInput
            id="profile-name"
            labelText="Nome"
            value={name}
            onChange={(event) => setName(event.target.value)}
            invalid={Boolean(errors.name)}
            invalidText={errors.name}
            required
          />
          <TextInput
            id="profile-email"
            type="email"
            labelText="E-mail"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            invalid={Boolean(errors.email)}
            invalidText={errors.email}
            required
          />
          <div className="section-actions">
            <Button renderIcon={Save} onClick={() => void save()} disabled={saving}>
              {saving ? 'Salvando…' : 'Salvar alterações'}
            </Button>
          </div>
        </Tile>
      </div>
    </div>
  );
}
