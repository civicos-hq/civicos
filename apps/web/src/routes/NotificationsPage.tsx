import { Link } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Button } from '@civicos/ui';
import { NotificationType, type Notification } from '@civicos/types';
import { api } from '../lib/api';
import { useNotifications } from '../hooks/useNotifications';

const TYPE_LABEL: Record<NotificationType, string> = {
  [NotificationType.ISSUE_UPDATE]: 'Issue update',
  [NotificationType.PETITION_UPDATE]: 'Petition update',
  [NotificationType.REPRESENTATIVE_RESPONSE]: 'Representative response',
  [NotificationType.COMMUNITY_UPDATE]: 'Community update',
  [NotificationType.SYSTEM]: 'System',
};

const TYPE_TONE: Record<NotificationType, string> = {
  [NotificationType.ISSUE_UPDATE]: 'bg-rose-100 text-rose-700',
  [NotificationType.PETITION_UPDATE]: 'bg-civic-100 text-civic-700',
  [NotificationType.REPRESENTATIVE_RESPONSE]: 'bg-amber-100 text-amber-700',
  [NotificationType.COMMUNITY_UPDATE]: 'bg-sky-100 text-sky-700',
  [NotificationType.SYSTEM]: 'bg-slate-200 text-slate-700',
};

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const minutes = Math.round(diff / 60_000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  if (days < 30) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}

export function NotificationsPage() {
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
      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
              Notification Center
            </p>
            <h1 className="mt-2 text-3xl font-semibold text-slate-900">Civic activity feed</h1>
            <p className="mt-1 text-sm text-slate-500">
              {unreadCount > 0 ? `${unreadCount} unread` : 'All caught up.'}
            </p>
          </div>
          <Button
            size="sm"
            variant="secondary"
            disabled={unreadCount === 0}
            loading={markAll.isPending}
            onClick={() => markAll.mutate()}
          >
            Mark all as read
          </Button>
        </div>
      </header>

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">Recent</h2>
        {notificationsQuery.isLoading ? (
          <p className="mt-4 text-sm text-slate-500">Loading…</p>
        ) : notifications.length === 0 ? (
          <p className="mt-4 text-sm text-slate-500">
            You don't have any notifications yet. They'll appear here when activity reaches you.
          </p>
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
  const content = (
    <article
      className={`flex flex-wrap items-start justify-between gap-3 rounded-xl border p-4 ${notification.read ? 'border-slate-200 bg-slate-50/40' : 'border-civic-200 bg-white'}`}
    >
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span
            className={`rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide ${TYPE_TONE[notification.type]}`}
          >
            {TYPE_LABEL[notification.type]}
          </span>
          {!notification.read && (
            <span className="h-2 w-2 rounded-full bg-civic-600" aria-label="Unread" />
          )}
          <span className="ml-auto text-xs font-medium text-slate-500">
            {timeAgo(notification.createdAt)}
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
