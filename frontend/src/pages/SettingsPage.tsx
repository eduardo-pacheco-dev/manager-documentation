import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  ArrowDown,
  ArrowUp,
  Checkmark,
  Cloud,
  DataBase,
  Save,
  SettingsAdjust,
  UserMultiple,
} from '@carbon/icons-react';
import {
  Button,
  Checkbox,
  InlineNotification,
  PasswordInput,
  RadioButton,
  RadioButtonGroup,
  Select,
  SelectItem,
  TextInput,
  Tile,
  Toggle,
} from '@carbon/react';
import { PageHeader } from '../components/PageHeader';
import { Loading } from '../components/Feedback';
import { ConfirmModal } from '../components/ConfirmModal';
import { BackupsPanel } from '../components/BackupsPanel';
import { useAuth } from '../contexts/AuthContext';
import { useToast } from '../contexts/ToastContext';
import { ApiError, api } from '../lib/api';
import { errorMessage } from '../lib/format';
import type { Module, SettingsResponse, StorageSettings, ThemePreference } from '../lib/types';

const SECTION_LINKS = [
  { id: 'aparencia', label: 'Aparência' },
  { id: 'modulos', label: 'Módulos' },
  { id: 'notificacoes', label: 'Notificações' },
  { id: 'sistema', label: 'Sistema', admin: true },
  { id: 'armazenamento', label: 'Armazenamento', admin: true },
  { id: 'backups', label: 'Backups', admin: true },
  { id: 'senha', label: 'Alterar senha' },
  { id: 'conta', label: 'Excluir conta', danger: true },
];

function SettingsSection({
  id,
  title,
  description,
  children,
  icon: Icon = SettingsAdjust,
  danger = false,
}: {
  id: string;
  title: string;
  description?: string;
  children: React.ReactNode;
  icon?: React.ComponentType<{ size?: number }>;
  danger?: boolean;
}) {
  return (
    <section id={id} className={`settings-section ${danger ? 'settings-section--danger' : ''}`}>
      <header className="settings-section__header">
        <div className="settings-section__icon"><Icon size={20} /></div>
        <div>
          <h2>{title}</h2>
          {description && <p>{description}</p>}
        </div>
      </header>
      <div className="settings-section__body">{children}</div>
    </section>
  );
}

