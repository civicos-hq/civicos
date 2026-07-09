// ─── Primitive aliases ────────────────────────────────────────────────────────

/** UUID v4 string — all entity IDs across CivicOS */
export type UUID = string;

/** ISO 8601 UTC datetime string */
export type ISODateTime = string;

// ─── Enums ───────────────────────────────────────────────────────────────────

export enum UserRole {
  CITIZEN = 'CITIZEN',
  REPRESENTATIVE = 'REPRESENTATIVE',
  GOVERNMENT_ADMIN = 'GOVERNMENT_ADMIN',
  NGO = 'NGO',
  MODERATOR = 'MODERATOR',
  PLATFORM_ADMIN = 'PLATFORM_ADMIN',
}

export enum RequestedAccountType {
  CITIZEN = 'CITIZEN',
  REPRESENTATIVE = 'REPRESENTATIVE',
  ORGANIZATION = 'ORGANIZATION',
}

export enum ApprovalStatus {
  NONE = 'NONE',
  PENDING = 'PENDING',
  APPROVED = 'APPROVED',
  NEEDS_CHANGES = 'NEEDS_CHANGES',
  REJECTED = 'REJECTED',
}

export enum IssueStatus {
  OPEN = 'OPEN',
  UNDER_REVIEW = 'UNDER_REVIEW',
  IN_PROGRESS = 'IN_PROGRESS',
  RESOLVED = 'RESOLVED',
  CLOSED = 'CLOSED',
}

export enum IssueCategory {
  INFRASTRUCTURE = 'INFRASTRUCTURE',
  HEALTH = 'HEALTH',
  EDUCATION = 'EDUCATION',
  SECURITY = 'SECURITY',
  ENVIRONMENT = 'ENVIRONMENT',
  UTILITIES = 'UTILITIES',
  TRANSPORT = 'TRANSPORT',
  OTHER = 'OTHER',
}

export enum PetitionStatus {
  DRAFT = 'DRAFT',
  ACTIVE = 'ACTIVE',
  CLOSED = 'CLOSED',
  SUCCESSFUL = 'SUCCESSFUL',
}

export enum NotificationType {
  ISSUE_UPDATE = 'ISSUE_UPDATE',
  PETITION_UPDATE = 'PETITION_UPDATE',
  REPRESENTATIVE_RESPONSE = 'REPRESENTATIVE_RESPONSE',
  COMMUNITY_UPDATE = 'COMMUNITY_UPDATE',
  CONSULTATION_UPDATE = 'CONSULTATION_UPDATE',
  SYSTEM = 'SYSTEM',
}

// ─── Domain interfaces ────────────────────────────────────────────────────────

