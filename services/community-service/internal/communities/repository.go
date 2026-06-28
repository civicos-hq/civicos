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

func (r *Repository) FindByID(id string) (*domain.Community, error) {
	var c domain.Community
	return &c, r.db.Where("id = ?", id).First(&c).Error
}

func (r *Repository) Create(c *domain.Community) error {
	return r.db.Create(c).Error
}
