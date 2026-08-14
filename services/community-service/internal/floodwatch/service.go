package floodwatch

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
)

// DefaultMatchRadiusKm is how far a gauge can be from a community's point
// and still be considered to cover it.
//
// 50km is deliberately generous. A river gauge sits on the river, not in
// the town, and the town that floods is usually downstream of the gauge
// that reads high. Under-matching means somebody is not warned; over-
// matching means somebody reads a warning about a river 40km away, which
// the UI states plainly by showing the distance and the river name. Of the
// two failures only one is dangerous.
const DefaultMatchRadiusKm = 50.0

// Store is the persistence this package needs.
type Store interface {
	// CommunitiesWithCoordinates returns only communities an admin has
	// located. Ones without a point are skipped entirely rather than
	// matched approximately — see Community.Latitude.
	CommunitiesWithCoordinates() ([]domain.Community, error)
	UpsertAlerts(alerts []domain.CommunityFloodAlert) error
	MarkNotified(id string, severity string, at time.Time) error
	// PendingNotifications returns rows whose severity has escalated since
	// citizens were last told.
	PendingNotifications() ([]domain.CommunityFloodAlert, error)
	ActiveForCommunity(communityID string, staleBefore time.Time) ([]domain.CommunityFloodAlert, error)
	MemberIDs(communityID string) ([]string, error)
}

// Notifier is the subset of notifications.Service used here.
type Notifier interface {
	Emit(userID string, t domain.NotificationType, title, body string, linkURL *string) error
}

type Service struct {
	client   *Client
	repo     Store
	notifier Notifier
	radiusKm float64
	region   string
}

func NewService(client *Client, repo Store, notifier Notifier, radiusKm float64, region string) *Service {
	if radiusKm <= 0 {
		radiusKm = DefaultMatchRadiusKm
	}
	if region == "" {
		region = "NG"
	}
	return &Service{client: client, repo: repo, notifier: notifier, radiusKm: radiusKm, region: region}
}

// haversineKm returns the great-circle distance between two points.
//
// Not planar: at Nigeria's latitudes a degree of longitude is ~110km at
// the equator but the country spans 4°N to 14°N, and treating degrees as
// a flat grid would distort north–south distances enough to matter at a
// 50km threshold.
func haversineKm(aLat, aLng, bLat, bLng float64) float64 {
	const earthRadiusKm = 6371.0
	rad := func(d float64) float64 { return d * math.Pi / 180 }

	dLat := rad(bLat - aLat)
	dLng := rad(bLng - aLng)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rad(aLat))*math.Cos(rad(bLat))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(h))
}

// Poll fetches the current forecast for the region, matches gauges to
// located communities, and persists the result.
//
// Returns the number of alerting pairings written, for the log line.
func (s *Service) Poll(ctx context.Context) (int, error) {
	statuses, err := s.client.SearchFloodStatusByRegion(ctx, s.region)
	if err != nil {
		// A partial page walk still returns rows. Log and continue with
		// what we have rather than discarding real warnings.
		if len(statuses) == 0 {
			return 0, err
		}
		log.Printf("floodwatch: %v — proceeding with %d partial results", err, len(statuses))
	}

	communities, err := s.repo.CommunitiesWithCoordinates()
	if err != nil {
		return 0, fmt.Errorf("floodwatch: load communities: %w", err)
	}
	if len(communities) == 0 {
		// Not an error. Nobody has been given coordinates yet, so there is
		// nothing to match against — say so once rather than silently
		// doing nothing every hour.
		log.Printf("floodwatch: no community has coordinates set; nothing to match (%d forecasts fetched)", len(statuses))
		return 0, nil
	}

	// Only alerting statuses are worth matching. NO_FLOODING is the
	// overwhelming majority of any national sweep, and pairing every quiet
	// gauge with every community is O(gauges × communities) of work whose
	// only output is rows meaning "nothing is happening".
	alerting := make([]FloodStatus, 0, len(statuses)/8)
	for _, st := range statuses {
		if st.Severity.IsAlerting() {
			alerting = append(alerting, st)
		}
	}

	now := time.Now().UTC()
	var matched []domain.CommunityFloodAlert
	gaugeIDs := map[string]struct{}{}

	for _, st := range alerting {
		for i := range communities {
			com := &communities[i]
			if com.Latitude == nil || com.Longitude == nil {
				continue
			}
			d := haversineKm(*com.Latitude, *com.Longitude, st.GaugeLocation.Latitude, st.GaugeLocation.Longitude)
			if d > s.radiusKm {
				continue
			}
			gaugeIDs[st.GaugeID] = struct{}{}
			alert := domain.CommunityFloodAlert{
				ID:             uuid.New().String(),
				CommunityID:    com.ID,
				GaugeID:        st.GaugeID,
				Severity:       string(st.Severity),
				Trend:          string(st.Trend),
				DistanceKm:     math.Round(d*10) / 10,
				GaugeLatitude:  st.GaugeLocation.Latitude,
				GaugeLongitude: st.GaugeLocation.Longitude,
				IssuedAt:       st.IssuedTime,
				LastSeenAt:     now,
			}
			if !st.ForecastRange.Start.IsZero() {
				start := st.ForecastRange.Start
				alert.ForecastStartAt = &start
			}
			if !st.ForecastRange.End.IsZero() {
				end := st.ForecastRange.End
				alert.ForecastEndAt = &end
			}
			matched = append(matched, alert)
		}
	}

	// Resolve river names for the gauges that actually matched. Done after
	// matching, not before: a national sweep touches far more gauges than
	// any community is near, and this is a second API call whose size
	// should track what we will display.
	if len(gaugeIDs) > 0 {
		ids := make([]string, 0, len(gaugeIDs))
		for id := range gaugeIDs {
			ids = append(ids, id)
		}
		if gauges, gErr := s.client.BatchGetGauges(ctx, ids); gErr != nil {
			// Non-fatal. An alert without a river name is less legible but
			// still worth showing — the severity is the part that matters.
			log.Printf("floodwatch: could not resolve gauge metadata: %v", gErr)
		} else {
			byID := make(map[string]Gauge, len(gauges))
			for _, g := range gauges {
				byID[g.GaugeID] = g
			}
			for i := range matched {
				if g, ok := byID[matched[i].GaugeID]; ok {
					if river := strings.TrimSpace(g.River); river != "" {
						matched[i].River = &river
					}
					if site := strings.TrimSpace(g.SiteName); site != "" {
						matched[i].SiteName = &site
					}
				}
			}
		}
	}

	if err := s.repo.UpsertAlerts(matched); err != nil {
		return 0, fmt.Errorf("floodwatch: persist alerts: %w", err)
	}
	return len(matched), nil
}

