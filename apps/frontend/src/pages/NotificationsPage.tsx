import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Checkmark, Notification as Bell, TrashCan } from '@carbon/icons-react';
import { Button, Pagination, Tag } from '@carbon/react';
import { PageHeader } from '../components/PageHeader';
import { EmptyState, Loading, NotificationIcon } from '../components/Feedback';
import { ConfirmModal } from '../components/ConfirmModal';
import { useToast } from '../contexts/ToastContext';
import { api } from '../lib/api';
import { errorMessage, relativeTime } from '../lib/format';
import type { AppNotification, NotificationsResponse } from '../lib/types';

const LEVEL_LABEL: Record<string, string> = {
  info: 'Informação',
  success: 'Sucesso',
  warning: 'Atenção',
  danger: 'Erro',
};

function NotificationLink({ notification }: { notification: AppNotification }) {
  if (!notification.url) return null;
  if (notification.url.startsWith('/')) {
    return <Link to={notification.url} className="notification-open">Abrir →</Link>;
  }
  return <a href={notification.url} target="_blank" rel="noreferrer" className="notification-open">Abrir →</a>;
}

export function NotificationsPage() {
  const toast = useToast();
  const [data, setData] = useState<NotificationsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [confirm, setConfirm] = useState<'clear' | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.get<NotificationsResponse>(`/notifications?page=${page}`);
      setData(response);
    } catch (error) {
      toast.error('Não foi possível carregar as notificações', errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [page, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  function notifyChanged() {
    window.dispatchEvent(new Event('notifications:changed'));
  }

  async function markRead(id: string) {
    try {
      await api.patch(`/notifications/${id}/read`);
      await load();
      notifyChanged();
    } catch (error) {
      toast.error('Não foi possível marcar como lida', errorMessage(error));
    }
  }

  async function markAllRead() {
    setBusy(true);
    try {
      await api.patch('/notifications/read-all');
      toast.success('Todas as notificações foram marcadas como lidas');
      await load();
      notifyChanged();
    } catch (error) {
      toast.error('Não foi possível atualizar as notificações', errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  async function remove(id: string) {
    try {
      await api.delete(`/notifications/${id}`);
      toast.success('Notificação removida');
      await load();
      notifyChanged();
    } catch (error) {
      toast.error('Não foi possível remover a notificação', errorMessage(error));
    }
  }

  async function clearAll() {
    setBusy(true);
    try {
      await api.delete('/notifications');
      setConfirm(null);
      setPage(1);
      toast.success('Todas as notificações foram removidas');
      await load();
      notifyChanged();
    } catch (error) {
      toast.error('Não foi possível limpar as notificações', errorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  const items = data?.data ?? [];
  const unread = data?.unread ?? 0;

  return (
    <div className="page">
      <PageHeader
        eyebrow="Conta"
        title="Notificações"
        description={unread > 0 ? `${unread} notificação(ões) não lida(s).` : 'Você está em dia com as notificações.'}
        actions={
          <>
            {unread > 0 && (
              <Button kind="tertiary" renderIcon={Checkmark} onClick={() => void markAllRead()} disabled={busy}>
                Marcar todas como lidas
              </Button>
            )}
            {items.length > 0 && (
              <Button kind="danger--ghost" renderIcon={TrashCan} onClick={() => setConfirm('clear')} disabled={busy}>
                Limpar tudo
              </Button>
            )}
          </>
        }
      />

      {loading ? (
        <Loading description="Carregando notificações…" />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Bell}
          title="Nenhuma notificação"
          description="Quando o sistema enviar um aviso, ele aparecerá aqui."
        />
      ) : (
        <div className="notification-list">
          {items.map((notification) => (
            <article
              key={notification.id}
              className={`notification-item ${notification.read_at ? '' : 'notification-item--unread'}`}
            >
              <div className={`notification-item__tile notification-item__tile--${notification.level}`}>
                <NotificationIcon level={notification.level} />
              </div>
              <div className="notification-item__body">
                <div className="notification-item__head">
                  <div className="tag-row">
                    <Tag type={notification.level === 'danger' ? 'red' : notification.level === 'warning' ? 'magenta' : notification.level === 'success' ? 'green' : 'blue'} size="sm">
                      {LEVEL_LABEL[notification.level] ?? notification.level}
                    </Tag>
                    {!notification.read_at && <Tag type="cool-gray" size="sm">Nova</Tag>}
                  </div>
                  <span className="notification-item__time">{relativeTime(notification.created_at)}</span>
                </div>
                <h2>{notification.title}</h2>
                <p>{notification.body}</p>
                <div className="notification-item__actions">
                  <NotificationLink notification={notification} />
                  {!notification.read_at && (
                    <Button kind="ghost" size="sm" renderIcon={Checkmark} onClick={() => void markRead(notification.id)}>
                      Marcar como lida
                    </Button>
                  )}
                  <Button kind="danger--ghost" size="sm" renderIcon={TrashCan} onClick={() => void remove(notification.id)}>
                    Remover
                  </Button>
                </div>
              </div>
            </article>
          ))}
        </div>
      )}

      {data && data.meta.total > 0 && (
        <Pagination
          page={page}
          pageSize={20}
          pageSizes={[20]}
          totalItems={data.meta.total}
          onChange={({ page: nextPage }) => setPage(nextPage)}
          itemRangeText={(min, max, total) => `${min}–${max} de ${total} notificações`}
          itemsPerPageText="Itens por página"
          pageRangeText={(_current, total) => `de ${total} página(s)`}
          pageSelectLabelText={(total) => `Página de ${total}`}
          pageText={(current) => `Página ${current}`}
          backwardText="Página anterior"
          forwardText="Próxima página"
        />
      )}

      <ConfirmModal
        open={confirm === 'clear'}
        title="Limpar notificações"
        message="Remover todas as notificações? Esta ação não pode ser desfeita."
        confirmLabel="Remover todas"
        loading={busy}
        onCancel={() => setConfirm(null)}
        onConfirm={() => void clearAll()}
      />
    </div>
  );
}
