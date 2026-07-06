package auth

import (
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository handles all User database queries for the auth domain.
// It is injected into AuthService — never instantiated directly inside services.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *Repository) FindByID(id string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *Repository) Create(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *Repository) CreateRegistration(
	user *domain.User,
	repApp *domain.RepresentativeApplication,
	orgApp *domain.OrganizationApplication,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if repApp != nil {
			if err := tx.Create(repApp).Error; err != nil {
				return err
			}
		}
		if orgApp != nil {
			if err := tx.Create(orgApp).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) CreateNotification(userID, title, body string, linkURL *string) error {
	row := map[string]any{
		"id":         uuid.New().String(),
		"type":       "SYSTEM",
		"title":      title,
		"body":       body,
		"read":       false,
		"user_id":    userID,
		"created_at": time.Now().UTC(),
	}
	if linkURL != nil && *linkURL != "" {
		row["link_url"] = *linkURL
	}
	return r.db.Table("notifications").Create(row).Error
}

func (r *Repository) UpdateCommunity(userID, communityID string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Update("community_id", communityID).Error
}

// UpdateProfile patches the user's name and/or email. Empty strings are
// treated as "leave unchanged" so the caller can opt into partial updates.
func (r *Repository) UpdateProfile(userID, name, email string) error {
	updates := map[string]any{}
	if name != "" {
		updates["name"] = name
	}
	if email != "" {
		updates["email"] = email
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Updates(updates).Error
}

// SetVerificationToken stores a hashed token + expiry. Always overwrites —
// resending invalidates any earlier outstanding link.
func (r *Repository) SetVerificationToken(userID, tokenHash string, expiresAt time.Time) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Updates(map[string]any{
		"email_verification_token_hash": tokenHash,
		"email_verification_expires_at": expiresAt,
	}).Error
}

func (r *Repository) FindByVerificationTokenHash(tokenHash string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("email_verification_token_hash = ?", tokenHash).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// MarkVerified flips the verified bit, stamps the time, and clears the
// outstanding token in a single update so the link can't be replayed.
func (r *Repository) MarkVerified(userID string) error {
	now := time.Now().UTC()
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Updates(map[string]any{
		"email_verified":                true,
		"email_verified_at":             now,
		"email_verification_token_hash": gorm.Expr("NULL"),
		"email_verification_expires_at": gorm.Expr("NULL"),
	}).Error
}

// SetPasswordResetToken stores the hashed reset token + expiry. Overwrites
// any previous one so a second "forgot password" request invalidates the
// earlier link.
func (r *Repository) SetPasswordResetToken(userID, tokenHash string, expiresAt time.Time) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Updates(map[string]any{
		"password_reset_token_hash": tokenHash,
		"password_reset_expires_at": expiresAt,
	}).Error
}

func (r *Repository) FindByPasswordResetTokenHash(tokenHash string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("password_reset_token_hash = ?", tokenHash).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// ResetPassword replaces the password hash and clears the outstanding reset
// token in one update so the link is strictly single-use.
func (r *Repository) ResetPassword(userID, newPasswordHash string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Updates(map[string]any{
		"password_hash":             newPasswordHash,
		"password_reset_token_hash": gorm.Expr("NULL"),
		"password_reset_expires_at": gorm.Expr("NULL"),
	}).Error
}

// SoftDelete anonymizes PII and stamps deleted_at in a single update.
// The row survives so authorship links (issues, petitions, comments)
// keep pointing at a valid FK. Post-delete queries see:
//   - email = "deleted-<uuid>@civicos.deleted" (unique, non-recoverable)
//   - name  = "[Deleted user]"
//   - avatar_url + community_id = NULL
//   - password_hash = argon2-style dummy (login refuses anyway)
//   - deleted_at, deletion_reason recorded
func (r *Repository) SoftDelete(userID string, reason *string) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"deleted_at":                    now,
		"email":                         "deleted-" + userID + "@civicos.deleted",
		"name":                          "[Deleted user]",
		"password_hash":                 "x", // never matches bcrypt.CompareHashAndPassword
		"avatar_url":                    gorm.Expr("NULL"),
		"community_id":                  gorm.Expr("NULL"),
		"email_verification_token_hash": gorm.Expr("NULL"),
		"email_verification_expires_at": gorm.Expr("NULL"),
		"password_reset_token_hash":     gorm.Expr("NULL"),
		"password_reset_expires_at":     gorm.Expr("NULL"),
	}
	if reason != nil {
		updates["deletion_reason"] = *reason
	}
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Updates(updates).Error
}
