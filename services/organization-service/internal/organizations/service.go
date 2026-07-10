package organizations

import (
	"errors"
	"net/http"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrgStore interface {
	FindAll(f ListFilters) ([]domain.Organization, error)
	FindByID(id string) (*domain.Organization, error)
	FindByIDs(ids []string) ([]domain.Organization, error)
	FindBySlug(slug string) (*domain.Organization, error)
	Update(id string, updates map[string]any) error
	FindMember(orgID, userID string) (*domain.OrgMember, error)
	FindMembershipsByUser(userID string) ([]domain.OrgMember, error)
	ListMembers(orgID string) ([]domain.OrgMember, error)
	AddMember(m *domain.OrgMember) error
	UpdateMemberRole(orgID, userID string, role domain.OrgMemberRole) error
	RemoveMember(orgID, userID string) error
	BumpMemberCount(orgID string, delta int) error
}

type Service struct{ repo OrgStore }

func NewService(repo OrgStore) *Service { return &Service{repo: repo} }

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
	// Verified is set exclusively by platform admins from the admin
	// console. The verified badge is a trust signal to citizens ("this
	// really is the Lagos Water Corp") so the toggle is deliberately
	// separate from other org edits — an audit-loggable action in its
	// own right (see admin/organizations page).
	Verified *bool `json:"verified"`
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
	if input.Verified != nil {
		updates["verified"] = *input.Verified
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

// MyMembership pairs a membership row with the org it references. The
// frontend uses this to render the "which org can I act as" picker
// without needing to fan out N GET-org calls after listing memberships.
type MyMembership struct {
	Organization domain.Organization `json:"organization"`
	Membership   domain.OrgMember    `json:"membership"`
}

// ListMyMemberships returns every org the caller belongs to, paired
// with the membership role. Ordered by joined_at ascending (older first)
// so the primary/first-joined org lands at the top of any picker UI.
func (s *Service) ListMyMemberships(userID string) ([]MyMembership, error) {
	memberships, err := s.repo.FindMembershipsByUser(userID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return []MyMembership{}, nil
	}
	ids := make([]string, len(memberships))
	for i, m := range memberships {
		ids[i] = m.OrganizationID
	}
	orgs, err := s.repo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}
	byID := map[string]domain.Organization{}
	for _, o := range orgs {
		byID[o.ID] = o
	}
	out := make([]MyMembership, 0, len(memberships))
	for _, m := range memberships {
		org, ok := byID[m.OrganizationID]
		if !ok {
			continue // stale membership row — org was deleted; skip
		}
		out = append(out, MyMembership{Organization: org, Membership: m})
	}
	return out, nil
}

// CanAdmin returns nil if the caller may perform standard admin
// actions on the org — creating content, editing, deleting members.
// Strict: **org membership is required.** A PLATFORM_ADMIN who is
// NOT a member of this org is refused so that content the org publishes
// (announcements, projects, consultations) is always attributed to the
// org itself, not to a platform admin acting on their behalf.
//
// For emergency-only actions where the platform must be able to
// intervene without joining the org first (e.g. closing a bad
// consultation, archiving an announcement), callers should use
// {@link CanClose} instead.
func (s *Service) CanAdmin(orgID, userID, userRole string) error {
	m, err := s.repo.FindMember(orgID, userID)
	if err != nil {
		return &AppError{Code: "FORBIDDEN", Message: "Only org owners or admins can do this", Status: http.StatusForbidden}
	}
	if m.Role != domain.MemberRoleOwner && m.Role != domain.MemberRoleAdmin {
		return &AppError{Code: "FORBIDDEN", Message: "Only org owners or admins can do this", Status: http.StatusForbidden}
	}
	return nil
}

// CanClose returns nil for the emergency-close subset of admin actions:
// closing a published consultation, archiving an announcement. Platform
// admins are allowed here because a bad-actor or compromised-account
// scenario needs a lever that doesn't require joining the org first.
// Everything CanClose allows is either destructive-to-visibility OR
// state-freezing, never authorship: the underlying record's authorId
// still names the org.
func (s *Service) CanClose(orgID, userID, userRole string) error {
	if userRole == "PLATFORM_ADMIN" {
		return nil
	}
	return s.CanAdmin(orgID, userID, userRole)
}

// CanReadInternal gates admin-only reads — response lists, analytics,
// draft content. Members of the org (any role including STAFF) and
// PLATFORM_ADMIN both qualify. This is separate from CanAdmin because
// a STAFF member should be able to look at their own org's analytics
// without being able to publish.
func (s *Service) CanReadInternal(orgID, userID, userRole string) error {
	if userRole == "PLATFORM_ADMIN" {
		return nil
	}
	if _, err := s.repo.FindMember(orgID, userID); err != nil {
		return &AppError{Code: "FORBIDDEN", Message: "Only members of this organization can view this", Status: http.StatusForbidden}
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
