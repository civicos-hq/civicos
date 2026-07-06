package applications

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	FindUserByID(id string) (*domain.User, error)
	FindRepresentativeByUserID(userID string) (*domain.RepresentativeApplication, error)
	FindOrganizationByUserID(userID string) (*domain.OrganizationApplication, error)
	FindRepresentativeByID(id string) (*domain.RepresentativeApplication, error)
	FindOrganizationByID(id string) (*domain.OrganizationApplication, error)
	ListRepresentativeApplications(f ListFilters) ([]domain.RepresentativeApplication, int64, error)
	ListOrganizationApplications(f ListFilters) ([]domain.OrganizationApplication, int64, error)
	UpsertRepresentativeApplication(userID string, app *domain.RepresentativeApplication) error
	UpsertOrganizationApplication(userID string, app *domain.OrganizationApplication) error
	CreateNotification(userID, title, body string, linkURL *string) error
	ApproveRepresentativeApplication(id, reviewerID string, note *string, reviewedAt time.Time) (*domain.RepresentativeApplication, error)
	ApproveOrganizationApplication(id, reviewerID string, note *string, reviewedAt time.Time, userRole domain.UserRole) (*domain.OrganizationApplication, error)
	ReviewRepresentativeApplication(id, reviewerID string, status domain.ApprovalStatus, note *string, reviewedAt time.Time) error
	ReviewOrganizationApplication(id, reviewerID string, status domain.ApprovalStatus, note *string, reviewedAt time.Time) error
}

type Service struct {
	repo Store
}

func NewService(repo Store) *Service {
	return &Service{repo: repo}
}

type MeResponse struct {
	RequestedAccountType      domain.RequestedAccountType       `json:"requestedAccountType"`
	ApprovalStatus            domain.ApprovalStatus             `json:"approvalStatus"`
	RepresentativeApplication *domain.RepresentativeApplication `json:"representativeApplication,omitempty"`
	OrganizationApplication   *domain.OrganizationApplication   `json:"organizationApplication,omitempty"`
}

type RepresentativeApplicationInput struct {
	FullName      string   `json:"fullName" binding:"required,min=2,max=100"`
	Title         string   `json:"title" binding:"required,min=2,max=50"`
	Position      string   `json:"position" binding:"required,min=2,max=100"`
	Constituency  string   `json:"constituency" binding:"required,min=2,max=120"`
	CommunityID   string   `json:"communityId" binding:"required,uuid4"`
	Party         *string  `json:"party"`
	Bio           *string  `json:"bio"`
	AvatarURL     *string  `json:"avatarUrl" binding:"omitempty,url"`
	OfficialEmail *string  `json:"officialEmail" binding:"omitempty,email"`
	OfficialPhone *string  `json:"officialPhone"`
	Website       *string  `json:"website" binding:"omitempty,url"`
	ProofURLs     []string `json:"proofUrls"`
}

type OrganizationApplicationInput struct {
	Name          string   `json:"name" binding:"required,min=2,max=120"`
	Slug          string   `json:"slug" binding:"required,min=2,max=120"`
	Kind          string   `json:"kind" binding:"required,oneof=GOVERNMENT AGENCY NGO UTILITY OTHER"`
	Jurisdiction  string   `json:"jurisdiction" binding:"required,oneof=NATIONAL STATE LGA COMMUNITY"`
	State         *string  `json:"state"`
	LGA           *string  `json:"lga"`
	Description   *string  `json:"description"`
	LogoURL       *string  `json:"logoUrl" binding:"omitempty,url"`
	OfficialEmail *string  `json:"officialEmail" binding:"omitempty,email"`
	OfficialPhone *string  `json:"officialPhone"`
	Website       *string  `json:"website" binding:"omitempty,url"`
	ProofURLs     []string `json:"proofUrls"`
}

type AdminApplicationSummary struct {
	Kind        string                `json:"kind"`
	ID          string                `json:"id"`
	Status      domain.ApprovalStatus `json:"status"`
	SubmittedAt time.Time             `json:"submittedAt"`
	ReviewedAt  *time.Time            `json:"reviewedAt,omitempty"`
	Headline    string                `json:"headline"`
	Subhead     string                `json:"subhead"`
	Applicant   domain.PublicUser     `json:"applicant"`
}

