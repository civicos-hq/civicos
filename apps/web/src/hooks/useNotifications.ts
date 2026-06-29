import { useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { ApiResponse, Notification } from '@civicos/types';
import { api } from '../lib/api';

const apiBase = import.meta.env.VITE_API_URL ?? 'http://localhost:3000';

export function useNotifications() {
  return useQuery({
    queryKey: ['notifications'],
    queryFn: async () => {
      const res =
        await api.get<ApiResponse<{ notifications: Notification[] }>>('/api/v1/notifications');
      return res.data.data.notifications;
    },
  });
}

export function useUnreadCount() {
  return useQuery({
    queryKey: ['notifications', 'unread-count'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ count: number }>>(
        '/api/v1/notifications/unread-count',
      );
      return res.data.data.count;
    },
    refetchInterval: 30_000,
  });
}

// useNotificationStream opens a Server-Sent Events connection to the gateway
// and invalidates the notification queries whenever a new event arrives, so
// the list and unread badge refresh without polling.
//
// Mount it once at a layout level — multiple concurrent subscriptions per tab
// would just open extra EventSources for no benefit.
export function useNotificationStream() {
  const qc = useQueryClient();

  useEffect(() => {
    const token = localStorage.getItem('accessToken');
    if (!token) return;

    const url = `${apiBase}/api/v1/notifications/stream?access_token=${encodeURIComponent(token)}`;
    const es = new EventSource(url);

    const refresh = () => {
      qc.invalidateQueries({ queryKey: ['notifications'] });
    };

    es.addEventListener('notification', refresh);
    es.onerror = () => {
      // EventSource auto-reconnects with exponential backoff; nothing to do.
    };

    return () => {
      es.removeEventListener('notification', refresh);
      es.close();
    };
  }, [qc]);
}
