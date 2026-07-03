import { useQuery } from '@tanstack/react-query';
import type { ApiResponse } from '@civicos/types';
import { api } from '../lib/api';

/**
 * Returns the set of issue IDs the current user has an active upvote on.
 * Used by IssueDetailPage (and any future list card) to render the button in
 * the correct "already upvoted" state instead of relying on client-only
 * optimistic state that resets on refresh.
 *
 * The endpoint is gated by JWT, so callers should skip using this hook when
 * `me` is not yet loaded — anonymous callers get an empty Set anyway.
 */
export function useUpvotedIssues() {
  return useQuery({
    queryKey: ['upvotedIssues'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ issueIds: string[] }>>('/api/v1/me/upvotes/issues');
      return new Set(res.data.data.issueIds);
    },
  });
}