type AdminApplicationDetail struct {
	Kind                      string                            `json:"kind"`
	ID                        string                            `json:"id"`
	Status                    domain.ApprovalStatus             `json:"status"`
	SubmittedAt               time.Time                         `json:"submittedAt"`
	ReviewedAt                *time.Time                        `json:"reviewedAt,omitempty"`
	RepresentativeApplication *domain.RepresentativeApplication `json:"representativeApplication,omitempty"`
	OrganizationApplication   *domain.OrganizationApplication   `json:"organizationApplication,omitempty"`
	Applicant                 domain.PublicUser                 `json:"applicant"`
}

type AdminListFilters struct {
	Kind   string
	Status string
	Search string
	Limit  int
	Offset int
}

type ReviewInput struct {
	Status string  `json:"status" binding:"required,oneof=APPROVED REJECTED"`
	Note   *string `json:"note"`
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (s *Service) GetMe(userID string) (*MeResponse, error) {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &AppError{Code: "USER_NOT_FOUND", Message: "User not found", Status: http.StatusNotFound}
		}
		return nil, err
	}
	out := &MeResponse{
		RequestedAccountType: user.RequestedAccountType,
		ApprovalStatus:       user.ApprovalStatus,
	}
	if user.RequestedAccountType == domain.AccountTypeRepresentative {
		app, err := s.repo.FindRepresentativeByUserID(userID)
		if err == nil {
			out.RepresentativeApplication = app
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if user.RequestedAccountType == domain.AccountTypeOrganization {
		app, err := s.repo.FindOrganizationByUserID(userID)
		if err == nil {
			out.OrganizationApplication = app
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) UpsertRepresentative(userID string, input RepresentativeApplicationInput) (*domain.RepresentativeApplication, error) {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &AppError{Code: "USER_NOT_FOUND", Message: "User not found", Status: http.StatusNotFound}
		}
		return nil, err
	}
	if user.ApprovalStatus == domain.ApprovalStatusApproved && user.Role == domain.RoleRepresentative {
		return nil, &AppError{Code: "APPLICATION_ALREADY_APPROVED", Message: "Representative approval is already complete", Status: http.StatusConflict}
	}
	now := time.Now().UTC()
	app := &domain.RepresentativeApplication{
		ID:               uuid.New().String(),
		UserID:           userID,
		Status:           domain.ApprovalStatusPending,
		FullName:         input.FullName,
		Title:            input.Title,
		Position:         input.Position,
		Constituency:     input.Constituency,
		CommunityID:      input.CommunityID,
		Party:            input.Party,
		Bio:              input.Bio,
		AvatarURL:        input.AvatarURL,
		OfficialEmail:    input.OfficialEmail,
		OfficialPhone:    input.OfficialPhone,
		Website:          input.Website,
		ProofURLs:        input.ProofURLs,
		SubmittedAt:      now,
		ReviewedAt:       nil,
		ReviewedByUserID: nil,
		ReviewNote:       nil,
	}
	if err := s.repo.UpsertRepresentativeApplication(userID, app); err != nil {
		return nil, err
	}
	link := "/profile"
	if err := s.repo.CreateNotification(userID, "Representative request updated", "Your representative application is pending admin review.", &link); err != nil {
		log.Printf("[applications.UpsertRepresentative] notification failed for user=%s: %v", userID, err)
	}
	return s.repo.FindRepresentativeByUserID(userID)
}

func (s *Service) UpsertOrganization(userID string, input OrganizationApplicationInput) (*domain.OrganizationApplication, error) {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &AppError{Code: "USER_NOT_FOUND", Message: "User not found", Status: http.StatusNotFound}
		}
		return nil, err
	}
	if user.ApprovalStatus == domain.ApprovalStatusApproved &&
		(user.Role == domain.RoleNGO || user.Role == domain.RoleGovernmentAdmin || user.Role == domain.RolePlatformAdmin) {
		return nil, &AppError{Code: "APPLICATION_ALREADY_APPROVED", Message: "Organization approval is already complete", Status: http.StatusConflict}
	}
	if err := validateOrganizationInput(input); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	app := &domain.OrganizationApplication{
		ID:               uuid.New().String(),
		UserID:           userID,
		Status:           domain.ApprovalStatusPending,
		Name:             input.Name,
		Slug:             strings.ToLower(strings.TrimSpace(input.Slug)),
		Kind:             input.Kind,
		Jurisdiction:     input.Jurisdiction,
		State:            input.State,
		LGA:              input.LGA,
		Description:      input.Description,
		LogoURL:          input.LogoURL,
		OfficialEmail:    input.OfficialEmail,
		OfficialPhone:    input.OfficialPhone,
		Website:          input.Website,
		ProofURLs:        input.ProofURLs,
		SubmittedAt:      now,
		ReviewedAt:       nil,
		ReviewedByUserID: nil,
		ReviewNote:       nil,
	}
	if err := s.repo.UpsertOrganizationApplication(userID, app); err != nil {
		return nil, err
	}
	link := "/profile"
	if err := s.repo.CreateNotification(userID, "Organization request updated", "Your organization application is pending admin review.", &link); err != nil {
		log.Printf("[applications.UpsertOrganization] notification failed for user=%s: %v", userID, err)
	}
	return s.repo.FindOrganizationByUserID(userID)
}

