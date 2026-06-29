package representatives

import (
	"strings"
	"time"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindAll(communityID string) ([]domain.Representative, error) {
	var list []domain.Representative
	q := r.db.Order("name asc")
	if communityID != "" {
		q = q.Where("community_id = ?", communityID)
	}
	return list, q.Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Representative, error) {
	var rep domain.Representative
	return &rep, r.db.Where("id = ?", id).First(&rep).Error
}

func (r *Repository) Create(rep *domain.Representative) error {
	return r.db.Create(rep).Error
}

func (r *Repository) Update(id string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&domain.Representative{}).Where("id = ?", id).Updates(updates).Error
}

// AddFollow inserts a follow row and bumps the rep's follower_count. Idempotent.
func (r *Repository) AddFollow(repID, userID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing domain.RepresentativeFollower
		if err := tx.Where("representative_id = ? AND user_id = ?", repID, userID).
			First(&existing).Error; err == nil {
			return nil
		}
		follow := domain.RepresentativeFollower{
			ID:               uuid.New().String(),
			RepresentativeID: repID,
			UserID:           userID,
			CreatedAt:        time.Now(),
		}
		if err := tx.Create(&follow).Error; err != nil {
			low := strings.ToLower(err.Error())
			if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
				return nil
			}
			return err
		}
		return tx.Model(&domain.Representative{}).Where("id = ?", repID).
			UpdateColumn("follower_count", gorm.Expr("follower_count + 1")).Error
	})
}

// RemoveFollow deletes the follow row and decrements follower_count if present.
func (r *Repository) RemoveFollow(repID, userID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("representative_id = ? AND user_id = ?", repID, userID).
			Delete(&domain.RepresentativeFollower{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		return tx.Model(&domain.Representative{}).Where("id = ?", repID).
			UpdateColumn("follower_count", gorm.Expr("GREATEST(follower_count - 1, 0)")).Error
	})
}

// FindFollowedIDs returns the rep IDs the given user follows.
func (r *Repository) FindFollowedIDs(userID string) ([]string, error) {
	var ids []string
	err := r.db.Model(&domain.RepresentativeFollower{}).
		Where("user_id = ?", userID).
		Pluck("representative_id", &ids).Error
	return ids, err
}

// FindFollowerIDs returns the user IDs following the given rep — used when
// fanning out notifications for official responses.
func (r *Repository) FindFollowerIDs(repID string) ([]string, error) {
	var ids []string
	err := r.db.Model(&domain.RepresentativeFollower{}).
		Where("representative_id = ?", repID).
		Pluck("user_id", &ids).Error
	return ids, err
}

func (r *Repository) ListComments(repID string) ([]domain.RepresentativeComment, error) {
	var list []domain.RepresentativeComment
	return list, r.db.Where("representative_id = ?", repID).
		Order("created_at asc").Find(&list).Error
}

func (r *Repository) AddComment(comment *domain.RepresentativeComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Representative{}).Where("id = ?", comment.RepresentativeID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	})
}
