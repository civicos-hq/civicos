package spend

import (
	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(rec *domain.SpendRecord) error { return r.db.Create(rec).Error }

func (r *Repository) Get(id string) (*domain.SpendRecord, error) {
	var rec domain.SpendRecord
	return &rec, r.db.Where("id = ?", id).First(&rec).Error
}

func (r *Repository) Update(rec *domain.SpendRecord) error { return r.db.Save(rec).Error }

func (r *Repository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&domain.SpendRecord{}).Error
}

// ListForCampaign returns spend newest-spent first. Ordered by SpentAt, not
// CreatedAt: a donor reading the account cares when the money left, not when
// the organization got round to typing it in.
func (r *Repository) ListForCampaign(campaignID string) ([]domain.SpendRecord, error) {
	var list []domain.SpendRecord
	return list, r.db.Where("campaign_id = ?", campaignID).
		Order("spent_at desc, created_at desc").Find(&list).Error
}

func (r *Repository) Campaign(id string) (*domain.Campaign, error) {
	var c domain.Campaign
	return &c, r.db.Where("id = ?", id).First(&c).Error
}

func (r *Repository) Milestone(id string) (*domain.Milestone, error) {
	var m domain.Milestone
	return &m, r.db.Where("id = ?", id).First(&m).Error
}
