package domain

import "time"

type UserRole string

const (
	RoleCitizen         UserRole = "CITIZEN"
	RoleRepresentative  UserRole = "REPRESENTATIVE"
	RoleGovernmentAdmin UserRole = "GOVERNMENT_ADMIN"
	RoleNGO             UserRole = "NGO"
	RoleModerator       UserRole = "MODERATOR"
	RolePlatformAdmin   UserRole = "PLATFORM_ADMIN"
)

type RequestedAccountType string

const (
	AccountTypeCitizen        RequestedAccountType = "CITIZEN"
	AccountTypeRepresentative RequestedAccountType = "REPRESENTATIVE"
	AccountTypeOrganization   RequestedAccountType = "ORGANIZATION"
)

type ApprovalStatus string

const (
	ApprovalStatusNone         ApprovalStatus = "NONE"
	ApprovalStatusPending      ApprovalStatus = "PENDING"
	ApprovalStatusApproved     ApprovalStatus = "APPROVED"
	ApprovalStatusNeedsChanges ApprovalStatus = "NEEDS_CHANGES"
	ApprovalStatusRejected     ApprovalStatus = "REJECTED"
)

// User is the core identity entity.
// All IDs are UUIDs — never expose sequential database IDs.
type User struct {
	ID                string   `gorm:"type:uuid;primaryKey" json:"id"`
	Email             string   `gorm:"uniqueIndex;not null" json:"email"`
	Name              string   `gorm:"not null" json:"name"`
	PasswordHash      string   `gorm:"not null" json:"-"` // never serialised
	Role              UserRole `gorm:"type:varchar(30);default:'CITIZEN'" json:"role"`
	AvatarURL         *string  `json:"avatarUrl,omitempty"`
	ActiveCommunityID *string  `gorm:"type:uuid;column:community_id" json:"activeCommunityId,omitempty"`
	// PrimaryCommunityID is the user's home constituency — the one they can
	// create issues, petitions, and rep-profile edits in. Set on first join
	// and only changed via the change-primary endpoint (30-day cooldown).
	// Distinct from ActiveCommunityID, which is the community the user is
	// currently viewing/acting in for signatures, comments, and upvotes.
	PrimaryCommunityID        *string                   `gorm:"type:uuid;index" json:"primaryCommunityId,omitempty"`
	PrimaryCommunityChangedAt *time.Time                `json:"primaryCommunityChangedAt,omitempty"`
	Memberships               []UserCommunityMembership `gorm:"foreignKey:UserID" json:"-"`

	// RequestedAccountType captures the signup intent separately from the
	// currently-effective role. A pending representative or organization
	// applicant stays low-privilege until an admin approves the request.
	RequestedAccountType RequestedAccountType `gorm:"type:varchar(30);not null;default:'CITIZEN'" json:"requestedAccountType"`
	ApprovalStatus       ApprovalStatus       `gorm:"type:varchar(20);not null;default:'NONE';index" json:"approvalStatus"`
	ApprovalReviewedAt   *time.Time           `json:"approvalReviewedAt,omitempty"`
	ApprovalReviewedByID *string              `gorm:"type:uuid" json:"approvalReviewedById,omitempty"`
	ApprovalNote         *string              `json:"approvalNote,omitempty"`

	EmailVerified              bool       `gorm:"not null;default:false" json:"emailVerified"`
	EmailVerifiedAt            *time.Time `json:"emailVerifiedAt,omitempty"`
	EmailVerificationTokenHash *string    `gorm:"index" json:"-"`
	EmailVerificationExpiresAt *time.Time `json:"-"`

	PasswordResetTokenHash *string    `gorm:"index" json:"-"`
	PasswordResetExpiresAt *time.Time `json:"-"`

	// Moderation state — set by an admin, never by the user themselves.
	// When BannedAt is non-nil the user is blocked from all authenticated
	// actions (their JWT still works until it expires, so a ban is more
	// of a soft-delete for now — the refresh-token consumption path
	// enforces the block on next rotation).
	BannedAt   *time.Time `gorm:"index" json:"bannedAt,omitempty"`
	BanReason  *string    `json:"banReason,omitempty"`
	BannedByID *string    `gorm:"type:uuid" json:"bannedById,omitempty"`

	// DeletedAt is the user-initiated soft-delete timestamp. Once set,
	// the user cannot log in and cannot refresh; every authenticated
	// path checks this flag. Content authored before deletion stays in
	// place with Name replaced by a placeholder — the audit trail +
	// public record survive, PII is scrubbed.
	DeletedAt      *time.Time `gorm:"index" json:"deletedAt,omitempty"`
	DeletionReason *string    `json:"deletionReason,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PublicUser is the safe view returned to API consumers.
type PublicUser struct {
	ID                        string                      `json:"id"`
	Email                     string                      `json:"email"`
	Name                      string                      `json:"name"`
	Role                      UserRole                    `json:"role"`
	AvatarURL                 *string                     `json:"avatarUrl,omitempty"`
	ActiveCommunityID         *string                     `json:"activeCommunityId,omitempty"`
	PrimaryCommunityID        *string                     `json:"primaryCommunityId,omitempty"`
	PrimaryCommunityChangedAt *time.Time                  `json:"primaryCommunityChangedAt,omitempty"`
	Memberships               []PublicCommunityMembership `json:"memberships"`
	RequestedAccountType      RequestedAccountType        `json:"requestedAccountType"`
	ApprovalStatus            ApprovalStatus              `json:"approvalStatus"`
	ApprovalReviewedAt        *time.Time                  `json:"approvalReviewedAt,omitempty"`
	ApprovalReviewedByID      *string                     `json:"approvalReviewedById,omitempty"`
	ApprovalNote              *string                     `json:"approvalNote,omitempty"`
	EmailVerified             bool                        `json:"emailVerified"`
	EmailVerifiedAt           *time.Time                  `json:"emailVerifiedAt,omitempty"`
	CreatedAt                 time.Time                   `json:"createdAt"`
}

type UserCommunityMembership struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      string    `gorm:"type:uuid;not null;uniqueIndex:idx_user_community_membership" json:"userId"`
	CommunityID string    `gorm:"type:uuid;not null;uniqueIndex:idx_user_community_membership;index" json:"communityId"`
	JoinedAt    time.Time `gorm:"not null;index" json:"joinedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PublicCommunityMembership struct {
	CommunityID string    `json:"communityId"`
	JoinedAt    time.Time `json:"joinedAt"`
}

