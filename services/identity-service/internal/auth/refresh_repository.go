package auth

import (
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"gorm.io/gorm"
)

// RefreshRepository persists opaque refresh tokens for rotation + replay
// detection. Deliberately separate from Repository so the mutation surface
// stays obvious in code review: it exposes only three real primitives —
// create, consume, revoke.
type RefreshRepository struct {
	db *gorm.DB
}

func NewRefreshRepository(db *gorm.DB) *RefreshRepository {
	return &RefreshRepository{db: db}
}

func (r *RefreshRepository) Create(t *domain.RefreshToken) error {
	return r.db.Create(t).Error
}

// FindByHash returns the (unconsumed / consumed / revoked) row for this token
// hash. Returning the full record — even consumed ones — is what lets the
// service detect replay: a "found but ConsumedAt != nil" result means an
// attacker reused a rotated token, and we should revoke the whole family.
func (r *RefreshRepository) FindByHash(hash string) (*domain.RefreshToken, error) {
	var tok domain.RefreshToken
	if err := r.db.Where("token_hash = ?", hash).First(&tok).Error; err != nil {
		return nil, err
	}
	return &tok, nil
}

// Consume marks a single token as consumed. Idempotent-safe: the WHERE clause
// filters out rows whose ConsumedAt is already set, so a racing double-refresh
// results in exactly one winner. Callers should re-read the row after this
// call and treat "not affected" as a replay signal.
func (r *RefreshRepository) Consume(id string, at time.Time) (int64, error) {
	res := r.db.Model(&domain.RefreshToken{}).
		Where("id = ? AND consumed_at IS NULL AND revoked_at IS NULL", id).
		Update("consumed_at", at)
	return res.RowsAffected, res.Error
}

// RevokeFamily marks every token in the family as revoked. Used both on
// intentional logout and on replay detection. Only touches rows that aren't
// already revoked, so the RevokedAt reflects the first revocation.
func (r *RefreshRepository) RevokeFamily(familyID string, at time.Time) error {
	return r.db.Model(&domain.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", at).Error
}

// RevokeAllForUser slams every live refresh token for a user. Used on
// account deletion — the user has decided to leave, so no live session
// should survive.
func (r *RefreshRepository) RevokeAllForUser(userID string, at time.Time) error {
	return r.db.Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", at).Error
}
