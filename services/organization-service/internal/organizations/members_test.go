package organizations

import (
	"strings"
	"testing"

	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type fakeMemberRepo struct {
	OrgStore
	users   map[string]*userRecord // keyed by lowercase email
	byID    map[string]*userRecord
	members map[string]*domain.OrgMember // keyed by orgID+"/"+userID
	bumps   int
}

func newFakeMemberRepo() *fakeMemberRepo {
	ada := &userRecord{ID: "user-1", Email: "Ada@Example.com", Name: "Ada Okafor", Role: "CITIZEN"}
	return &fakeMemberRepo{
		users:   map[string]*userRecord{"ada@example.com": ada},
		byID:    map[string]*userRecord{"user-1": ada},
		members: map[string]*domain.OrgMember{},
	}
}

func (f *fakeMemberRepo) FindUserByEmail(email string) (*userRecord, error) {
	// Mirrors the repository's LOWER(email) = LOWER(?) match.
	if u, ok := f.users[strings.ToLower(strings.TrimSpace(email))]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeMemberRepo) FindUserByID(id string) (*userRecord, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeMemberRepo) FindMember(orgID, userID string) (*domain.OrgMember, error) {
	if m, ok := f.members[orgID+"/"+userID]; ok {
		return m, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeMemberRepo) AddMember(m *domain.OrgMember) error {
	f.members[m.OrganizationID+"/"+m.UserID] = m
	return nil
}

func (f *fakeMemberRepo) BumpMemberCount(orgID string, delta int) error {
	f.bumps += delta
	return nil
}

func (f *fakeMemberRepo) UpdateMemberRole(orgID, userID string, role domain.OrgMemberRole) error {
	m, ok := f.members[orgID+"/"+userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	m.Role = role
	return nil
}

func (f *fakeMemberRepo) UpdateMemberTitle(orgID, userID string, title *string) error {
	m, ok := f.members[orgID+"/"+userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	m.Title = title
	return nil
}

func TestAddMemberByEmail(t *testing.T) {
	repo := newFakeMemberRepo()
	svc := NewService(repo)
	title := "Head of Distribution"

	m, err := svc.AddMember("org-1", AddMemberInput{
		// Deliberately a different case from the stored address: nobody
		// types their own email the same way twice.
		Email: "ADA@example.com",
		Role:  "STAFF",
		Title: &title,
	})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if m.UserID != "user-1" {
		t.Fatalf("expected user-1, got %s", m.UserID)
	}
	if m.Title == nil || *m.Title != title {
		t.Fatalf("expected title %q, got %v", title, m.Title)
	}
	if repo.bumps != 1 {
		t.Fatalf("expected member count bumped once, got %d", repo.bumps)
	}
}

// Name and platform role come from `users`, never from the request. The old
// shape took both from the client, so a caller could file a colleague under
// any name they liked.
func TestAddMemberIgnoresClientSuppliedIdentity(t *testing.T) {
	repo := newFakeMemberRepo()
	svc := NewService(repo)

	m, err := svc.AddMember("org-1", AddMemberInput{Email: "ada@example.com", Role: "ADMIN"})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if m.UserName != "Ada Okafor" {
		t.Fatalf("name must come from users, got %q", m.UserName)
	}
	if m.UserRole != "CITIZEN" {
		t.Fatalf("platform role must come from users, got %q", m.UserRole)
	}
}

// A representative can be a member of an organization like anyone else —
// a councillor sitting on a water board's oversight committee is both an
// elected official and a member of that org. Nothing about the platform
// role restricts this.
func TestAddMemberAcceptsARepresentative(t *testing.T) {
	repo := newFakeMemberRepo()
	rep := &userRecord{ID: "user-2", Email: "rep@example.com", Name: "Bode Adeyemi", Role: "REPRESENTATIVE"}
	repo.users["rep@example.com"] = rep
	repo.byID["user-2"] = rep
	svc := NewService(repo)

	m, err := svc.AddMember("org-1", AddMemberInput{Email: "rep@example.com", Role: "ADMIN"})
	if err != nil {
		t.Fatalf("a representative must be addable: %v", err)
	}
	if m.UserRole != "REPRESENTATIVE" {
		t.Fatalf("expected REPRESENTATIVE, got %q", m.UserRole)
	}
}

func TestAddMemberValidation(t *testing.T) {
	cases := []struct {
		name  string
		input AddMemberInput
		want  string
	}{
		{"no identifier", AddMemberInput{Role: "STAFF"}, "VALIDATION_ERROR"},
		{"unknown role", AddMemberInput{Email: "ada@example.com", Role: "SUPERUSER"}, "INVALID_ROLE"},
		{"unknown email", AddMemberInput{Email: "nobody@example.com", Role: "STAFF"}, "USER_NOT_FOUND"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(newFakeMemberRepo())
			_, err := svc.AddMember("org-1", tc.input)
			var appErr *AppError
			if !asAppErr(err, &appErr) || appErr.Code != tc.want {
				t.Fatalf("expected %s, got %v", tc.want, err)
			}
		})
	}
}

// The "no such account" message must not confirm or deny that an address is
// registered. Membership is invitation-only; this endpoint is not an
// account-existence oracle for whoever happens to own an org.
func TestUnknownEmailDoesNotLeakAccountExistence(t *testing.T) {
	svc := NewService(newFakeMemberRepo())
	_, err := svc.AddMember("org-1", AddMemberInput{Email: "someone@example.com", Role: "STAFF"})
	var appErr *AppError
	if !asAppErr(err, &appErr) {
		t.Fatalf("expected an AppError, got %v", err)
	}
	if containsAny(appErr.Message, []string{"someone@example.com", "not registered", "no account with"}) {
		t.Fatalf("message leaks whether the address exists: %q", appErr.Message)
	}
}

func TestAddMemberRejectsDuplicateAndDeleted(t *testing.T) {
	repo := newFakeMemberRepo()
	svc := NewService(repo)

	if _, err := svc.AddMember("org-1", AddMemberInput{Email: "ada@example.com", Role: "STAFF"}); err != nil {
		t.Fatalf("first add: %v", err)
	}
	_, err := svc.AddMember("org-1", AddMemberInput{Email: "ada@example.com", Role: "ADMIN"})
	var appErr *AppError
	if !asAppErr(err, &appErr) || appErr.Code != "ALREADY_MEMBER" {
		t.Fatalf("expected ALREADY_MEMBER, got %v", err)
	}

	when := "2026-01-01T00:00:00Z"
	repo.users["ada@example.com"].DeletedAt = &when
	_, err = svc.AddMember("org-2", AddMemberInput{Email: "ada@example.com", Role: "STAFF"})
	if !asAppErr(err, &appErr) || appErr.Code != "USER_DELETED" {
		t.Fatalf("expected USER_DELETED, got %v", err)
	}
}

func TestUpdateMemberRoleAndTitle(t *testing.T) {
	repo := newFakeMemberRepo()
	svc := NewService(repo)
	if _, err := svc.AddMember("org-1", AddMemberInput{Email: "ada@example.com", Role: "STAFF"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	newTitle := "Managing Director"
	if err := svc.UpdateMember("org-1", "user-1", UpdateMemberInput{Role: "ADMIN", Title: &newTitle}); err != nil {
		t.Fatalf("update: %v", err)
	}
	m := repo.members["org-1/user-1"]
	if m.Role != domain.MemberRoleAdmin {
		t.Fatalf("expected ADMIN, got %s", m.Role)
	}
	if m.Title == nil || *m.Title != newTitle {
		t.Fatalf("expected title %q, got %v", newTitle, m.Title)
	}

	// Omitting the title must leave the existing one, not blank it.
	if err := svc.UpdateMember("org-1", "user-1", UpdateMemberInput{Role: "STAFF"}); err != nil {
		t.Fatalf("update without title: %v", err)
	}
	if m.Title == nil || *m.Title != newTitle {
		t.Fatalf("title must survive a role-only update, got %v", m.Title)
	}
}

// An all-whitespace title should become no title, not a blank line in the
// member list.
func TestBlankTitleBecomesNil(t *testing.T) {
	repo := newFakeMemberRepo()
	svc := NewService(repo)
	blank := "   "

	m, err := svc.AddMember("org-1", AddMemberInput{Email: "ada@example.com", Role: "STAFF", Title: &blank})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if m.Title != nil {
		t.Fatalf("expected no title, got %q", *m.Title)
	}
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
