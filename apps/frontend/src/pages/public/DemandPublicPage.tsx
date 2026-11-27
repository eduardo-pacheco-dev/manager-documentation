import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { Button, InlineLoading } from '@carbon/react';
import { PublicShell } from '../../components/PublicShell';
import { api, ApiError } from '../../lib/api';
import { errorMessage } from '../../lib/format';

type DemandFile = {
  name: string;
  size: string;
  url: string;
};

type DemandPublicResponse = {
  title: string;
  heading: string;
  intro: string;
  description: string;
  scope: string;
  due_date: string;
  client_name: string;
  is_upload: boolean;
  files: DemandFile[];
  uploaded: number;
  error: string;
};

type DemandPublicPageProps = {
  mode: 'upload' | 'download';
};

export function DemandPublicPage({ mode }: DemandPublicPageProps) {
  const { token } = useParams();
  const [searchParams] = useSearchParams();
  const [data, setData] = useState<DemandPublicResponse | null>(null);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [uploading, setUploading] = useState(false);
  const loaded = useRef(false);

  useEffect(() => {
    if (loaded.current) return;
    loaded.current = true;
    const uploaded = searchParams.get('uploaded');
    const queryError = searchParams.get('error');
    if (uploaded) setMessage(`${uploaded} arquivo(s) enviado(s) com sucesso. O responsável pela demanda foi notificado.`);
    if (queryError) setError(queryError);
    if (!token) {
      setError('Este link está desativado, expirou ou não existe.');
      return;
    }
    api
      .get<DemandPublicResponse>(`/public/demands/${mode}/${encodeURIComponent(token)}`)
      .then(setData)
      .catch((err: unknown) => {
        setError(
          err instanceof ApiError && err.status === 410
            ? 'Este link está desativado, expirou ou não existe.'
            : 'Não foi possível carregar esta demanda.',
        );
      });
  }, [mode, searchParams, token]);

  async function handleUpload(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;
    const form = event.currentTarget;
    const payload = new FormData(form);
    setUploading(true);
    setError('');
    setMessage('');
    try {
      const result = await api.upload<{ uploaded: number }>(
        `/public/demands/upload/${encodeURIComponent(token)}`,
        payload,
      );
      setMessage(`${result.uploaded} arquivo(s) enviado(s) com sucesso. O responsável pela demanda foi notificado.`);
      form.reset();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setUploading(false);
    }
  }

  const meta = mode === 'upload' ? 'Envio de arquivos' : 'Download de arquivos';

  if (error && !data) {
    return (
      <PublicShell meta={meta}>
        <div className="public-unavailable">
          <p className="public-eyebrow">Demanda</p>
          <h1>Link indisponível</h1>
          <p>{error}</p>
        </div>
      </PublicShell>
    );
  }

  if (!data) {
    return (
      <PublicShell meta={meta}>
        <InlineLoading description="Carregando demanda…" />
      </PublicShell>
    );
  }

  return (
    <PublicShell meta={meta}>
      <p className="public-eyebrow">Demanda</p>
      <h1 className="public-title">{data.heading}</h1>
      <p className="public-lead">{data.intro}</p>

      <div className="public-tags">
        {data.due_date && <span className="public-tag public-tag--blue">Prazo {data.due_date}</span>}
        {data.client_name && <span className="public-tag">Cliente: {data.client_name}</span>}
        <span className={mode === 'upload' ? 'public-tag public-tag--green' : 'public-tag public-tag--blue'}>
          {mode === 'upload' ? 'Link de envio' : 'Link de download'}
        </span>
      </div>

      {message && <div className="public-alert public-alert--success">{message}</div>}
      {error && <div className="public-alert public-alert--error">{error}</div>}
      {data.uploaded > 0 && !message && (
        <div className="public-alert public-alert--success">
          {data.uploaded} arquivo(s) enviado(s) com sucesso. O responsável pela demanda foi notificado.
        </div>
      )}
      {data.error && !error && <div className="public-alert public-alert--error">{data.error}</div>}

      {data.description && (
        <section className="public-panel">
          <h2>Descrição</h2>
          <p className="public-hint">Resumo da demanda.</p>
          <div className="public-description">{data.description}</div>
          {(data.due_date || data.client_name) && (
            <div className="public-meta">
              {data.due_date && (
                <div><span>Prazo</span><strong>{data.due_date}</strong></div>
              )}
              {data.client_name && (
                <div><span>Cliente</span><strong>{data.client_name}</strong></div>
              )}
            </div>
          )}
        </section>
      )}

      {data.scope && (
        <section className="public-panel">
          <h2>Escopo da atividade</h2>
          <p className="public-hint">Detalhamento operacional solicitado.</p>
          <div className="public-description">{data.scope}</div>
        </section>
      )}

      {mode === 'upload' ? (
        <section className="public-panel">
          <h2>Enviar arquivos</h2>
          <p className="public-hint">
            Envie documentos, evidências, fotos ou planilhas relacionados à demanda. Até 20 arquivos por demanda, com no máximo 50 MB cada.
          </p>
          <form className="public-form" onSubmit={(event) => void handleUpload(event)}>
            <label>
              Seu nome (opcional)
              <input type="text" name="name" maxLength={255} placeholder="Nome do colaborador" />
            </label>
            <label>
              Arquivos
              <input type="file" name="files" multiple required />
            </label>
            <Button type="submit" disabled={uploading}>
              {uploading ? 'Enviando…' : 'Enviar arquivos'}
            </Button>
          </form>
        </section>
      ) : (
        <section className="public-panel">
          <h2>Arquivos para download</h2>
          <p className="public-hint">Baixe os arquivos disponibilizados nesta demanda.</p>
          {data.files.length > 0 ? (
            <ul className="public-file-list">
              {data.files.map((file) => (
                <li className="public-file-row" key={file.url}>
                  <span className="public-file-icon" aria-hidden="true">DOC</span>
                  <div className="public-file-info">
                    <strong>{file.name}</strong>
                    <span>{file.size}</span>
                  </div>
                  <a className="public-download" href={file.url} download>Baixar</a>
                </li>
              ))}
            </ul>
          ) : (
            <p className="public-empty">Nenhum arquivo disponível para download.</p>
          )}
        </section>
      )}

      <p className="public-notice">Este link é exclusivo para esta demanda. Não o compartilhe com pessoas não envolvidas.</p>
    </PublicShell>
  );
}
