package applications

import (
	"errors"
	"strings"
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type ListFilters struct {
	Status          string
	Search          string
	SubmittedBefore *time.Time
	Limit           int
	Offset          int
}

type reviewerRecord struct {
	ID   string
	Name string
}

type representativeRecord struct {
	ID            string `gorm:"type:uuid;primaryKey"`
	Name          string
	Title         string
	Position      string
	Constituency  string
	Party         *string
	Bio           *string
	AvatarURL     *string `gorm:"column:avatar_url"`
	Email         *string
	Phone         *string
	Website       *string
	CommunityID   string `gorm:"type:uuid"`
	ResponseRate  int
	FollowerCount int
	CommentCount  int
	CreatedByID   string `gorm:"type:uuid"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (representativeRecord) TableName() string { return "representatives" }

type organizationRecord struct {
	ID                string `gorm:"type:uuid;primaryKey"`
	Name              string
	Slug              string
	Kind              string
	Jurisdiction      string
	State             *string
	LGA               *string
	Description       *string
	LogoURL           *string `gorm:"column:logo_url"`
	Email             *string
	Phone             *string
	Website           *string
	Verified          bool
	MemberCount       int
	AnnouncementCount int
	ProjectCount      int
	AssignmentCount   int
	CreatedByID       string `gorm:"type:uuid"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (organizationRecord) TableName() string { return "organizations" }

type orgMemberRecord struct {
	ID             string `gorm:"type:uuid;primaryKey"`
	OrganizationID string `gorm:"type:uuid"`
	UserID         string `gorm:"type:uuid"`
	UserName       string
	UserRole       string
	Role           string
	JoinedAt       time.Time
}

func (orgMemberRecord) TableName() string { return "org_members" }

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

func (r *Repository) ListReviewHistory(kind domain.RequestedAccountType, applicationID string) ([]domain.ApplicationReviewEvent, error) {
	var list []domain.ApplicationReviewEvent
	if err := r.db.
		Where("application_kind = ? AND application_id = ?", kind, applicationID).
		Order("created_at desc").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
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
	if f.SubmittedBefore != nil {
		q = q.Where("representative_applications.submitted_at < ?", *f.SubmittedBefore)
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
	if f.SubmittedBefore != nil {
		q = q.Where("organization_applications.submitted_at < ?", *f.SubmittedBefore)
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

func (r *Repository) CreateNotification(userID, title, body string, linkURL *string) error {
	row := map[string]any{
		"id":         uuid.New().String(),
		"type":       "SYSTEM",
		"title":      title,
		"body":       body,
		"read":       false,
		"user_id":    userID,
		"created_at": time.Now().UTC(),
	}
	if linkURL != nil && *linkURL != "" {
		row["link_url"] = *linkURL
	}
	return r.db.Table("notifications").Create(row).Error
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
		reviewer, err := findReviewer(tx, reviewerID)
		if err != nil {
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
		if err := tx.Model(&domain.User{}).Where("id = ?", app.UserID).Updates(userUpdates).Error; err != nil {
			return err
		}
		return createReviewEvent(tx, domain.AccountTypeRepresentative, app.ID, app.UserID, reviewer, status, note, reviewedAt)
	})
}

func (r *Repository) ApproveRepresentativeApplication(id, reviewerID string, note *string, reviewedAt time.Time) (*domain.RepresentativeApplication, error) {
	var out domain.RepresentativeApplication
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var app domain.RepresentativeApplication
		if err := tx.Where("id = ?", id).First(&app).Error; err != nil {
			return err
		}
		var user domain.User
		if err := tx.Where("id = ?", app.UserID).First(&user).Error; err != nil {
			return err
		}
		reviewer, err := findReviewer(tx, reviewerID)
		if err != nil {
			return err
		}
		profileID := app.ApprovedProfileID
		if profileID == nil || *profileID == "" {
			id := uuid.New().String()
			profileID = &id
			now := reviewedAt
			rep := &representativeRecord{
				ID:           id,
				Name:         app.FullName,
				Title:        app.Title,
				Position:     app.Position,
				Constituency: app.Constituency,
				Party:        app.Party,
				Bio:          app.Bio,
				AvatarURL:    app.AvatarURL,
				Email:        app.OfficialEmail,
				Phone:        app.OfficialPhone,
				Website:      app.Website,
				CommunityID:  app.CommunityID,
				CreatedByID:  app.UserID,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := tx.Create(rep).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&domain.RepresentativeApplication{}).Where("id = ?", id).Updates(map[string]any{
			"status":              domain.ApprovalStatusApproved,
			"reviewed_at":         reviewedAt,
			"reviewed_by_user_id": reviewerID,
			"review_note":         nullableString(note),
			"approved_profile_id": *profileID,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.User{}).Where("id = ?", app.UserID).Updates(map[string]any{
			"role":                    domain.RoleRepresentative,
			"requested_account_type":  domain.AccountTypeRepresentative,
			"approval_status":         domain.ApprovalStatusApproved,
			"approval_reviewed_at":    reviewedAt,
			"approval_reviewed_by_id": reviewerID,
			"approval_note":           nullableString(note),
		}).Error; err != nil {
			return err
		}
		if err := createReviewEvent(tx, domain.AccountTypeRepresentative, app.ID, app.UserID, reviewer, domain.ApprovalStatusApproved, note, reviewedAt); err != nil {
			return err
		}
		return tx.Where("id = ?", id).First(&out).Error
	})
	return &out, err
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
		reviewer, err := findReviewer(tx, reviewerID)
		if err != nil {
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
		if err := tx.Model(&domain.User{}).Where("id = ?", app.UserID).Updates(userUpdates).Error; err != nil {
			return err
		}
		return createReviewEvent(tx, domain.AccountTypeOrganization, app.ID, app.UserID, reviewer, status, note, reviewedAt)
	})
}

func (r *Repository) ApproveOrganizationApplication(id, reviewerID string, note *string, reviewedAt time.Time, userRole domain.UserRole) (*domain.OrganizationApplication, error) {
	var out domain.OrganizationApplication
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var app domain.OrganizationApplication
		if err := tx.Where("id = ?", id).First(&app).Error; err != nil {
			return err
		}
		var user domain.User
		if err := tx.Where("id = ?", app.UserID).First(&user).Error; err != nil {
			return err
		}
		reviewer, err := findReviewer(tx, reviewerID)
		if err != nil {
			return err
		}
		orgID := app.ApprovedOrganizationID
		if orgID == nil || *orgID == "" {
			id := uuid.New().String()
			orgID = &id
			now := reviewedAt
			org := &organizationRecord{
				ID:           id,
				Name:         app.Name,
				Slug:         app.Slug,
				Kind:         app.Kind,
				Jurisdiction: app.Jurisdiction,
				State:        app.State,
				LGA:          app.LGA,
				Description:  app.Description,
				LogoURL:      app.LogoURL,
				Email:        app.OfficialEmail,
				Phone:        app.OfficialPhone,
				Website:      app.Website,
				CreatedByID:  app.UserID,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := tx.Create(org).Error; err != nil {
				return err
			}
			member := &orgMemberRecord{
				ID:             uuid.New().String(),
				OrganizationID: id,
				UserID:         user.ID,
				UserName:       user.Name,
				UserRole:       string(userRole),
				Role:           "OWNER",
				JoinedAt:       reviewedAt,
			}
			if err := tx.Create(member).Error; err != nil {
				return err
			}
			if err := tx.Model(&organizationRecord{}).Where("id = ?", id).Update("member_count", 1).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&domain.OrganizationApplication{}).Where("id = ?", id).Updates(map[string]any{
			"status":                   domain.ApprovalStatusApproved,
			"reviewed_at":              reviewedAt,
			"reviewed_by_user_id":      reviewerID,
			"review_note":              nullableString(note),
			"approved_organization_id": *orgID,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.User{}).Where("id = ?", app.UserID).Updates(map[string]any{
			"role":                    userRole,
			"requested_account_type":  domain.AccountTypeOrganization,
			"approval_status":         domain.ApprovalStatusApproved,
			"approval_reviewed_at":    reviewedAt,
			"approval_reviewed_by_id": reviewerID,
			"approval_note":           nullableString(note),
		}).Error; err != nil {
			return err
		}
		if err := createReviewEvent(tx, domain.AccountTypeOrganization, app.ID, app.UserID, reviewer, domain.ApprovalStatusApproved, note, reviewedAt); err != nil {
			return err
		}
		return tx.Where("id = ?", id).First(&out).Error
	})
	return &out, err
}

func findReviewer(tx *gorm.DB, reviewerID string) (*reviewerRecord, error) {
	var reviewer reviewerRecord
	if err := tx.Model(&domain.User{}).Select("id", "name").Where("id = ?", reviewerID).First(&reviewer).Error; err != nil {
		return nil, err
	}
	return &reviewer, nil
}

func createReviewEvent(
	tx *gorm.DB,
	kind domain.RequestedAccountType,
	applicationID string,
	applicantUserID string,
	reviewer *reviewerRecord,
	status domain.ApprovalStatus,
	note *string,
	reviewedAt time.Time,
) error {
	return tx.Create(&domain.ApplicationReviewEvent{
		ID:              uuid.New().String(),
		ApplicationKind: kind,
		ApplicationID:   applicationID,
		ApplicantUserID: applicantUserID,
		ReviewerUserID:  reviewer.ID,
		ReviewerName:    reviewer.Name,
		Status:          status,
		Note:            normalizeStringPtr(note),
		CreatedAt:       reviewedAt,
	}).Error
}

func nullableString(v *string) any {
	if v == nil || strings.TrimSpace(*v) == "" {
		return gorm.Expr("NULL")
	}
	return strings.TrimSpace(*v)
}

func normalizeStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
