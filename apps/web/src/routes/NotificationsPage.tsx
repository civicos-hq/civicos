import { Link } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Button } from '@civicos/ui';
import { NotificationType, type Notification } from '@civicos/types';
import { api } from '../lib/api';
import { useNotifications } from '../hooks/useNotifications';
import { useEnumLabels } from '../hooks/useEnumLabels';
import { useRelativeTime } from '../hooks/useRelativeTime';
import { PageHeader, useTodayMeta } from '../components/PageHeader';
import { EmptyState } from '../components/EmptyState';
import { Bell } from 'lucide-react';

const TYPE_TONE: Record<NotificationType, string> = {
  [NotificationType.ISSUE_UPDATE]: 'bg-rose-100 text-rose-700',
  [NotificationType.PETITION_UPDATE]: 'bg-civic-100 text-civic-700',
  [NotificationType.REPRESENTATIVE_RESPONSE]: 'bg-amber-100 text-amber-700',
  [NotificationType.COMMUNITY_UPDATE]: 'bg-sky-100 text-sky-700',
  [NotificationType.CONSULTATION_UPDATE]: 'bg-emerald-100 text-emerald-700',
  [NotificationType.SYSTEM]: 'bg-slate-200 text-slate-700',
};

export function NotificationsPage() {
  const { t } = useTranslation();
  const meta = useTodayMeta();
  const queryClient = useQueryClient();
  const notificationsQuery = useNotifications();

  const markRead = useMutation({
    mutationFn: async (id: string) => {
      await api.patch(`/api/v1/notifications/${id}/read`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  const markAll = useMutation({
    mutationFn: async () => {
      await api.post('/api/v1/notifications/read-all');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  const notifications = notificationsQuery.data ?? [];
  const unreadCount = notifications.filter((n) => !n.read).length;

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow={t('notificationsPage.eyebrow')}
        title={t('notificationsPage.title')}
        subtitle={
          unreadCount > 0
            ? t('notificationsPage.unreadCount', { count: unreadCount })
            : t('notificationsPage.allCaughtUp')
        }
        meta={meta}
        actions={
          <Button
            size="sm"
            variant="secondary"
            disabled={unreadCount === 0}
            loading={markAll.isPending}
            onClick={() => markAll.mutate()}
          >
            {t('notificationsPage.markAllRead')}
          </Button>
        }
      />

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">{t('notificationsPage.recent')}</h2>
        {notificationsQuery.isLoading ? (
          <p className="mt-4 text-sm text-slate-600">{t('common.loading')}</p>
        ) : notifications.length === 0 ? (
          <div className="mt-4">
            <EmptyState icon={<Bell className="h-5 w-5" />} title={t('notificationsPage.empty')} />
          </div>
        ) : (
          <div className="mt-4 grid gap-3">
            {notifications.map((n) => (
              <NotificationRow
                key={n.id}
                notification={n}
                onClick={() => !n.read && markRead.mutate(n.id)}
              />
            ))}
          </div>
        )}
      </section>
    </section>
  );
}

function NotificationRow({
  notification,
  onClick,
}: {
  notification: Notification;
  onClick: () => void;
}) {
  const { t } = useTranslation();
  const enums = useEnumLabels();
  const relative = useRelativeTime();

  const content = (
    <article
      className={`flex flex-wrap items-start justify-between gap-3 rounded-xl border p-4 ${notification.read ? 'border-slate-200 bg-slate-50/40' : 'border-civic-200 bg-white'}`}
    >
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span
            className={`rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide ${TYPE_TONE[notification.type]}`}
          >
            {enums.notificationType(notification.type)}
          </span>
          {!notification.read && (
            <span
              className="h-2 w-2 rounded-full bg-civic-600"
              aria-label={t('notificationsPage.unread')}
            />
          )}
          <span className="ml-auto text-xs font-medium text-slate-600">
            {relative(notification.createdAt)}
          </span>
        </div>
        <h3 className="mt-2 font-semibold text-slate-900">{notification.title}</h3>
        <p className="mt-1 text-sm text-slate-600">{notification.body}</p>
      </div>
    </article>
  );

  if (notification.linkUrl) {
    return (
      <Link to={notification.linkUrl} onClick={onClick} className="block">
        {content}
      </Link>
    );
  }
  return (
    <button type="button" onClick={onClick} className="block w-full text-left">
      {content}
    </button>
  );
}
