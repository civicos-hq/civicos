package notifications

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// The event must describe the committed row exactly. community-service
// pushes it to the browser without re-reading the database, so anything
// wrong or missing here is wrong in the user's notification tray.
func TestEventFor_CarriesTheRealRow(t *testing.T) {
	link := "/campaigns/flood-relief-sabon-gari"
	created := time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC)
	row := &Notification{
		ID: "11111111-2222-3333-4444-555555555555", Type: TypeAnnouncementUpdate,
		Title: "Flood relief update", Body: "Two boreholes completed.",
		Read: false, LinkURL: &link, UserID: "user-1", CreatedAt: created,
	}

	e := eventFor(row)

	if e.ID != row.ID {
		t.Fatalf("id = %q, want the row's own id — the client cannot mark a phantom id as read", e.ID)
	}
	if e.Type != string(TypeAnnouncementUpdate) || e.Title != row.Title || e.Body != row.Body {
		t.Fatalf("event does not describe the row: %+v", e)
	}
	if e.UserID != "user-1" {
		t.Fatalf("userId = %q — the hub routes on this, so a wrong value delivers to the wrong person", e.UserID)
	}
	if e.LinkURL == nil || *e.LinkURL != link {
		t.Fatalf("link not carried: %v", e.LinkURL)
	}
	if !e.CreatedAt.Equal(created) {
		t.Fatalf("createdAt = %v, want %v — the client sorts on this", e.CreatedAt, created)
	}
}

// The wire format is a cross-module contract: community-service declares its
// own struct and decodes these bytes. The JSON keys must match what the SSE
// stream already sends to browsers.
func TestEvent_WireFormatKeys(t *testing.T) {
	link := "/x"
	b, err := json.Marshal(eventFor(&Notification{
		ID: "id-1", Type: TypeConsultationUpdate, Title: "T", Body: "B",
		LinkURL: &link, UserID: "u-1", CreatedAt: time.Now().UTC(),
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"id", "type", "title", "body", "read", "linkUrl", "userId", "createdAt"} {
		if _, ok := m[key]; !ok {
			t.Errorf("wire format is missing %q — the bridge on the other side decodes by these names", key)
		}
	}
}

// Realtime is an enhancement. No broker configured must yield no bus and no
// error, and a nil bus must be safe to hold.
func TestConnectNATS_NoURLIsNotAnError(t *testing.T) {
	if bus := ConnectNATS(""); bus != nil {
		t.Fatal("no URL should yield no bus")
	}
}

// A typed-nil *NATSBus must not panic if one ever reaches a call site.
func TestNATSBus_NilIsSafe(t *testing.T) {
	var b *NATSBus
	b.PublishNotification(Event{ID: "x", UserID: "u"})
	b.Close()
}

// community-service owns the notifications schema; this service only mirrors
// the enum. Nothing structural keeps the two in sync — they are separate Go
// modules — so a value changed on one side would silently produce
// notifications the client cannot filter or route.
//
// This reads the canonical source and compares. It skips rather than fails
// when the sibling checkout is not present, so it can never become a false
// alarm in an environment that builds one service in isolation.
func TestNotificationTypes_MatchCommunityService(t *testing.T) {
	const canonical = "../../../community-service/internal/domain/models.go"
	src, err := os.ReadFile(canonical)
	if err != nil {
		t.Skipf("canonical enum not readable here (%v) — skipping drift check", err)
	}
	text := string(src)

	mirrored := []NotificationType{
		TypeConsultationUpdate, TypeAnnouncementUpdate,
		TypeCampaignApproved, TypeDonationReceived, TypeMilestoneCompleted,
		TypeCampaignUpdate, TypeFundingGoalReached, TypeCampaignCompleted,
	}
	for _, v := range mirrored {
		want := `NotificationType = "` + string(v) + `"`
		if !strings.Contains(text, want) {
			t.Errorf("%q is not declared in community-service — the two enums have drifted", v)
		}
	}
}
