import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
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
  /** Absent when the campaign has no spend reporting available. */
  spend?: SpendSummary;
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
    // narrowSymbol, or Nigerians are shown "NGN 100,000" instead of
    // "₦100,000": Intl only picks the ₦ glyph for an explicitly Nigerian
    // locale, and the app passes i18n.language ("en", "ha", "ig"…), never
    // "en-NG". Applies to every locale we ship.
    currencyDisplay: 'narrowSymbol',
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

// ─── Transparency dashboard (Phase 4) ───────────────────────────────────

/**
 * The organization's own account of what it did with the money.
 *
 * Named `reported`, never `spent`, throughout. Donations settle straight to
 * the organization's Paystack sub-account, so CivicOS never holds the money
 * and cannot verify a single line of this. It is a claim published under the
 * organization's name, and the UI has to say so.
 */
export interface SpendSummary {
  reportedMinor: number;
  unreportedMinor: number;
  /** The organization reports spending more than it raised here — legitimate
   *  (they may have topped up from other funds) and surfaced rather than
   *  clamped, so the arithmetic on the page still adds up. */
  exceedsReceived: boolean;
  recordCount: number;
  perMilestone: Record<string, number>;
}

export interface SpendRecord {
  id: string;
  milestoneId: string;
  amountMinor: number;
  currency: string;
  description: string;
  spentAt: string;
  receiptUrl?: string | null;
  publishedBy: string;
  publishedAt: string;
}

export interface FundingUpdate {
  id: string;
  campaignId?: string | null;
  title?: string | null;
  body: string;
  attachmentUrls: string[];
  authorName: string;
  createdAt: string;
}

export function useCampaignSpend(campaignId: string | undefined) {
  return useQuery({
    queryKey: ['campaign-spend', campaignId],
    enabled: !!campaignId,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ spend: SpendRecord[] }>>(
        `/api/v1/campaigns/${campaignId}/spend`,
      );
      return res.data.data?.spend ?? [];
    },
  });
}

export function useCampaignUpdates(campaignId: string | undefined) {
  return useQuery({
    queryKey: ['campaign-updates', campaignId],
    enabled: !!campaignId,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ updates: FundingUpdate[] }>>(
        `/api/v1/campaigns/${campaignId}/updates`,
      );
      return res.data.data?.updates ?? [];
    },
  });
}

/**
 * Formats minor units WITHOUT rounding away the kobo.
 *
 * `formatMoney` rounds to whole units, which is right for a progress bar but
 * wrong for the accounting section: a donor reading "₦62 reported" against a
 * stored ₦62.50 sees figures that do not sum to the total shown above them.
 * A transparency page whose arithmetic does not add up undermines the exact
 * thing it exists to demonstrate.
 */
export function formatMoneyExact(minor: number, currency: string, locale?: string): string {
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency,
    currencyDisplay: 'narrowSymbol',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(minor / 100);
}

/** What share of received money has been accounted for, 0-100. */
export function accountedPercent(reportedMinor: number, receivedMinor: number): number {
  if (receivedMinor <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((reportedMinor / receivedMinor) * 100)));
}

// ─── Org console (Phase 4) ──────────────────────────────────────────────

/**
 * Whether the signed-in user may publish on behalf of this campaign's
 * organization.
 *
 * This gates VISIBILITY only. Every write is authorised again server-side
 * against the owning org — a client-side role check decides what to render,
 * never what is permitted.
 */
export function useCanManageCampaign(organizationId: string | undefined) {
  const q = useQuery({
    queryKey: ['my-organizations'],
    // A signed-out visitor has no token; asking would just 401 on every
    // campaign page view.
    enabled: !!organizationId && !!localStorage.getItem('accessToken'),
    retry: false,
    queryFn: async () => {
      const res = await api.get<
        ApiResponse<{
          memberships: { organization: { id: string }; membership: { role: string } }[];
        }>
      >('/api/v1/me/organizations');
      return res.data.data?.memberships ?? [];
    },
  });
  const m = (q.data ?? []).find((x) => x.organization?.id === organizationId);
  // STAFF can read an org's internals but must not publish financial claims
  // in its name.
  return m?.membership?.role === 'OWNER' || m?.membership?.role === 'ADMIN';
}

