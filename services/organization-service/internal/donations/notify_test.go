package donations

import (
	"context"
	"strings"
	"testing"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/notifications"
)

type sentNote struct {
	users []string
	kind  notifications.NotificationType
	title string
	body  string
}

type fakeNotifier struct{ sent []sentNote }

func (f *fakeNotifier) EmitMany(users []string, t notifications.NotificationType, title, body string, _ *string) {
	f.sent = append(f.sent, sentNote{users: users, kind: t, title: title, body: body})
}

type fakeAudience struct{ org, donors []string }

func (f *fakeAudience) OrgMembers(string) []string { return f.org }
func (f *fakeAudience) Donors(string) []string     { return f.donors }
func (f *fakeAudience) Stakeholders(string, string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range append(append([]string{}, f.donors...), f.org...) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (f *fakeNotifier) of(kind notifications.NotificationType) []sentNote {
	var out []sentNote
	for _, n := range f.sent {
		if n.kind == kind {
			out = append(out, n)
		}
	}
	return out
}

func withNotify(t *testing.T, st *fakeStore, svc *Service, d *domain.Donation) (*fakeNotifier, *fakeAudience) {
	t.Helper()
	n := &fakeNotifier{}
	a := &fakeAudience{org: []string{"org-user-1"}, donors: []string{"donor-1"}}
	svc.WithNotifications(n, a)
	st.campaigns[d.CampaignID].Title = "Flood relief for Sabon Gari"
	st.campaigns[d.CampaignID].Slug = "flood-relief-sabon-gari"
	return n, a
}

// The organization needs to know money arrived. Previous donors do not need
// a notification for every subsequent donation — that would make the tray
// unusable on a busy campaign, and they already have their own receipt.
func TestNotify_DonationReceivedGoesToTheOrgNotToDonors(t *testing.T) {
	st, svc, d := newSettledFixture(t, 250_000)
	n, _ := withNotify(t, st, svc, d)

	body := st.p.body(d.ProviderRef, 250_000)
	if err := svc.HandleWebhook(context.Background(), body, st.p.sigFor(body)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	got := n.of(notifications.TypeDonationReceived)
	if len(got) != 1 {
		t.Fatalf("DONATION_RECEIVED count = %d, want 1", len(got))
	}
	for _, u := range got[0].users {
		if u == "donor-1" {
			t.Fatal("a previous donor was notified about someone else's donation")
		}
	}
	if !strings.Contains(got[0].title, "₦2,500.00") {
		t.Errorf("amount should be in the title, got %q", got[0].title)
	}
}

// Goal reached is announced at the crossing, to everyone with a stake.
func TestNotify_GoalReachedAnnouncedOnceToEveryone(t *testing.T) {
	st, svc, d := newSettledFixture(t, 100_000_000) // fixture goal is 100_000_000
	n, _ := withNotify(t, st, svc, d)

	body := st.p.body(d.ProviderRef, 100_000_000)
	if err := svc.HandleWebhook(context.Background(), body, st.p.sigFor(body)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	got := n.of(notifications.TypeFundingGoalReached)
	if len(got) != 1 {
		t.Fatalf("FUNDING_GOAL_REACHED count = %d, want 1", len(got))
	}
	users := strings.Join(got[0].users, ",")
	if !strings.Contains(users, "donor-1") || !strings.Contains(users, "org-user-1") {
		t.Fatalf("goal-reached audience = %v, want donors and org members", got[0].users)
	}
}

// Short of the goal, nothing is announced. Telling donors a target was hit
// when it was not would be the most damaging notification in the system.
func TestNotify_NoGoalAnnouncementBelowTarget(t *testing.T) {
	st, svc, d := newSettledFixture(t, 250_000) // far short of 100_000_000
	n, _ := withNotify(t, st, svc, d)

	body := st.p.body(d.ProviderRef, 250_000)
	if err := svc.HandleWebhook(context.Background(), body, st.p.sigFor(body)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if got := n.of(notifications.TypeFundingGoalReached); len(got) != 0 {
		t.Fatalf("announced goal reached at %d of %d", 250_000, 100_000_000)
	}
}

// Paystack retries deliveries. A replay must not re-announce anything.
func TestNotify_ReplayedWebhookAnnouncesNothingTwice(t *testing.T) {
	st, svc, d := newSettledFixture(t, 100_000_000)
	n, _ := withNotify(t, st, svc, d)

	body := st.p.body(d.ProviderRef, 100_000_000)
	if err := svc.HandleWebhook(context.Background(), body, st.p.sigFor(body)); err != nil {
		t.Fatalf("first: %v", err)
	}
	replay := st.p.bodyWithEvent(d.ProviderRef, 100_000_000, 424242)
	if err := svc.HandleWebhook(context.Background(), replay, st.p.sigFor(replay)); err != nil {
		t.Fatalf("replay: %v", err)
	}

	if got := n.of(notifications.TypeDonationReceived); len(got) != 1 {
		t.Fatalf("DONATION_RECEIVED sent %d times after a replay", len(got))
	}
	if got := n.of(notifications.TypeFundingGoalReached); len(got) != 1 {
		t.Fatalf("FUNDING_GOAL_REACHED sent %d times after a replay", len(got))
	}
}

// A donation recovered by reconciliation days later still notifies: the
// money moved, and when we found out does not change who deserves to know.
func TestNotify_RecoveredDonationStillAnnounces(t *testing.T) {
	st, svc, d := newSettledFixture(t, 100_000_000)
	n, _ := withNotify(t, st, svc, d)
	aged(d, 2*3600*1e9) // older than the grace period
	st.p.says(d.ProviderRef, TransactionStatus{Succeeded: true, AmountMinor: 100_000_000, Currency: "NGN"})

	if _, err := svc.Reconcile(context.Background(), ReconcileOptions{}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if got := n.of(notifications.TypeDonationReceived); len(got) != 1 {
		t.Fatalf("a recovered donation announced %d times, want 1", len(got))
	}
	if got := n.of(notifications.TypeFundingGoalReached); len(got) != 1 {
		t.Fatalf("goal reached announced %d times on recovery, want 1", len(got))
	}
}

// Notifications are optional infrastructure. Settlement must not depend on
// them being wired.
func TestNotify_SettlesWithoutANotifier(t *testing.T) {
	st, svc, d := newSettledFixture(t, 250_000)
	body := st.p.body(d.ProviderRef, 250_000)
	if err := svc.HandleWebhook(context.Background(), body, st.p.sigFor(body)); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if st.donations[d.ID].Status != domain.DonationSettled {
		t.Fatal("settlement failed with no notifier attached")
	}
}
