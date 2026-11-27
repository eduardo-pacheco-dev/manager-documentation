import { useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { InlineLoading } from '@carbon/react';
import { PublicShell } from '../../components/PublicShell';
import { ApiError, api } from '../../lib/api';

type ShortLinkFile = {
  name: string;
  size: string;
  type: string;
  url: string;
};

type ShortLinkResponse = {
  code?: string;
  title?: string;
  description?: string;
  target_url?: string;
  target_host?: string;
  files?: ShortLinkFile[];
  file_count?: number;
  total_size?: string;
  expires?: string;
  redirect_url?: string;
};

export function ShortLinkPage() {
  const { code } = useParams();
  const [data, setData] = useState<ShortLinkResponse | null>(null);
  const [error, setError] = useState('');
  const loaded = useRef(false);

  useEffect(() => {
    if (loaded.current) return;
    loaded.current = true;
    if (!code) {
      setError('Este link está desativado, expirou ou não existe.');
      return;
    }
    api
      .get<ShortLinkResponse>(`/public/links/${encodeURIComponent(code)}`)
      .then((result) => {
        if (result.redirect_url) {
          window.location.replace(result.redirect_url);
          return;
        }
        setData(result);
      })
      .catch((err: unknown) => {
        setError(
          err instanceof ApiError && err.status === 410
            ? 'Este link está desativado, expirou ou não existe.'
            : 'Não foi possível carregar este link.',
        );
      });
  }, [code]);

  if (error) {
    return (
      <PublicShell meta="Link compartilhado">
        <div className="public-unavailable">
          <p className="public-eyebrow">Link compartilhado</p>
          <h1>Link indisponível</h1>
          <p>{error}</p>
        </div>
      </PublicShell>
    );
  }

  if (!data) {
    return (
      <PublicShell meta="Link compartilhado">
        <InlineLoading description="Carregando link…" />
      </PublicShell>
    );
  }

  const files = data.files ?? [];
  const fileCount = data.file_count ?? files.length;

  return (
    <PublicShell meta="Compartilhamento de arquivos">
      <p className="public-eyebrow">Link compartilhado</p>
      <h1 className="public-title">{data.title || data.code}</h1>
      {data.description && <p className="public-lead">{data.description}</p>}

      <div className="public-tags">
        <span className="public-tag public-tag--blue">{fileCount} arquivo(s)</span>
        {data.total_size && <span className="public-tag">{data.total_size}</span>}
        {data.expires && <span className="public-tag public-tag--green">Expira em {data.expires}</span>}
      </div>

      {data.target_url && (
        <section className="public-panel">
          <h2>Destino do link</h2>
          <p className="public-hint">Este link também aponta para uma página externa.</p>
          <div className="public-target">
            <a className="public-button" href={data.target_url} target="_blank" rel="noopener noreferrer">
              Abrir destino
            </a>
            <span className="public-target__host">{data.target_host || data.target_url}</span>
          </div>
        </section>
      )}

      <section className="public-panel">
        <h2>Arquivos para download</h2>
        <p className="public-hint">Baixe os arquivos disponibilizados neste link.</p>
        {files.length > 0 ? (
          <ul className="public-file-list">
            {files.map((file) => (
              <li className="public-file-row" key={file.url}>
                <span className="public-file-icon" aria-hidden="true">DOC</span>
                <div className="public-file-info">
                  <strong>{file.name}</strong>
                  <span>{file.type} · {file.size}</span>
                </div>
                <a className="public-download" href={file.url} download>Baixar</a>
              </li>
            ))}
          </ul>
        ) : (
          <p className="public-empty">Nenhum arquivo anexado a este link.</p>
        )}
      </section>

      <p className="public-notice">
        Os arquivos são disponibilizados pelo proprietário do link. Em caso de dúvida, entre em contato com quem compartilhou.
      </p>
    </PublicShell>
  );
}