// RefreshToken records a single opaque refresh token. Rotation:
//
//   - Every /refresh call CONSUMES the presented token and issues a fresh one
//     in the same family. Consuming = setting ConsumedAt.
//   - FamilyID is stable across a rotation chain (typically = the initial
//     token's ID). It's what lets us nuke a whole session on replay.
//   - Presenting a token whose ConsumedAt is already set = replay = theft.
//     We revoke every row where FamilyID matches, forcing the attacker (and
//     legitimate user) to sign in again. This is the OWASP pattern.
//
// The raw token is 32 bytes of crypto/rand hex — never stored. We keep only
// SHA256(raw) in TokenHash so leaking the DB can't hijack live sessions.
type RefreshToken struct {
	ID         string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     string     `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash  string     `gorm:"uniqueIndex;not null" json:"-"`
	FamilyID   string     `gorm:"type:uuid;not null;index" json:"familyId"`
	ExpiresAt  time.Time  `gorm:"not null" json:"expiresAt"`
	ConsumedAt *time.Time `json:"consumedAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// IsUsable reports whether this token can still be exchanged. Callers should
// treat an "already consumed" result as a replay signal and revoke the family.
func (r *RefreshToken) IsUsable(now time.Time) bool {
	if r.RevokedAt != nil {
		return false
	}
	if r.ConsumedAt != nil {
		return false
	}
	return now.Before(r.ExpiresAt)
}

// ── Moderation infrastructure ────────────────────────────────────────────
// AuditLog is the immutable trail of administrative actions across the
// platform. Every PLATFORM_ADMIN write records a row here so that
// admin power is transparent and reviewable — a Phase 1 "Trust" objective
// baked in from day one.
//
// Cross-service note: community-service and organization-service can
// write to this table by re-declaring the model without AutoMigrate;
// identity-service is the source of truth for the schema.
type AuditLog struct {
	ID         string    `gorm:"type:uuid;primaryKey" json:"id"`
	ActorID    string    `gorm:"type:uuid;not null;index" json:"actorId"`
	ActorName  string    `gorm:"not null" json:"actorName"`
	ActorRole  string    `gorm:"not null" json:"actorRole"`
	Action     string    `gorm:"not null;index" json:"action"`
	TargetType string    `gorm:"not null;index" json:"targetType"`
	TargetID   string    `gorm:"type:uuid;not null;index" json:"targetId"`
	Metadata   string    `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	IPAddress  *string   `json:"ipAddress,omitempty"`
	UserAgent  *string   `json:"userAgent,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

type FlagReason string
type FlagStatus string
type FlaggableType string

const (
	FlagReasonSpam    FlagReason = "SPAM"
	FlagReasonAbuse   FlagReason = "ABUSE"
	FlagReasonMisinfo FlagReason = "MISINFO"
	FlagReasonHate    FlagReason = "HATE"
	FlagReasonOther   FlagReason = "OTHER"

	FlagStatusPending   FlagStatus = "PENDING"
	FlagStatusReviewed  FlagStatus = "REVIEWED"
	FlagStatusHidden    FlagStatus = "HIDDEN"
	FlagStatusDismissed FlagStatus = "DISMISSED"

	FlaggableIssue           FlaggableType = "ISSUE"
	FlaggableIssueComment    FlaggableType = "ISSUE_COMMENT"
	FlaggablePetition        FlaggableType = "PETITION"
	FlaggablePetitionComment FlaggableType = "PETITION_COMMENT"
	FlaggableRepComment      FlaggableType = "REPRESENTATIVE_COMMENT"
	FlaggableAnnouncement    FlaggableType = "ANNOUNCEMENT"
	FlaggableProgressUpdate  FlaggableType = "PROGRESS_UPDATE"
)

// ContentFlag is a citizen's report of content that violates the
// platform's rules. Compound uniqueness ({content_type, content_id,
// reporter_id}) enforces one flag per user per content — repeated
// signal escalates via the reporter's next action, not by mashing the
// button. Moderators consume the flag queue via /api/v1/flags; the
// resolution is another row in the AuditLog.
type ContentFlag struct {
	ID             string        `gorm:"type:uuid;primaryKey" json:"id"`
	ContentType    FlaggableType `gorm:"type:varchar(30);not null;uniqueIndex:idx_flag_dedup;index" json:"contentType"`
	ContentID      string        `gorm:"type:uuid;not null;uniqueIndex:idx_flag_dedup;index" json:"contentId"`
	ReporterID     string        `gorm:"type:uuid;not null;uniqueIndex:idx_flag_dedup" json:"reporterId"`
	ReporterName   string        `gorm:"not null" json:"reporterName"`
	Reason         FlagReason    `gorm:"type:varchar(20);not null" json:"reason"`
	Description    *string       `json:"description,omitempty"`
	Status         FlagStatus    `gorm:"type:varchar(20);default:'PENDING';index" json:"status"`
	ResolvedByID   *string       `gorm:"type:uuid" json:"resolvedById,omitempty"`
	ResolvedByName *string       `json:"resolvedByName,omitempty"`
	ResolutionNote *string       `json:"resolutionNote,omitempty"`
	ResolvedAt     *time.Time    `json:"resolvedAt,omitempty"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}

type ApplicationReviewEvent struct {
	ID              string               `gorm:"type:uuid;primaryKey" json:"id"`
	ApplicationKind RequestedAccountType `gorm:"type:varchar(30);not null;index:idx_application_review_lookup" json:"applicationKind"`
	ApplicationID   string               `gorm:"type:uuid;not null;index:idx_application_review_lookup" json:"applicationId"`
	ApplicantUserID string               `gorm:"type:uuid;not null;index" json:"applicantUserId"`
	ReviewerUserID  string               `gorm:"type:uuid;not null;index" json:"reviewerUserId"`
	ReviewerName    string               `gorm:"not null" json:"reviewerName"`
	Status          ApprovalStatus       `gorm:"type:varchar(20);not null;index" json:"status"`
	Note            *string              `json:"note,omitempty"`
	CreatedAt       time.Time            `gorm:"not null;index" json:"createdAt"`
}

// RepresentativeApplication stores the data a user submits when asking to be
// approved as a representative. The approval status is duplicated onto User as
// a quick auth/profile summary, while this record holds the reviewable payload.
type RepresentativeApplication struct {
	ID                string         `gorm:"type:uuid;primaryKey" json:"id"`
	UserID            string         `gorm:"type:uuid;not null;uniqueIndex" json:"userId"`
	Status            ApprovalStatus `gorm:"type:varchar(20);not null;default:'PENDING';index" json:"status"`
	FullName          string         `gorm:"not null" json:"fullName"`
	Title             string         `gorm:"not null" json:"title"`
	Position          string         `gorm:"not null" json:"position"`
	Constituency      string         `gorm:"not null" json:"constituency"`
	CommunityID       string         `gorm:"type:uuid;not null;index" json:"communityId"`
	Party             *string        `json:"party,omitempty"`
	Bio               *string        `json:"bio,omitempty"`
	AvatarURL         *string        `json:"avatarUrl,omitempty"`
	OfficialEmail     *string        `json:"officialEmail,omitempty"`
	OfficialPhone     *string        `json:"officialPhone,omitempty"`
	Website           *string        `json:"website,omitempty"`
	ProofURLs         []string       `gorm:"serializer:json" json:"proofUrls,omitempty"`
	SubmittedAt       time.Time      `gorm:"not null;index" json:"submittedAt"`
	ReviewedAt        *time.Time     `json:"reviewedAt,omitempty"`
	ReviewedByUserID  *string        `gorm:"type:uuid" json:"reviewedByUserId,omitempty"`
	ReviewNote        *string        `json:"reviewNote,omitempty"`
	ApprovedProfileID *string        `gorm:"type:uuid" json:"approvedProfileId,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

// OrganizationApplication stores a pending request to create an organization
// account. Approval later creates the actual organization and promotes the
// applicant's platform role as needed.
type OrganizationApplication struct {
	ID                     string         `gorm:"type:uuid;primaryKey" json:"id"`
	UserID                 string         `gorm:"type:uuid;not null;uniqueIndex" json:"userId"`
	Status                 ApprovalStatus `gorm:"type:varchar(20);not null;default:'PENDING';index" json:"status"`
	Name                   string         `gorm:"not null" json:"name"`
	Slug                   string         `gorm:"not null;index" json:"slug"`
	Kind                   string         `gorm:"type:varchar(20);not null" json:"kind"`
	Jurisdiction           string         `gorm:"type:varchar(20);not null" json:"jurisdiction"`
	State                  *string        `json:"state,omitempty"`
	LGA                    *string        `json:"lga,omitempty"`
	Description            *string        `json:"description,omitempty"`
	LogoURL                *string        `json:"logoUrl,omitempty"`
	OfficialEmail          *string        `json:"officialEmail,omitempty"`
	OfficialPhone          *string        `json:"officialPhone,omitempty"`
	Website                *string        `json:"website,omitempty"`
	ProofURLs              []string       `gorm:"serializer:json" json:"proofUrls,omitempty"`
	SubmittedAt            time.Time      `gorm:"not null;index" json:"submittedAt"`
	ReviewedAt             *time.Time     `json:"reviewedAt,omitempty"`
	ReviewedByUserID       *string        `gorm:"type:uuid" json:"reviewedByUserId,omitempty"`
	ReviewNote             *string        `json:"reviewNote,omitempty"`
	ApprovedOrganizationID *string        `gorm:"type:uuid" json:"approvedOrganizationId,omitempty"`
	CreatedAt              time.Time      `json:"createdAt"`
	UpdatedAt              time.Time      `json:"updatedAt"`
}

func (u *User) ToPublic() PublicUser {
	requestedType := u.RequestedAccountType
	if requestedType == "" {
		requestedType = AccountTypeCitizen
	}
	approvalStatus := u.ApprovalStatus
	if approvalStatus == "" {
		approvalStatus = ApprovalStatusNone
	}
	return PublicUser{
		ID:                        u.ID,
		Email:                     u.Email,
		Name:                      u.Name,
		Role:                      u.Role,
		AvatarURL:                 u.AvatarURL,
		ActiveCommunityID:         u.ActiveCommunityID,
		PrimaryCommunityID:        u.PrimaryCommunityID,
		PrimaryCommunityChangedAt: u.PrimaryCommunityChangedAt,
		Memberships:               ToPublicMemberships(u.Memberships),
		RequestedAccountType:      requestedType,
		ApprovalStatus:            approvalStatus,
		ApprovalReviewedAt:        u.ApprovalReviewedAt,
		ApprovalReviewedByID:      u.ApprovalReviewedByID,
		ApprovalNote:              u.ApprovalNote,
		EmailVerified:             u.EmailVerified,
		EmailVerifiedAt:           u.EmailVerifiedAt,
		CreatedAt:                 u.CreatedAt,
	}
}

func ToPublicMemberships(memberships []UserCommunityMembership) []PublicCommunityMembership {
	if len(memberships) == 0 {
		return []PublicCommunityMembership{}
	}
	out := make([]PublicCommunityMembership, 0, len(memberships))
	for _, membership := range memberships {
		out = append(out, PublicCommunityMembership{
			CommunityID: membership.CommunityID,
			JoinedAt:    membership.JoinedAt,
		})
	}
	return out
}
