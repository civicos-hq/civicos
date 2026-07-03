package organizations

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrgStore interface {
	FindAll(f ListFilters) ([]domain.Organization, error)
	FindByID(id string) (*domain.Organization, error)
	FindBySlug(slug string) (*domain.Organization, error)
	Create(o *domain.Organization) error
	Update(id string, updates map[string]any) error
	FindMember(orgID, userID string) (*domain.OrgMember, error)
	ListMembers(orgID string) ([]domain.OrgMember, error)
	AddMember(m *domain.OrgMember) error
	UpdateMemberRole(orgID, userID string, role domain.OrgMemberRole) error
	RemoveMember(orgID, userID string) error
	BumpMemberCount(orgID string, delta int) error
}

type Service struct{ repo OrgStore }

func NewService(repo OrgStore) *Service { return &Service{repo: repo} }

type CreateInput struct {
	Name         string  `json:"name" binding:"required,min=2"`
	Slug         string  `json:"slug" binding:"required,min=2"`
	Kind         string  `json:"kind" binding:"required"`
	Jurisdiction string  `json:"jurisdiction" binding:"required"`
	State        *string `json:"state"`
	LGA          *string `json:"lga"`
	Description  *string `json:"description"`
	LogoURL      *string `json:"logoUrl"`
	Email        *string `json:"email"`
	Phone        *string `json:"phone"`
	Website      *string `json:"website"`
}

type UpdateInput struct {
	Name         *string `json:"name"`
	Kind         *string `json:"kind"`
	Jurisdiction *string `json:"jurisdiction"`
	State        *string `json:"state"`
	LGA          *string `json:"lga"`
	Description  *string `json:"description"`
	LogoURL      *string `json:"logoUrl"`
	Email        *string `json:"email"`
	Phone        *string `json:"phone"`
	Website      *string `json:"website"`
}

