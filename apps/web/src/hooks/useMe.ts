import { useQuery } from '@tanstack/react-query';
import type { ApiResponse, User } from '@civicos/types';
import { api } from '../lib/api';

export function useMe() {
  return useQuery({
    queryKey: ['me'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ user: User }>>('/api/v1/auth/me');
      return res.data.data.user;
    },
  });
}

/**
 * useMe for pages a signed-out visitor can reach.
 *
 * Any 401 in the app triggers a refresh attempt and, failing that, a hard
 * redirect to /login (`lib/api.ts` → `forceSignOut`). `/auth/me` 401s for an
 * anonymous visitor, so calling `useMe()` on a public page bounces the very
 * people it is public for — including anyone following a shared campaign link,
 * which is the last page that should demand a session.
 *
 * Gating on the presence of a token means an anonymous visitor never makes
 * the request at all. `undefined` means "not signed in", and callers should
 * treat it that way rather than as "still loading".
 */
export function useOptionalMe() {
  const hasToken =
    typeof window !== 'undefined' && Boolean(window.localStorage.getItem('accessToken'));
  return useQuery({
    queryKey: ['me'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ user: User }>>('/api/v1/auth/me');
      return res.data.data.user;
    },
    enabled: hasToken,
  });
}