func (s *Service) ListAdmin(f AdminListFilters) ([]AdminApplicationSummary, int64, error) {
	kind := strings.ToUpper(strings.TrimSpace(f.Kind))
	repoFilters := ListFilters{Status: strings.ToUpper(strings.TrimSpace(f.Status)), Search: f.Search, Limit: f.Limit, Offset: f.Offset}
	switch kind {
	case "", "ALL":
		return s.listAdminAll(repoFilters)
	case string(domain.AccountTypeRepresentative):
		items, total, err := s.repo.ListRepresentativeApplications(repoFilters)
		if err != nil {
			return nil, 0, err
		}
		out := make([]AdminApplicationSummary, 0, len(items))
		for _, item := range items {
			applicant, err := s.repo.FindUserByID(item.UserID)
			if err != nil {
				return nil, 0, err
			}
			out = append(out, representativeSummary(&item, applicant))
		}
		return out, total, nil
	case string(domain.AccountTypeOrganization):
		items, total, err := s.repo.ListOrganizationApplications(repoFilters)
		if err != nil {
			return nil, 0, err
		}
		out := make([]AdminApplicationSummary, 0, len(items))
		for _, item := range items {
			applicant, err := s.repo.FindUserByID(item.UserID)
			if err != nil {
				return nil, 0, err
			}
			out = append(out, organizationSummary(&item, applicant))
		}
		return out, total, nil
	default:
		return nil, 0, &AppError{Code: "INVALID_APPLICATION_KIND", Message: "Unknown application kind", Status: http.StatusBadRequest}
	}
}

func (s *Service) GetAdmin(kind, id string) (*AdminApplicationDetail, error) {
	switch normalizeKind(kind) {
	case domain.AccountTypeRepresentative:
		app, err := s.repo.FindRepresentativeByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &AppError{Code: "APPLICATION_NOT_FOUND", Message: "Application not found", Status: http.StatusNotFound}
			}
			return nil, err
		}
		applicant, err := s.repo.FindUserByID(app.UserID)
		if err != nil {
			return nil, err
		}
		return &AdminApplicationDetail{
			Kind:                      string(domain.AccountTypeRepresentative),
			ID:                        app.ID,
			Status:                    app.Status,
			SubmittedAt:               app.SubmittedAt,
			ReviewedAt:                app.ReviewedAt,
			RepresentativeApplication: app,
			Applicant:                 applicant.ToPublic(),
		}, nil
	case domain.AccountTypeOrganization:
		app, err := s.repo.FindOrganizationByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &AppError{Code: "APPLICATION_NOT_FOUND", Message: "Application not found", Status: http.StatusNotFound}
			}
			return nil, err
		}
		applicant, err := s.repo.FindUserByID(app.UserID)
		if err != nil {
			return nil, err
		}
		return &AdminApplicationDetail{
			Kind:                    string(domain.AccountTypeOrganization),
			ID:                      app.ID,
			Status:                  app.Status,
			SubmittedAt:             app.SubmittedAt,
			ReviewedAt:              app.ReviewedAt,
			OrganizationApplication: app,
			Applicant:               applicant.ToPublic(),
		}, nil
	default:
		return nil, &AppError{Code: "INVALID_APPLICATION_KIND", Message: "Unknown application kind", Status: http.StatusBadRequest}
	}
}

