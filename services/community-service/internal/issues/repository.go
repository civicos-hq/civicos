package issues

import (
	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindAll(communityID, status string) ([]domain.Issue, error) {
	var list []domain.Issue
	q := r.db.Order("created_at desc")
	if communityID != "" { q = q.Where("community_id = ?", communityID) }
	if status != "" { q = q.Where("status = ?", status) }
	return list, q.Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Issue, error) {
	var issue domain.Issue
	return &issue, r.db.Where("id = ?", id).First(&issue).Error
}

func (r *Repository) Create(issue *domain.Issue) error {
	return r.db.Create(issue).Error
}

func (r *Repository) IncrementUpvote(id string) error {
	return r.db.Model(&domain.Issue{}).Where("id = ?", id).
		UpdateColumn("upvote_count", gorm.Expr("upvote_count + 1")).Error
}

func (r *Repository) UpdateStatus(id string, status domain.IssueStatus) error {
	return r.db.Model(&domain.Issue{}).Where("id = ?", id).
		Update("status", status).Error
}
