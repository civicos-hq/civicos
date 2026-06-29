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
