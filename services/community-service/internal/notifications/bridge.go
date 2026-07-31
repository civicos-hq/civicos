package notifications

import (
	"encoding/json"
	"log"
	"time"

	"github.com/civicos/community-service/internal/domain"
	"github.com/nats-io/nats.go"
)

// The realtime bridge for notifications written by other services.
//
// community-service owns the notifications table and the SSE hub, but it is
// not the only writer: organization-service writes rows directly for
// announcements, consultations and campaign events. Those rows never passed
// through Service.Emit, so they never reached the hub — they sat in the
// database until the user's next fetch.
//
// This subscribes to those writes and pushes them to connected browsers.
//
// It deliberately does NOT persist anything. The publishing service already
// committed the row before announcing it; writing again here would give
// every cross-service notification a duplicate. The one job of this bridge
// is delivery.
//
// Losing an event costs a realtime push and nothing more — the row is in the
// table and the REST list remains the source of truth, which is the same
// contract the hub already documents for slow subscribers.

// SubjectNotificationCreated must match the publisher's subject in
// organization-service/internal/notifications/bus.go. Each service declares
// it separately because they are separate Go modules; the string is the
// contract.
const SubjectNotificationCreated = "civicos.notifications.created"

// event mirrors the publisher's wire format.
type event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Read      bool      `json:"read"`
	LinkURL   *string   `json:"linkUrl,omitempty"`
	UserID    string    `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

// toNotification converts a wire event into the shape the hub fans out.
// Deliberately a plain mapping with no lookup: the row already exists, and
// re-reading it here would turn a realtime nicety into a database load
// proportional to notification volume.
func toNotification(e event) domain.Notification {
	return domain.Notification{
		ID:        e.ID,
		Type:      domain.NotificationType(e.Type),
		Title:     e.Title,
		Body:      e.Body,
		Read:      e.Read,
		LinkURL:   e.LinkURL,
		UserID:    e.UserID,
		CreatedAt: e.CreatedAt,
	}
}

// Bridge holds the subscription so it can be shut down cleanly.
type Bridge struct{ nc *nats.Conn }

// StartBridge subscribes to notifications written by other services and
// pushes them to the hub.
//
// Returns nil when no URL is configured or the broker cannot be reached.
// That is not a failure worth blocking startup for: without it, cross-service
// notifications behave exactly as they do today.
func StartBridge(url string, hub *Hub) *Bridge {
	if url == "" || hub == nil {
		log.Printf("events: no NATS_URL — cross-service notifications will appear on next fetch, not in realtime")
		return nil
	}
	nc, err := nats.Connect(url,
		nats.Name("civicos-community-service"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Printf("events: NATS disconnected (%v) — realtime bridge paused", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Printf("events: NATS reconnected — realtime bridge resumed")
		}),
	)
	if err != nil {
		log.Printf("events: could not reach NATS at %s (%v) — cross-service notifications will appear on next fetch", url, err)
		return nil
	}

	if _, err := nc.Subscribe(SubjectNotificationCreated, func(m *nats.Msg) {
		var e event
		if err := json.Unmarshal(m.Data, &e); err != nil {
			log.Printf("events: undecodable notification event: %v", err)
			return
		}
		if e.UserID == "" {
			return
		}
		// Push only. The publisher already wrote this row.
		hub.Publish(toNotification(e))
	}); err != nil {
		log.Printf("events: subscribe failed (%v) — realtime bridge disabled", err)
		nc.Close()
		return nil
	}

	log.Printf("events: NATS bridge active at %s — notifications from other services push in realtime", url)
	return &Bridge{nc: nc}
}

func (b *Bridge) Close() {
	if b != nil && b.nc != nil {
		_ = b.nc.Drain()
	}
}
