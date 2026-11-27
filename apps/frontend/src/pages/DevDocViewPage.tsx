import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, Edit, TrashCan } from '@carbon/icons-react';
import { Breadcrumb, BreadcrumbItem, Button, InlineLoading, Tag } from '@carbon/react';
import { Loading, PublishedTag } from '../components/Feedback';
import { ConfirmModal } from '../components/ConfirmModal';
import { WikiNavigation } from '../components/WikiNavigation';
import { useToast } from '../contexts/ToastContext';
import { api } from '../lib/api';
import { errorMessage, fullDate, relativeTime } from '../lib/format';
import type { DocShowResponse } from '../lib/types';

export function DevDocViewPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const toast = useToast();
  const [data, setData] = useState<DocShowResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    let active = true;
    setLoading(true);
    api
      .get<{ data: DocShowResponse }>(`/dev-docs/${id}`)
      .then((response) => {
        if (active) setData(response.data);
      })
      .catch(() => {
        if (active) setNotFound(true);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [id]);

  async function remove() {
    setDeleting(true);
    try {
      await api.delete(`/dev-docs/${id}`);
      toast.success('Documento excluído com sucesso');
      navigate('/dev-docs');
    } catch (error) {
      toast.error('Não foi possível excluir', errorMessage(error));
    } finally {
      setDeleting(false);
    }
  }

  if (loading) return <Loading description="Carregando documento…" />;

  if (notFound || !data) {
    return (
      <div className="status-page">
        <p className="status-page__code">404</p>
        <h1>Documento não encontrado</h1>
        <p>O documento pode ter sido excluído ou o endereço está incorreto.</p>
        <Link to="/dev-docs"><Button>Voltar para a wiki</Button></Link>
      </div>
    );
  }

  const { document } = data;

  return (
    <div className="docs-layout docs-layout--view">
      <WikiNavigation groups={data.navigation} />
      <article className="doc-view">
        <Breadcrumb className="doc-breadcrumb" noTrailingSlash>
          <BreadcrumbItem>
            <Link to="/dev-docs">Wiki</Link>
          </BreadcrumbItem>
          <BreadcrumbItem>
            <Link to={`/dev-docs?category=${document.category}`}>{data.category.label}</Link>
          </BreadcrumbItem>
          <BreadcrumbItem isCurrentPage>{document.title}</BreadcrumbItem>
        </Breadcrumb>

        <header className="doc-view__header">
          <div className="tag-row">
            <Tag type="blue" size="sm">{data.category.label}</Tag>
            <PublishedTag published={document.is_published} />
          </div>
          <h1>{document.title}</h1>
          {document.summary && <p className="doc-view__summary">{document.summary}</p>}
          <div className="doc-view__meta">
            <span>{document.author_name ?? 'Autor removido'}</span>
            <span>Criado em {fullDate(document.created_at)}</span>
            <span>Atualizado {relativeTime(document.updated_at)}</span>
          </div>
          <div className="doc-view__actions">
            <Button kind="ghost" renderIcon={ArrowLeft} onClick={() => navigate('/dev-docs')}>Wiki</Button>
            <Button renderIcon={Edit} onClick={() => navigate(`/dev-docs/${document.id}/edit`)}>Editar</Button>
            <Button kind="danger--ghost" renderIcon={TrashCan} onClick={() => setConfirmOpen(true)}>Excluir</Button>
          </div>
        </header>

        <div className="markdown-body" dangerouslySetInnerHTML={{ __html: data.html }} />

        <footer className="doc-view__footer">
          <Link to={`/dev-docs?category=${document.category}`}>Ver mais em {data.category.label}</Link>
          <Link to="/dev-docs">Voltar para a wiki</Link>
        </footer>
      </article>

      <ConfirmModal
        open={confirmOpen}
        title="Excluir documento"
        message={`Excluir a página “${document.title}”? Essa ação não pode ser desfeita.`}
        confirmLabel="Excluir documento"
        loading={deleting}
        onCancel={() => setConfirmOpen(false)}
        onConfirm={() => void remove()}
      />
      {deleting && <InlineLoading description="Excluindo…" />}
    </div>
  );
}
