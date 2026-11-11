import { useEffect, useState } from 'react';
import { Cube } from '@carbon/icons-react';
import { InlineLoading, Tag, Tile } from '@carbon/react';
import { PageHeader } from '../../components/PageHeader';
import { api } from '../../lib/api';

export function ExamplePage() {
  const [data, setData] = useState<{ title: string; message: string; version: string; features: string[] } | null>(null);

  useEffect(() => {
    api.get<{ data: typeof data }>('/example').then((response) => setData(response.data)).catch(() => undefined);
  }, []);

  return (
    <div className="page page--narrow">
      <PageHeader
        eyebrow="Módulo nativo"
        title="Exemplo"
        description="O antigo plugin Example foi convertido em um módulo nativo do sistema."
      />
      {!data ? <InlineLoading description="Carregando módulo…" /> : (
        <>
          <Tile className="module-main__header">
            <div className="module-icon-title">
              <Cube size={28} />
              <div>
                <h2>{data.title}</h2>
                <p>{data.message}</p>
              </div>
              <Tag type="green" size="sm">v{data.version}</Tag>
            </div>
          </Tile>
          <Tile className="module-subcard">
            <h3>Recursos</h3>
            <ul className="feature-list">
              {data.features.map((feature) => <li key={feature}>{feature}</li>)}
            </ul>
          </Tile>
        </>
      )}
    </div>
  );
}
