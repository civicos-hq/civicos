import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ApiResponse, OrgMember, OrgMemberRole } from '@civicos/types';
import { api } from '../lib/api';

export function useOrgMembers(orgId: string | undefined) {
  return useQuery({
    queryKey: ['org-members', orgId],
    enabled: Boolean(orgId),
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ members: OrgMember[] }>>(
        `/api/v1/organizations/${orgId}/members`,
      );
      return res.data.data.members ?? [];
    },
  });
}

export interface AddMemberArgs {
  /** The normal path — an org owner knows a colleague's email, not their ID. */
  email: string;
  role: OrgMemberRole;
  title?: string;
}

export function useAddOrgMember(orgId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (args: AddMemberArgs) => {
      const res = await api.post<ApiResponse<{ member: OrgMember }>>(
        `/api/v1/organizations/${orgId}/members`,
        { email: args.email, role: args.role, title: args.title || undefined },
      );
      return res.data.data.member;
    },
    onSuccess: () => invalidate(queryClient, orgId),
  });
}

export function useUpdateOrgMember(orgId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (args: { userId: string; role: OrgMemberRole; title?: string }) => {
      await api.patch(`/api/v1/organizations/${orgId}/members/${args.userId}`, {
        role: args.role,
        ...(args.title === undefined ? {} : { title: args.title }),
      });
    },
    onSuccess: () => invalidate(queryClient, orgId),
  });
}

export function useRemoveOrgMember(orgId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (userId: string) => {
      await api.delete(`/api/v1/organizations/${orgId}/members/${userId}`);
    },
    onSuccess: () => invalidate(queryClient, orgId),
  });
}

function invalidate(queryClient: ReturnType<typeof useQueryClient>, orgId: string | undefined) {
  void queryClient.invalidateQueries({ queryKey: ['org-members', orgId] });
  // The member count on the org record moved too.
  void queryClient.invalidateQueries({ queryKey: ['my-organizations'] });
}
