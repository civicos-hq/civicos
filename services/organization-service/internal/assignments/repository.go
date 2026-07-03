package assignments

import (
	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindByOrg(orgID, status string) ([]domain.IssueAssignment, error) {
	q := r.db.Where("organization_id = ?", orgID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var list []domain.IssueAssignment
	return list, q.Order("created_at desc").Find(&list).Error
}

func (r *Repository) FindByIssue(issueID string) ([]domain.IssueAssignment, error) {
	var list []domain.IssueAssignment
	return list, r.db.Where("issue_id = ?", issueID).Order("created_at desc").Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.IssueAssignment, error) {
	var a domain.IssueAssignment
	return &a, r.db.Where("id = ?", id).First(&a).Error
}

func (r *Repository) FindExact(orgID, issueID string) (*domain.IssueAssignment, error) {
	var a domain.IssueAssignment
	return &a, r.db.Where("organization_id = ? AND issue_id = ?", orgID, issueID).First(&a).Error
}

func (r *Repository) Create(a *domain.IssueAssignment) error {
	return r.db.Create(a).Error
}

func (r *Repository) Update(id string, updates map[string]any) error {
	return r.db.Model(&domain.IssueAssignment{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) Delete(id string) error {
	return r.db.Delete(&domain.IssueAssignment{}, "id = ?", id).Error
}

func (r *Repository) BumpOrgCount(orgID string, delta int) error {
	return r.db.Model(&domain.Organization{}).Where("id = ?", orgID).
		Update("assignment_count", gorm.Expr("assignment_count + ?", delta)).Error
}
