package audience

import "testing"

// Someone at the organization may well have donated. Without the dedupe they
// would get two identical rows in their tray for one event.
func TestDedupe_CollapsesTheOverlapAndKeepsOrder(t *testing.T) {
	got := Dedupe([]string{"donor-1", "donor-2", "org-1", "donor-1", "org-1"})
	want := []string{"donor-1", "donor-2", "org-1"}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order changed: got %v, want %v — donors are listed first deliberately", got, want)
		}
	}
}

// A missing user id would address a notification to nobody and, worse, could
// write a row with an empty recipient.
func TestDedupe_DropsEmpties(t *testing.T) {
	if got := Dedupe([]string{"", "a", "", "b"}); len(got) != 2 {
		t.Fatalf("got %v, want the two real ids", got)
	}
	if got := Dedupe(nil); len(got) != 0 {
		t.Fatalf("nil input should yield nothing, got %v", got)
	}
}

// A nil resolver must not panic: notifications are optional infrastructure.
func TestResolver_NilIsSafe(t *testing.T) {
	var r *Resolver
	if got := r.OrgMembers("org"); got != nil {
		t.Fatalf("got %v", got)
	}
	if got := r.Donors("campaign"); got != nil {
		t.Fatalf("got %v", got)
	}
}
