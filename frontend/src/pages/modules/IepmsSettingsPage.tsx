import { Link } from 'react-router-dom';
import { SettingsAdjust } from '@carbon/icons-react';
import { Breadcrumb, BreadcrumbItem, Tile } from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { EmptyState } from '../../components/Feedback';

export function IepmsSettingsPage() {
  return (
    <div className="page">
      <Breadcrumb noTrailingSlash className="doc-breadcrumb">
        <BreadcrumbItem><Link to="/iepms-users">Usuários iEPMS</Link></BreadcrumbItem>
        <BreadcrumbItem isCurrentPage>Configurações</BreadcrumbItem>
      </Breadcrumb>

      <PageHeader
        eyebrow="iEPMS"
        title="Configurações"
        description="Configurações do módulo de usuários do iEPMS."
      />

      <Tile>
        <EmptyState
          icon={SettingsAdjust}
          title="Página em branco"
          description="As configurações do iEPMS serão adicionadas aqui em breve."
        />
      </Tile>
    </div>
  );
}
