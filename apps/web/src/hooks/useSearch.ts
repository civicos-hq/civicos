import { useEffect, useState } from 'react';
import { useQuery, keepPreviousData } from '@tanstack/react-query';
import type {
  Announcement,
  ApiResponse,
  Consultation,
  Issue,
  Organization,
  Petition,
  Project,
  Representative,
} from '@civicos/types';
import { api } from '../lib/api';

export interface SearchResult {
  issues: Issue[];
  petitions: Petition[];
  representatives: Representative[];
  organizations: Organization[];
  consultations: Consultation[];
  announcements: Announcement[];
  projects: Project[];
}

const empty: SearchResult = {
  issues: [],
  petitions: [],
  representatives: [],
  organizations: [],
  consultations: [],
  announcements: [],
  projects: [],
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
