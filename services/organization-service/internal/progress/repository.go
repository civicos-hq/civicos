package progress

import (
	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type ListFilters struct {
	OrgID     string
	IssueID   string
	ProjectID string
	PublicOnly bool
}

// See announcements/repository.go for the hide-filter rationale.
const hideFilter = `id NOT IN (SELECT content_id FROM content_flags WHERE content_type = 'PROGRESS_UPDATE' AND status = 'HIDDEN')`

func (r *Repository) Find(f ListFilters) ([]domain.ProgressUpdate, error) {
	q := r.db.Model(&domain.ProgressUpdate{}).Where(hideFilter)
	if f.OrgID != "" {
		q = q.Where("organization_id = ?", f.OrgID)
	}
	if f.IssueID != "" {
		q = q.Where("issue_id = ?", f.IssueID)
	}
	if f.ProjectID != "" {
		q = q.Where("project_id = ?", f.ProjectID)
	}
	if f.PublicOnly {
		q = q.Where("is_public = ?", true)
	}
	var list []domain.ProgressUpdate
	return list, q.Order("created_at desc").Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.ProgressUpdate, error) {
	// Deep-link protection — see announcements/repository.go for the
	// rationale.
	var p domain.ProgressUpdate
	return &p, r.db.
		Where("id = ?", id).
		Where(hideFilter).
		First(&p).Error
}

func (r *Repository) Create(p *domain.ProgressUpdate) error {
	return r.db.Create(p).Error
}

func (r *Repository) Delete(id string) error {
	return r.db.Delete(&domain.ProgressUpdate{}, "id = ?", id).Error
}
