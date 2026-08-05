import { useQuery } from '@tanstack/react-query';
import type { ApiResponse } from '@civicos/types';
import { api } from '../lib/api';

/**
 * The unauthenticated activity ticker on the marketing homepage.
 *
 * Every other feed on CivicOS is signed-in and personalised. This one is read
 * by visitors who have no account, so the server returns only what is already
 * public — kind, title, status, state/LGA, timestamp. No author names, no
 * bodies, nothing in draft or under review.
 */

export type PublicActivityKind =
  'issue' | 'petition' | 'consultation' | 'announcement' | 'campaign' | 'repAnnouncement';

export interface PublicActivityItem {
  kind: PublicActivityKind;
  title: string;
  status?: string;
  state?: string;
  lga?: string;
  at: string;
}

/**
 * `refetchInterval` is what makes the panel's "Live" label true. Without it the
 * ticker would rotate through a snapshot taken at page load and still claim to
 * be live — which is the exact dishonesty this endpoint exists to remove.
 *
 * No auth header is required, and none of the app's 401 redirect behaviour can
 * fire from here, so this is safe to call on a page signed-out visitors see.
 */
export function usePublicActivity(limit = 12) {
  return useQuery({
    queryKey: ['public-activity', limit],
    staleTime: 30_000,
    refetchInterval: 60_000,
    retry: false,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ activity: PublicActivityItem[] }>>(
        `/api/v1/discover/public-activity?limit=${limit}`,
      );
      return res.data.data.activity ?? [];
    },
  });
}
