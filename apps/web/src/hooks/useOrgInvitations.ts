import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type {
  ApiResponse,
  OrgInvitation,
  OrgInvitationPreview,
  OrgMember,
  OrgMemberRole,
} from '@civicos/types';
import { api } from '../lib/api';

export function useOrgInvitations(orgId: string | undefined) {
  return useQuery({
    queryKey: ['org-invitations', orgId],
    enabled: Boolean(orgId),
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ invitations: OrgInvitation[] }>>(
        `/api/v1/organizations/${orgId}/invitations`,
      );
      return res.data.data.invitations ?? [];
    },
  });
}

export function useInviteToOrg(orgId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (args: { email: string; role: OrgMemberRole; title?: string }) => {
      const res = await api.post<ApiResponse<{ invitation: OrgInvitation }>>(
        `/api/v1/organizations/${orgId}/invitations`,
        { email: args.email, role: args.role, title: args.title || undefined },
      );
      return res.data.data.invitation;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['org-invitations', orgId] });
    },
  });
}

export function useRevokeInvitation(orgId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (invitationId: string) => {
      await api.delete(`/api/v1/invitations/${invitationId}`);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['org-invitations', orgId] });
    },
  });
}

/**
 * Reads an invitation from its token. Unauthenticated on purpose — the
 * invitee needs to see what they have been asked to join before deciding
 * whether to create an account.
 *
 * `retry: false` because every failure here is terminal: the link is
 * expired, used or withdrawn, and the server returns one indistinguishable
 * error for all three. Retrying just delays the message.
 */
export function useInvitationPreview(token: string | undefined) {
  return useQuery({
    queryKey: ['invitation', token],
    enabled: Boolean(token),
    retry: false,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ invitation: OrgInvitationPreview }>>(
        `/api/v1/invitations/${encodeURIComponent(token!)}`,
      );
      return res.data.data.invitation;
    },
  });
}

export function useAcceptInvitation(token: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await api.post<ApiResponse<{ member: OrgMember }>>(
        `/api/v1/invitations/${encodeURIComponent(token!)}/accept`,
      );
      return res.data.data.member;
    },
    onSuccess: () => {
      // The caller now belongs to an org they did not before — this is what
      // makes the sidebar entry and the /org redirect appear.
      void queryClient.invalidateQueries({ queryKey: ['my-organizations'] });
    },
  });
}
