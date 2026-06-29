import { useQuery } from '@tanstack/react-query';
import type { ApiResponse, Notification } from '@civicos/types';
import { api } from '../lib/api';

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
