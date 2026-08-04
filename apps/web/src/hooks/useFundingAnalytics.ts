import { useQuery } from '@tanstack/react-query';
import type { ApiResponse } from '@civicos/types';
import { api } from '../lib/api';

/**
 * Funding analytics for one organization.
 *
 * Every money figure is what settled THROUGH CivicOS. Donations go straight to
 * the organization's own bank account, so none of this is a balance. The API
 * returns its own caveats in `notes` — render them; they are not optional
 * decoration.
 */

export interface AnalyticsMoney {
  currency: string;
  amountMinor: number;
  donationCount: number;
}

export interface AnalyticsTrendPoint {
  periodStart: string;
  amountMinor: number;
  count: number;
}

export interface OrgFundingAnalytics {
  organizationId: string;
  fundsRaised: AnalyticsMoney[];
  donors: {
    /** Signed-in donors only — see attributableDonations. */
    uniqueDonors: number;
    repeatDonors: number;
    /**
     * How many settled donations carry a user at all. Without this the two
     * counts above read as totals when they are a floor.
     */
    attributableDonations: number;
    totalDonations: number;
    averageDonation: AnalyticsMoney[];
  };
  campaigns: {
    total: number;
    byStatus: Record<string, number>;
    everPublished: number;
    completed: number;
    reported: number;
    completionRate: number;
    reportingRate: number;
  };
  trend: AnalyticsTrendPoint[];
  topCampaigns: {
    id: string;
    title: string;
    slug: string;
    status: string;
    currency: string;
    goalMinor: number;
    raisedMinor: number;
    donorCount: number;
    percentOfGoal: number;
  }[];
  generatedAt: string;
  notes: string[];
}

export function useOrgFundingAnalytics(orgId: string | undefined, weeks = 12, enabled = true) {
  return useQuery({
    queryKey: ['org-funding-analytics', orgId, weeks],
    enabled: Boolean(orgId) && enabled,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ analytics: OrgFundingAnalytics }>>(
        `/api/v1/organizations/${orgId}/funding-analytics`,
        { params: { weeks } },
      );
      return res.data.data.analytics;
    },
  });
}

/** First currency, or a zeroed placeholder. One entry in practice today. */
export function primaryMoney(list: AnalyticsMoney[] | undefined): AnalyticsMoney {
  return list?.[0] ?? { currency: 'NGN', amountMinor: 0, donationCount: 0 };
}
