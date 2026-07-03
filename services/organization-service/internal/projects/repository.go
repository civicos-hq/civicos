package projects

import (
	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type ListFilters struct {
	OrgID       string
	CommunityID string
	Status      string
}

func (r *Repository) Find(f ListFilters) ([]domain.Project, error) {
	q := r.db.Model(&domain.Project{})
	if f.OrgID != "" {
		q = q.Where("organization_id = ?", f.OrgID)
	}
	if f.CommunityID != "" {
		q = q.Where("community_id = ?", f.CommunityID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var list []domain.Project
	return list, q.Order("created_at desc").Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Project, error) {
	var p domain.Project
	return &p, r.db.Where("id = ?", id).First(&p).Error
}

func (r *Repository) Create(p *domain.Project) error {
	return r.db.Create(p).Error
}

func (r *Repository) Update(id string, updates map[string]any) error {
	return r.db.Model(&domain.Project{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) Delete(id string) error {
	return r.db.Delete(&domain.Project{}, "id = ?", id).Error
}

func (r *Repository) BumpOrgCount(orgID string, delta int) error {
	return r.db.Model(&domain.Organization{}).Where("id = ?", orgID).
		Update("project_count", gorm.Expr("project_count + ?", delta)).Error
}
