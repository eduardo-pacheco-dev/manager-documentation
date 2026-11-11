import { useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Button, Checkbox, InlineNotification, PasswordInput, TextInput, Theme } from '@carbon/react';
import { useAuth } from '../contexts/AuthContext';
import { ApiError } from '../lib/api';

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [remember, setRemember] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [generalError, setGeneralError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const from = (location.state as { from?: string } | null)?.from ?? '/dashboard';

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setErrors({});
    setGeneralError('');
    try {
      await login(email, password, remember);
      navigate(from, { replace: true });
    } catch (error) {
      if (error instanceof ApiError) {
        setErrors(error.fields);
        if (!Object.keys(error.fields).length) setGeneralError(error.message);
      } else {
        setGeneralError('Não foi possível entrar. Tente novamente.');
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
            <p className="auth-brand__eyebrow">Central de conhecimento</p>
            <h1>Documentação técnica, usuários e configurações em um só lugar.</h1>
            <p>Uma plataforma enxuta, responsiva e construída com o IBM Carbon Design System.</p>
            <ul>
              <li>Wiki em Markdown com categorias e busca</li>
              <li>Dashboard personalizável por usuário</li>
              <li>API documentada com tokens de acesso</li>
            </ul>
          </div>
          <p className="auth-brand__footer">Go · React · IBM Carbon · IBM Plex</p>
        </aside>

        <main className="auth-panel">
          <div className="auth-form">
            <p className="auth-form__eyebrow">Acesse sua conta</p>
            <h2>Entrar</h2>
            <p className="auth-form__lead">Use o e-mail e a senha cadastrados.</p>

            {generalError && (
              <InlineNotification
                kind="error"
                title="Não foi possível entrar"
                subtitle={generalError}
                lowContrast
                hideCloseButton
              />
            )}

            <form onSubmit={handleSubmit} className="auth-form__fields">
              <TextInput
                id="email"
                type="email"
                labelText="E-mail"
                placeholder="voce@empresa.com"
                autoComplete="email"
                autoFocus
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                invalid={Boolean(errors.email)}
                invalidText={errors.email}
                required
              />
              <PasswordInput
                id="password"
                labelText="Senha"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                invalid={Boolean(errors.password)}
                invalidText={errors.password}
                required
              />
              <Checkbox
                id="remember"
                labelText="Lembrar de mim"
                checked={remember}
                onChange={(_event, { checked }) => setRemember(checked)}
              />
              <Button type="submit" disabled={submitting} className="auth-submit">
                {submitting ? 'Entrando…' : 'Entrar'}
              </Button>
            </form>

            <p className="auth-form__switch">
              Ainda não tem conta? <Link to="/register">Cadastre-se</Link>
            </p>
          </div>
        </main>
      </div>
    </Theme>
  );
}
