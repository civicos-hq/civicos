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
  communityId?: UUID;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
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
  communityId: UUID;
  responseRate: number;
  followerCount: number;
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
  createdAt: ISODateTime;
}

export interface IssueComment extends Comment {
  issueId: UUID;
}

export interface PetitionComment extends Comment {
  petitionId: UUID;
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