func (s *Service) Review(kind, id, reviewerID string, input ReviewInput) (*AdminApplicationDetail, error) {
	status := domain.ApprovalStatus(strings.ToUpper(strings.TrimSpace(input.Status)))
	if status != domain.ApprovalStatusApproved && status != domain.ApprovalStatusRejected {
		return nil, &AppError{Code: "INVALID_REVIEW_STATUS", Message: "Review status must be APPROVED or REJECTED", Status: http.StatusBadRequest}
	}
	now := time.Now().UTC()

	switch normalizeKind(kind) {
	case domain.AccountTypeRepresentative:
		existing, err := s.repo.FindRepresentativeByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &AppError{Code: "APPLICATION_NOT_FOUND", Message: "Application not found", Status: http.StatusNotFound}
			}
			return nil, err
		}
		if existing.Status == domain.ApprovalStatusApproved {
			return nil, &AppError{Code: "APPLICATION_ALREADY_APPROVED", Message: "This application is already approved", Status: http.StatusConflict}
		}
		if status == domain.ApprovalStatusApproved {
			if _, err := s.repo.ApproveRepresentativeApplication(id, reviewerID, input.Note, now); err != nil {
				return nil, err
			}
			detail, err := s.GetAdmin(kind, id)
			if err != nil {
				return nil, err
			}
			link := "/profile"
			if err := s.repo.CreateNotification(detail.Applicant.ID, "Representative request approved", "Your representative account has been approved.", &link); err != nil {
				log.Printf("[applications.Review] representative approval notification failed for user=%s: %v", detail.Applicant.ID, err)
			}
			return detail, nil
		}
		if err := s.repo.ReviewRepresentativeApplication(id, reviewerID, status, input.Note, now); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &AppError{Code: "APPLICATION_NOT_FOUND", Message: "Application not found", Status: http.StatusNotFound}
			}
			return nil, err
		}
		detail, err := s.GetAdmin(kind, id)
		if err != nil {
			return nil, err
		}
		link := "/profile"
		if err := s.repo.CreateNotification(detail.Applicant.ID, "Representative request needs changes", "Your representative application was reviewed and needs changes before approval.", &link); err != nil {
			log.Printf("[applications.Review] representative rejection notification failed for user=%s: %v", detail.Applicant.ID, err)
		}
		return detail, nil
	case domain.AccountTypeOrganization:
		existing, err := s.repo.FindOrganizationByID(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &AppError{Code: "APPLICATION_NOT_FOUND", Message: "Application not found", Status: http.StatusNotFound}
			}
			return nil, err
		}
		if existing.Status == domain.ApprovalStatusApproved {
			return nil, &AppError{Code: "APPLICATION_ALREADY_APPROVED", Message: "This application is already approved", Status: http.StatusConflict}
		}
		if status == domain.ApprovalStatusApproved {
			role := organizationRoleForKind(existing.Kind)
			if _, err := s.repo.ApproveOrganizationApplication(id, reviewerID, input.Note, now, role); err != nil {
				return nil, err
			}
			detail, err := s.GetAdmin(kind, id)
			if err != nil {
				return nil, err
			}
			link := "/profile"
			if err := s.repo.CreateNotification(detail.Applicant.ID, "Organization request approved", "Your organization account has been approved.", &link); err != nil {
				log.Printf("[applications.Review] organization approval notification failed for user=%s: %v", detail.Applicant.ID, err)
			}
			return detail, nil
		}
		if err := s.repo.ReviewOrganizationApplication(id, reviewerID, status, input.Note, now); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &AppError{Code: "APPLICATION_NOT_FOUND", Message: "Application not found", Status: http.StatusNotFound}
			}
			return nil, err
		}
		detail, err := s.GetAdmin(kind, id)
		if err != nil {
			return nil, err
		}
		link := "/profile"
		if err := s.repo.CreateNotification(detail.Applicant.ID, "Organization request needs changes", "Your organization application was reviewed and needs changes before approval.", &link); err != nil {
			log.Printf("[applications.Review] organization rejection notification failed for user=%s: %v", detail.Applicant.ID, err)
		}
		return detail, nil
	default:
		return nil, &AppError{Code: "INVALID_APPLICATION_KIND", Message: "Unknown application kind", Status: http.StatusBadRequest}
	}
}

