package communities

import (
	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindAll() ([]domain.Community, error) {
	var list []domain.Community
	return list, r.db.Order("name asc").Find(&list).Error
}

// FindByLocation narrows the list to a state and/or LGA. Empty string means
// "no constraint on this field" so the caller can filter by state only, LGA
// only (unlikely), or both.
func (r *Repository) FindByLocation(state, lga string) ([]domain.Community, error) {
	q := r.db.Model(&domain.Community{})
	if state != "" {
		q = q.Where("state = ?", state)
	}
	if lga != "" {
		q = q.Where("lga = ?", lga)
	}
	var list []domain.Community
	return list, q.Order("name asc").Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Community, error) {
	var c domain.Community
	return &c, r.db.Where("id = ?", id).First(&c).Error
}

func (r *Repository) Create(c *domain.Community) error {
	return r.db.Create(c).Error
}
