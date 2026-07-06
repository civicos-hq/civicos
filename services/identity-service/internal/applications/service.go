package applications

import (
	"errors"
	"net/http"
	"regexp"
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
	UpsertRepresentativeApplication(userID string, app *domain.RepresentativeApplication) error
	UpsertOrganizationApplication(userID string, app *domain.OrganizationApplication) error
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
	return s.repo.FindOrganizationByUserID(userID)
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
