import { useQuery } from '@tanstack/react-query';
import type { ApiResponse, CommunityFloodAlert, FloodAttribution } from '@civicos/types';
import { api } from '../lib/api';

interface FloodAlertsResponse {
  alerts: CommunityFloodAlert[];
  attribution: FloodAttribution;
}

/**
 * Current flood forecasts for a community.
 *
 * Returns an empty list when the feature is off, when the community has no
 * coordinates, or when nothing is forecast — the banner renders nothing in
 * all three cases. That is deliberate: "no banner" must never be read as
 * "you are safe", so there is no quiet-state UI to mistake for one.
 */
export function useFloodAlerts(communityId: string | undefined) {
  return useQuery({
    queryKey: ['flood-alerts', communityId],
    enabled: Boolean(communityId),
    // Forecasts refresh upstream on a slow cadence and the poller writes
    // hourly; refetching harder would just add load for identical data.
    staleTime: 5 * 60 * 1000,
    // A failed fetch shows nothing rather than an error. The banner is
    // additive — a broken third-party call must not break the page a
    // citizen came to read.
    retry: false,
    queryFn: async (): Promise<FloodAlertsResponse> => {
      const res = await api.get<ApiResponse<FloodAlertsResponse>>(
        `/api/v1/communities/${communityId}/flood-alerts`,
      );
      return { alerts: res.data.data.alerts ?? [], attribution: res.data.data.attribution };
    },
  });
}