export interface CreateSpendInput {
  milestoneId: string;
  amountMinor: number;
  description: string;
  spentAt: string;
  receiptUrl?: string;
}

export function useCreateSpend(campaignId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateSpendInput) => {
      const res = await api.post<ApiResponse<{ spend: SpendRecord }>>(
        `/api/v1/campaigns/${campaignId}/spend`,
        input,
      );
      return res.data.data?.spend;
    },
    onSuccess: () => {
      // The campaign detail carries the summary totals, so both have to
      // refetch or the page shows a new record against a stale total.
      qc.invalidateQueries({ queryKey: ['campaign-spend', campaignId] });
      qc.invalidateQueries({ queryKey: ['public-campaign'] });
    },
  });
}

export function useDeleteSpend(campaignId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (spendId: string) => {
      await api.delete(`/api/v1/spend/${spendId}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['campaign-spend', campaignId] });
      qc.invalidateQueries({ queryKey: ['public-campaign'] });
    },
  });
}

export interface CreateUpdateInput {
  campaignId: string;
  title?: string;
  body: string;
  attachmentUrls?: string[];
}

export function useCreateFundingUpdate(
  organizationId: string | undefined,
  campaignId: string | undefined,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateUpdateInput) => {
      const res = await api.post<ApiResponse<{ update: FundingUpdate }>>(
        `/api/v1/organizations/${organizationId}/progress-updates`,
        input,
      );
      return res.data.data?.update;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['campaign-updates', campaignId] });
    },
  });
}

// ─── Org campaign management ────────────────────────────────────────────

/**
 * A campaign as its own organization sees it — including the statuses the
 * public never does (DRAFT, PENDING_REVIEW, NEEDS_CHANGES, REJECTED) and the
 * reviewer's note.
 *
 * `reviewNote` is why this is a separate type from PublicCampaign: it is a
 * private conversation between the platform and the organization, and the
 * public DTO deliberately omits it.
 */
export interface OrgCampaign {
  id: string;
  slug: string;
  title: string;
  summary: string;
  status: string;
  category: CampaignCategory;
  currency: string;
  goalMinor: number;
  raisedMinor: number;
  donorCount: number;
  reviewNote?: string | null;
  isEmergency: boolean;
  createdAt?: string;
  // The org-scoped endpoint returns the full campaign, so the edit form can
  // be populated from the list without a second fetch.
  description?: string;
  state?: string | null;
  lga?: string | null;
}

/**
 * Maps a category enum to its translation key.
 *
 * The enum is SCREAMING_SNAKE (`EMERGENCY_RELIEF`) but the locale files are
 * keyed camelCase (`emergencyRelief`), so translating the raw value silently
 * falls through to the fallback and shows the enum to the user.
 */
export function categoryKey(c: CampaignCategory): string {
  return c.toLowerCase().replace(/_([a-z])/g, (_, ch: string) => ch.toUpperCase());
}

export function useOrgCampaigns(organizationId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ['org-campaigns', organizationId],
    enabled: !!organizationId && enabled,
    retry: false,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ campaigns: OrgCampaign[] }>>(
        `/api/v1/organizations/${organizationId}/campaigns`,
      );
      return res.data.data?.campaigns ?? [];
    },
  });
}

export interface CreateCampaignInput {
  title: string;
  summary: string;
  description: string;
  category: CampaignCategory;
  goalMinor: number;
  state?: string;
  lga?: string;
  isEmergency?: boolean;
}

export function useCreateCampaign(organizationId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateCampaignInput) => {
      const res = await api.post<ApiResponse<{ campaign: OrgCampaign }>>(
        `/api/v1/organizations/${organizationId}/campaigns`,
        input,
      );
      return res.data.data?.campaign;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['org-campaigns', organizationId] }),
  });
}

/**
 * Edits campaign content.
 *
 * The server permits this only while a campaign is DRAFT or NEEDS_CHANGES —
 * once it is in review or live, content is frozen, because a donor who reads
 * a campaign and gives money must be giving to the thing they read. The UI
 * mirrors that rather than letting someone fill in a form only to be
 * refused.
 */
export function useUpdateCampaign(organizationId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (v: { campaignId: string; input: Partial<CreateCampaignInput> }) => {
      await api.patch(`/api/v1/campaigns/${v.campaignId}`, v.input);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['org-campaigns', organizationId] });
      qc.invalidateQueries({ queryKey: ['public-campaign'] });
    },
  });
}

/** Whether an organization may still change this campaign's content. */
export function isEditable(status: string): boolean {
  return status === 'DRAFT' || status === 'NEEDS_CHANGES';
}

function invalidatePlan(
  qc: ReturnType<typeof useQueryClient>,
  campaignId?: string,
  orgId?: string,
) {
  qc.invalidateQueries({ queryKey: ['milestones', campaignId] });
  // The public page renders the plan and the spend summary keys off it.
  qc.invalidateQueries({ queryKey: ['public-campaign'] });
  qc.invalidateQueries({ queryKey: ['org-campaigns', orgId] });
}

export function useCreateMilestone(organizationId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (v: { campaignId: string; title: string; targetMinor: number }) => {
      await api.post(`/api/v1/campaigns/${v.campaignId}/milestones`, {
        title: v.title,
        targetMinor: v.targetMinor,
      });
      return v.campaignId;
    },
    // Invalidate the plan too, not just the campaign list. Without this the
    // spend-plan editor keeps showing a stale plan after adding to it, and
    // the remaining-capacity figure it computes goes wrong with it.
    onSuccess: (campaignId) => invalidatePlan(qc, campaignId, organizationId),
  });
}

/**
 * Drives the org-side lifecycle steps: submit for review, then publish once
 * the platform has approved.
 *
 * Deliberately does NOT cover review, pause, resume or archive — those are
 * platform-admin actions and live in the admin console. An organization
 * approving its own fundraiser would defeat the point of review.
 */
export function useCampaignLifecycle(organizationId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (v: { campaignId: string; action: 'submit' | 'publish' | 'complete' }) => {
      await api.post(`/api/v1/campaigns/${v.campaignId}/${v.action}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['org-campaigns', organizationId] });
      qc.invalidateQueries({ queryKey: ['public-campaign'] });
    },
  });
}

