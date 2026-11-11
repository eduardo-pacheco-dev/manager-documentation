import { Link, useNavigate } from 'react-router-dom';
import { Button } from '@carbon/react';

export function NotFoundPage() {
  const navigate = useNavigate();
  return (
    <div className="status-page">
      <p className="status-page__code">404</p>
      <h1>Página não encontrada</h1>
      <p>O endereço acessado não existe ou foi movido.</p>
      <Button onClick={() => navigate('/dashboard')}>Voltar ao dashboard</Button>
    </div>
  );
}

export function ForbiddenPage() {
  return (
    <div className="status-page">
      <p className="status-page__code">403</p>
      <h1>Acesso restrito</h1>
      <p>Você não tem permissão para acessar esta área.</p>
      <Link to="/dashboard"><Button>Voltar ao dashboard</Button></Link>
    </div>
  );
}
