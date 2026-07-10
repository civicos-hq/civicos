import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { Announcement, AnnouncementStatus, ApiResponse } from '@civicos/types';
import { api } from '../lib/api';

// The org-owner surface uses /organizations/:id/announcements which
// returns both drafts + published to org members. The citizen surface
// uses /announcements for the global published feed (see below).

// usePublishedAnnouncements is the citizen-facing global feed — every
// published announcement across every organization, newest first, with
// a server-imposed default cap of 20 rows (up to 100 if requested).
export function usePublishedAnnouncements(limit?: number) {
  return useQuery({
    queryKey: ['published-announcements', limit ?? 20],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ announcements: Announcement[] }>>(
        '/api/v1/announcements',
        { params: limit ? { limit } : undefined },
      );
      return res.data.data.announcements;
    },
  });
}

export function useOrgAnnouncements(orgId: string | undefined) {
  return useQuery({
    queryKey: ['org-announcements', orgId],
    enabled: !!orgId,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ announcements: Announcement[] }>>(
        `/api/v1/organizations/${orgId}/announcements`,
      );
      return res.data.data.announcements;
    },
  });
}

export function useAnnouncement(id: string | undefined) {
  return useQuery({
    queryKey: ['announcement', id],
    enabled: !!id,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ announcement: Announcement }>>(
        `/api/v1/announcements/${id}`,
      );
      return res.data.data.announcement;
    },
  });
}

// invalidateAnnouncement invalidates every query that depends on one
// announcement or the org's list. Used by all admin mutations.
function invalidateAnnouncement(
  qc: ReturnType<typeof useQueryClient>,
  id: string | undefined,
  orgId: string | undefined,
) {
  qc.invalidateQueries({ queryKey: ['announcement', id] });
  qc.invalidateQueries({ queryKey: ['org-announcements', orgId] });
}

export interface CreateAnnouncementInput {
  title: string;
  body: string;
  /** When true, the server publishes immediately instead of leaving as DRAFT. */
  publish?: boolean;
}

export interface UpdateAnnouncementInput {
  title?: string;
  body?: string;
}

export function useCreateAnnouncement(orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateAnnouncementInput) => {
      const res = await api.post<ApiResponse<{ announcement: Announcement }>>(
        `/api/v1/organizations/${orgId}/announcements`,
        input,
      );
      return res.data.data.announcement;
    },
    onSuccess: (_, __) => invalidateAnnouncement(qc, undefined, orgId),
  });
}

export function useUpdateAnnouncement(id: string | undefined, orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateAnnouncementInput) => {
      await api.patch(`/api/v1/announcements/${id}`, input);
    },
    onSuccess: () => invalidateAnnouncement(qc, id, orgId),
  });
}

export function usePublishAnnouncement(id: string | undefined, orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.post(`/api/v1/announcements/${id}/publish`);
    },
    onSuccess: () => invalidateAnnouncement(qc, id, orgId),
  });
}

export function useArchiveAnnouncement(id: string | undefined, orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.post(`/api/v1/announcements/${id}/archive`);
    },
    onSuccess: () => invalidateAnnouncement(qc, id, orgId),
  });
}

export function useDeleteAnnouncement(id: string | undefined, orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete(`/api/v1/announcements/${id}`);
    },
    onSuccess: () => invalidateAnnouncement(qc, id, orgId),
  });
}

// AnnouncementStatus is re-exported for pages so they don't need a
// second import from @civicos/types when the only symbol they touch is
// the enum.
export type { AnnouncementStatus };
