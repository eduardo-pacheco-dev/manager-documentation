import { useEffect, useRef } from 'react';
import { PublicShell } from '../components/PublicShell';

const SWAGGER_CSS = 'https://unpkg.com/swagger-ui-dist@5/swagger-ui.css';
const SWAGGER_JS = 'https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js';

type SwaggerUIBundleFn = ((options: Record<string, unknown>) => void) & {
  presets?: { apis?: unknown };
};

declare global {
  interface Window {
    SwaggerUIBundle?: SwaggerUIBundleFn;
    ui?: unknown;
  }
}

export function ApiDocsPage() {
  const loaded = useRef(false);

  useEffect(() => {
    if (loaded.current) return;
    loaded.current = true;

    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = SWAGGER_CSS;
    document.head.appendChild(link);

    const script = document.createElement('script');
    script.src = SWAGGER_JS;
    script.onload = () => {
      if (window.SwaggerUIBundle) {
        window.ui = window.SwaggerUIBundle({
          url: '/docs',
          dom_id: '#swagger-ui',
          deepLinking: true,
          presets: [window.SwaggerUIBundle.presets?.apis].filter(Boolean),
        });
      }
    };
    document.body.appendChild(script);

    return () => {
      link.remove();
      script.remove();
      window.ui = undefined;
    };
  }, []);

  return (
    <PublicShell meta="Documentação da API">
      <p className="public-eyebrow">API</p>
      <h1 className="public-title">Documentation Center API</h1>
      <p className="public-lead">Especificação OpenAPI disponível em <code>/docs</code>.</p>
      <div className="api-docs-container" id="swagger-ui" />
    </PublicShell>
  );
}
