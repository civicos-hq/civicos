package notifications

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// The notification event bus.
//
// organization-service writes notification rows directly into the shared
// table, but the SSE hub that pushes them to connected browsers lives in
// community-service. Without a bridge, anything this service emits — a
// consultation opening, an announcement, a campaign update — sits in the
// database until the user's next fetch.
//
// The split of responsibility is deliberate and is what keeps this safe:
//
//   - **This service writes the row.** Exactly once, as it does today.
//   - **The event carries the already-written row**, id and all.
//   - **community-service only pushes it to the hub.** It never writes.
//
// So NATS being down costs realtime delivery and nothing else: the
// notification is still in the table and still appears on the next fetch,
// which is precisely today's behaviour. That fallback is why this can use
// core NATS with no persistence — the hub already documents itself as lossy
// ("the REST list endpoint is the source of truth"), so a dropped realtime
// hint is a supported outcome, not data loss.

// SubjectNotificationCreated carries a notification row that has already
// been persisted. Subscribers must push it to their SSE hub and must NOT
// write it again.
const SubjectNotificationCreated = "civicos.notifications.created"

// Event is the wire format. Field names match the JSON the SSE stream
// already sends to browsers, so the bridge on the other side is a decode
// and a hand-off rather than a translation.
type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Read      bool      `json:"read"`
	LinkURL   *string   `json:"linkUrl,omitempty"`
	UserID    string    `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

// eventFor describes a committed row on the wire.
//
// The real row id is carried deliberately: community-service pushes this
// straight to the browser without re-reading, so a placeholder would hand
// the client something it can never mark as read.
func eventFor(row *Notification) Event {
	return Event{
		ID:        row.ID,
		Type:      string(row.Type),
		Title:     row.Title,
		Body:      row.Body,
		Read:      row.Read,
		LinkURL:   row.LinkURL,
		UserID:    row.UserID,
		CreatedAt: row.CreatedAt,
	}
}

// Bus publishes notification events. Optional: a nil Bus means realtime
// push is off and notifications behave exactly as they did before.
type Bus interface {
	PublishNotification(Event)
	Close()
}

// NATSBus is the core-NATS implementation.
type NATSBus struct{ nc *nats.Conn }

// ConnectNATS dials the event bus. Returns nil (not an error) when no URL is
// configured or the broker is unreachable: realtime push is an enhancement,
// and a service that refuses to start because a nice-to-have is down is
// worse than one that quietly falls back to fetch-on-load.
func ConnectNATS(url string) Bus {
	if url == "" {
		log.Printf("events: no NATS_URL — notifications will appear on next fetch, not in realtime")
		return nil
	}
	nc, err := nats.Connect(url,
		nats.Name("civicos-organization-service"),
		nats.MaxReconnects(-1), // keep trying forever; this is a background nicety
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Printf("events: NATS disconnected (%v) — falling back to fetch-on-load", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Printf("events: NATS reconnected — realtime notifications resumed")
		}),
	)
	if err != nil {
		log.Printf("events: could not reach NATS at %s (%v) — notifications will appear on next fetch", url, err)
		return nil
	}
	log.Printf("events: NATS connected at %s — realtime notifications enabled", url)
	return &NATSBus{nc: nc}
}

// PublishNotification fires and forgets. Errors are logged, never returned:
// the row is already committed, and a failed publish costs a realtime push,
// not a notification.
func (b *NATSBus) PublishNotification(e Event) {
	if b == nil || b.nc == nil {
		return
	}
	payload, err := json.Marshal(e)
	if err != nil {
		log.Printf("events: could not encode notification %s: %v", e.ID, err)
		return
	}
	if err := b.nc.Publish(SubjectNotificationCreated, payload); err != nil {
		log.Printf("events: publish failed for notification %s: %v", e.ID, err)
	}
}

func (b *NATSBus) Close() {
	if b != nil && b.nc != nil {
		// Drain rather than Close so anything already queued still goes out.
		_ = b.nc.Drain()
	}
}
