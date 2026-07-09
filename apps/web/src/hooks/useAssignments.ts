import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ApiResponse, AssignmentStatus, IssueAssignment } from '@civicos/types';
import { api } from '../lib/api';

// Assignments record "this org has taken responsibility for this issue."
// Two list surfaces:
//   - Org's inbox: what have we claimed? — org member read
//   - Issue's page: which orgs are on it? — public read

export interface CreateAssignmentInput {
  issueId: string;
  note?: string;
}

export interface UpdateAssignmentStatusInput {
  status: AssignmentStatus;
  note?: string;
}

export function useOrgAssignments(orgId: string | undefined) {
  return useQuery({
    queryKey: ['org-assignments', orgId],
    enabled: !!orgId,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ assignments: IssueAssignment[] }>>(
        `/api/v1/organizations/${orgId}/assignments`,
      );
      return res.data.data.assignments;
    },
  });
}

export function useIssueAssignments(issueId: string | undefined) {
  return useQuery({
    queryKey: ['issue-assignments', issueId],
    enabled: !!issueId,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ assignments: IssueAssignment[] }>>(
        `/api/v1/issues/${issueId}/assignments`,
      );
      return res.data.data.assignments;
    },
  });
}

function invalidateAssignment(
  qc: ReturnType<typeof useQueryClient>,
  orgId: string | undefined,
  issueId: string | undefined,
) {
  if (orgId) qc.invalidateQueries({ queryKey: ['org-assignments', orgId] });
  if (issueId) qc.invalidateQueries({ queryKey: ['issue-assignments', issueId] });
}

export function useCreateAssignment(orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateAssignmentInput) => {
      const res = await api.post<ApiResponse<{ assignment: IssueAssignment }>>(
        `/api/v1/organizations/${orgId}/assignments`,
        input,
      );
      return res.data.data.assignment;
    },
    onSuccess: (assignment) => invalidateAssignment(qc, orgId, assignment.issueId),
  });
}

export function useUpdateAssignmentStatus(
  assignmentId: string | undefined,
  orgId: string | undefined,
  issueId: string | undefined,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateAssignmentStatusInput) => {
      await api.patch(`/api/v1/assignments/${assignmentId}`, input);
    },
    onSuccess: () => invalidateAssignment(qc, orgId, issueId),
  });
}

export function useDeleteAssignment(
  assignmentId: string | undefined,
  orgId: string | undefined,
  issueId: string | undefined,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete(`/api/v1/assignments/${assignmentId}`);
    },
    onSuccess: () => invalidateAssignment(qc, orgId, issueId),
  });
}
