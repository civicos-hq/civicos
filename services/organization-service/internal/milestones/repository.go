package milestones

import (
	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindByCampaign(campaignID string) ([]domain.Milestone, error) {
	var list []domain.Milestone
	return list, r.db.Where("campaign_id = ?", campaignID).
		Order("position asc").Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Milestone, error) {
	var m domain.Milestone
	return &m, r.db.Where("id = ?", id).First(&m).Error
}

func (r *Repository) Create(m *domain.Milestone) error {
	return r.db.Create(m).Error
}

func (r *Repository) Update(id string, updates map[string]any) error {
	return r.db.Model(&domain.Milestone{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) Delete(id string) error {
	return r.db.Delete(&domain.Milestone{}, "id = ?", id).Error
}

// NextPosition returns the position to append at. Milestones are a spend
// plan read top-to-bottom, so append order is the default rather than
// making every client compute an index.
func (r *Repository) NextPosition(campaignID string) (int, error) {
	var max *int
	err := r.db.Model(&domain.Milestone{}).
		Where("campaign_id = ?", campaignID).
		Select("MAX(position)").Scan(&max).Error
	if err != nil || max == nil {
		return 1, err
	}
	return *max + 1, nil
}

// SumTargetsExcluding totals milestone targets for a campaign, optionally
// ignoring one milestone. The exclusion is what makes an edit checkable:
// when raising milestone #2's target we need the total *without* its old
// value to compare against the goal.
func (r *Repository) SumTargetsExcluding(campaignID, excludeID string) (int64, error) {
	q := r.db.Model(&domain.Milestone{}).Where("campaign_id = ?", campaignID)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	var total int64
	// COALESCE — SUM over zero rows is NULL and will not scan into int64.
	err := q.Select("COALESCE(SUM(target_minor), 0)").Scan(&total).Error
	return total, err
}

// Campaign loads the parent. Milestones authorize through their campaign
// (which knows its organization), so every write needs it.
func (r *Repository) Campaign(campaignID string) (*domain.Campaign, error) {
	var c domain.Campaign
	return &c, r.db.Where("id = ?", campaignID).First(&c).Error
}
