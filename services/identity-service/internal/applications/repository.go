package applications

import (
	"errors"
	"strings"
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type ListFilters struct {
	Status string
	Search string
	Limit  int
	Offset int
}

func (r *Repository) FindUserByID(id string) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindRepresentativeByUserID(userID string) (*domain.RepresentativeApplication, error) {
	var app domain.RepresentativeApplication
	if err := r.db.Where("user_id = ?", userID).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *Repository) FindOrganizationByUserID(userID string) (*domain.OrganizationApplication, error) {
	var app domain.OrganizationApplication
	if err := r.db.Where("user_id = ?", userID).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *Repository) FindRepresentativeByID(id string) (*domain.RepresentativeApplication, error) {
	var app domain.RepresentativeApplication
	if err := r.db.Where("id = ?", id).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *Repository) FindOrganizationByID(id string) (*domain.OrganizationApplication, error) {
	var app domain.OrganizationApplication
	if err := r.db.Where("id = ?", id).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *Repository) ListRepresentativeApplications(f ListFilters) ([]domain.RepresentativeApplication, int64, error) {
	q := r.db.Model(&domain.RepresentativeApplication{}).
		Joins("JOIN users ON representative_applications.user_id = users.id")

	if f.Status != "" {
		q = q.Where("representative_applications.status = ?", f.Status)
	}
	if f.Search != "" {
		term := "%" + strings.ToLower(f.Search) + "%"
		q = q.Where(
			"LOWER(users.email) LIKE ? OR LOWER(users.name) LIKE ? OR LOWER(representative_applications.full_name) LIKE ? OR LOWER(representative_applications.position) LIKE ? OR LOWER(representative_applications.constituency) LIKE ?",
			term, term, term, term, term,
		)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 {
		f.Limit = 25
	}

	var list []domain.RepresentativeApplication
	if err := q.
		Order("representative_applications.submitted_at desc").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) ListOrganizationApplications(f ListFilters) ([]domain.OrganizationApplication, int64, error) {
	q := r.db.Model(&domain.OrganizationApplication{}).
		Joins("JOIN users ON organization_applications.user_id = users.id")

	if f.Status != "" {
		q = q.Where("organization_applications.status = ?", f.Status)
	}
	if f.Search != "" {
		term := "%" + strings.ToLower(f.Search) + "%"
		q = q.Where(
			"LOWER(users.email) LIKE ? OR LOWER(users.name) LIKE ? OR LOWER(organization_applications.name) LIKE ? OR LOWER(organization_applications.slug) LIKE ?",
			term, term, term, term,
		)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 {
		f.Limit = 25
	}

	var list []domain.OrganizationApplication
	if err := q.
		Order("organization_applications.submitted_at desc").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) UpsertRepresentativeApplication(userID string, app *domain.RepresentativeApplication) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.User{}).Where("id = ?", userID).Updates(map[string]any{
			"requested_account_type":  domain.AccountTypeRepresentative,
			"approval_status":         domain.ApprovalStatusPending,
			"approval_reviewed_at":    gorm.Expr("NULL"),
			"approval_reviewed_by_id": gorm.Expr("NULL"),
			"approval_note":           gorm.Expr("NULL"),
		}).Error; err != nil {
			return err
		}

		var existing domain.RepresentativeApplication
		err := tx.Where("user_id = ?", userID).First(&existing).Error
		if err == nil {
			app.ID = existing.ID
			app.CreatedAt = existing.CreatedAt
			return tx.Model(&existing).Select("*").Updates(app).Error
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(app).Error
	})
}

func (r *Repository) UpsertOrganizationApplication(userID string, app *domain.OrganizationApplication) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.User{}).Where("id = ?", userID).Updates(map[string]any{
			"requested_account_type":  domain.AccountTypeOrganization,
			"approval_status":         domain.ApprovalStatusPending,
			"approval_reviewed_at":    gorm.Expr("NULL"),
			"approval_reviewed_by_id": gorm.Expr("NULL"),
			"approval_note":           gorm.Expr("NULL"),
		}).Error; err != nil {
			return err
		}

		var existing domain.OrganizationApplication
		err := tx.Where("user_id = ?", userID).First(&existing).Error
		if err == nil {
			app.ID = existing.ID
			app.CreatedAt = existing.CreatedAt
			return tx.Model(&existing).Select("*").Updates(app).Error
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(app).Error
	})
}

func (r *Repository) ReviewRepresentativeApplication(
	id, reviewerID string,
	status domain.ApprovalStatus,
	note *string,
	reviewedAt time.Time,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var app domain.RepresentativeApplication
		if err := tx.Where("id = ?", id).First(&app).Error; err != nil {
			return err
		}
		appUpdates := map[string]any{
			"status":              status,
			"reviewed_at":         reviewedAt,
			"reviewed_by_user_id": reviewerID,
			"review_note":         nullableString(note),
		}
		if err := tx.Model(&domain.RepresentativeApplication{}).Where("id = ?", id).Updates(appUpdates).Error; err != nil {
			return err
		}
		userUpdates := map[string]any{
			"approval_status":         status,
			"approval_reviewed_at":    reviewedAt,
			"approval_reviewed_by_id": reviewerID,
			"approval_note":           nullableString(note),
		}
		return tx.Model(&domain.User{}).Where("id = ?", app.UserID).Updates(userUpdates).Error
	})
}

func (r *Repository) ReviewOrganizationApplication(
	id, reviewerID string,
	status domain.ApprovalStatus,
	note *string,
	reviewedAt time.Time,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var app domain.OrganizationApplication
		if err := tx.Where("id = ?", id).First(&app).Error; err != nil {
			return err
		}
		appUpdates := map[string]any{
			"status":              status,
			"reviewed_at":         reviewedAt,
			"reviewed_by_user_id": reviewerID,
			"review_note":         nullableString(note),
		}
		if err := tx.Model(&domain.OrganizationApplication{}).Where("id = ?", id).Updates(appUpdates).Error; err != nil {
			return err
		}
		userUpdates := map[string]any{
			"approval_status":         status,
			"approval_reviewed_at":    reviewedAt,
			"approval_reviewed_by_id": reviewerID,
			"approval_note":           nullableString(note),
		}
		return tx.Model(&domain.User{}).Where("id = ?", app.UserID).Updates(userUpdates).Error
	})
}

func nullableString(v *string) any {
	if v == nil || strings.TrimSpace(*v) == "" {
		return gorm.Expr("NULL")
	}
	return strings.TrimSpace(*v)
}