func validateOrganizationInput(input OrganizationApplicationInput) error {
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if !slugRe.MatchString(slug) {
		return &AppError{Code: "INVALID_SLUG", Message: "Slug must be lowercase, alphanumeric, and hyphen-separated", Status: http.StatusBadRequest}
	}
	if (input.Jurisdiction == "STATE" || input.Jurisdiction == "LGA" || input.Jurisdiction == "COMMUNITY") &&
		(input.State == nil || strings.TrimSpace(*input.State) == "") {
		return &AppError{Code: "STATE_REQUIRED", Message: "This organization jurisdiction requires a state", Status: http.StatusBadRequest}
	}
	if (input.Jurisdiction == "LGA" || input.Jurisdiction == "COMMUNITY") &&
		(input.LGA == nil || strings.TrimSpace(*input.LGA) == "") {
		return &AppError{Code: "LGA_REQUIRED", Message: "This organization jurisdiction requires an LGA", Status: http.StatusBadRequest}
	}
	return nil
}

func (s *Service) listAdminAll(f ListFilters) ([]AdminApplicationSummary, int64, error) {
	effectiveLimit := f.Limit
	if effectiveLimit <= 0 {
		effectiveLimit = 25
	}
	if f.Offset > 0 {
		effectiveLimit += f.Offset
	}

	repItems, repTotal, err := s.repo.ListRepresentativeApplications(ListFilters{
		Status: f.Status, Search: f.Search, Limit: effectiveLimit, Offset: 0,
	})
	if err != nil {
		return nil, 0, err
	}
	orgItems, orgTotal, err := s.repo.ListOrganizationApplications(ListFilters{
		Status: f.Status, Search: f.Search, Limit: effectiveLimit, Offset: 0,
	})
	if err != nil {
		return nil, 0, err
	}

	out := make([]AdminApplicationSummary, 0, len(repItems)+len(orgItems))
	for _, item := range repItems {
		applicant, err := s.repo.FindUserByID(item.UserID)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, representativeSummary(&item, applicant))
	}
	for _, item := range orgItems {
		applicant, err := s.repo.FindUserByID(item.UserID)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, organizationSummary(&item, applicant))
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].SubmittedAt.After(out[j].SubmittedAt)
	})

	start := f.Offset
	if start > len(out) {
		start = len(out)
	}
	end := start + f.Limit
	if f.Limit <= 0 || end > len(out) {
		end = len(out)
	}
	return out[start:end], repTotal + orgTotal, nil
}

func representativeSummary(app *domain.RepresentativeApplication, applicant *domain.User) AdminApplicationSummary {
	return AdminApplicationSummary{
		Kind:        string(domain.AccountTypeRepresentative),
		ID:          app.ID,
		Status:      app.Status,
		SubmittedAt: app.SubmittedAt,
		ReviewedAt:  app.ReviewedAt,
		Headline:    app.FullName,
		Subhead:     app.Position + " · " + app.Constituency,
		Applicant:   applicant.ToPublic(),
	}
}

func organizationSummary(app *domain.OrganizationApplication, applicant *domain.User) AdminApplicationSummary {
	return AdminApplicationSummary{
		Kind:        string(domain.AccountTypeOrganization),
		ID:          app.ID,
		Status:      app.Status,
		SubmittedAt: app.SubmittedAt,
		ReviewedAt:  app.ReviewedAt,
		Headline:    app.Name,
		Subhead:     app.Kind + " · " + app.Jurisdiction,
		Applicant:   applicant.ToPublic(),
	}
}

func normalizeKind(raw string) domain.RequestedAccountType {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(domain.AccountTypeRepresentative):
		return domain.AccountTypeRepresentative
	case string(domain.AccountTypeOrganization):
		return domain.AccountTypeOrganization
	default:
		return ""
	}
}

func organizationRoleForKind(kind string) domain.UserRole {
	if strings.ToUpper(strings.TrimSpace(kind)) == "NGO" {
		return domain.RoleNGO
	}
	return domain.RoleGovernmentAdmin
}
