package repannouncements

import (
	"errors"

	"github.com/civicos/community-service/internal/domain"
	"github.com/civicos/community-service/internal/moderation"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Representative(id string) (*domain.Representative, error) {
	var rep domain.Representative
	if err := r.db.Where("id = ?", id).First(&rep).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rep, nil
}

func (r *Repository) Create(a *domain.RepresentativeAnnouncement) error {
	return r.db.Create(a).Error
}

func (r *Repository) FindByID(id string) (*domain.RepresentativeAnnouncement, error) {
	var a domain.RepresentativeAnnouncement
	if err := r.db.Where("id = ?", id).First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *Repository) Update(id string, fields map[string]any) error {
	return r.db.Model(&domain.RepresentativeAnnouncement{}).Where("id = ?", id).Updates(fields).Error
}

func (r *Repository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&domain.RepresentativeAnnouncement{}).Error
}

// ListPublic returns only what a citizen may see. Drafts have never been
// public and archived ones have been withdrawn; an allow-list rather than a
// deny-list, so a status added later defaults to hidden.
func (r *Repository) ListPublic(repID string) ([]domain.RepresentativeAnnouncement, error) {
	out := []domain.RepresentativeAnnouncement{}
	err := r.db.Where("representative_id = ? AND status = ?", repID, domain.AnnouncementPublished).
		Order("published_at DESC").
		Find(&out).Error
	return out, err
}

// ListAll is the owner's view. Ordered by creation rather than publication so
// drafts, which have no published_at, do not sort to the bottom where they
// would be forgotten.
func (r *Repository) ListAll(repID string) ([]domain.RepresentativeAnnouncement, error) {
	out := []domain.RepresentativeAnnouncement{}
	err := r.db.Where("representative_id = ?", repID).
		Order("created_at DESC").
		Find(&out).Error
	return out, err
}

// FollowerIDs is the publish fan-out audience.
func (r *Repository) FollowerIDs(repID string) ([]string, error) {
	out := []string{}
	err := r.db.Table("representative_followers").
		Where("representative_id = ?", repID).
		Pluck("user_id", &out).Error
	return out, err
}

// ─── Comments ───────────────────────────────────────────────────────────

// ListComments returns the thread oldest-first, with hidden comments replaced
// by a placeholder rather than removed.
//
// Same choice as every other thread on CivicOS: a gap where a comment was
// reads as censorship, while a visible placeholder says something was hidden
// and leaves the moderation decision reviewable.
func (r *Repository) ListComments(annID string) ([]domain.RepresentativeAnnouncementComment, error) {
	list := []domain.RepresentativeAnnouncementComment{}
	if err := r.db.
		Where("announcement_id = ?", annID).
		Order("created_at asc").
		Find(&list).Error; err != nil {
		return nil, err
	}
	ids := make([]string, len(list))
	for i, c := range list {
		ids[i] = c.ID
	}
	hidden := moderation.HiddenSet(r.db, "REPRESENTATIVE_ANNOUNCEMENT_COMMENT", ids)
	for i := range list {
		if _, ok := hidden[list[i].ID]; ok {
			list[i].IsHidden = true
			list[i].Content = moderation.PlaceholderContent
			list[i].AuthorName = moderation.PlaceholderAuthorName
		}
	}
	return list, nil
}

// AddComment writes the comment and bumps the announcement's counter in one
// transaction, so a list can never show a count the thread does not support.
func (r *Repository) AddComment(c *domain.RepresentativeAnnouncementComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(c).Error; err != nil {
			return err
		}
		return tx.Model(&domain.RepresentativeAnnouncement{}).
			Where("id = ?", c.AnnouncementID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	})
}
