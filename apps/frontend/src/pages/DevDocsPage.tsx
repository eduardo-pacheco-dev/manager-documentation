import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import {
  Add,
  ArrowRight,
  Book,
  Edit,
  Search as SearchIcon,
  TrashCan,
} from '@carbon/icons-react';
import {
  Button,
  ClickableTile,
  Pagination,
  Search,
  Select,
  SelectItem,
  Tag,
} from '@carbon/react';
import { PageHeader } from '../components/PageHeader';
import { EmptyState, Loading, PublishedTag } from '../components/Feedback';
import { ConfirmModal } from '../components/ConfirmModal';
import { WikiNavigation } from '../components/WikiNavigation';
import { useToast } from '../contexts/ToastContext';
import { api } from '../lib/api';
import { errorMessage, formatNumber, relativeTime, truncate } from '../lib/format';
import type { DocsResponse, DocSummary } from '../lib/types';

export function DevDocsPage() {
  const navigate = useNavigate();
  const toast = useToast();
  const [searchParams] = useSearchParams();
  const categoryParam = searchParams.get('category') ?? '';
  const [data, setData] = useState<DocsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [category, setCategory] = useState(categoryParam);
  const [page, setPage] = useState(1);
  const [confirm, setConfirm] = useState<DocSummary | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    setCategory(categoryParam);
    setPage(1);
  }, [categoryParam]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedSearch(search.trim());
      setPage(1);
    }, 400);
    return () => window.clearTimeout(timer);
  }, [search]);

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({ page: String(page) });
    if (debouncedSearch) params.set('search', debouncedSearch);
    if (category) params.set('category', category);
    try {
      const response = await api.get<DocsResponse>(`/dev-docs?${params.toString()}`);
      setData(response);
    } catch (error) {
      toast.error('Não foi possível carregar a documentação', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [page, debouncedSearch, category, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  async function removeDoc() {
    if (!confirm) return;
    setDeleting(true);
    try {
      await api.delete(`/dev-docs/${confirm.id}`);
      toast.success('Documento excluído com sucesso');
      setConfirm(null);
      await load();
    } catch (error) {
      toast.error('Não foi possível excluir', errorMessage(error));
    } finally {
      setDeleting(false);
    }
  }

  const navigation = data?.navigation ?? [];
  const categories = data?.categories ?? [];
  const counts = new Map<string, number>();
  navigation.forEach((group) => counts.set(group.category, group.items.length));
  const hasFilters = Boolean(debouncedSearch || category);
  const docs = data?.data ?? [];

  return (
    <div className="page">
      <PageHeader
        eyebrow="Administração"
        title="Documentação Dev"
        description="Wiki técnica em Markdown com categorias, rascunhos e busca."
        actions={<Button renderIcon={Add} onClick={() => navigate('/dev-docs/new')}>Novo documento</Button>}
      />

      {data && (
        <div className="docs-stats">
          <div><strong>{formatNumber(data.stats.total)}</strong><span>documentos</span></div>
          <div><strong>{formatNumber(data.stats.published)}</strong><span>publicados</span></div>
          <div><strong>{formatNumber(data.stats.drafts)}</strong><span>rascunhos</span></div>
          <div><strong>{formatNumber(data.stats.categories)}</strong><span>categorias</span></div>
        </div>
      )}

      <div className="docs-layout">
        <WikiNavigation groups={navigation} />
        <div className="docs-main">
          <div className="docs-toolbar">
            <Search
              size="lg"
              labelText="Buscar documentos"
              placeholder="Buscar por título, resumo ou conteúdo"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              onClear={() => setSearch('')}
            />
            <Select
              id="category-filter"
              labelText="Categoria"
              value={category}
              size="sm"
              onChange={(event) => {
                setCategory(event.target.value);
                setPage(1);
              }}
            >
              <SelectItem value="" text="Todas as categorias" />
              {categories.map((item) => (
                <SelectItem key={item.key} value={item.key} text={item.label} />
              ))}
            </Select>
          </div>

          {!hasFilters && categories.length > 0 && (
            <div className="category-grid">
              {categories.map((item) => (
                <ClickableTile
                  key={item.key}
                  className="category-card"
                  onClick={() => {
                    setCategory(item.key);
                    setPage(1);
                  }}
                >
                  <div className="category-card__icon"><Book size={20} /></div>
                  <strong>{item.label}</strong>
                  <span>{counts.get(item.key) ?? 0} {counts.get(item.key) === 1 ? 'documento' : 'documentos'}</span>
                  <ArrowRight size={16} className="category-card__arrow" />
                </ClickableTile>
              ))}
            </div>
          )}

          {loading ? (
            <Loading description="Carregando documentos…" />
          ) : docs.length === 0 ? (
            <EmptyState
              icon={SearchIcon}
              title={hasFilters ? 'Nenhuma página encontrada' : 'Nenhuma página cadastrada'}
              description={hasFilters ? 'Tente outra busca ou remova os filtros.' : 'Crie o primeiro documento da wiki.'}
              action={<Button renderIcon={Add} onClick={() => navigate('/dev-docs/new')}>Novo documento</Button>}
            />
          ) : (
            <div className="doc-list">
              {docs.map((doc) => (
                <article key={doc.id} className="doc-card">
                  <div className="doc-card__head">
                    <div className="tag-row">
                      <Tag type="blue" size="sm">{categories.find((item) => item.key === doc.category)?.label ?? doc.category}</Tag>
                      <PublishedTag published={doc.is_published} />
                    </div>
                    <div className="doc-card__actions">
                      <Button
                        kind="ghost"
                        size="sm"
                        hasIconOnly
                        renderIcon={Edit}
                        iconDescription="Editar"
                        onClick={() => navigate(`/dev-docs/${doc.id}/edit`)}
                      />
                      <Button
                        kind="danger--ghost"
                        size="sm"
                        hasIconOnly
                        renderIcon={TrashCan}
                        iconDescription="Excluir"
                        onClick={() => setConfirm(doc)}
                      />
                    </div>
                  </div>
                  <Link to={`/dev-docs/${doc.id}`} className="doc-card__title">{doc.title}</Link>
                  {doc.summary && <p className="doc-card__summary">{truncate(doc.summary, 240)}</p>}
                  <div className="doc-card__meta">
                    <span>{doc.author_name ?? 'Autor removido'}</span>
                    <span>Atualizado {relativeTime(doc.updated_at)}</span>
                  </div>
                </article>
              ))}
            </div>
          )}

          {data && data.meta.total > 0 && (
            <Pagination
              page={page}
              pageSize={12}
              pageSizes={[12]}
              totalItems={data.meta.total}
              onChange={({ page: nextPage }) => setPage(nextPage)}
              itemRangeText={(min, max, total) => `${min}–${max} de ${total} documentos`}
              itemsPerPageText="Itens por página"
              pageRangeText={(_current, total) => `de ${total} página(s)`}
              pageSelectLabelText={(total) => `Página de ${total}`}
              pageText={(current) => `Página ${current}`}
              backwardText="Página anterior"
              forwardText="Próxima página"
            />
          )}
        </div>
      </div>

      <ConfirmModal
        open={Boolean(confirm)}
        title="Excluir documento"
        message={`Excluir a página “${confirm?.title ?? ''}”? Essa ação não pode ser desfeita.`}
        confirmLabel="Excluir documento"
        loading={deleting}
        onCancel={() => setConfirm(null)}
        onConfirm={() => void removeDoc()}
      />
    </div>
  );
}
