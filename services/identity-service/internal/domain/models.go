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
