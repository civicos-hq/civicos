import { useEffect, useState } from 'react';
import { useQuery, keepPreviousData } from '@tanstack/react-query';
import type {
  Announcement,
  ApiResponse,
  Community,
  Consultation,
  Issue,
  Organization,
  Petition,
  Project,
  Representative,
} from '@civicos/types';
import { api } from '../lib/api';

/**
 * A campaign as search returns it — enough to decide whether this is the one
 * you were looking for. The full shape lives on the campaign page.
 */
export interface SearchCampaign {
  id: string;
  slug: string;
  title: string;
  summary: string;
  category: string;
  status: string;
  currency: string;
  goalMinor: number;
  raisedMinor: number;
  isEmergency: boolean;
  state?: string | null;
  lga?: string | null;
}

export interface SearchResult {
  /**
   * Communities are the one result kind that is an entry point rather
   * than a destination: someone searching "University of Abuja" is
   * almost always trying to join it, not to read something inside it.
   * They lead the results for that reason.
   */
  communities: Community[];
  issues: Issue[];
  petitions: Petition[];
  representatives: Representative[];
  organizations: Organization[];
  consultations: Consultation[];
  announcements: Announcement[];
  projects: Project[];
  campaigns: SearchCampaign[];
  repAnnouncements: SearchRepAnnouncement[];
}

/** A representative's public statement, as search returns it. */
export interface SearchRepAnnouncement {
  id: string;
  representativeId: string;
  representativeName: string;
  title: string;
  body: string;
  communityId: string;
  commentCount: number;
  publishedAt?: string | null;
}

const empty: SearchResult = {
  communities: [],
  issues: [],
  petitions: [],
  representatives: [],
  organizations: [],
  consultations: [],
  announcements: [],
  projects: [],
  campaigns: [],
  repAnnouncements: [],
};

// useDebouncedValue returns `value` after `delay` ms of stillness. Avoids
// hammering the API on every keystroke while the user is still typing.
function useDebouncedValue<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(t);
  }, [value, delay]);
  return debounced;
}

export function useSearch(query: string) {
  const q = useDebouncedValue(query.trim(), 200);
  const enabled = q.length >= 2;

  const queryResult = useQuery({
    queryKey: ['search', q],
    queryFn: async (): Promise<SearchResult> => {
      const res = await api.get<ApiResponse<SearchResult>>('/api/v1/search', {
        params: { q },
      });
      return res.data.data;
    },
    enabled,
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  });

  return {
    ...queryResult,
    debouncedQuery: q,
    data: queryResult.data ?? empty,
    enabled,
  };
}
