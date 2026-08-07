package organizations

import (
	"testing"

	"github.com/civicos/organization-service/internal/domain"
)

func operateFixture() *Service {
	repo := newFakeMemberRepo()
	repo.members["org-1/staff-1"] = &domain.OrgMember{
		OrganizationID: "org-1", UserID: "staff-1", Role: domain.MemberRoleStaff,
	}
	repo.members["org-1/admin-1"] = &domain.OrgMember{
		OrganizationID: "org-1", UserID: "admin-1", Role: domain.MemberRoleAdmin,
	}
	repo.members["org-1/owner-1"] = &domain.OrgMember{
		OrganizationID: "org-1", UserID: "owner-1", Role: domain.MemberRoleOwner,
	}
	return NewService(repo)
}

// The whole point of the change: a field officer can record work.
func TestStaffCanOperate(t *testing.T) {
	svc := operateFixture()
	for _, user := range []string{"staff-1", "admin-1", "owner-1"} {
		if err := svc.CanOperate("org-1", user); err != nil {
			t.Fatalf("%s must be able to operate: %v", user, err)
		}
	}
}

// ...and the boundary it must not cross. STAFF records work; it does not
// commit the organization to anything.
func TestStaffStillCannotAdmin(t *testing.T) {
	svc := operateFixture()
	if err := svc.CanAdmin("org-1", "staff-1", "CITIZEN"); err == nil {
		t.Fatal("STAFF must not pass CanAdmin — that gate guards announcements, campaigns, member management and org edits")
	}
	// Elevating the platform role changes nothing: org permissions come
	// from org membership.
	if err := svc.CanAdmin("org-1", "staff-1", "PLATFORM_ADMIN"); err == nil {
		t.Fatal("a platform role must not substitute for an org role")
	}
	for _, user := range []string{"admin-1", "owner-1"} {
		if err := svc.CanAdmin("org-1", user, "CITIZEN"); err != nil {
			t.Fatalf("%s must pass CanAdmin: %v", user, err)
		}
	}
}

// Non-members get nothing, whatever their platform role. CanOperate is
// deliberately narrower than CanReadInternal, which does admit
// PLATFORM_ADMIN for oversight reads — an operational record carries the
// organization's voice and has to come from the organization.
func TestCanOperateRejectsNonMembers(t *testing.T) {
	svc := operateFixture()
	for _, user := range []string{"stranger", "", "admin-of-another-org"} {
		if err := svc.CanOperate("org-1", user); err == nil {
			t.Fatalf("%q must not be able to operate on org-1", user)
		}
	}
	// Membership is per-organization, not global.
	if err := svc.CanOperate("org-2", "staff-1"); err == nil {
		t.Fatal("membership in org-1 must not grant anything in org-2")
	}
}
