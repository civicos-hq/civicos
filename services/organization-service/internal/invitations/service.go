package invitations

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InvitationTTL is how long an invitation link stays usable.
//
// Two weeks: long enough to survive someone's annual leave, short enough
// that a forwarded or leaked link does not stay live indefinitely. An
// expired invitation is not deleted — the inviter can see it lapsed and
// send another.
const InvitationTTL = 14 * 24 * time.Hour

type Store interface {
	FindUserByEmail(email string) (*userRecord, error)
	FindUserByID(id string) (*userRecord, error)
	FindOrganization(id string) (*domain.Organization, error)
	FindMember(orgID, userID string) (*domain.OrgMember, error)
	FindByTokenHash(hash string) (*domain.OrgInvitation, error)
	FindByID(id string) (*domain.OrgInvitation, error)
	ListPending(orgID string) ([]domain.OrgInvitation, error)
	ReplacePending(inv *domain.OrgInvitation, revokedByID string) error
	Revoke(id, byUserID string) error
	Accept(invID, userID string, member *domain.OrgMember) error
}

// Mailer is the subset of the mailer contract this package needs.
type Mailer interface {
	Send(to, subject, htmlBody, textBody string) error
}

type Service struct {
	repo   Store
	mailer Mailer
	appURL string
}

func NewService(repo Store, m Mailer, appURL string) *Service {
	return &Service{repo: repo, mailer: m, appURL: appURL}
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }

type CreateInput struct {
	Email string  `json:"email" binding:"required,email"`
	Role  string  `json:"role" binding:"required"`
	Title *string `json:"title" binding:"omitempty,max=120"`
}

