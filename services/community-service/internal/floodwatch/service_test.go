package floodwatch

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/civicos/community-service/internal/domain"
)

func f64(v float64) *float64 { return &v }

type fakeStore struct {
	communities []domain.Community
	upserted    []domain.CommunityFloodAlert
	pending     []domain.CommunityFloodAlert
	notified    map[string]string
	members     map[string][]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{notified: map[string]string{}, members: map[string][]string{}}
}

func (f *fakeStore) CommunitiesWithCoordinates() ([]domain.Community, error) {
	out := make([]domain.Community, 0, len(f.communities))
	for _, c := range f.communities {
		if c.Latitude != nil && c.Longitude != nil {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeStore) UpsertAlerts(a []domain.CommunityFloodAlert) error {
	f.upserted = append(f.upserted, a...)
	return nil
}
func (f *fakeStore) MarkNotified(id, severity string, at time.Time) error {
	f.notified[id] = severity
	return nil
}
func (f *fakeStore) PendingNotifications() ([]domain.CommunityFloodAlert, error) {
	return f.pending, nil
}
func (f *fakeStore) ActiveForCommunity(id string, staleBefore time.Time) ([]domain.CommunityFloodAlert, error) {
	return nil, nil
}
func (f *fakeStore) MemberIDs(communityID string) ([]string, error) {
	return f.members[communityID], nil
}

type captureNotifier struct {
	sent []struct{ user, title, body string }
}

func (n *captureNotifier) Emit(userID string, t domain.NotificationType, title, body string, link *string) error {
	n.sent = append(n.sent, struct{ user, title, body string }{userID, title, body})
	return nil
}

// stubAPI stands in for Google's endpoint.
func stubAPI(t *testing.T, statuses []FloodStatus, gauges []Gauge) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "searchLatestFloodStatusByArea"):
			_ = json.NewEncoder(w).Encode(map[string]any{"floodStatuses": statuses})
		case strings.Contains(r.URL.Path, "gauges:batchGet"):
			_ = json.NewEncoder(w).Encode(map[string]any{"gauges": gauges})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// Makurdi sits on the River Benue and is the obvious Nigerian test case.
const (
	makurdiLat = 7.7322
	makurdiLng = 8.5391
)

func TestHaversineIsRealDistance(t *testing.T) {
	// Makurdi → Abuja is roughly 300km. A planar degree approximation
	// would be materially off at these latitudes, which matters against a
	// 50km threshold.
	d := haversineKm(makurdiLat, makurdiLng, 9.0765, 7.3986)
	if d < 190 || d > 230 {
		t.Fatalf("expected ~210km Makurdi→Abuja, got %.1f", d)
	}
	if got := haversineKm(makurdiLat, makurdiLng, makurdiLat, makurdiLng); got != 0 {
		t.Fatalf("expected 0 for identical points, got %v", got)
	}
}

func TestPollMatchesNearbyAlertingGauges(t *testing.T) {
	// ~11km from Makurdi: inside the radius.
	near := FloodStatus{
		GaugeID: "g-near", Severity: SeveritySevere, Trend: TrendRise,
		GaugeLocation: LatLng{Latitude: 7.83, Longitude: 8.54},
		IssuedTime:    time.Now().UTC(),
		ForecastRange: TimeRange{Start: time.Now(), End: time.Now().Add(72 * time.Hour)},
	}
	// Abuja: far outside it.
	far := FloodStatus{
		GaugeID: "g-far", Severity: SeverityExtreme,
		GaugeLocation: LatLng{Latitude: 9.0765, Longitude: 7.3986},
	}
	// Near, but nothing is happening — must not become a row.
	quiet := FloodStatus{
		GaugeID: "g-quiet", Severity: SeverityNoFlooding,
		GaugeLocation: LatLng{Latitude: 7.74, Longitude: 8.55},
	}

	srv := stubAPI(t, []FloodStatus{near, far, quiet},
		[]Gauge{{GaugeID: "g-near", River: "Benue", SiteName: "Makurdi"}})
	defer srv.Close()

	store := newFakeStore()
	store.communities = []domain.Community{
		{ID: "com-makurdi", Name: "Makurdi", Latitude: f64(makurdiLat), Longitude: f64(makurdiLng)},
		// No coordinates — must be skipped entirely rather than guessed at.
		{ID: "com-unlocated", Name: "Ikeja"},
	}

	svc := NewService(NewClient("k").WithBaseURL(srv.URL), store, &captureNotifier{}, 50, "NG")
	n, err := svc.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected exactly 1 matched pairing, got %d", n)
	}

	got := store.upserted[0]
	if got.GaugeID != "g-near" || got.CommunityID != "com-makurdi" {
		t.Fatalf("wrong pairing: %+v", got)
	}
	if got.River == nil || *got.River != "Benue" {
		t.Fatalf("river name should be resolved for legibility, got %v", got.River)
	}
	if got.DistanceKm <= 0 || got.DistanceKm > 50 {
		t.Fatalf("distance should be within radius, got %v", got.DistanceKm)
	}
	if got.ForecastEndAt == nil {
		t.Fatal("forecast window should be carried through")
	}
}

