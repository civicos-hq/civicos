package domain

import "time"

type UserRole string

const (
	RoleCitizen        UserRole = "CITIZEN"
	RoleRepresentative UserRole = "REPRESENTATIVE"
	RoleGovernmentAdmin UserRole = "GOVERNMENT_ADMIN"
	RoleNGO            UserRole = "NGO"
	RoleModerator      UserRole = "MODERATOR"
	RolePlatformAdmin  UserRole = "PLATFORM_ADMIN"
)

// User is the core identity entity.
// All IDs are UUIDs — never expose sequential database IDs.
type User struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	Name         string    `gorm:"not null" json:"name"`
	PasswordHash string    `gorm:"not null" json:"-"` // never serialised
	Role         UserRole  `gorm:"type:varchar(30);default:'CITIZEN'" json:"role"`
	AvatarURL    *string   `json:"avatarUrl,omitempty"`
	CommunityID  *string   `gorm:"type:uuid" json:"communityId,omitempty"`

	EmailVerified                 bool       `gorm:"not null;default:false" json:"emailVerified"`
	EmailVerifiedAt               *time.Time `json:"emailVerifiedAt,omitempty"`
	EmailVerificationTokenHash    *string    `gorm:"index" json:"-"`
	EmailVerificationExpiresAt    *time.Time `json:"-"`

	PasswordResetTokenHash        *string    `gorm:"index" json:"-"`
	PasswordResetExpiresAt        *time.Time `json:"-"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PublicUser is the safe view returned to API consumers.
type PublicUser struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	Name            string     `json:"name"`
	Role            UserRole   `json:"role"`
	AvatarURL       *string    `json:"avatarUrl,omitempty"`
	CommunityID     *string    `json:"communityId,omitempty"`
	EmailVerified   bool       `json:"emailVerified"`
	EmailVerifiedAt *time.Time `json:"emailVerifiedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// RefreshToken records a single opaque refresh token. Rotation:
//
//	- Every /refresh call CONSUMES the presented token and issues a fresh one
//	  in the same family. Consuming = setting ConsumedAt.
//	- FamilyID is stable across a rotation chain (typically = the initial
//	  token's ID). It's what lets us nuke a whole session on replay.
//	- Presenting a token whose ConsumedAt is already set = replay = theft.
//	  We revoke every row where FamilyID matches, forcing the attacker (and
//	  legitimate user) to sign in again. This is the OWASP pattern.
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

	FlaggableIssue          FlaggableType = "ISSUE"
	FlaggableIssueComment   FlaggableType = "ISSUE_COMMENT"
	FlaggablePetition       FlaggableType = "PETITION"
	FlaggablePetitionComment FlaggableType = "PETITION_COMMENT"
	FlaggableRepComment     FlaggableType = "REPRESENTATIVE_COMMENT"
	FlaggableAnnouncement   FlaggableType = "ANNOUNCEMENT"
	FlaggableProgressUpdate FlaggableType = "PROGRESS_UPDATE"
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

func (u *User) ToPublic() PublicUser {
	return PublicUser{
		ID:              u.ID,
		Email:           u.Email,
		Name:            u.Name,
		Role:            u.Role,
		AvatarURL:       u.AvatarURL,
		CommunityID:     u.CommunityID,
		EmailVerified:   u.EmailVerified,
		EmailVerifiedAt: u.EmailVerifiedAt,
		CreatedAt:       u.CreatedAt,
	}
}