// Preview is the unauthenticated view of an invitation, shown on the accept
// page before anyone signs in.
//
// Deliberately thin. It names the organization and the role so the person
// can see what they are being asked to join, and nothing else — the page is
// reachable by anyone holding the link, so it must not become a way to read
// an organization's internal state or to confirm who else was invited.
type Preview struct {
	OrganizationID   string    `json:"organizationId"`
	OrganizationName string    `json:"organizationName"`
	Role             string    `json:"role"`
	Title            *string   `json:"title,omitempty"`
	InvitedByName    string    `json:"invitedByName"`
	Email            string    `json:"email"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

func validRole(r string) bool {
	switch domain.OrgMemberRole(r) {
	case domain.MemberRoleOwner, domain.MemberRoleAdmin, domain.MemberRoleStaff:
		return true
	}
	return false
}

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func trimTitle(t *string) *string {
	if t == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*t)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// Create issues an invitation and emails the link.
func (s *Service) Create(orgID string, in CreateInput, inviterID, inviterName string) (*domain.OrgInvitation, error) {
	if !validRole(in.Role) {
		return nil, &AppError{Code: "INVALID_ROLE", Message: "Unknown member role", Status: http.StatusBadRequest}
	}
	email := normalizeEmail(in.Email)

	org, err := s.repo.FindOrganization(orgID)
	if err != nil {
		return nil, &AppError{Code: "ORG_NOT_FOUND", Message: "Organization not found", Status: http.StatusNotFound}
	}

	// If the address already belongs to a member, say so rather than
	// sending a link that would fail on acceptance.
	if existing, uErr := s.repo.FindUserByEmail(email); uErr == nil {
		if _, mErr := s.repo.FindMember(orgID, existing.ID); mErr == nil {
			return nil, &AppError{
				Code:    "ALREADY_MEMBER",
				Message: "That person is already a member of this organization",
				Status:  http.StatusConflict,
			}
		}
	}

	raw, err := newToken()
	if err != nil {
		return nil, fmt.Errorf("generate invitation token: %w", err)
	}
	inv := &domain.OrgInvitation{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Email:          email,
		Role:           domain.OrgMemberRole(in.Role),
		Title:          trimTitle(in.Title),
		TokenHash:      hashToken(raw),
		ExpiresAt:      time.Now().Add(InvitationTTL).UTC(),
		InvitedByID:    inviterID,
		InvitedByName:  inviterName,
	}

	// Supersede any outstanding invitation for this address rather than
	// colliding with the pending-uniqueness index.
	if err := s.repo.ReplacePending(inv, inviterID); err != nil {
		return nil, err
	}

	// Fire-and-log, matching how registration treats its verification mail:
	// a mailer failure must not roll back the invitation, because the
	// inviter can resend and the record is what matters.
	if err := s.sendEmail(org, inv, raw); err != nil {
		log.Printf("[invitations.Create] email failed for org=%s invite=%s: %v", orgID, inv.ID, err)
	}
	return inv, nil
}

func (s *Service) sendEmail(org *domain.Organization, inv *domain.OrgInvitation, rawToken string) error {
	link := fmt.Sprintf("%s/invitations/%s", strings.TrimRight(s.appURL, "/"), url.PathEscape(rawToken))
	subject, html, text := InvitationEmail(org.Name, inv.InvitedByName, string(inv.Role), inv.Title, link, inv.ExpiresAt)
	return s.mailer.Send(inv.Email, subject, html, text)
}

func (s *Service) ListPending(orgID string) ([]domain.OrgInvitation, error) {
	list, err := s.repo.ListPending(orgID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return []domain.OrgInvitation{}, nil
	}
	return list, nil
}

func (s *Service) Revoke(invID, byUserID string) error {
	if err := s.repo.Revoke(invID, byUserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &AppError{
				Code:    "INVITATION_NOT_PENDING",
				Message: "That invitation has already been accepted or revoked",
				Status:  http.StatusConflict,
			}
		}
		return err
	}
	return nil
}

// OrganizationOf returns the org an invitation belongs to, so the handler
// can run the CanAdmin check against the right organization before
// revoking. Kept separate from Revoke so authorization stays in the
// handler layer with every other permission decision.
func (s *Service) OrganizationOf(invID string) (string, error) {
	inv, err := s.repo.FindByID(invID)
	if err != nil {
		return "", &AppError{Code: "INVITATION_NOT_FOUND", Message: "Invitation not found", Status: http.StatusNotFound}
	}
	return inv.OrganizationID, nil
}

// lookup resolves a raw token to a still-usable invitation.
//
// Every failure returns the same code and message. A page reachable by
// anyone holding a URL must not distinguish "no such invitation" from
// "that one was revoked" — the difference would confirm that an
// organization invited a particular address, which is exactly what someone
// probing tokens would want to learn.
func (s *Service) lookup(rawToken string) (*domain.OrgInvitation, error) {
	notUsable := &AppError{
		Code:    "INVITATION_INVALID",
		Message: "This invitation link is not valid. It may have expired, been used already, or been withdrawn.",
		Status:  http.StatusNotFound,
	}
	if strings.TrimSpace(rawToken) == "" {
		return nil, notUsable
	}
	inv, err := s.repo.FindByTokenHash(hashToken(rawToken))
	if err != nil {
		return nil, notUsable
	}
	if !inv.Pending(time.Now()) {
		return nil, notUsable
	}
	return inv, nil
}

func (s *Service) Preview(rawToken string) (*Preview, error) {
	inv, err := s.lookup(rawToken)
	if err != nil {
		return nil, err
	}
	org, err := s.repo.FindOrganization(inv.OrganizationID)
	if err != nil {
		return nil, &AppError{Code: "ORG_NOT_FOUND", Message: "Organization not found", Status: http.StatusNotFound}
	}
	return &Preview{
		OrganizationID:   org.ID,
		OrganizationName: org.Name,
		Role:             string(inv.Role),
		Title:            inv.Title,
		InvitedByName:    inv.InvitedByName,
		Email:            inv.Email,
		ExpiresAt:        inv.ExpiresAt,
	}, nil
}

// Accept turns an invitation into a membership for the signed-in caller.
//
// The caller's account email must match the address the invitation was
// sent to. Org membership is not cosmetic — an ADMIN can publish in the
// organization's name and launch fundraising campaigns — so a forwarded
// link must not be enough to get in. Matching on email means the invitation
// grants access to the person the organization chose, not to whoever opened
// the message.
func (s *Service) Accept(rawToken, userID string) (*domain.OrgMember, error) {
	inv, err := s.lookup(rawToken)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return nil, &AppError{Code: "USER_NOT_FOUND", Message: "No such user", Status: http.StatusNotFound}
	}
	if user.DeletedAt != nil {
		return nil, &AppError{Code: "USER_DELETED", Message: "That account has been deleted", Status: http.StatusBadRequest}
	}
	if normalizeEmail(user.Email) != inv.Email {
		return nil, &AppError{
			Code: "INVITATION_EMAIL_MISMATCH",
			// Names both addresses: the usual cause is being signed in to a
			// personal account while the invitation went to a work one, and
			// without both the person cannot tell what to do next.
			Message: fmt.Sprintf("This invitation was sent to %s, but you are signed in as %s. Sign in with the invited address to accept.", inv.Email, user.Email),
			Status:  http.StatusForbidden,
		}
	}

	if _, mErr := s.repo.FindMember(inv.OrganizationID, userID); mErr == nil {
		return nil, &AppError{
			Code:    "ALREADY_MEMBER",
			Message: "You are already a member of this organization",
			Status:  http.StatusConflict,
		}
	}

	member := &domain.OrgMember{
		ID:             uuid.New().String(),
		OrganizationID: inv.OrganizationID,
		UserID:         userID,
		UserName:       user.Name,
		UserRole:       user.Role,
		Title:          inv.Title,
		Role:           inv.Role,
		JoinedAt:       time.Now().UTC(),
	}
	if err := s.repo.Accept(inv.ID, userID, member); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Lost a race with a concurrent acceptance.
			return nil, &AppError{
				Code:    "INVITATION_INVALID",
				Message: "This invitation link is not valid. It may have expired, been used already, or been withdrawn.",
				Status:  http.StatusNotFound,
			}
		}
		return nil, err
	}
	return member, nil
}
