package notifications

import (
	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListByUser(userID string, limit int) ([]domain.Notification, error) {
	var list []domain.Notification
	q := r.db.Where("user_id = ?", userID).Order("created_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return list, q.Find(&list).Error
}

func (r *Repository) CountUnread(userID string) (int64, error) {
	var n int64
	err := r.db.Model(&domain.Notification{}).
		Where("user_id = ? AND read = false", userID).
		Count(&n).Error
	return n, err
}

func (r *Repository) Create(n *domain.Notification) error {
	return r.db.Create(n).Error
}

func (r *Repository) MarkRead(id, userID string) error {
	return r.db.Model(&domain.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read", true).Error
}

func (r *Repository) MarkAllRead(userID string) error {
	return r.db.Model(&domain.Notification{}).
		Where("user_id = ? AND read = false", userID).
		Update("read", true).Error
}
