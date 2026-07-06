import { useQuery } from '@tanstack/react-query';
import type { ApiResponse, Community } from '@civicos/types';
import { api } from '../lib/api';

export function useCommunities() {
  return useQuery({
    queryKey: ['communities'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ communities: Community[] }>>('/api/v1/communities');
      return res.data.data.communities;
    },
  });
}
