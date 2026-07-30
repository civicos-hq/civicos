package campaigns

import (
	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type ListFilters struct {
	OrgID       string
	CommunityID string
	Category    string
	Status      string
	State       string
	LGA         string
	// PublicOnly restricts results to statuses a citizen may see. Phase 1
	// has no public route, but the filter lives here so the public list in
	// Phase 4 cannot accidentally leak drafts or rejected campaigns — the
	// allow-list is one place, not scattered across handlers.
	PublicOnly bool
	// EmergencyOnly powers the admin review queue's fast path for the
	// spec's "Approve emergency campaigns" capability.
	EmergencyOnly bool
}

// publicStatuses is the allow-list for citizen-visible campaigns.
// Deliberately an allow-list rather than a deny-list: a new status added
// later defaults to hidden, which is the safe direction for a surface that
// will eventually ask people for money.
var publicStatuses = []domain.CampaignStatus{
	domain.CampaignPublished,
	domain.CampaignFunded,
	domain.CampaignCompleted,
	domain.CampaignReported,
}

func (r *Repository) Find(f ListFilters) ([]domain.Campaign, error) {
	q := r.db.Model(&domain.Campaign{})
	if f.OrgID != "" {
		q = q.Where("organization_id = ?", f.OrgID)
	}
	if f.CommunityID != "" {
		q = q.Where("community_id = ?", f.CommunityID)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.State != "" {
		q = q.Where("state = ?", f.State)
	}
	if f.LGA != "" {
		q = q.Where("lga = ?", f.LGA)
	}
	if f.PublicOnly {
		q = q.Where("status IN ?", publicStatuses)
	}
	if f.EmergencyOnly {
		q = q.Where("is_emergency = ?", true)
	}
	var list []domain.Campaign
	return list, q.Order("created_at desc").Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Campaign, error) {
	var c domain.Campaign
	return &c, r.db.Where("id = ?", id).First(&c).Error
}

func (r *Repository) FindBySlug(slug string) (*domain.Campaign, error) {
	var c domain.Campaign
	return &c, r.db.Where("slug = ?", slug).First(&c).Error
}

func (r *Repository) Create(c *domain.Campaign) error {
	return r.db.Create(c).Error
}

func (r *Repository) Update(id string, updates map[string]any) error {
	return r.db.Model(&domain.Campaign{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) Delete(id string) error {
	return r.db.Delete(&domain.Campaign{}, "id = ?", id).Error
}

// SlugExists reports whether a slug is taken. Used to suffix duplicates at
// create time rather than surfacing a raw unique-constraint violation.
func (r *Repository) SlugExists(slug string) (bool, error) {
	var n int64
	err := r.db.Model(&domain.Campaign{}).Where("slug = ?", slug).Count(&n).Error
	return n > 0, err
}

// Org loads the owning organization. Campaigns may be drafted by any org,
// but only a funding-eligible one can be submitted for review — the spec's
// "Trust First" principle, enforced server-side rather than assumed from
// the UI. The service calls Organization.FundingEligible() on the result so
// the eligibility rule lives in one place.
func (r *Repository) Org(orgID string) (*domain.Organization, error) {
	var org domain.Organization
	if err := r.db.Where("id = ?", orgID).First(&org).Error; err != nil {
		return nil, err
	}
	return &org, nil
}

// OrgNames resolves organization display names in one query, so the public
// list doesn't issue a lookup per row.
func (r *Repository) OrgNames(ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID   string
		Name string
	}
	if err := r.db.Model(&domain.Organization{}).
		Select("id, name").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row.Name
	}
	return out, nil
}

// MilestonesFor loads a campaign's spend plan for the public detail page.
// Lives here rather than crossing into the milestones package so the public
// read is a single dependency-free path.
func (r *Repository) MilestonesFor(campaignID string) ([]domain.Milestone, error) {
	var list []domain.Milestone
	return list, r.db.Where("campaign_id = ?", campaignID).
		Order("position asc").Find(&list).Error
}

// CountMilestones and SumMilestoneTargets back the submit-for-review
// checks. Kept on this repository (rather than reaching into the
// milestones package) so the campaigns service has no cross-package
// dependency for a read it needs on its own hot path.
func (r *Repository) CountMilestones(campaignID string) (int64, error) {
	var n int64
	err := r.db.Model(&domain.Milestone{}).Where("campaign_id = ?", campaignID).Count(&n).Error
	return n, err
}

func (r *Repository) SumMilestoneTargets(campaignID string) (int64, error) {
	// COALESCE because SUM over zero rows is NULL, which will not scan
	// into int64.
	var total int64
	err := r.db.Model(&domain.Milestone{}).
		Where("campaign_id = ?", campaignID).
		Select("COALESCE(SUM(target_minor), 0)").
		Scan(&total).Error
	return total, err
}
