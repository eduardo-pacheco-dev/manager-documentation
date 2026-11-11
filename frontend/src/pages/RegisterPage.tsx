import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Button, InlineNotification, PasswordInput, TextInput, Theme } from '@carbon/react';
import { useAuth } from '../contexts/AuthContext';
import { ApiError } from '../lib/api';

export function RegisterPage() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ name: '', email: '', password: '', confirmation: '' });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [generalError, setGeneralError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setErrors({});
    setGeneralError('');
    try {
      await register({
        name: form.name,
        email: form.email,
        password: form.password,
        password_confirmation: form.confirmation,
      });
      navigate('/dashboard', { replace: true });
    } catch (error) {
      if (error instanceof ApiError) {
        setErrors(error.fields);
        if (!Object.keys(error.fields).length) setGeneralError(error.message);
      } else {
        setGeneralError('Não foi possível criar a conta. Tente novamente.');
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Theme theme="g100">
      <div className="auth-shell">
        <aside className="auth-brand">
          <div className="auth-brand__top">
            <span className="auth-brand__mark">DC</span>
            <span>Documentation Center</span>
          </div>
          <div className="auth-brand__copy">
            <p className="auth-brand__eyebrow">Comece agora</p>
            <h1>Crie sua conta e organize o conhecimento do time.</h1>
            <p>Cadastre-se para acessar o painel, acompanhar notificações e personalizar seu ambiente.</p>
            <ul>
              <li>Tema claro, escuro ou automático</li>
              <li>Widgets configuráveis no dashboard</li>
              <li>Notificações internas e por e-mail</li>
            </ul>
          </div>
          <p className="auth-brand__footer">Go · React · IBM Carbon · IBM Plex</p>
        </aside>

        <main className="auth-panel">
          <div className="auth-form">
            <p className="auth-form__eyebrow">Nova conta</p>
            <h2>Criar conta</h2>
            <p className="auth-form__lead">Preencha os dados abaixo para começar.</p>

            {generalError && (
              <InlineNotification
                kind="error"
                title="Não foi possível criar a conta"
                subtitle={generalError}
                lowContrast
                hideCloseButton
              />
            )}

            <form onSubmit={handleSubmit} className="auth-form__fields">
              <TextInput
                id="name"
                labelText="Nome"
                placeholder="Seu nome"
                autoComplete="name"
                value={form.name}
                onChange={(event) => setForm({ ...form, name: event.target.value })}
                invalid={Boolean(errors.name)}
                invalidText={errors.name}
                required
              />
              <TextInput
                id="email"
                type="email"
                labelText="E-mail"
                placeholder="voce@empresa.com"
                autoComplete="email"
                value={form.email}
                onChange={(event) => setForm({ ...form, email: event.target.value })}
                invalid={Boolean(errors.email)}
                invalidText={errors.email}
                required
              />
              <PasswordInput
                id="password"
                labelText="Senha"
                helperText="Use pelo menos 8 caracteres."
                autoComplete="new-password"
                value={form.password}
                onChange={(event) => setForm({ ...form, password: event.target.value })}
                invalid={Boolean(errors.password)}
                invalidText={errors.password}
                required
              />
              <PasswordInput
                id="password_confirmation"
                labelText="Confirmar senha"
                autoComplete="new-password"
                value={form.confirmation}
                onChange={(event) => setForm({ ...form, confirmation: event.target.value })}
                invalid={Boolean(errors.password_confirmation)}
                invalidText={errors.password_confirmation}
                required
              />
              <Button type="submit" disabled={submitting} className="auth-submit">
                {submitting ? 'Criando conta…' : 'Criar conta'}
              </Button>
            </form>

            <p className="auth-form__switch">
              Já tem uma conta? <Link to="/login">Entrar</Link>
            </p>
          </div>
        </main>
      </div>
    </Theme>
  );
}
