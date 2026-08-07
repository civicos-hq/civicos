import { keepPreviousData, useQuery } from '@tanstack/react-query';
import type { ApiResponse, Community } from '@civicos/types';
import { api } from '../lib/api';

/** Matches the server's MaxPageSize in communities/service.go. */
export const MAX_COMMUNITY_PAGE = 100;

export interface CommunityQuery {
  /** Free-text search over name, description, LGA and state. */
  q?: string;
  state?: string;
  lga?: string;
  limit?: number;
  offset?: number;
  enabled?: boolean;
}

export interface CommunityPage {
  communities: Community[];
  total: number;
  limit: number;
  offset: number;
}

async function fetchCommunities(params: Record<string, string | number>): Promise<CommunityPage> {
  const res = await api.get<ApiResponse<CommunityPage>>('/api/v1/communities', { params });
  const data = res.data.data;
  return {
    communities: data.communities ?? [],
    total: data.total ?? 0,
    limit: data.limit ?? 0,
    offset: data.offset ?? 0,
  };
}

/**
 * One page of communities, filtered and searched server-side.
 *
 * The endpoint is paginated, so this no longer returns "every community" —
 * callers that need a specific set (the ones a user has joined) must ask
 * for them by ID via useCommunitiesByID rather than filtering a page.
 */
export function useCommunities({
  q,
  state,
  lga,
  limit,
  offset,
  enabled = true,
}: CommunityQuery = {}) {
  const params: Record<string, string | number> = {};
  if (q?.trim()) params.q = q.trim();
  if (state) params.state = state;
  if (lga) params.lga = lga;
  // Explicit undefined check, not a truthiness one: limit is optional but
  // a caller passing 0 should not be silently upgraded to the default.
  if (limit !== undefined) params.limit = limit;
  if (offset) params.offset = offset;

  return useQuery({
    queryKey: [
      'communities',
      { q: q?.trim() ?? '', state: state ?? '', lga: lga ?? '', limit, offset },
    ],
    enabled,
    // Keeps the previous page on screen while the next one loads, so
    // typing in the search box doesn't flash an empty list on every
    // keystroke.
    placeholderData: keepPreviousData,
    queryFn: () => fetchCommunities(params),
  });
}

/**
 * Resolves a known set of communities by ID.
 *
 * Membership gives us IDs, not communities, and a joined community is not
 * necessarily on whatever page of search results is currently displayed —
 * so it has to be fetched explicitly rather than filtered out of a page.
 */
export function useCommunitiesByID(ids: string[]) {
  // Sorted so the cache key is stable regardless of membership order.
  const sorted = [...ids].sort();
  return useQuery({
    queryKey: ['communities', 'byId', sorted],
    enabled: sorted.length > 0,
    queryFn: () => fetchCommunities({ ids: sorted.join(','), limit: MAX_COMMUNITY_PAGE }),
    select: (page: CommunityPage) => page.communities,
  });
}

/**
 * A single community by ID, for the several places that only need to name
 * the one a user is currently in. Previously these each pulled the entire
 * community table and ran `.find()` over it.
 */
export function useCommunity(id: string | undefined | null) {
  const query = useCommunitiesByID(id ? [id] : []);
  return { ...query, data: query.data?.[0] };
}

/**
 * Every community, paged through to completion.
 *
 * Only for the org-side `<select>` pickers that genuinely enumerate the
 * whole list. Prefer useCommunities (searchable, paged) anywhere a person
 * is choosing from more than a screenful — which is why the citizen
 * surfaces no longer use this.
 */
export function useAllCommunities() {
  return useQuery({
    queryKey: ['communities', 'all'],
    queryFn: async () => {
      const all: Community[] = [];
      for (let offset = 0; ; offset += MAX_COMMUNITY_PAGE) {
        const page = await fetchCommunities({ limit: MAX_COMMUNITY_PAGE, offset });
        all.push(...page.communities);
        if (page.communities.length < MAX_COMMUNITY_PAGE || all.length >= page.total) break;
      }
      return all;
    },
  });
}
