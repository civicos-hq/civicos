import { useQuery } from '@tanstack/react-query';
import type { ApiResponse } from '@civicos/types';
import { api } from '../lib/api';

export function useFollowedReps() {
  return useQuery({
    queryKey: ['followedReps'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ representativeIds: string[] }>>(
        '/api/v1/me/follows/representatives',
      );
      return new Set(res.data.data.representativeIds);
    },
  });
}
