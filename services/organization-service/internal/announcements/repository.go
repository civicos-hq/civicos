package announcements

import (
	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// hideFilter drops any announcement that has a HIDDEN moderator flag.
// content_flags lives in identity-service's schema; the shared-DB
// architecture lets us cross-reference it. If services move to isolated
// DBs later, this becomes an HTTP call or a NATS-fed materialized view.
const hideFilter = `id NOT IN (SELECT content_id FROM content_flags WHERE content_type = 'ANNOUNCEMENT' AND status = 'HIDDEN')`

func (r *Repository) FindByOrg(orgID string, includeDrafts bool) ([]domain.Announcement, error) {
	q := r.db.Where("organization_id = ?", orgID).Where(hideFilter)
	if !includeDrafts {
		q = q.Where("status = ?", domain.AnnouncementPublished)
	}
	var list []domain.Announcement
	return list, q.Order("COALESCE(published_at, created_at) desc").Find(&list).Error
}

func (r *Repository) FindPublished(limit int) ([]domain.Announcement, error) {
	var list []domain.Announcement
	q := r.db.
		Where("status = ?", domain.AnnouncementPublished).
		Where(hideFilter).
		Order("published_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return list, q.Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Announcement, error) {
	var a domain.Announcement
	return &a, r.db.Where("id = ?", id).First(&a).Error
}

func (r *Repository) Create(a *domain.Announcement) error {
	return r.db.Create(a).Error
}

func (r *Repository) Update(id string, updates map[string]any) error {
	return r.db.Model(&domain.Announcement{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) Delete(id string) error {
	return r.db.Delete(&domain.Announcement{}, "id = ?", id).Error
}

func (r *Repository) BumpOrgCount(orgID string, delta int) error {
	return r.db.Model(&domain.Organization{}).Where("id = ?", orgID).
		Update("announcement_count", gorm.Expr("announcement_count + ?", delta)).Error
}
