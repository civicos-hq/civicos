package invitations

import (
	"strings"
	"testing"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type fakeStore struct {
	usersByEmail map[string]*userRecord
	usersByID    map[string]*userRecord
	orgs         map[string]*domain.Organization
	members      map[string]*domain.OrgMember     // orgID/userID
	invites      map[string]*domain.OrgInvitation // by ID
}

func newFakeStore() *fakeStore {
	ada := &userRecord{ID: "user-ada", Email: "ada@water.example", Name: "Ada Okafor", Role: "CITIZEN"}
	return &fakeStore{
		usersByEmail: map[string]*userRecord{"ada@water.example": ada},
		usersByID:    map[string]*userRecord{"user-ada": ada},
		orgs:         map[string]*domain.Organization{"org-1": {ID: "org-1", Name: "Abuja Water Board"}},
		members:      map[string]*domain.OrgMember{},
		invites:      map[string]*domain.OrgInvitation{},
	}
}

func (f *fakeStore) FindUserByEmail(email string) (*userRecord, error) {
	if u, ok := f.usersByEmail[normalizeEmail(email)]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeStore) FindUserByID(id string) (*userRecord, error) {
	if u, ok := f.usersByID[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeStore) FindOrganization(id string) (*domain.Organization, error) {
	if o, ok := f.orgs[id]; ok {
		return o, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeStore) FindMember(orgID, userID string) (*domain.OrgMember, error) {
	if m, ok := f.members[orgID+"/"+userID]; ok {
		return m, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeStore) FindByTokenHash(hash string) (*domain.OrgInvitation, error) {
	for _, inv := range f.invites {
		if inv.TokenHash == hash {
			return inv, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeStore) FindByID(id string) (*domain.OrgInvitation, error) {
	if inv, ok := f.invites[id]; ok {
		return inv, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeStore) ListPending(orgID string) ([]domain.OrgInvitation, error) {
	var out []domain.OrgInvitation
	for _, inv := range f.invites {
		if inv.OrganizationID == orgID && inv.Pending(time.Now()) {
			out = append(out, *inv)
		}
	}
	return out, nil
}
func (f *fakeStore) ReplacePending(inv *domain.OrgInvitation, revokedByID string) error {
	now := time.Now().UTC()
	for _, existing := range f.invites {
		if existing.OrganizationID == inv.OrganizationID && existing.Email == inv.Email &&
			existing.AcceptedAt == nil && existing.RevokedAt == nil {
			existing.RevokedAt = &now
			existing.RevokedByID = &revokedByID
		}
	}
	f.invites[inv.ID] = inv
	return nil
}
func (f *fakeStore) Revoke(id, byUserID string) error {
	inv, ok := f.invites[id]
	if !ok || inv.AcceptedAt != nil || inv.RevokedAt != nil {
		return gorm.ErrRecordNotFound
	}
	now := time.Now().UTC()
	inv.RevokedAt = &now
	inv.RevokedByID = &byUserID
	return nil
}
func (f *fakeStore) Accept(invID, userID string, member *domain.OrgMember) error {
	inv, ok := f.invites[invID]
	if !ok || inv.AcceptedAt != nil || inv.RevokedAt != nil {
		return gorm.ErrRecordNotFound
	}
	now := time.Now().UTC()
	inv.AcceptedAt = &now
	inv.AcceptedUserID = &userID
	f.members[member.OrganizationID+"/"+member.UserID] = member
	return nil
}

type captureMailer struct {
	to, subject, text string
	sends             int
}

func (m *captureMailer) Send(to, subject, htmlBody, textBody string) error {
	m.to, m.subject, m.text = to, subject, textBody
	m.sends++
	return nil
}

func fixture() (*Service, *fakeStore, *captureMailer) {
	store := newFakeStore()
	mail := &captureMailer{}
	return NewService(store, mail, "https://civicos.ng"), store, mail
}

// The token in the email is the only copy; the database keeps a hash.
func tokenFromEmail(t *testing.T, body string) string {
	t.Helper()
	const marker = "/invitations/"
	i := strings.Index(body, marker)
	if i < 0 {
		t.Fatalf("no invitation link in email body: %q", body)
	}
	tail := body[i+len(marker):]
	if end := strings.IndexAny(tail, " \n\r"); end >= 0 {
		return tail[:end]
	}
	return tail
}

func TestCreateSendsALinkAndStoresOnlyAHash(t *testing.T) {
	svc, store, mail := fixture()

	inv, err := svc.Create("org-1", CreateInput{Email: "New.Hire@Water.example", Role: "STAFF"}, "owner-1", "Bode")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if mail.sends != 1 || mail.to != "new.hire@water.example" {
		t.Fatalf("expected one mail to the normalised address, got %d to %q", mail.sends, mail.to)
	}
	if inv.Email != "new.hire@water.example" {
		t.Fatalf("email must be stored lower-cased, got %q", inv.Email)
	}

	raw := tokenFromEmail(t, mail.text)
	if raw == "" {
		t.Fatal("no token in the link")
	}
	// The raw token must not be recoverable from what was persisted.
	stored := store.invites[inv.ID]
	if stored.TokenHash == raw {
		t.Fatal("the raw token was stored instead of a hash")
	}
	if stored.TokenHash != hashToken(raw) {
		t.Fatal("stored hash does not match the token that was emailed")
	}
}

// A person who was invited and signs in with that address gets in, at the
// level the inviter chose — not one they picked themselves.
func TestAcceptCreatesMembershipAtTheInvitedLevel(t *testing.T) {
	svc, store, mail := fixture()
	title := "Head of Distribution"
	if _, err := svc.Create("org-1", CreateInput{Email: "ada@water.example", Role: "ADMIN", Title: &title}, "owner-1", "Bode"); err != nil {
		t.Fatalf("create: %v", err)
	}
	raw := tokenFromEmail(t, mail.text)

	member, err := svc.Accept(raw, "user-ada")
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if member.Role != domain.MemberRoleAdmin {
		t.Fatalf("expected ADMIN, got %s", member.Role)
	}
	if member.Title == nil || *member.Title != title {
		t.Fatalf("expected title carried over, got %v", member.Title)
	}
	if member.UserName != "Ada Okafor" || member.UserRole != "CITIZEN" {
		t.Fatalf("identity must come from users, got %q/%q", member.UserName, member.UserRole)
	}
	if _, ok := store.members["org-1/user-ada"]; !ok {
		t.Fatal("membership was not created")
	}
}

// The security property the whole design rests on: a forwarded link does
// not get you in. Org ADMIN can publish and run fundraising campaigns.
func TestForwardedLinkCannotBeAcceptedByAnotherAccount(t *testing.T) {
	svc, store, mail := fixture()
	other := &userRecord{ID: "user-other", Email: "someone.else@example.com", Name: "Chidi", Role: "CITIZEN"}
	store.usersByEmail[other.Email] = other
	store.usersByID[other.ID] = other

	if _, err := svc.Create("org-1", CreateInput{Email: "ada@water.example", Role: "ADMIN"}, "owner-1", "Bode"); err != nil {
		t.Fatalf("create: %v", err)
	}
	raw := tokenFromEmail(t, mail.text)

	_, err := svc.Accept(raw, "user-other")
	var appErr *AppError
	if !asAppErr(err, &appErr) || appErr.Code != "INVITATION_EMAIL_MISMATCH" {
		t.Fatalf("expected INVITATION_EMAIL_MISMATCH, got %v", err)
	}
	// The message has to name both addresses or the person cannot work out
	// what to do next.
	for _, want := range []string{"ada@water.example", "someone.else@example.com"} {
		if !strings.Contains(appErr.Message, want) {
			t.Fatalf("message should name %q, got %q", want, appErr.Message)
		}
	}
	if _, ok := store.members["org-1/user-other"]; ok {
		t.Fatal("a mismatched account must not have been made a member")
	}
}

func TestInvitationIsSingleUse(t *testing.T) {
	svc, _, mail := fixture()
	if _, err := svc.Create("org-1", CreateInput{Email: "ada@water.example", Role: "STAFF"}, "owner-1", "Bode"); err != nil {
		t.Fatalf("create: %v", err)
	}
	raw := tokenFromEmail(t, mail.text)

	if _, err := svc.Accept(raw, "user-ada"); err != nil {
		t.Fatalf("first accept: %v", err)
	}
	_, err := svc.Accept(raw, "user-ada")
	if err == nil {
		t.Fatal("a consumed invitation must not be reusable")
	}
}

// Revoked and expired links must be indistinguishable from links that
// never existed — otherwise probing tokens confirms who an organization
// invited.
func TestUnusableInvitationsAreIndistinguishable(t *testing.T) {
	svc, store, mail := fixture()
	if _, err := svc.Create("org-1", CreateInput{Email: "ada@water.example", Role: "STAFF"}, "owner-1", "Bode"); err != nil {
		t.Fatalf("create: %v", err)
	}
	raw := tokenFromEmail(t, mail.text)
	var inv *domain.OrgInvitation
	for _, i := range store.invites {
		inv = i
	}

	// Revoked.
	if err := svc.Revoke(inv.ID, "owner-1"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	_, revokedErr := svc.Preview(raw)
	_, unknownErr := svc.Preview("a-token-that-never-existed")

	var a, b *AppError
	if !asAppErr(revokedErr, &a) || !asAppErr(unknownErr, &b) {
		t.Fatalf("expected AppErrors, got %v / %v", revokedErr, unknownErr)
	}
	if a.Code != b.Code || a.Message != b.Message || a.Status != b.Status {
		t.Fatalf("revoked and unknown must be indistinguishable:\n  revoked: %+v\n  unknown: %+v", a, b)
	}

	// Expired behaves the same way.
	inv.RevokedAt = nil
	inv.RevokedByID = nil
	inv.ExpiresAt = time.Now().Add(-time.Hour)
	_, expiredErr := svc.Preview(raw)
	var cErr *AppError
	if !asAppErr(expiredErr, &cErr) || cErr.Code != b.Code || cErr.Message != b.Message {
		t.Fatalf("expired must match unknown, got %+v", cErr)
	}
}

// Re-inviting supersedes rather than colliding with the pending-uniqueness
// index, and the superseded link stops working.
func TestReinvitingSupersedesTheOldLink(t *testing.T) {
	svc, _, mail := fixture()
	if _, err := svc.Create("org-1", CreateInput{Email: "ada@water.example", Role: "STAFF"}, "owner-1", "Bode"); err != nil {
		t.Fatalf("first invite: %v", err)
	}
	first := tokenFromEmail(t, mail.text)

	if _, err := svc.Create("org-1", CreateInput{Email: "ada@water.example", Role: "ADMIN"}, "owner-1", "Bode"); err != nil {
		t.Fatalf("second invite: %v", err)
	}
	second := tokenFromEmail(t, mail.text)
	if first == second {
		t.Fatal("re-inviting must mint a new token")
	}

	if _, err := svc.Preview(first); err == nil {
		t.Fatal("the superseded link must stop working")
	}
	p, err := svc.Preview(second)
	if err != nil {
		t.Fatalf("the new link must work: %v", err)
	}
	if p.Role != "ADMIN" {
		t.Fatalf("expected the new role, got %s", p.Role)
	}
}

func TestCreateRejectsExistingMemberAndBadRole(t *testing.T) {
	svc, store, _ := fixture()
	store.members["org-1/user-ada"] = &domain.OrgMember{OrganizationID: "org-1", UserID: "user-ada"}

	_, err := svc.Create("org-1", CreateInput{Email: "ada@water.example", Role: "STAFF"}, "owner-1", "Bode")
	var appErr *AppError
	if !asAppErr(err, &appErr) || appErr.Code != "ALREADY_MEMBER" {
		t.Fatalf("expected ALREADY_MEMBER, got %v", err)
	}

	_, err = svc.Create("org-1", CreateInput{Email: "new@water.example", Role: "SUPERUSER"}, "owner-1", "Bode")
	if !asAppErr(err, &appErr) || appErr.Code != "INVALID_ROLE" {
		t.Fatalf("expected INVALID_ROLE, got %v", err)
	}
}

// The preview is reachable by anyone holding the link, so it must not leak
// anything about the organization beyond what the invitation is for.
func TestPreviewExposesOnlyWhatTheInviteeNeeds(t *testing.T) {
	svc, _, mail := fixture()
	if _, err := svc.Create("org-1", CreateInput{Email: "ada@water.example", Role: "STAFF"}, "owner-1", "Bode"); err != nil {
		t.Fatalf("create: %v", err)
	}
	p, err := svc.Preview(tokenFromEmail(t, mail.text))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if p.OrganizationName != "Abuja Water Board" || p.Role != "STAFF" || p.InvitedByName != "Bode" {
		t.Fatalf("preview missing what the page needs: %+v", p)
	}
	if p.Email != "ada@water.example" {
		t.Fatalf("preview should name the invited address so a signed-in user can tell why it mismatched, got %q", p.Email)
	}
}

func asAppErr(err error, target **AppError) bool {
	if e, ok := err.(*AppError); ok {
		*target = e
		return true
	}
	return false
}
