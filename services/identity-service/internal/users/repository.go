package users

import (
	"strings"

	"github.com/civicos/identity-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type ListFilters struct {
	Search string // matches email or name
	Role   string
	Banned string // "true" | "false" | "" (any)
	Limit  int
	Offset int
}

func (r *Repository) Find(f ListFilters) ([]domain.User, int64, error) {
	q := r.db.Model(&domain.User{})
	if f.Search != "" {
		term := "%" + strings.ToLower(f.Search) + "%"
		q = q.Where("LOWER(email) LIKE ? OR LOWER(name) LIKE ?", term, term)
	}
	if f.Role != "" {
		q = q.Where("role = ?", f.Role)
	}
	switch f.Banned {
	case "true":
		q = q.Where("banned_at IS NOT NULL")
	case "false":
		q = q.Where("banned_at IS NULL")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if f.Limit <= 0 {
		f.Limit = 25
	}
	var list []domain.User
	if err := q.
		Preload("Memberships").
		Order("created_at desc").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) FindByID(id string) (*domain.User, error) {
	var u domain.User
	return &u, r.db.Preload("Memberships").Where("id = ?", id).First(&u).Error
}

func (r *Repository) Update(id string, updates map[string]any) error {
	return r.db.Model(&domain.User{}).Where("id = ?", id).Updates(updates).Error
}