export interface User {
  id: UUID;
  email: string;
  name: string;
  role: UserRole;
  avatarUrl?: string;
  activeCommunityId?: UUID;
  primaryCommunityId?: UUID;
  primaryCommunityChangedAt?: ISODateTime;
  memberships: CommunityMembership[];
  requestedAccountType: RequestedAccountType;
  approvalStatus: ApprovalStatus;
  approvalReviewedAt?: ISODateTime;
  approvalReviewedById?: UUID;
  approvalNote?: string;
  emailVerified: boolean;
  emailVerifiedAt?: ISODateTime;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface CommunityMembership {
  communityId: UUID;
  joinedAt: ISODateTime;
}

export interface Community {
  id: UUID;
  name: string;
  slug: string;
  description?: string;
  state: string;
  lga: string;
  country: string;
  logoUrl?: string;
  memberCount: number;
  createdAt: ISODateTime;
}

export interface Representative {
  id: UUID;
  name: string;
  title: string;
  position: string;
  constituency: string;
  party?: string;
  bio?: string;
  avatarUrl?: string;
  email?: string;
  phone?: string;
  website?: string;
  communityId: UUID;
  responseRate: number;
  followerCount: number;
  commentCount: number;
  createdById: UUID;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface Issue {
  id: UUID;
  title: string;
  description: string;
  category: IssueCategory;
  status: IssueStatus;
  location?: string;
  imageUrls: string[];
  upvoteCount: number;
  commentCount: number;
  communityId: UUID;
  reportedById: UUID;
  reportedBy?: Pick<User, 'id' | 'name' | 'avatarUrl'>;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface Petition {
  id: UUID;
  title: string;
  description: string;
  goal: number;
  signatureCount: number;
  commentCount: number;
  status: PetitionStatus;
  deadline?: ISODateTime;
  imageUrls: string[];
  communityId: UUID;
  createdById: UUID;
  createdBy?: Pick<User, 'id' | 'name' | 'avatarUrl'>;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface Comment {
  id: UUID;
  content: string;
  authorId: UUID;
  authorName: string;
  authorRole: UserRole;
  isOfficialResponse: boolean;
  // isHidden is true when a moderator has hidden this comment. The
  // server replaces content and authorName with placeholders on the way
  // out; the UI just needs this flag to render the row in a muted style
  // (see CommentsSection).
  isHidden?: boolean;
  createdAt: ISODateTime;
}

export interface IssueComment extends Comment {
  issueId: UUID;
}

export interface PetitionComment extends Comment {
  petitionId: UUID;
}

export interface RepresentativeComment extends Comment {
  representativeId: UUID;
}

export interface Notification {
  id: UUID;
  type: NotificationType;
  title: string;
  body: string;
  read: boolean;
  linkUrl?: string;
  userId: UUID;
  createdAt: ISODateTime;
}

// ─── API envelope types ───────────────────────────────────────────────────────

export interface ApiResponse<T> {
  success: true;
  data: T;
  message?: string;
  requestId?: string;
}

export interface ApiError {
  success: false;
  code: string;
  message: string;
  details?: Record<string, string[]>;
  requestId?: string;
}

export interface PaginatedData<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  hasMore: boolean;
}

// ─── Auth types ───────────────────────────────────────────────────────────────

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

export interface JwtPayload {
  sub: UUID;
  email: string;
  role: UserRole;
  iat?: number;
  exp?: number;
}

export interface RepresentativeApplication {
  id: UUID;
  userId: UUID;
  status: ApprovalStatus;
  fullName: string;
  title: string;
  position: string;
  constituency: string;
  communityId: UUID;
  party?: string;
  bio?: string;
  avatarUrl?: string;
  officialEmail?: string;
  officialPhone?: string;
  website?: string;
  proofUrls: string[];
  submittedAt: ISODateTime;
  reviewedAt?: ISODateTime;
  reviewedByUserId?: UUID;
  reviewNote?: string;
  approvedProfileId?: UUID;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface OrganizationApplication {
  id: UUID;
  userId: UUID;
  status: ApprovalStatus;
  name: string;
  slug: string;
  kind: OrgKind;
  jurisdiction: OrgJurisdiction;
  state?: string;
  lga?: string;
  description?: string;
  logoUrl?: string;
  officialEmail?: string;
  officialPhone?: string;
  website?: string;
  proofUrls: string[];
  submittedAt: ISODateTime;
  reviewedAt?: ISODateTime;
  reviewedByUserId?: UUID;
  reviewNote?: string;
  approvedOrganizationId?: UUID;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

// ─── Organization types ───────────────────────────────────────────────────────

export enum OrgKind {
  GOVERNMENT = 'GOVERNMENT',
  AGENCY = 'AGENCY',
  NGO = 'NGO',
  UTILITY = 'UTILITY',
  OTHER = 'OTHER',
}

export enum OrgJurisdiction {
  NATIONAL = 'NATIONAL',
  STATE = 'STATE',
  LGA = 'LGA',
  COMMUNITY = 'COMMUNITY',
}

export enum OrgMemberRole {
  OWNER = 'OWNER',
  ADMIN = 'ADMIN',
  STAFF = 'STAFF',
}

export enum AnnouncementStatus {
  DRAFT = 'DRAFT',
  PUBLISHED = 'PUBLISHED',
  ARCHIVED = 'ARCHIVED',
}

export enum ProjectStatus {
  PLANNED = 'PLANNED',
  ACTIVE = 'ACTIVE',
  PAUSED = 'PAUSED',
  COMPLETED = 'COMPLETED',
  CANCELLED = 'CANCELLED',
}

export enum AssignmentStatus {
  RECEIVED = 'RECEIVED',
  IN_PROGRESS = 'IN_PROGRESS',
  COMPLETED = 'COMPLETED',
  REJECTED = 'REJECTED',
}

export interface Organization {
  id: UUID;
  name: string;
  slug: string;
  kind: OrgKind;
  jurisdiction: OrgJurisdiction;
  state?: string;
  lga?: string;
  description?: string;
  logoUrl?: string;
  email?: string;
  phone?: string;
  website?: string;
  verified: boolean;
  memberCount: number;
  announcementCount: number;
  projectCount: number;
  assignmentCount: number;
  createdById: UUID;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface OrgMember {
  id: UUID;
  organizationId: UUID;
  userId: UUID;
  userName: string;
  userRole: UserRole;
  role: OrgMemberRole;
  joinedAt: ISODateTime;
}

/**
 * Returned by `GET /me/organizations` — each org the caller belongs to
 * paired with the membership row so the frontend knows the caller's
 * role in that org without a second request.
 */
export interface MyOrgMembership {
  organization: Organization;
  membership: OrgMember;
}

export interface Announcement {
  id: UUID;
  organizationId: UUID;
  title: string;
  body: string;
  status: AnnouncementStatus;
  publishedAt?: ISODateTime;
  authorId: UUID;
  authorName: string;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface Project {
  id: UUID;
  organizationId: UUID;
  title: string;
  description: string;
  status: ProjectStatus;
  startDate?: ISODateTime;
  expectedEndDate?: ISODateTime;
  budgetKobo?: number;
  communityId?: UUID;
  createdById: UUID;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface IssueAssignment {
  id: UUID;
  organizationId: UUID;
  issueId: UUID;
  status: AssignmentStatus;
  note?: string;
  assignedById: UUID;
  assignedByName: string;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface ProgressUpdate {
  id: UUID;
  organizationId: UUID;
  issueId?: UUID;
  projectId?: UUID;
  body: string;
  isPublic: boolean;
  authorId: UUID;
  authorName: string;
  createdAt: ISODateTime;
}

// ─── Consultations ───────────────────────────────────────────────────────────

export enum ConsultationStatus {
  DRAFT = 'DRAFT',
  PUBLISHED = 'PUBLISHED',
  CLOSED = 'CLOSED',
}

export enum ConsultationQuestionType {
  SHORT_TEXT = 'SHORT_TEXT',
  LONG_TEXT = 'LONG_TEXT',
  SINGLE_CHOICE = 'SINGLE_CHOICE',
  MULTI_CHOICE = 'MULTI_CHOICE',
  YES_NO = 'YES_NO',
}

/**
 * A structured feedback ask published by an organization.
 *
 * `communityId` is an audience label — it's shown to citizens but NOT
 * enforced on response submission. Any verified user can answer.
 */
export interface Consultation {
  id: UUID;
  organizationId: UUID;
  communityId?: UUID;
  title: string;
  summary: string;
  description: string;
  coverImageUrl?: string;
  status: ConsultationStatus;
  opensAt?: ISODateTime;
  closesAt?: ISODateTime;
  responseCount: number;
  authorId: UUID;
  authorName: string;
  publishedAt?: ISODateTime;
  closedAt?: ISODateTime;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

/**
 * A question inside a consultation. `options` is only meaningful for
 * SINGLE_CHOICE and MULTI_CHOICE. YES_NO answers are encoded as the
 * selection `"YES"` or `"NO"` on the answer.
 */
export interface ConsultationQuestion {
  id: UUID;
  consultationId: UUID;
  position: number;
  prompt: string;
  helpText?: string;
  type: ConsultationQuestionType;
  options: string[];
  required: boolean;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface ConsultationResponse {
  id: UUID;
  consultationId: UUID;
  userId: UUID;
  submittedAt: ISODateTime;
  createdAt: ISODateTime;
}

/**
 * A single question's answer inside a response. Exactly one of
 * `textValue` or `selections` carries data based on the question type.
 */
export interface ConsultationAnswer {
  id: UUID;
  responseId: UUID;
  questionId: UUID;
  textValue?: string;
  selections?: string[];
}

/**
 * Bundled shape returned by the admin `GET /consultations/:id/responses`
 * — one response with its answers grouped.
 */
export interface ConsultationResponseWithAnswers {
  response: ConsultationResponse;
  answers: ConsultationAnswer[];
}

/**
 * The "close the loop" primitive. One outcome per consultation;
 * re-publishing overwrites.
 */
export interface ConsultationOutcome {
  id: UUID;
  consultationId: UUID;
  summary: string;
  decisions: string;
  nextSteps: string;
  authorId: UUID;
  authorName: string;
  publishedAt: ISODateTime;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

/**
 * Per-question rollup returned by `GET /consultations/:id/analytics`.
 * `optionCounts` for choice + Yes/No questions; `textValues` for text
 * questions (capped at 100 samples server-side).
 */
export interface ConsultationAggregate {
  questionId: UUID;
  prompt: string;
  type: ConsultationQuestionType;
  answerCount: number;
  optionCounts?: Record<string, number>;
  textValues?: string[];
}

// ─── Consultation request DTOs ──────────────────────────────────────

export interface CreateConsultationInput {
  title: string;
  summary: string;
  description: string;
  coverImageUrl?: string;
  communityId?: UUID;
  opensAt?: ISODateTime;
  closesAt?: ISODateTime;
}

export type UpdateConsultationInput = Partial<CreateConsultationInput>;

export interface ConsultationQuestionInput {
  prompt: string;
  helpText?: string;
  type: ConsultationQuestionType;
  options?: string[];
  required?: boolean;
  /** Omit to append at the end of the current question list. */
  position?: number;
}

export interface ConsultationAnswerInput {
  questionId: UUID;
  textValue?: string;
  selections?: string[];
}

export interface SubmitConsultationResponseInput {
  answers: ConsultationAnswerInput[];
}

export interface ConsultationOutcomeInput {
  summary: string;
  decisions: string;
  nextSteps: string;
}

/** Map of questionId → new zero-based position. */
export type ConsultationQuestionOrdering = Record<UUID, number>;
