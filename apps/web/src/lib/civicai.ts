// Thin client for the CivicAI service. Every AI endpoint lives behind
// /api/v1/ai/* on the gateway and returns the standard { success, data }
// envelope. Feature hooks (classify-issue, summarize, draft-announcement)
// consume this module so we have one place to bump versioning or wire
// tracing later.

import type { ApiResponse } from '@civicos/types';
import { api } from './api';

// Keep enum values in sync with the civicai-service classify package. The
// authoritative list is also mirrored by IssueCategory in @civicos/types —
// we don't reuse the enum directly here because Gemini could add
// non-issue categories (e.g. petition-topic) in future endpoints.
export type CivicAICategory =
  | 'INFRASTRUCTURE'
  | 'HEALTH'
  | 'EDUCATION'
  | 'SECURITY'
  | 'ENVIRONMENT'
  | 'UTILITIES'
  | 'TRANSPORT'
  | 'OTHER';

export type CivicAISeverity = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';

export interface IssueClassification {
  category: CivicAICategory;
  severity: CivicAISeverity;
  suggestedTags: string[];
  reasoning: string;
  confidence: number;
  model: string;
  generatedAt: string;
}

export interface ClassifyIssueInput {
  title: string;
  description: string;
  communityId?: string;
}

export async function classifyIssue(
  input: ClassifyIssueInput,
  signal?: AbortSignal,
): Promise<IssueClassification> {
  const res = await api.post<ApiResponse<{ classification: IssueClassification }>>(
    '/api/v1/ai/classify-issue',
    input,
    { signal },
  );
  return res.data.data.classification;
}

// ─── Summarize ────────────────────────────────────────────────────────────

export type SummarizableResource = 'petition' | 'issue' | 'consultation';

export interface DiscussionSummary {
  resource: SummarizableResource;
  resourceId: string;
  title: string;
  tldr: string;
  themes: string[];
  sentiment: { positive: number; neutral: number; negative: number };
  topAsks: string[];
  recommendedActions: string[];
  commentsAnalyzed: number;
  officialResponders: string[];
  model: string;
  generatedAt: string;
  cached: boolean;
}

export async function summarizeDiscussion(
  resource: SummarizableResource,
  id: string,
  signal?: AbortSignal,
): Promise<DiscussionSummary> {
  const res = await api.post<ApiResponse<{ summary: DiscussionSummary }>>(
    '/api/v1/ai/summarize',
    { resource, id },
    { signal },
  );
  return res.data.data.summary;
}

// ─── Announcement drafting ────────────────────────────────────────────────

export type DraftTone = 'formal' | 'friendly' | 'urgent' | 'empathetic';
export type DraftAudience = 'all' | 'members';

export interface AnnouncementDraft {
  title: string;
  body: string;
  keyPoints: string[];
  tone: DraftTone;
  audience: DraftAudience;
  model: string;
  generatedAt: string;
}

export interface DraftAnnouncementInput {
  brief: string;
  tone: DraftTone;
  audience: DraftAudience;
  orgName?: string;
  orgKind?: string;
}

export async function draftAnnouncement(
  input: DraftAnnouncementInput,
  signal?: AbortSignal,
): Promise<AnnouncementDraft> {
  const res = await api.post<ApiResponse<{ draft: AnnouncementDraft }>>(
    '/api/v1/ai/draft-announcement',
    input,
    { signal },
  );
  return res.data.data.draft;
}

// ─── Community insights ───────────────────────────────────────────────────

export interface CommunityInsights {
  communityId: string;
  tldr: string;
  themes: string[];
  sentimentMix: { positive: number; neutral: number; negative: number };
  topAsks: string[];
  recommendedActions: string[];
  activity: { issueCount: number; petitionCount: number; commentCount: number };
  model: string;
  generatedAt: string;
  cached: boolean;
}

export async function getCommunityInsights(
  communityId: string,
  signal?: AbortSignal,
): Promise<CommunityInsights> {
  const res = await api.get<ApiResponse<{ insights: CommunityInsights }>>(
    '/api/v1/ai/community-insights',
    { params: { communityId }, signal },
  );
  return res.data.data.insights;
}

