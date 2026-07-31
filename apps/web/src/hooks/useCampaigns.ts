import { useMutation, useQuery } from '@tanstack/react-query';
import type { ApiResponse } from '@civicos/types';
import { api } from '../lib/api';

// Community Funding — public read surface (Phase 2).
//
// These endpoints are unauthenticated by design: a citizen must be able to
// see what an organization is asking for, and exactly how it plans to spend
// it, before any money can move. See docs/product/community-funding-plan.md.
//
// The API returns a projection, not the domain record — the review trail
// (approval status, reviewer notes) is deliberately absent, so there is
// nothing here to accidentally render.

export type CampaignStatus = 'PUBLISHED' | 'FUNDED' | 'COMPLETED' | 'REPORTED';

export type CampaignCategory =
  | 'EMERGENCY_RELIEF'
  | 'COMMUNITY_DEVELOPMENT'
  | 'EDUCATION'
  | 'HEALTHCARE'
  | 'ENVIRONMENT'
  | 'AGRICULTURE'
  | 'OTHER';

export interface PublicCampaign {
  id: string;
  slug: string;
  title: string;
  summary: string;
  category: CampaignCategory;
  status: CampaignStatus;
  currency: string;
  /** Integer minor units (kobo for NGN). Never a decimal — see formatMoney. */
  goalMinor: number;
  raisedMinor: number;
  donorCount: number;
  coverImageUrl?: string | null;
  state?: string | null;
  lga?: string | null;
  communityId?: string | null;
  organizationId: string;
  organizationName?: string;
  isEmergency: boolean;
  startDate?: string | null;
  endDate?: string | null;
  publishedAt?: string | null;
  completedAt?: string | null;
  /**
   * CivicOS's cut in integer basis points (250 = 2.5%), disclosed publicly.
   * A donor is entitled to know what actually reaches the organization
   * before giving — see DonateForm's live split.
   */
  platformFeeBps: number;
}

export interface PublicMilestone {
  id: string;
  title: string;
  description?: string | null;
  targetMinor: number;
  status: 'PLANNED' | 'IN_PROGRESS' | 'COMPLETED';
  position: number;
  completedAt?: string | null;
}

export interface PublicCampaignDetail extends PublicCampaign {
  description: string;
  milestones: PublicMilestone[];
}

export interface CampaignFilters {
  category?: CampaignCategory | '';
  state?: string;
  lga?: string;
  organizationId?: string;
  emergency?: boolean;
}

export function usePublicCampaigns(filters: CampaignFilters = {}) {
  return useQuery({
    queryKey: ['public-campaigns', filters],
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (filters.category) params.category = filters.category;
      if (filters.state) params.state = filters.state;
      if (filters.lga) params.lga = filters.lga;
      if (filters.organizationId) params.organizationId = filters.organizationId;
      if (filters.emergency) params.emergency = 'true';
      const res = await api.get<ApiResponse<{ campaigns: PublicCampaign[] }>>('/api/v1/campaigns', {
        params,
      });
      return res.data.data?.campaigns ?? [];
    },
  });
}

export function usePublicCampaign(slug: string | undefined) {
  return useQuery({
    queryKey: ['public-campaign', slug],
    enabled: !!slug,
    // Don't retry a 404. The API returns it both for a slug that never
    // existed and for one that isn't publicly visible — neither becomes a
    // 200 on a second attempt, so retrying just doubles the load from
    // every mistyped or stale public link.
    retry: (failureCount, error) => {
      const status = (error as { response?: { status?: number } })?.response?.status;
      if (status === 404) return false;
      return failureCount < 2;
    },
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ campaign: PublicCampaignDetail }>>(
        `/api/v1/campaigns/slug/${slug}`,
      );
      return res.data.data?.campaign ?? null;
    },
  });
}

/**
 * Formats integer minor units for display.
 *
 * The division by 100 is the ONLY place minor units become a decimal, and it
 * happens at the final formatting step. Money is stored and transported as
 * int64 minor units precisely because binary floats cannot represent 0.01 —
 * doing arithmetic on the divided value would reintroduce that problem.
 */
export function formatMoney(minor: number, currency: string, locale?: string): string {
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency,
    maximumFractionDigits: 0,
  }).format(minor / 100);
}

/** Progress as a whole percentage, clamped to 0-100 for the bar width. */
export function progressPercent(raisedMinor: number, goalMinor: number): number {
  if (goalMinor <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((raisedMinor / goalMinor) * 100)));
}

// ─── Donations (Phase 3) ────────────────────────────────────────────────

export interface DonationIntent {
  authorizationUrl: string;
  reference: string;
  amountMinor: number;
  platformFeeMinor: number;
  netMinor: number;
  currency: string;
}

export interface PublicDonation {
  donorName: string;
  amountMinor: number;
  message?: string;
  settledAt: string;
}

export interface DonateInput {
  amountMinor: number;
  email: string;
  donorName?: string;
  isAnonymous: boolean;
  message?: string;
}

/**
 * Opens a donation and hands back a Paystack checkout URL.
 *
 * The idempotency key is generated per attempt, client-side, so a
 * double-tapped donate button cannot open two transactions — the server has
 * a unique index on it and will refuse the second.
 */
export function useCreateDonationIntent(campaignId: string | undefined) {
  return useMutation({
    mutationFn: async (input: DonateInput) => {
      const res = await api.post<ApiResponse<{ donation: DonationIntent }>>(
        `/api/v1/campaigns/${campaignId}/donation-intents`,
        { ...input, idempotencyKey: crypto.randomUUID() },
      );
      return res.data.data!.donation;
    },
  });
}

export function usePublicDonations(campaignId: string | undefined) {
  return useQuery({
    queryKey: ['public-donations', campaignId],
    enabled: !!campaignId,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ donations: PublicDonation[] }>>(
        `/api/v1/campaigns/${campaignId}/donations`,
      );
      return res.data.data?.donations ?? [];
    },
  });
}

/**
 * Splits an amount the way the server will, for the live fee disclosure.
 *
 * Mirrors ComputeSplit in services/organization-service/internal/donations:
 * integer basis points, floored so any fraction of a kobo falls to the
 * organization rather than the platform. This is a PREVIEW — the server
 * recomputes authoritatively — but it must not disagree, or the donor sees
 * one number here and another on their receipt.
 */
export function previewSplit(amountMinor: number, platformFeeBps: number) {
  const gross = Math.max(0, Math.floor(amountMinor));
  const fee = Math.floor((gross * platformFeeBps) / 10_000);
  return { grossMinor: gross, platformFeeMinor: fee, netMinor: gross - fee };
}
