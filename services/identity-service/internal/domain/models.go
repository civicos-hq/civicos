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
	CommunityID  *string   `json:"communityId,omitempty"`

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
	UserID     string     `gorm:"not null;index" json:"userId"`
	TokenHash  string     `gorm:"uniqueIndex;not null" json:"-"`
	FamilyID   string     `gorm:"not null;index" json:"familyId"`
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