type AddMemberInput struct {
	UserID   string `json:"userId" binding:"required"`
	UserName string `json:"userName" binding:"required"`
	UserRole string `json:"userRole" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

type UpdateMemberInput struct {
	Role string `json:"role" binding:"required"`
}

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (s *Service) List(f ListFilters) ([]domain.Organization, error) {
	return s.repo.FindAll(f)
}

func (s *Service) Get(id string) (*domain.Organization, error) {
	o, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "ORG_NOT_FOUND", Message: "Organization not found", Status: http.StatusNotFound}
	}
	return o, err
}

func (s *Service) Create(input CreateInput, createdByID, createdByName, createdByRole string) (*domain.Organization, error) {
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if !slugRe.MatchString(slug) {
		return nil, &AppError{Code: "INVALID_SLUG", Message: "Slug must be lowercase, alphanumeric, and hyphen-separated", Status: http.StatusBadRequest}
	}
	if !validKind(input.Kind) {
		return nil, &AppError{Code: "INVALID_KIND", Message: "Unknown organization kind", Status: http.StatusBadRequest}
	}
	if !validJurisdiction(input.Jurisdiction) {
		return nil, &AppError{Code: "INVALID_JURISDICTION", Message: "Unknown jurisdiction", Status: http.StatusBadRequest}
	}
	if _, err := s.repo.FindBySlug(slug); err == nil {
		return nil, &AppError{Code: "SLUG_TAKEN", Message: "That slug is already in use", Status: http.StatusConflict}
	}
	o := &domain.Organization{
		ID:           uuid.New().String(),
		Name:         input.Name,
		Slug:         slug,
		Kind:         domain.OrgKind(input.Kind),
		Jurisdiction: domain.OrgJurisdiction(input.Jurisdiction),
		State:        input.State,
		LGA:          input.LGA,
		Description:  input.Description,
		LogoURL:      input.LogoURL,
		Email:        input.Email,
		Phone:        input.Phone,
		Website:      input.Website,
		CreatedByID:  createdByID,
	}
	if err := s.repo.Create(o); err != nil {
		return nil, err
	}
	// The creator becomes the first OWNER — otherwise nobody could ever
	// administer the org they just made.
	owner := &domain.OrgMember{
		ID:             uuid.New().String(),
		OrganizationID: o.ID,
		UserID:         createdByID,
		UserName:       createdByName,
		UserRole:       createdByRole,
		Role:           domain.MemberRoleOwner,
		JoinedAt:       time.Now().UTC(),
	}
	if err := s.repo.AddMember(owner); err != nil {
		return nil, err
	}
	_ = s.repo.BumpMemberCount(o.ID, 1)
	o.MemberCount = 1
	return o, nil
}

func (s *Service) Update(id string, input UpdateInput) (*domain.Organization, error) {
	updates := map[string]any{}
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Kind != nil {
		if !validKind(*input.Kind) {
			return nil, &AppError{Code: "INVALID_KIND", Message: "Unknown organization kind", Status: http.StatusBadRequest}
		}
		updates["kind"] = *input.Kind
	}
	if input.Jurisdiction != nil {
		if !validJurisdiction(*input.Jurisdiction) {
			return nil, &AppError{Code: "INVALID_JURISDICTION", Message: "Unknown jurisdiction", Status: http.StatusBadRequest}
		}
		updates["jurisdiction"] = *input.Jurisdiction
	}
	if input.State != nil {
		updates["state"] = *input.State
	}
	if input.LGA != nil {
		updates["lga"] = *input.LGA
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.LogoURL != nil {
		updates["logo_url"] = *input.LogoURL
	}
	if input.Email != nil {
		updates["email"] = *input.Email
	}
	if input.Phone != nil {
		updates["phone"] = *input.Phone
	}
	if input.Website != nil {
		updates["website"] = *input.Website
	}
	if len(updates) > 0 {
		if err := s.repo.Update(id, updates); err != nil {
			return nil, err
		}
	}
	return s.Get(id)
}

func (s *Service) ListMembers(orgID string) ([]domain.OrgMember, error) {
	return s.repo.ListMembers(orgID)
}

func (s *Service) IsMember(orgID, userID string) (*domain.OrgMember, error) {
	m, err := s.repo.FindMember(orgID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "NOT_A_MEMBER", Message: "You are not a member of this organization", Status: http.StatusForbidden}
	}
	return m, err
}

// CanAdmin returns nil if the caller may perform admin actions on the org
// (create announcements, change members, etc.) — OWNER, ADMIN, or a
// platform-role acting from outside the org.
func (s *Service) CanAdmin(orgID, userID, userRole string) error {
	if userRole == "PLATFORM_ADMIN" {
		return nil
	}
	m, err := s.repo.FindMember(orgID, userID)
	if err != nil {
		return &AppError{Code: "FORBIDDEN", Message: "Only org owners, admins, or platform admins can do this", Status: http.StatusForbidden}
	}
	if m.Role != domain.MemberRoleOwner && m.Role != domain.MemberRoleAdmin {
		return &AppError{Code: "FORBIDDEN", Message: "Only org owners or admins can do this", Status: http.StatusForbidden}
	}
	return nil
}

func (s *Service) AddMember(orgID string, input AddMemberInput) (*domain.OrgMember, error) {
	if !validMemberRole(input.Role) {
		return nil, &AppError{Code: "INVALID_ROLE", Message: "Unknown member role", Status: http.StatusBadRequest}
	}
	if _, err := s.repo.FindMember(orgID, input.UserID); err == nil {
		return nil, &AppError{Code: "ALREADY_MEMBER", Message: "User is already a member", Status: http.StatusConflict}
	}
	m := &domain.OrgMember{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		UserID:         input.UserID,
		UserName:       input.UserName,
		UserRole:       input.UserRole,
		Role:           domain.OrgMemberRole(input.Role),
		JoinedAt:       time.Now().UTC(),
	}
	if err := s.repo.AddMember(m); err != nil {
		return nil, err
	}
	_ = s.repo.BumpMemberCount(orgID, 1)
	return m, nil
}

func (s *Service) UpdateMember(orgID, userID string, input UpdateMemberInput) error {
	if !validMemberRole(input.Role) {
		return &AppError{Code: "INVALID_ROLE", Message: "Unknown member role", Status: http.StatusBadRequest}
	}
	return s.repo.UpdateMemberRole(orgID, userID, domain.OrgMemberRole(input.Role))
}

func (s *Service) RemoveMember(orgID, userID string) error {
	if err := s.repo.RemoveMember(orgID, userID); err != nil {
		return err
	}
	_ = s.repo.BumpMemberCount(orgID, -1)
	return nil
}

func validKind(k string) bool {
	switch domain.OrgKind(k) {
	case domain.OrgKindGovernment, domain.OrgKindAgency, domain.OrgKindNGO, domain.OrgKindUtility, domain.OrgKindOther:
		return true
	}
	return false
}

func validJurisdiction(j string) bool {
	switch domain.OrgJurisdiction(j) {
	case domain.JurisdictionNational, domain.JurisdictionState, domain.JurisdictionLGA, domain.JurisdictionCommunity:
		return true
	}
	return false
}

func validMemberRole(r string) bool {
	switch domain.OrgMemberRole(r) {
	case domain.MemberRoleOwner, domain.MemberRoleAdmin, domain.MemberRoleStaff:
		return true
	}
	return false
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
