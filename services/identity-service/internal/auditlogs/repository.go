package auditlogs

import (
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type ListFilters struct {
	ActorID    string
	Action     string // prefix match — 'flag.' catches all flag actions
	TargetType string
	TargetID   string
	Since      *time.Time
	Until      *time.Time
	Limit      int
	Offset     int
}

func (r *Repository) Find(f ListFilters) ([]domain.AuditLog, int64, error) {
	q := r.db.Model(&domain.AuditLog{})
	if f.ActorID != "" {
		q = q.Where("actor_id = ?", f.ActorID)
	}
	if f.Action != "" {
		q = q.Where("action LIKE ?", f.Action+"%")
	}
	if f.TargetType != "" {
		q = q.Where("target_type = ?", f.TargetType)
	}
	if f.TargetID != "" {
		q = q.Where("target_id = ?", f.TargetID)
	}
	if f.Since != nil {
		q = q.Where("created_at >= ?", *f.Since)
	}
	if f.Until != nil {
		q = q.Where("created_at <= ?", *f.Until)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if f.Limit <= 0 {
		f.Limit = 50
	}
	var list []domain.AuditLog
	if err := q.
		Order("created_at desc").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