export function SettingsPage() {
  const { patchUser, refresh, logout } = useAuth();
  const toast = useToast();
  const navigate = useNavigate();
  const [data, setData] = useState<SettingsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [theme, setTheme] = useState<ThemePreference>('system');
  const [modules, setModules] = useState<Module[]>([]);
  const [emailNotifications, setEmailNotifications] = useState(true);
  const [timezone, setTimezone] = useState('UTC');
  const [storage, setStorage] = useState<StorageSettings | null>(null);
  const [secrets, setSecrets] = useState({
    awsAccessKey: '',
    awsSecretKey: '',
    cloudflareAccessKey: '',
    cloudflareSecretKey: '',
  });
  const [passwords, setPasswords] = useState({ current: '', next: '', confirmation: '' });
  const [deletePassword, setDeletePassword] = useState('');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [saving, setSaving] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [storageErrors, setStorageErrors] = useState<Record<string, string>>({});

  const load = async () => {
    try {
      const response = await api.get<{ data: SettingsResponse }>('/settings');
      const settings = response.data;
      setData(settings);
      setTheme(settings.theme);
      setModules(settings.modules);
      setEmailNotifications(settings.email_notifications);
      setTimezone(settings.timezone);
      setStorage(settings.storage ?? null);
    } catch (error) {
      toast.error('Não foi possível carregar as configurações', errorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const links = useMemo(
    () => SECTION_LINKS.filter((link) => !link.admin || data?.is_admin),
    [data?.is_admin],
  );

  async function saveTheme() {
    setSaving('theme');
    try {
      await api.put('/settings/theme', { theme });
      patchUser({ theme });
      toast.success('Preferência de tema salva com sucesso');
    } catch (error) {
      toast.error('Não foi possível salvar o tema', errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  function moveModule(index: number, direction: -1 | 1) {
    const target = index + direction;
    if (target < 0 || target >= modules.length) return;
    const next = [...modules];
    [next[index], next[target]] = [next[target], next[index]];
    setModules(next);
  }

  function toggleModule(key: string, enabled: boolean) {
    setModules((current) =>
      current.map((module) => (module.key === key ? { ...module, enabled } : module)),
    );
  }

  function moveSection(key: string) {
    setModules((current) =>
      current.map((module) =>
        module.key === key
          ? { ...module, section: module.section === 'sidebar' ? 'modules' : 'sidebar' }
          : module,
      ),
    );
  }

  async function saveModules() {
    setSaving('modules');
    try {
      const payload = {
        modules: modules.filter((module) => module.toggleable && module.enabled).map((module) => module.key),
        order: modules.map((module) => module.key),
        sections: Object.fromEntries(modules.map((module) => [module.key, module.section])),
      };
      const response = await api.put<{ data: { modules: Module[] } }>('/settings/modules', payload);
      setModules(response.data.modules);
      await refresh();
      toast.success('Módulos atualizados com sucesso');
    } catch (error) {
      if (error instanceof ApiError) toast.error('Não foi possível salvar os módulos', error.message);
      else toast.error('Não foi possível salvar os módulos', errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function saveNotifications() {
    setSaving('notifications');
    try {
      await api.put('/settings/notifications', { email_notifications: emailNotifications });
      toast.success('Preferências de notificação atualizadas');
    } catch (error) {
      toast.error('Não foi possível salvar as preferências', errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function saveTimezone() {
    setSaving('timezone');
    try {
      await api.put('/settings/system', { timezone });
      toast.success('Fuso horário atualizado com sucesso');
    } catch (error) {
      if (error instanceof ApiError && error.fields.timezone) {
        toast.error('Fuso horário inválido', error.fields.timezone);
      } else {
        toast.error('Não foi possível salvar o fuso horário', errorMessage(error));
      }
    } finally {
      setSaving(null);
    }
  }

  async function saveStorage() {
    if (!storage) return;
    setSaving('storage');
    setStorageErrors({});
    try {
      const response = await api.put<{ data: { storage: StorageSettings } }>('/settings/storage', {
        provider: storage.provider,
        max_upload_mb: Number(storage.max_upload_mb),
        aws_access_key: secrets.awsAccessKey,
        aws_secret_key: secrets.awsSecretKey,
        aws_region: storage.aws_region,
        aws_bucket: storage.aws_bucket,
        aws_endpoint: storage.aws_endpoint,
        aws_public_url: storage.aws_public_url,
        aws_use_path_style: storage.aws_use_path_style,
        cloudflare_account_id: storage.cloudflare_account_id,
        cloudflare_access_key_id: secrets.cloudflareAccessKey,
        cloudflare_secret_access_key: secrets.cloudflareSecretKey,
        cloudflare_bucket: storage.cloudflare_bucket,
        cloudflare_public_url: storage.cloudflare_public_url,
      });
      setStorage(response.data.storage);
      setSecrets({ awsAccessKey: '', awsSecretKey: '', cloudflareAccessKey: '', cloudflareSecretKey: '' });
      toast.success('Configurações de armazenamento salvas');
    } catch (error) {
      if (error instanceof ApiError) {
        setStorageErrors(error.fields);
        toast.error('Não foi possível salvar o armazenamento', error.message);
      } else {
        toast.error('Não foi possível salvar o armazenamento', errorMessage(error));
      }
    } finally {
      setSaving(null);
    }
  }

  async function savePassword() {
    setSaving('password');
    setFieldErrors({});
    try {
      await api.put('/settings/password', {
        current_password: passwords.current,
        password: passwords.next,
        password_confirmation: passwords.confirmation,
      });
      setPasswords({ current: '', next: '', confirmation: '' });
      toast.success('Senha atualizada com sucesso');
    } catch (error) {
      if (error instanceof ApiError) {
        setFieldErrors(error.fields);
        toast.error('Não foi possível atualizar a senha', error.message);
      } else {
        toast.error('Não foi possível atualizar a senha', errorMessage(error));
      }
    } finally {
      setSaving(null);
    }
  }

  async function deleteAccount() {
    setSaving('account');
    try {
      await api.delete('/settings/account', { delete_password: deletePassword });
      toast.success('Sua conta foi excluída com sucesso');
      await logout();
      navigate('/login');
    } catch (error) {
      if (error instanceof ApiError && error.fields.delete_password) {
        toast.error('Não foi possível excluir a conta', error.fields.delete_password);
      } else {
        toast.error('Não foi possível excluir a conta', errorMessage(error));
      }
      setDeleteOpen(false);
    } finally {
      setSaving(null);
    }
  }

  if (loading || !data) return <Loading description="Carregando configurações…" />;

  return (
    <div className="page">
      <PageHeader
        eyebrow="Conta"
        title="Configurações"
        description="Personalize a aparência, os módulos, as notificações, os backups e a segurança da sua conta."
      />

      <div className="settings-layout">
        <nav className="settings-nav" aria-label="Seções de configurações">
          {links.map((link) => (
            <a key={link.id} href={`#${link.id}`} className={link.danger ? 'is-danger' : ''}>
              {link.label}
            </a>
          ))}
        </nav>

        <div className="settings-content">
          <SettingsSection
            id="aparencia"
            title="Aparência"
            description="Escolha como o sistema deve ser exibido neste dispositivo."
            icon={SettingsAdjust}
          >
            <RadioButtonGroup
              legendText="Tema"
              name="theme"
              valueSelected={theme}
              onChange={(value) => setTheme(value as ThemePreference)}
            >
              <RadioButton labelText="Sistema — segue a preferência do dispositivo" value="system" id="theme-system" />
              <RadioButton labelText="Claro — sempre com fundo claro" value="light" id="theme-light" />
              <RadioButton labelText="Escuro — sempre com fundo escuro" value="dark" id="theme-dark" />
            </RadioButtonGroup>
            <div className="section-actions">
              <Button renderIcon={Save} onClick={() => void saveTheme()} disabled={saving === 'theme'}>
                {saving === 'theme' ? 'Salvando…' : 'Salvar aparência'}
              </Button>
            </div>
          </SettingsSection>

          <SettingsSection
            id="modulos"
            title="Módulos"
            description="Ative, ordene e escolha onde cada módulo aparece na navegação."
            icon={UserMultiple}
          >
            <div className="module-list">
              {modules.map((module, index) => (
                <div key={module.key} className={`module-row ${module.enabled ? '' : 'module-row--disabled'}`}>
                  <div className="module-row__main">
                    {module.toggleable ? (
                      <Checkbox
                        id={`module-${module.key}`}
                        labelText={module.label}
                        checked={module.enabled}
                        onChange={(_event, { checked }) => toggleModule(module.key, checked)}
                      />
                    ) : (
                      <strong>{module.label}</strong>
                    )}
                    <span className="module-row__meta">
                      {module.section === 'sidebar' ? 'Sidebar' : 'Menu Módulos'}
                      {!module.toggleable && ' · sempre ativo'}
                    </span>
                  </div>
                  <div className="module-row__actions">
                    <Button
                      kind="ghost"
                      size="sm"
                      hasIconOnly
                      renderIcon={ArrowUp}
                      iconDescription="Mover para cima"
                      onClick={() => moveModule(index, -1)}
                      disabled={index === 0}
                    />
                    <Button
                      kind="ghost"
                      size="sm"
                      hasIconOnly
                      renderIcon={ArrowDown}
                      iconDescription="Mover para baixo"
                      onClick={() => moveModule(index, 1)}
                      disabled={index === modules.length - 1}
                    />
                    <Button kind="tertiary" size="sm" onClick={() => moveSection(module.key)}>
                      Mover para {module.section === 'sidebar' ? 'Módulos' : 'Sidebar'}
                    </Button>
                  </div>
                </div>
              ))}
            </div>
            <div className="section-actions">
              <Button renderIcon={Save} onClick={() => void saveModules()} disabled={saving === 'modules'}>
                {saving === 'modules' ? 'Salvando…' : 'Salvar módulos'}
              </Button>
            </div>
          </SettingsSection>

          <SettingsSection
            id="notificacoes"
            title="Notificações"
            description="Notificações no sistema são sempre exibidas. O e-mail é opcional."
            icon={Checkmark}
          >
            <Toggle
              id="email-notifications"
              labelText="Receber notificações por e-mail"
              labelA="Desativado"
              labelB="Ativado"
              toggled={emailNotifications}
              onToggle={(checked) => setEmailNotifications(checked)}
            />
            <div className="section-actions">
              <Button renderIcon={Save} onClick={() => void saveNotifications()} disabled={saving === 'notifications'}>
                {saving === 'notifications' ? 'Salvando…' : 'Salvar notificações'}
              </Button>
            </div>
          </SettingsSection>

          {data.is_admin && (
            <SettingsSection
              id="sistema"
              title="Sistema"
              description="Fuso horário usado para exibir datas e executar tarefas."
              icon={DataBase}
            >
              <Select
                id="timezone"
                labelText="Fuso horário"
                value={timezone}
                onChange={(event) => setTimezone(event.target.value)}
              >
                {data.timezones.map((zone) => (
                  <SelectItem key={zone} value={zone} text={zone} />
                ))}
              </Select>
              <div className="section-actions">
                <Button renderIcon={Save} onClick={() => void saveTimezone()} disabled={saving === 'timezone'}>
                  {saving === 'timezone' ? 'Salvando…' : 'Salvar fuso horário'}
                </Button>
              </div>
            </SettingsSection>
          )}

          {data.is_admin && storage && (
            <SettingsSection
              id="armazenamento"
              title="Armazenamento"
              description="Provedor usado para armazenar imagens enviadas pela API e pelo editor."
              icon={Cloud}
            >
              <TextInput
                id="max-upload"
                type="number"
                labelText="Tamanho máximo (MB)"
                value={String(storage.max_upload_mb)}
                onChange={(event) => setStorage({ ...storage, max_upload_mb: Number(event.target.value) })}
                invalid={Boolean(storageErrors.max_upload_mb)}
                invalidText={storageErrors.max_upload_mb}
              />
              <RadioButtonGroup
                legendText="Provedor"
                name="storage-provider"
                valueSelected={storage.provider}
                onChange={(value) => setStorage({ ...storage, provider: value as StorageSettings['provider'] })}
              >
                <RadioButton labelText="Local (disco do servidor)" value="local" id="storage-local" />
                <RadioButton labelText="AWS S3" value="aws" id="storage-aws" />
                <RadioButton labelText="Cloudflare R2" value="cloudflare" id="storage-cloudflare" />
              </RadioButtonGroup>

              {storage.provider === 'local' && (
                <InlineNotification
                  kind="info"
                  title="Armazenamento local"
                  subtitle="As imagens são gravadas no diretório UPLOAD_DIR e servidas por /uploads."
                  lowContrast
                  hideCloseButton
                />
              )}

              {storage.provider === 'aws' && (
                <div className="form-grid">
                  <TextInput
                    id="aws-access-key"
                    labelText="Access Key ID"
                    value={secrets.awsAccessKey}
                    helperText={storage.aws_access_key_set ? 'Já configurada. Deixe em branco para manter.' : ''}
                    onChange={(event) => setSecrets({ ...secrets, awsAccessKey: event.target.value })}
                    invalid={Boolean(storageErrors.aws_access_key)}
                    invalidText={storageErrors.aws_access_key}
                  />
                  <PasswordInput
                    id="aws-secret-key"
                    labelText="Secret Access Key"
                    value={secrets.awsSecretKey}
                    helperText={storage.aws_secret_key_set ? 'Já configurada. Deixe em branco para manter.' : ''}
                    onChange={(event) => setSecrets({ ...secrets, awsSecretKey: event.target.value })}
                    invalid={Boolean(storageErrors.aws_secret_key)}
                    invalidText={storageErrors.aws_secret_key}
                  />
                  <TextInput
                    id="aws-region"
                    labelText="Região"
                    value={storage.aws_region}
                    onChange={(event) => setStorage({ ...storage, aws_region: event.target.value })}
                    invalid={Boolean(storageErrors.aws_region)}
                    invalidText={storageErrors.aws_region}
                  />
                  <TextInput
                    id="aws-bucket"
                    labelText="Bucket"
                    value={storage.aws_bucket}
                    onChange={(event) => setStorage({ ...storage, aws_bucket: event.target.value })}
                    invalid={Boolean(storageErrors.aws_bucket)}
                    invalidText={storageErrors.aws_bucket}
                  />
                  <TextInput
                    id="aws-endpoint"
                    labelText="Endpoint (opcional)"
                    value={storage.aws_endpoint}
                    onChange={(event) => setStorage({ ...storage, aws_endpoint: event.target.value })}
                    invalid={Boolean(storageErrors.aws_endpoint)}
                    invalidText={storageErrors.aws_endpoint}
                  />
                  <TextInput
                    id="aws-public-url"
                    labelText="URL pública (opcional)"
                    value={storage.aws_public_url}
                    onChange={(event) => setStorage({ ...storage, aws_public_url: event.target.value })}
                    invalid={Boolean(storageErrors.aws_public_url)}
                    invalidText={storageErrors.aws_public_url}
                  />
                  <Checkbox
                    id="aws-path-style"
                    labelText="Usar path-style endpoint"
                    checked={storage.aws_use_path_style}
                    onChange={(_event, { checked }) => setStorage({ ...storage, aws_use_path_style: checked })}
                  />
                </div>
              )}

              {storage.provider === 'cloudflare' && (
                <div className="form-grid">
                  <TextInput
                    id="cf-account"
                    labelText="Account ID"
                    value={storage.cloudflare_account_id}
                    onChange={(event) => setStorage({ ...storage, cloudflare_account_id: event.target.value })}
                    invalid={Boolean(storageErrors.cloudflare_account_id)}
                    invalidText={storageErrors.cloudflare_account_id}
                  />
                  <TextInput
                    id="cf-access"
                    labelText="Access Key ID"
                    value={secrets.cloudflareAccessKey}
                    helperText={storage.cloudflare_access_key_set ? 'Já configurada. Deixe em branco para manter.' : ''}
                    onChange={(event) => setSecrets({ ...secrets, cloudflareAccessKey: event.target.value })}
                    invalid={Boolean(storageErrors.cloudflare_access_key_id)}
                    invalidText={storageErrors.cloudflare_access_key_id}
                  />
                  <PasswordInput
                    id="cf-secret"
                    labelText="Secret Access Key"
                    value={secrets.cloudflareSecretKey}
                    helperText={storage.cloudflare_secret_key_set ? 'Já configurada. Deixe em branco para manter.' : ''}
                    onChange={(event) => setSecrets({ ...secrets, cloudflareSecretKey: event.target.value })}
                    invalid={Boolean(storageErrors.cloudflare_secret_access_key)}
                    invalidText={storageErrors.cloudflare_secret_access_key}
                  />
                  <TextInput
                    id="cf-bucket"
                    labelText="Bucket"
                    value={storage.cloudflare_bucket}
                    onChange={(event) => setStorage({ ...storage, cloudflare_bucket: event.target.value })}
                    invalid={Boolean(storageErrors.cloudflare_bucket)}
                    invalidText={storageErrors.cloudflare_bucket}
                  />
                  <TextInput
                    id="cf-public-url"
                    labelText="URL pública (opcional)"
                    value={storage.cloudflare_public_url}
                    onChange={(event) => setStorage({ ...storage, cloudflare_public_url: event.target.value })}
                    invalid={Boolean(storageErrors.cloudflare_public_url)}
                    invalidText={storageErrors.cloudflare_public_url}
                  />
                </div>
              )}

              <div className="section-actions">
                <Button renderIcon={Save} onClick={() => void saveStorage()} disabled={saving === 'storage'}>
                  {saving === 'storage' ? 'Salvando…' : 'Salvar armazenamento'}
                </Button>
              </div>
            </SettingsSection>
          )}

          {data.is_admin && (
            <SettingsSection
              id="backups"
              title="Backups"
              description="Gere, importe, baixe e restaure cópias de segurança do banco de dados, uploads e anexos."
              icon={DataBase}
            >
              <BackupsPanel />
            </SettingsSection>
          )}

          <SettingsSection
            id="senha"
            title="Alterar senha"
            description="Use pelo menos 8 caracteres e não repita a senha atual."
            icon={Checkmark}
          >
            <PasswordInput
              id="current-password"
              labelText="Senha atual"
              value={passwords.current}
              onChange={(event) => setPasswords({ ...passwords, current: event.target.value })}
              invalid={Boolean(fieldErrors.current_password)}
              invalidText={fieldErrors.current_password}
            />
            <div className="form-row">
              <PasswordInput
                id="new-password"
                labelText="Nova senha"
                value={passwords.next}
                onChange={(event) => setPasswords({ ...passwords, next: event.target.value })}
                invalid={Boolean(fieldErrors.password)}
                invalidText={fieldErrors.password}
              />
              <PasswordInput
                id="confirm-password"
                labelText="Confirmar nova senha"
                value={passwords.confirmation}
                onChange={(event) => setPasswords({ ...passwords, confirmation: event.target.value })}
                invalid={Boolean(fieldErrors.password_confirmation)}
                invalidText={fieldErrors.password_confirmation}
              />
            </div>
            <div className="section-actions">
              <Button renderIcon={Save} onClick={() => void savePassword()} disabled={saving === 'password'}>
                {saving === 'password' ? 'Salvando…' : 'Atualizar senha'}
              </Button>
            </div>
          </SettingsSection>

          <SettingsSection
            id="conta"
            title="Excluir conta"
            description="Esta ação remove permanentemente sua conta, documentos e notificações."
            icon={DataBase}
            danger
          >
            <PasswordInput
              id="delete-password"
              labelText="Confirme sua senha"
              value={deletePassword}
              onChange={(event) => setDeletePassword(event.target.value)}
            />
            <div className="section-actions">
              <Button kind="danger" onClick={() => setDeleteOpen(true)} disabled={!deletePassword}>
                Excluir minha conta
              </Button>
            </div>
          </SettingsSection>
        </div>
      </div>

      <ConfirmModal
        open={deleteOpen}
        title="Excluir conta"
        message="Excluir sua conta permanentemente? Esta ação não pode ser desfeita."
        confirmLabel="Excluir conta"
        loading={saving === 'account'}
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => void deleteAccount()}
      />
    </div>
  );
}
