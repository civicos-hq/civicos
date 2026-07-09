import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type {
  ApiResponse,
  Consultation,
  ConsultationOutcome,
  ConsultationQuestion,
  ConsultationStatus,
  SubmitConsultationResponseInput,
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
