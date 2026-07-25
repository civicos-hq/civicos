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

export type SummarizableResource = 'petition' | 'issue';

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
