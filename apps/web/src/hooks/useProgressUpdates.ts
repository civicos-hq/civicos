import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ApiResponse, ProgressUpdate } from '@civicos/types';
import { api } from '../lib/api';

// Progress updates hang off either an assigned issue or a project.
// Public updates are readable by anyone; the internal-note flag lets
// orgs post notes their members can see but citizens can't.
//
// Note: an older `useIssueProgressUpdates` exists in OrganizationDetailPage
// for the read side. This module is the primary place for the write hooks
// and the project-side reads; the older hook stays for backwards compat
// until we consolidate.

export interface CreateProgressUpdateInput {
  issueId?: string;
  projectId?: string;
  body: string;
  isPublic?: boolean;
}

export function useProjectProgressUpdates(projectId: string | undefined) {
  return useQuery({
    queryKey: ['project-progress', projectId],
    enabled: !!projectId,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ updates: ProgressUpdate[] }>>(
        `/api/v1/projects/${projectId}/progress-updates`,
      );
      return res.data.data.updates;
    },
  });
}

// useCreateProgressUpdate is scoped to a specific org — the caller must
// pass the orgId because the API mounts creation under
// /organizations/:id/progress-updates. Invalidates whichever list the
// created update belongs to.
export function useCreateProgressUpdate(orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateProgressUpdateInput) => {
      const res = await api.post<ApiResponse<{ update: ProgressUpdate }>>(
        `/api/v1/organizations/${orgId}/progress-updates`,
        input,
      );
      return res.data.data.update;
    },
    onSuccess: (update) => {
      if (update.issueId) {
        qc.invalidateQueries({ queryKey: ['issue-progress', update.issueId] });
      }
      if (update.projectId) {
        qc.invalidateQueries({ queryKey: ['project-progress', update.projectId] });
      }
    },
  });
}

export function useDeleteProgressUpdate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({
      updateId,
      issueId,
      projectId,
    }: {
      updateId: string;
      issueId?: string;
      projectId?: string;
    }) => {
      await api.delete(`/api/v1/progress-updates/${updateId}`);
      return { issueId, projectId };
    },
    onSuccess: ({ issueId, projectId }) => {
      if (issueId) qc.invalidateQueries({ queryKey: ['issue-progress', issueId] });
      if (projectId) qc.invalidateQueries({ queryKey: ['project-progress', projectId] });
    },
  });
}
