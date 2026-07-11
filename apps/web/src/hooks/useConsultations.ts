import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type {
  ApiResponse,
  Consultation,
  ConsultationAggregate,
  ConsultationOutcome,
  ConsultationOutcomeInput,
  ConsultationQuestion,
  ConsultationQuestionInput,
  ConsultationStatus,
  CreateConsultationInput,
  MyOrgMembership,
  SubmitConsultationResponseInput,
  UpdateConsultationInput,
} from '@civicos/types';
import { api } from '../lib/api';
import { getApiError } from '../lib/api';

type ListFilters = {
  organizationId?: string;
  communityId?: string;
  status?: ConsultationStatus | '';
};

export function useConsultations(filters: ListFilters = {}) {
  return useQuery({
    queryKey: ['consultations', filters],
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (filters.organizationId) params.organizationId = filters.organizationId;
      if (filters.communityId) params.communityId = filters.communityId;
      if (filters.status) params.status = filters.status;
      const res = await api.get<ApiResponse<{ consultations: Consultation[] }>>(
        '/api/v1/consultations',
        { params },
      );
      return res.data.data.consultations;
    },
  });
}

export function useConsultation(id: string | undefined) {
  return useQuery({
    queryKey: ['consultation', id],
    enabled: !!id,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ consultation: Consultation }>>(
        `/api/v1/consultations/${id}`,
      );
      return res.data.data.consultation;
    },
  });
}

export function useConsultationQuestions(id: string | undefined) {
  return useQuery({
    queryKey: ['consultation-questions', id],
    enabled: !!id,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ questions: ConsultationQuestion[] }>>(
        `/api/v1/consultations/${id}/questions`,
      );
      return res.data.data.questions;
    },
  });
}

// useConsultationOutcome resolves to `null` when the outcome hasn't been
// published yet (server returns 404 OUTCOME_NOT_FOUND). Any other error
// bubbles up as a normal TanStack Query error so the UI can surface it.
export function useConsultationOutcome(id: string | undefined) {
  return useQuery({
    queryKey: ['consultation-outcome', id],
    enabled: !!id,
    queryFn: async () => {
      try {
        const res = await api.get<ApiResponse<{ outcome: ConsultationOutcome }>>(
          `/api/v1/consultations/${id}/outcome`,
        );
        return res.data.data.outcome;
      } catch (err) {
        const apiErr = getApiError(err);
        if (apiErr?.code === 'OUTCOME_NOT_FOUND') return null;
        throw err;
      }
    },
  });
}

// useMyConsultationResponses returns a Set of consultation IDs the caller
// has already responded to, so the list page can badge "you've participated"
// without an extra request per row.
export function useMyConsultationResponses() {
  return useQuery({
    queryKey: ['my-consultation-responses'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ consultationIds: string[] }>>(
        '/api/v1/me/consultations/responses',
      );
      return new Set(res.data.data.consultationIds);
    },
  });
}

export function useSubmitConsultationResponse(consultationId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: SubmitConsultationResponseInput) => {
      await api.post(`/api/v1/consultations/${consultationId}/responses`, input);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['consultation', consultationId] });
      qc.invalidateQueries({ queryKey: ['consultations'] });
      qc.invalidateQueries({ queryKey: ['my-consultation-responses'] });
    },
  });
}

// ─── Org-owner surface ──────────────────────────────────────────────

// useMyOrganizations lists the orgs the caller belongs to, paired with
// the membership role. Empty array means "you don't own any orgs" —
// the sidebar entry is hidden in that case.
export function useMyOrganizations() {
  return useQuery({
    queryKey: ['my-organizations'],
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ memberships: MyOrgMembership[] }>>(
        '/api/v1/me/organizations',
      );
      return res.data.data.memberships;
    },
  });
}

// invalidateConsultation invalidates every query that depends on one
// consultation. Used by all admin mutations so the frontend reflects
// server state without individual invalidation calls in every hook.
function invalidateConsultation(qc: ReturnType<typeof useQueryClient>, id: string | undefined) {
  qc.invalidateQueries({ queryKey: ['consultation', id] });
  qc.invalidateQueries({ queryKey: ['consultation-questions', id] });
  qc.invalidateQueries({ queryKey: ['consultation-outcome', id] });
  qc.invalidateQueries({ queryKey: ['consultation-analytics', id] });
  qc.invalidateQueries({ queryKey: ['consultation-responses', id] });
  qc.invalidateQueries({ queryKey: ['consultations'] });
}

export function useCreateConsultation(orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateConsultationInput) => {
      const res = await api.post<ApiResponse<{ consultation: Consultation }>>(
        `/api/v1/organizations/${orgId}/consultations`,
        input,
      );
      return res.data.data.consultation;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['consultations'] }),
  });
}

export function useUpdateConsultation(id: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateConsultationInput) => {
      await api.patch(`/api/v1/consultations/${id}`, input);
    },
    onSuccess: () => invalidateConsultation(qc, id),
  });
}

export function useDeleteConsultation(id: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete(`/api/v1/consultations/${id}`);
    },
    onSuccess: () => invalidateConsultation(qc, id),
  });
}

export function usePublishConsultation(id: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.post(`/api/v1/consultations/${id}/publish`);
    },
    onSuccess: () => invalidateConsultation(qc, id),
  });
}

export function useCloseConsultation(id: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.post(`/api/v1/consultations/${id}/close`);
    },
    onSuccess: () => invalidateConsultation(qc, id),
  });
}

export function useAddConsultationQuestion(consultationId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ConsultationQuestionInput) => {
      const res = await api.post<ApiResponse<{ question: ConsultationQuestion }>>(
        `/api/v1/consultations/${consultationId}/questions`,
        input,
      );
      return res.data.data.question;
    },
    onSuccess: () => invalidateConsultation(qc, consultationId),
  });
}

export function useUpdateConsultationQuestion(
  consultationId: string | undefined,
  questionId: string | undefined,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ConsultationQuestionInput) => {
      await api.patch(`/api/v1/consultation-questions/${questionId}`, input);
    },
    onSuccess: () => invalidateConsultation(qc, consultationId),
  });
}

export function useDeleteConsultationQuestion(
  consultationId: string | undefined,
  questionId: string | undefined,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete(`/api/v1/consultation-questions/${questionId}`);
    },
    onSuccess: () => invalidateConsultation(qc, consultationId),
  });
}

// useReorderConsultationQuestions writes a full ordering map — every
// question in the consultation must appear. The server rejects partial
// orderings so the client sends the complete snapshot after each drop.
export function useReorderConsultationQuestions(consultationId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (ordering: Record<string, number>) => {
      await api.patch(`/api/v1/consultations/${consultationId}/questions/reorder`, { ordering });
    },
    onSuccess: () => invalidateConsultation(qc, consultationId),
  });
}

export function useConsultationAnalytics(id: string | undefined) {
  return useQuery({
    queryKey: ['consultation-analytics', id],
    enabled: !!id,
    queryFn: async () => {
      const res = await api.get<
        ApiResponse<{
          consultation: Consultation;
          responseCount: number;
          questions: ConsultationAggregate[];
        }>
      >(`/api/v1/consultations/${id}/analytics`);
      return res.data.data;
    },
  });
}

export function usePublishConsultationOutcome(consultationId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ConsultationOutcomeInput) => {
      const res = await api.post<ApiResponse<{ outcome: ConsultationOutcome }>>(
        `/api/v1/consultations/${consultationId}/outcome`,
        input,
      );
      return res.data.data.outcome;
    },
    onSuccess: () => invalidateConsultation(qc, consultationId),
  });
}
