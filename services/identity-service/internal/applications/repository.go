package applications

import (
	"errors"

	"github.com/civicos/identity-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
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
