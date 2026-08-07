import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { ApiResponse, Organization } from '@civicos/types';
import { api } from '../lib/api';

/**
 * Creates (or returns) the caller's constituency office.
 *
 * An elected representative publishes campaigns, projects, consultations
 * and announcements through an organization of kind REPRESENTATIVE_OFFICE.
 * Once this returns, the representative is the OWNER of that org and every
 * existing `/org/:orgId/*` screen works for them unchanged — there is no
 * separate representative-flavoured version of any of it.
 *
 * The endpoint is idempotent, so calling it twice is harmless; it is still
 * behind an explicit button rather than fired on page load, because
 * creating "Office of Senator X" puts a new, publicly listed entity into
 * the organization directory and that should be someone's decision.
 */
export function useProvisionRepresentativeOffice() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const res = await api.post<ApiResponse<{ organization: Organization }>>(
        '/api/v1/me/representative-office',
      );
      return res.data.data.organization;
    },
    onSuccess: async () => {
      // The sidebar entry and the /org landing redirect both key off this.
      await queryClient.invalidateQueries({ queryKey: ['my-organizations'] });
    },
  });
}