// ─── Spend plan management ──────────────────────────────────────────────

export interface Milestone {
  id: string;
  campaignId: string;
  title: string;
  description?: string | null;
  targetMinor: number;
  status: 'PLANNED' | 'IN_PROGRESS' | 'COMPLETED';
  position: number;
  completedAt?: string | null;
}

export function useMilestones(campaignId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ['milestones', campaignId],
    enabled: !!campaignId && enabled,
    retry: false,
    // The plan is edited inside a dialog. Refetching on window focus or on a
    // stale timer swapped the list out underneath the user mid-edit, which
    // read as flicker. Mutations invalidate this key explicitly, so it stays
    // correct without background churn.
    refetchOnWindowFocus: false,
    staleTime: 30_000,
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ milestones: Milestone[] }>>(
        `/api/v1/campaigns/${campaignId}/milestones`,
      );
      return res.data.data?.milestones ?? [];
    },
  });
}

export function useDeleteMilestone(orgId: string | undefined, campaignId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (milestoneId: string) => {
      await api.delete(`/api/v1/milestones/${milestoneId}`);
    },
    onSuccess: () => invalidatePlan(qc, campaignId, orgId),
  });
}

/**
 * Marks a milestone complete.
 *
 * Deliberately available while a campaign is LIVE, unlike every other change
 * to the plan: this is progress reporting, not rewriting what donors were
 * shown. It is also what fires the MILESTONE_COMPLETED notification to
 * everyone who funded the campaign.
 */
export function useCompleteMilestone(orgId: string | undefined, campaignId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (milestoneId: string) => {
      await api.patch(`/api/v1/milestones/${milestoneId}`, { status: 'COMPLETED' });
    },
    onSuccess: () => invalidatePlan(qc, campaignId, orgId),
  });
}

/**
 * Deletes a draft campaign.
 *
 * DRAFT only, and the server agrees: anything already submitted is archived
 * rather than deleted, so a campaign that reviewers or donors have seen
 * leaves a trail instead of vanishing.
 */
export function useDeleteCampaign(orgId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (campaignId: string) => {
      await api.delete(`/api/v1/campaigns/${campaignId}`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['org-campaigns', orgId] }),
  });
}

/** Only a draft may be deleted; everything later is archived instead. */
export function isDeletable(status: string): boolean {
  return status === 'DRAFT';
}