// NotifyEscalations tells members about forecasts that have got worse
// since they were last told.
//
// Escalation only, deliberately. An hourly poll against an unchanged
// SEVERE would produce an hourly alarm, and a person who mutes CivicOS
// during a flood because it will not stop buzzing is worse off than one
// who was never notified. A forecast that eases is reflected on the
// community page but does not push.
func (s *Service) NotifyEscalations() error {
	pending, err := s.repo.PendingNotifications()
	if err != nil {
		return err
	}
	now := time.Now().UTC()

	for _, alert := range pending {
		members, mErr := s.repo.MemberIDs(alert.CommunityID)
		if mErr != nil {
			log.Printf("floodwatch: members for community=%s: %v", alert.CommunityID, mErr)
			continue
		}
		title, body := AlertCopy(alert)
		link := "/community"
		for _, userID := range members {
			if eErr := s.notifier.Emit(userID, domain.NotificationFloodAlert, title, body, &link); eErr != nil {
				log.Printf("floodwatch: notify user=%s: %v", userID, eErr)
			}
		}
		// Stamped even when the fan-out partly failed. Retrying the whole
		// set would re-notify everyone who did receive it, and a duplicate
		// flood alarm costs more trust than a missed one costs here — the
		// banner on the community page does not depend on this.
		if err := s.repo.MarkNotified(alert.ID, alert.Severity, now); err != nil {
			log.Printf("floodwatch: mark notified id=%s: %v", alert.ID, err)
		}
	}
	return nil
}

// ActiveForCommunity returns the forecasts currently attached to a
// community.
//
// staleAfter drops rows the upstream has stopped reporting. A forecast
// that is no longer being issued is unknown, not clear, and leaving it on
// screen would let a stale warning look current — or, worse, let a stale
// quiet reading look like reassurance.
func (s *Service) ActiveForCommunity(communityID string, staleAfter time.Duration) ([]domain.CommunityFloodAlert, error) {
	alerts, err := s.repo.ActiveForCommunity(communityID, time.Now().UTC().Add(-staleAfter))
	if err != nil {
		return nil, err
	}
	if alerts == nil {
		return []domain.CommunityFloodAlert{}, nil
	}
	return alerts, nil
}

// AlertCopy renders the notification text.
//
// Attribution is in the body, not a footnote, and CivicOS offers no advice
// of its own — it names the source and points at the official channel.
// The wording never says a citizen is safe.
func AlertCopy(a domain.CommunityFloodAlert) (title, body string) {
	where := "a river near you"
	if a.River != nil && *a.River != "" {
		where = *a.River
	}

	switch Severity(a.Severity) {
	case SeverityExtreme:
		title = "Extreme flood risk forecast near you"
	case SeveritySevere:
		title = "Severe flood risk forecast near you"
	default:
		title = "Above-normal water levels forecast near you"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Google Flood Hub forecasts %s water levels on %s, about %.0fkm away",
		strings.ToLower(strings.ReplaceAll(a.Severity, "_", "-")), where, a.DistanceKm)
	if a.ForecastEndAt != nil {
		fmt.Fprintf(&b, ", through %s", a.ForecastEndAt.Format("2 Jan"))
	}
	b.WriteString(". ")
	if Trend(a.Trend) == TrendRise {
		b.WriteString("Levels are forecast to keep rising. ")
	}
	b.WriteString("This is a third-party forecast, not a CivicOS prediction and not an official warning — follow NEMA and NiMet for official guidance.")
	return title, b.String()
}
