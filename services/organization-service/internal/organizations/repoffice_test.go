package organizations

import (
	"strings"
	"testing"

	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

// fakeOfficeRepo implements only what ProvisionRepresentativeOffice needs;
// the rest of OrgStore panics loudly so a future caller can't quietly rely
// on unimplemented behaviour.
type fakeOfficeRepo struct {
	OrgStore
	reps        map[string]*representativeRecord // keyed by user ID
	communities map[string]*communityRecord
	offices     map[string]*domain.Organization // keyed by representative ID
	members     []*domain.OrgMember
	createCalls int
	// raceWinner stands in for a concurrent provision that committed
	// between our existence check and our insert. When set, CreateOffice
	// fails the way the partial unique index would and the winner's row is
	// what a re-read finds.
	raceWinner *domain.Organization
}

func newFakeOfficeRepo() *fakeOfficeRepo {
	return &fakeOfficeRepo{
		reps:        map[string]*representativeRecord{},
		communities: map[string]*communityRecord{},
		offices:     map[string]*domain.Organization{},
	}
}

func (f *fakeOfficeRepo) FindRepresentativeByUserID(userID string) (*representativeRecord, error) {
	if rec, ok := f.reps[userID]; ok {
		return rec, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeOfficeRepo) FindCommunity(id string) (*communityRecord, error) {
	if c, ok := f.communities[id]; ok {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeOfficeRepo) FindMember(orgID, userID string) (*domain.OrgMember, error) {
	for _, m := range f.members {
		if m.OrganizationID == orgID && m.UserID == userID {
			return m, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeOfficeRepo) FindOfficeByRepresentativeID(repID string) (*domain.Organization, error) {
	if o, ok := f.offices[repID]; ok {
		return o, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeOfficeRepo) CreateOffice(org *domain.Organization, owner *domain.OrgMember) error {
	f.createCalls++
	if f.raceWinner != nil {
		// The index rejects us; the winner's row is now what a re-read
		// finds.
		f.offices[*f.raceWinner.RepresentativeID] = f.raceWinner
		f.raceWinner = nil
		return gorm.ErrDuplicatedKey
	}
	f.offices[*org.RepresentativeID] = org
	f.members = append(f.members, owner)
	return nil
}

func fixture(t *testing.T) (*Service, *fakeOfficeRepo) {
	t.Helper()
	repo := newFakeOfficeRepo()
	repo.reps["user-1"] = &representativeRecord{
		ID:           "rep-1",
		Name:         "Ada Okafor",
		Title:        "Senator",
		Position:     "Senator, Federal Republic",
		Constituency: "Enugu East",
		CommunityID:  "community-1",
	}
	repo.communities["community-1"] = &communityRecord{
		ID: "community-1", Name: "Enugu East", State: "Enugu", LGA: "Enugu East",
	}
	return NewService(repo), repo
}

func TestProvisionCreatesOfficeWithOwner(t *testing.T) {
	svc, repo := fixture(t)

	org, err := svc.ProvisionRepresentativeOffice("user-1", "REPRESENTATIVE")
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	if org.Kind != domain.OrgKindRepresentativeOffice {
		t.Fatalf("expected REPRESENTATIVE_OFFICE, got %s", org.Kind)
	}
	if org.RepresentativeID == nil || *org.RepresentativeID != "rep-1" {
		t.Fatalf("office must link back to the profile, got %v", org.RepresentativeID)
	}
	if org.Name != "Office of Senator Ada Okafor" {
		t.Fatalf("unexpected office name: %q", org.Name)
	}
	// Placed where the representative serves, so campaigns tier correctly.
	if org.State == nil || *org.State != "Enugu" || org.LGA == nil || *org.LGA != "Enugu East" {
		t.Fatalf("expected Enugu/Enugu East, got %v/%v", org.State, org.LGA)
	}

	// The caller must be able to administer it, or the office is unusable.
	if len(repo.members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(repo.members))
	}
	if repo.members[0].Role != domain.MemberRoleOwner || repo.members[0].UserID != "user-1" {
		t.Fatalf("expected user-1 as OWNER, got %+v", repo.members[0])
	}
	// The point of the whole design: CanAdmin is the single gate in front of
	// creating campaigns, projects, consultations and announcements. If the
	// representative passes it for their own office, they get all four
	// without any of those modules changing.
	if err := svc.CanAdmin(org.ID, "user-1", "REPRESENTATIVE"); err != nil {
		t.Fatalf("representative must be able to administer their own office: %v", err)
	}
	// And nobody else does.
	if err := svc.CanAdmin(org.ID, "someone-else", "REPRESENTATIVE"); err == nil {
		t.Fatal("another representative must not be able to administer this office")
	}
}

// Provisioning starts an office in exactly the position a newly registered
// organization is in: it can draft, and it cannot take a naira.
func TestProvisionedOfficeCannotYetRaiseMoney(t *testing.T) {
	svc, _ := fixture(t)

	org, err := svc.ProvisionRepresentativeOffice("user-1", "REPRESENTATIVE")
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	if org.Verified {
		t.Fatal("a freshly provisioned office must not be verified")
	}
	eligible, missing := org.FundingEligible()
	if eligible {
		t.Fatal("a freshly provisioned office must not be funding-eligible")
	}
	for _, want := range []string{"organization verification", "connected payout account"} {
		found := false
		for _, m := range missing {
			if m == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %q among missing requirements, got %v", want, missing)
		}
	}
}

func TestProvisionIsIdempotent(t *testing.T) {
	svc, repo := fixture(t)

	first, err := svc.ProvisionRepresentativeOffice("user-1", "REPRESENTATIVE")
	if err != nil {
		t.Fatalf("first provision: %v", err)
	}
	second, err := svc.ProvisionRepresentativeOffice("user-1", "REPRESENTATIVE")
	if err != nil {
		t.Fatalf("second provision: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected the same office, got %s then %s", first.ID, second.ID)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected exactly 1 create, got %d", repo.createCalls)
	}
}

// Losing the check-then-create race must return the office that won, not
// an error: the caller asked for an office and there now is one.
func TestProvisionSurvivesLosingTheRace(t *testing.T) {
	svc, repo := fixture(t)
	// A concurrent provision — a double-clicked button, a client retry —
	// commits between our existence check and our insert.
	repo.raceWinner = &domain.Organization{ID: "org-winner", RepresentativeID: strptr("rep-1")}

	got, err := svc.ProvisionRepresentativeOffice("user-1", "REPRESENTATIVE")
	if err != nil {
		t.Fatalf("losing the race must not surface as an error, got %v", err)
	}
	if got.ID != "org-winner" {
		t.Fatalf("expected the winning office, got %s", got.ID)
	}
}

func TestProvisionRejectsNonRepresentative(t *testing.T) {
	svc, _ := fixture(t)

	_, err := svc.ProvisionRepresentativeOffice("user-1", "CITIZEN")
	var appErr *AppError
	if !asAppErr(err, &appErr) || appErr.Code != "NOT_A_REPRESENTATIVE" {
		t.Fatalf("expected NOT_A_REPRESENTATIVE, got %v", err)
	}
}

// A REPRESENTATIVE account with no claimed profile is an administrative
// gap, not a forbidden action — and the error has to say who fixes it.
func TestProvisionRejectsUnclaimedProfile(t *testing.T) {
	svc, repo := fixture(t)
	delete(repo.reps, "user-1")

	_, err := svc.ProvisionRepresentativeOffice("user-1", "REPRESENTATIVE")
	var appErr *AppError
	if !asAppErr(err, &appErr) || appErr.Code != "REPRESENTATIVE_UNCLAIMED" {
		t.Fatalf("expected REPRESENTATIVE_UNCLAIMED, got %v", err)
	}
	if !strings.Contains(appErr.Message, "admin") {
		t.Fatalf("error must name who can fix it, got %q", appErr.Message)
	}
}

func TestOfficeNameDoesNotDoubleTheTitle(t *testing.T) {
	cases := []struct {
		name, title, want string
	}{
		{"Ada Okafor", "Senator", "Office of Senator Ada Okafor"},
		// Some profiles store the title inside the name; don't produce
		// "Office of Senator Senator Ada Okafor".
		{"Senator Ada Okafor", "Senator", "Office of Senator Ada Okafor"},
		{"Ada Okafor", "", "Office of Ada Okafor"},
	}
	for _, tc := range cases {
		got := officeName(&representativeRecord{Name: tc.name, Title: tc.title})
		if got != tc.want {
			t.Fatalf("officeName(%q, %q) = %q, want %q", tc.name, tc.title, got, tc.want)
		}
	}
}

func TestJurisdictionHeuristic(t *testing.T) {
	cases := map[string]domain.OrgJurisdiction{
		"Senator, Federal Republic":        domain.JurisdictionNational,
		"Member, House of Representatives": domain.JurisdictionNational,
		"Governor":                         domain.JurisdictionState,
		"Councillor, Ward 3":               domain.JurisdictionLGA,
		"Local Government Chairman":        domain.JurisdictionLGA,
		// Unrecognised titles fall to the NARROWEST reach, so a bad guess
		// under-claims rather than letting an office look national.
		"Community liaison": domain.JurisdictionCommunity,
		"":                  domain.JurisdictionCommunity,
	}
	for position, want := range cases {
		if got := jurisdictionFor(position); got != want {
			t.Fatalf("jurisdictionFor(%q) = %s, want %s", position, got, want)
		}
	}
}

func strptr(s string) *string { return &s }

func asAppErr(err error, target **AppError) bool {
	if e, ok := err.(*AppError); ok {
		*target = e
		return true
	}
	return false
}
