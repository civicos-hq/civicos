import { useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { ApiResponse, Notification } from '@civicos/types';
import { API_BASE, api, refreshAccessToken } from '../lib/api';

// Reuse the scheme-resolved base from lib/api — VITE_API_URL is a bare host
// on Render, and a scheme-less EventSource URL resolves relative to the
// static-site origin instead of the gateway.
const apiBase = API_BASE;

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
// Access tokens now live 15 minutes, so any long-lived SSE connection will
// eventually be reopened past its issue time. The browser's built-in
// EventSource reconnect would just re-use the same expired token in the URL
// query param, so we manage reconnect ourselves:
//
//   - When the connection dies quickly after opening (< 5s), we treat it as
//     an auth failure and force a refresh-token rotation before retrying.
//   - Reconnect uses exponential backoff, capped at 30s.
//   - After ~10 consecutive failures we give up passively — the axios
//     interceptor will bounce the user to /login on their next request.
//
// Mount this once at a layout level — multiple concurrent subscriptions per
// tab would just open extra EventSources for no benefit.
export function useNotificationStream() {
  const qc = useQueryClient();

  useEffect(() => {
    let currentES: EventSource | null = null;
    let reconnectTimer: number | null = null;
    let disposed = false;
    let backoff = 1000;
    let consecutiveFailures = 0;
    let openedAt = 0;

    function connect() {
      if (disposed) return;
      const token = localStorage.getItem('accessToken');
      if (!token) return;

      const url = `${apiBase}/api/v1/notifications/stream?access_token=${encodeURIComponent(token)}`;
      openedAt = Date.now();
      const es = new EventSource(url);
      currentES = es;

      es.addEventListener('open', () => {
        // A successful open resets the backoff so a transient blip doesn't
        // penalise the next disconnect.
        backoff = 1000;
        consecutiveFailures = 0;
      });

      es.addEventListener('notification', () => {
        qc.invalidateQueries({ queryKey: ['notifications'] });
      });

      es.onerror = async () => {
        // Close the browser's own retry loop — we're running our own with
        // refresh-aware behaviour.
        es.close();
        currentES = null;
        if (disposed) return;

        consecutiveFailures++;
        const lifetime = Date.now() - openedAt;
        // A connection that dies quickly probably got rejected on auth.
        // Force a rotation before the next attempt so we don't retry with
        // the same expired token in the query param.
        if (lifetime < 5000) {
          await refreshAccessToken();
        }

        // Passive give-up after many failures — the user has probably
        // signed out, lost network, or something worse. The axios
        // interceptor handles the sign-out redirect on next request.
        if (consecutiveFailures > 10) return;

        reconnectTimer = window.setTimeout(connect, backoff);
        backoff = Math.min(backoff * 2, 30_000);
      };
    }

    connect();

    return () => {
      disposed = true;
      if (reconnectTimer !== null) window.clearTimeout(reconnectTimer);
      currentES?.close();
    };
  }, [qc]);
}
