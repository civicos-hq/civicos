import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ApiResponse } from '@civicos/types';
import { api } from '../lib/api';

/**
 * A representative's announcements — them speaking to their constituents,
 * rather than replying inside someone else's thread.
 */

export type RepAnnouncementStatus = 'DRAFT' | 'PUBLISHED' | 'ARCHIVED';

export interface RepAnnouncement {
  id: string;
  representativeId: string;
  communityId: string;
  title: string;
  body: string;
  status: RepAnnouncementStatus;
  publishedAt?: string | null;
  authorId: string;
  authorName: string;
  commentCount: number;
  isHidden: boolean;
  createdAt: string;
  updatedAt: string;
}

/** Public list — published only. No account needed. */
export function useRepAnnouncements(repId: string | undefined) {
  return useQuery({
    queryKey: ['rep-announcements', repId],
    enabled: Boolean(repId),
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ announcements: RepAnnouncement[] }>>(
        `/api/v1/representatives/${repId}/announcements`,
      );
      return res.data.data.announcements ?? [];
    },
  });
}

/**
 * The owner's view: drafts and archived included.
 *
 * Doubles as the ownership check. The server is the only thing that knows who
 * a profile belongs to, and it answers 403 for anyone else — so rather than
 * teaching the client a second copy of that rule, we ask. A failed query means
 * "not yours", and the authoring UI simply does not render.
 *
 * `enabled` should gate this to accounts that could plausibly own a profile,
 * so ordinary citizens never generate a pointless 403.
 */
export function useRepAnnouncementsManage(repId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ['rep-announcements-manage', repId],
    enabled: Boolean(repId) && enabled,
    retry: false, // a 403 is an answer, not a failure worth retrying
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ announcements: RepAnnouncement[] }>>(
        `/api/v1/representatives/${repId}/announcements/manage`,
      );
      return res.data.data.announcements ?? [];
    },
  });
}

function useInvalidate(repId: string | undefined) {
  const qc = useQueryClient();
  return () => {
    qc.invalidateQueries({ queryKey: ['rep-announcements', repId] });
    qc.invalidateQueries({ queryKey: ['rep-announcements-manage', repId] });
  };
}

export function useCreateRepAnnouncement(repId: string | undefined) {
  const invalidate = useInvalidate(repId);
  return useMutation({
    mutationFn: async (input: { title: string; body: string }) => {
      const res = await api.post<ApiResponse<{ announcement: RepAnnouncement }>>(
        `/api/v1/representatives/${repId}/announcements`,
        input,
      );
      return res.data.data.announcement;
    },
    onSuccess: invalidate,
  });
}

export function useUpdateRepAnnouncement(repId: string | undefined) {
  const invalidate = useInvalidate(repId);
  return useMutation({
    mutationFn: async ({ id, ...input }: { id: string; title?: string; body?: string }) => {
      const res = await api.patch<ApiResponse<{ announcement: RepAnnouncement }>>(
        `/api/v1/representatives/${repId}/announcements/${id}`,
        input,
      );
      return res.data.data.announcement;
    },
    onSuccess: invalidate,
  });
}

/** Publishing notifies every follower, so it is always an explicit action. */
export function usePublishRepAnnouncement(repId: string | undefined) {
  const invalidate = useInvalidate(repId);
  return useMutation({
    mutationFn: (id: string) =>
      api.post(`/api/v1/representatives/${repId}/announcements/${id}/publish`),
    onSuccess: invalidate,
  });
}

export function useArchiveRepAnnouncement(repId: string | undefined) {
  const invalidate = useInvalidate(repId);
  return useMutation({
    mutationFn: (id: string) =>
      api.post(`/api/v1/representatives/${repId}/announcements/${id}/archive`),
    onSuccess: invalidate,
  });
}

/** Only drafts can be deleted; the server enforces it. */
export function useDeleteRepAnnouncement(repId: string | undefined) {
  const invalidate = useInvalidate(repId);
  return useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/representatives/${repId}/announcements/${id}`),
    onSuccess: invalidate,
  });
}