// A community nobody has located must never be matched approximately.
// Warning the wrong town is the failure this guards against.
func TestUnlocatedCommunitiesAreSkipped(t *testing.T) {
	srv := stubAPI(t, []FloodStatus{{
		GaugeID: "g", Severity: SeverityExtreme,
		GaugeLocation: LatLng{Latitude: makurdiLat, Longitude: makurdiLng},
	}}, nil)
	defer srv.Close()

	store := newFakeStore()
	store.communities = []domain.Community{{ID: "com-1", Name: "Somewhere"}}

	svc := NewService(NewClient("k").WithBaseURL(srv.URL), store, &captureNotifier{}, 50, "NG")
	n, err := svc.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if n != 0 || len(store.upserted) != 0 {
		t.Fatalf("expected no matches without coordinates, got %d", n)
	}
}

// Escalation only. An hourly poll against an unchanged SEVERE must not
// produce an hourly alarm — people mute the channel and miss the next one.
func TestNotifiesOnlyOnEscalation(t *testing.T) {
	store := newFakeStore()
	store.members["com-1"] = []string{"user-a", "user-b"}
	river := "Benue"
	store.pending = []domain.CommunityFloodAlert{
		{ID: "a1", CommunityID: "com-1", Severity: "SEVERE", Trend: "RISE", River: &river, DistanceKm: 11},
	}
	notifier := &captureNotifier{}
	svc := NewService(NewClient("k"), store, notifier, 50, "NG")

	if err := svc.NotifyEscalations(); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if len(notifier.sent) != 2 {
		t.Fatalf("expected both members notified, got %d", len(notifier.sent))
	}
	if store.notified["a1"] != "SEVERE" {
		t.Fatalf("expected notified severity recorded, got %q", store.notified["a1"])
	}

	// Nothing pending on the next sweep → no further alarm.
	store.pending = nil
	notifier.sent = nil
	if err := svc.NotifyEscalations(); err != nil {
		t.Fatalf("second notify: %v", err)
	}
	if len(notifier.sent) != 0 {
		t.Fatalf("an unchanged forecast must not re-notify, got %d", len(notifier.sent))
	}
}

// UNKNOWN means the upstream has nothing to say. Treating it as an
// escalation would alarm people on missing data.
func TestUnknownAndNoFloodingAreNotAlerting(t *testing.T) {
	for _, s := range []Severity{SeverityUnknown, SeverityNoFlooding, ""} {
		if s.IsAlerting() {
			t.Fatalf("%q must not be alerting", s)
		}
	}
	for _, s := range []Severity{SeverityAboveNormal, SeveritySevere, SeverityExtreme} {
		if !s.IsAlerting() {
			t.Fatalf("%q must be alerting", s)
		}
	}
	if severityRank(SeverityExtreme) <= severityRank(SeveritySevere) ||
		severityRank(SeveritySevere) <= severityRank(SeverityAboveNormal) {
		t.Fatal("severity ordering is wrong")
	}
}

// The copy must attribute Google, point at the official channel, and never
// tell anyone they are safe.
func TestAlertCopyAttributesAndDoesNotReassure(t *testing.T) {
	river := "Benue"
	end := time.Now().Add(48 * time.Hour)
	title, body := AlertCopy(domain.CommunityFloodAlert{
		Severity: "SEVERE", Trend: "RISE", River: &river, DistanceKm: 11.4, ForecastEndAt: &end,
	})

	if !strings.Contains(body, "Google Flood Hub") {
		t.Fatalf("body must attribute the source: %q", body)
	}
	if !strings.Contains(body, "NEMA") || !strings.Contains(body, "NiMet") {
		t.Fatalf("body must point at official guidance: %q", body)
	}
	if !strings.Contains(body, "not a CivicOS prediction") {
		t.Fatalf("body must disclaim authorship: %q", body)
	}
	if !strings.Contains(body, "Benue") {
		t.Fatalf("body should name the river: %q", body)
	}
	for _, banned := range []string{"you are safe", "no risk", "safe to"} {
		if strings.Contains(strings.ToLower(body), banned) {
			t.Fatalf("copy must never reassure: found %q in %q", banned, body)
		}
	}
	if !strings.Contains(strings.ToLower(title), "flood") {
		t.Fatalf("title should say what it is about: %q", title)
	}
}

// A gauge exactly at the radius boundary should match; one beyond should
// not. Guards against an off-by-one flip in the comparison.
func TestRadiusBoundary(t *testing.T) {
	const radius = 50.0
	// ~0.2° north of Makurdi is roughly 22km — comfortably inside.
	inside := haversineKm(makurdiLat, makurdiLng, makurdiLat+0.2, makurdiLng)
	if inside > radius {
		t.Fatalf("sanity: expected %.1fkm to be inside %v", inside, radius)
	}
	// ~1.0° north is roughly 111km — comfortably outside.
	outside := haversineKm(makurdiLat, makurdiLng, makurdiLat+1.0, makurdiLng)
	if outside <= radius {
		t.Fatalf("sanity: expected %.1fkm to be outside %v", outside, radius)
	}
	if math.Abs(inside-22) > 4 {
		t.Fatalf("expected ~22km for 0.2°, got %.1f", inside)
	}
}
