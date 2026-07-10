import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ApiResponse, Project, ProjectStatus } from '@civicos/types';
import { api } from '../lib/api';

// Projects are org-owned records with a status lifecycle
// (PLANNED / ACTIVE / PAUSED / COMPLETED / CANCELLED). Unlike
// consultations or announcements the underlying API doesn't expose
// dedicated `/publish` or `/close` endpoints — the caller changes state
// by PATCHing `status` on the project itself.

export interface CreateProjectInput {
  title: string;
  description: string;
  status?: ProjectStatus;
  startDate?: string;
  expectedEndDate?: string;
  budgetKobo?: number;
  communityId?: string;
}

export type UpdateProjectInput = Partial<CreateProjectInput>;

// useCitizenProjects is the global project browse — accepts optional
// filter fields matching the /api/v1/projects query params. Any field
// left undefined means "no filter" (server returns everything matching
// the other fields, or all rows if none supplied).
export interface CitizenProjectFilters {
  organizationId?: string;
  communityId?: string;
  status?: ProjectStatus | '';
}

export function useCitizenProjects(filters: CitizenProjectFilters = {}) {
  return useQuery({
    queryKey: ['citizen-projects', filters],
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (filters.organizationId) params.organizationId = filters.organizationId;
      if (filters.communityId) params.communityId = filters.communityId;
      if (filters.status) params.status = filters.status;
      const res = await api.get<ApiResponse<{ projects: Project[] }>>('/api/v1/projects', {
        params,
      });
      return res.data.data.projects;
    },
  });
}

export function useOrgProjects(orgId: string | undefined) {
  return useQuery({
    queryKey: ['org-projects', orgId],
    enabled: !!orgId,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ projects: Project[] }>>(
        `/api/v1/organizations/${orgId}/projects`,
      );
      return res.data.data.projects;
    },
  });
}

export function useProject(id: string | undefined) {
  return useQuery({
    queryKey: ['project', id],
    enabled: !!id,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ project: Project }>>(`/api/v1/projects/${id}`);
      return res.data.data.project;
    },
  });
}

function invalidateProject(
  qc: ReturnType<typeof useQueryClient>,
  id: string | undefined,
  orgId: string | undefined,
) {
  qc.invalidateQueries({ queryKey: ['project', id] });
  qc.invalidateQueries({ queryKey: ['org-projects', orgId] });
}

export function useCreateProject(orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateProjectInput) => {
      const res = await api.post<ApiResponse<{ project: Project }>>(
        `/api/v1/organizations/${orgId}/projects`,
        input,
      );
      return res.data.data.project;
    },
    onSuccess: () => invalidateProject(qc, undefined, orgId),
  });
}

export function useUpdateProject(id: string | undefined, orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateProjectInput) => {
      await api.patch(`/api/v1/projects/${id}`, input);
    },
    onSuccess: () => invalidateProject(qc, id, orgId),
  });
}

export function useDeleteProject(id: string | undefined, orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete(`/api/v1/projects/${id}`);
    },
    onSuccess: () => invalidateProject(qc, id, orgId),
  });
}

// Budget helpers — the API stores kobo (₦1 = 100 kobo) so the UI has
// to convert on both sides. Kept here so pages don't sprinkle math.
export function kobopToNaira(kobo: number | undefined | null): number | '' {
  if (kobo === undefined || kobo === null) return '';
  return kobo / 100;
}

export function nairaToKobo(naira: number | string): number | undefined {
  const n = typeof naira === 'string' ? parseFloat(naira) : naira;
  if (!Number.isFinite(n)) return undefined;
  return Math.round(n * 100);
}