// ─── Community Funding ────────────────────────────────────────────────────
//
// Six advisory surfaces for campaigns. Every one of them returns Provenance,
// and every UI that renders their output must show the AI badge — these
// drafts are about money other people gave, and a reader needs to know a
// model wrote the words.
//
// None of these endpoints writes anything. Accepting a suggestion is always a
// separate, explicit action by the person looking at it.

/** Present on every campaign AI response. */
export interface AIProvenance {
  model: string;
  generatedAt: string;
  /** Always true. Rendered, not just carried. */
  advisory: boolean;
}

export type CampaignAICategory =
  | 'EMERGENCY_RELIEF'
  | 'COMMUNITY_DEVELOPMENT'
  | 'EDUCATION'
  | 'HEALTHCARE'
  | 'ENVIRONMENT'
  | 'AGRICULTURE'
  | 'OTHER';

export interface CampaignClassification extends AIProvenance {
  category: CampaignAICategory;
  isEmergency: boolean;
  suggestedTags: string[];
  reasoning: string;
  confidence: number;
}

export async function classifyCampaign(
  input: { title: string; description: string },
  signal?: AbortSignal,
): Promise<CampaignClassification> {
  const res = await api.post<ApiResponse<{ classification: CampaignClassification }>>(
    '/api/v1/ai/classify-campaign',
    input,
    { signal },
  );
  return res.data.data.classification;
}

export interface CampaignDraftMilestone {
  title: string;
  description: string;
  targetMinor: number;
}

export interface CampaignDraft extends AIProvenance {
  title: string;
  summary: string;
  description: string;
  milestones: CampaignDraftMilestone[];
  /** Things a reviewer will ask for that the brief does not yet supply. */
  warnings: string[];
}

export interface DraftCampaignInput {
  brief: string;
  category?: string;
  goalMinor?: number;
  currency?: string;
  state?: string;
  lga?: string;
  organizationName?: string;
  isEmergency?: boolean;
}

export async function draftCampaign(
  input: DraftCampaignInput,
  signal?: AbortSignal,
): Promise<CampaignDraft> {
  const res = await api.post<ApiResponse<{ draft: CampaignDraft }>>(
    '/api/v1/ai/draft-campaign',
    input,
    { signal },
  );
  return res.data.data.draft;
}

export interface CampaignImpact extends AIProvenance {
  summary: string;
  highlights: string[];
  /** What a reader cannot tell from what has been published. Not decoration. */
  gaps: string[];
}

export async function summarizeCampaignImpact(
  campaignId: string,
  signal?: AbortSignal,
): Promise<CampaignImpact> {
  const res = await api.post<ApiResponse<{ impact: CampaignImpact }>>(
    '/api/v1/ai/summarize-campaign-impact',
    { campaignId },
    { signal },
  );
  return res.data.data.impact;
}

export interface DonorUpdateDraft extends AIProvenance {
  title: string;
  body: string;
  warnings: string[];
}

export async function draftDonorUpdate(
  input: { campaignId: string; brief: string },
  signal?: AbortSignal,
): Promise<DonorUpdateDraft> {
  const res = await api.post<ApiResponse<{ update: DonorUpdateDraft }>>(
    '/api/v1/ai/draft-donor-update',
    input,
    { signal },
  );
  return res.data.data.update;
}

export interface CompletionReportDraft extends AIProvenance {
  body: string;
  /** Computed from the ledger server-side, never generated by the model. */
  unaccountedMinor: number;
  currency: string;
  mustExplain: boolean;
  warnings: string[];
}

export async function draftCompletionReport(
  input: { campaignId: string; brief: string },
  signal?: AbortSignal,
): Promise<CompletionReportDraft> {
  const res = await api.post<ApiResponse<{ report: CompletionReportDraft }>>(
    '/api/v1/ai/draft-completion-report',
    input,
    { signal },
  );
  return res.data.data.report;
}
