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
