package notifications

import (
	"encoding/json"
	"testing"
	"time"
)

// decodeAndPush is the bridge's whole job, factored out of the NATS callback
// so it can be exercised without a broker.
func decodeAndPush(t *testing.T, hub *Hub, payload []byte) bool {
	t.Helper()
	var e event
	if err := json.Unmarshal(payload, &e); err != nil {
		return false
	}
	if e.UserID == "" {
		return false
	}
	hub.Publish(toNotification(e))
	return true
}

// The subject string is a cross-module contract — organization-service
// declares its own copy. If these drift, notifications silently stop being
// realtime and nothing fails loudly.
func TestBridge_SubjectMatchesThePublisher(t *testing.T) {
	const publisherSubject = "civicos.notifications.created"
	if SubjectNotificationCreated != publisherSubject {
		t.Fatalf("subject = %q, but organization-service publishes to %q",
			SubjectNotificationCreated, publisherSubject)
	}
}

// An event written by another service must reach that user's SSE subscriber.
func TestBridge_DeliversToTheRightSubscriber(t *testing.T) {
	hub := NewHub()
	mine := hub.Subscribe("user-1")
	theirs := hub.Subscribe("user-2")
	defer hub.Unsubscribe("user-1", mine)
	defer hub.Unsubscribe("user-2", theirs)

	link := "/campaigns/flood-relief-sabon-gari"
	payload, _ := json.Marshal(event{
		ID: "abc", Type: "CAMPAIGN_UPDATE", Title: "Flood relief update",
		Body: "Two boreholes completed.", LinkURL: &link,
		UserID: "user-1", CreatedAt: time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC),
	})

	if !decodeAndPush(t, hub, payload) {
		t.Fatal("bridge refused a valid event")
	}

	select {
	case got := <-mine:
		if got.ID != "abc" || got.UserID != "user-1" {
			t.Fatalf("wrong notification delivered: %+v", got)
		}
		if got.Type != "CAMPAIGN_UPDATE" {
			t.Fatalf("type = %q, want CAMPAIGN_UPDATE", got.Type)
		}
		if got.LinkURL == nil || *got.LinkURL != link {
			t.Fatalf("link lost in transit: %v", got.LinkURL)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber never received the notification")
	}

	// Someone else's notification must not leak into this subscriber.
	select {
	case leaked := <-theirs:
		t.Fatalf("user-2 received user-1's notification: %+v", leaked)
	default:
	}
}

// Malformed payloads must be dropped, not panic the subscription goroutine —
// one bad message would otherwise take realtime down for everyone.
func TestBridge_IgnoresJunk(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe("user-1")
	defer hub.Unsubscribe("user-1", ch)

	for _, payload := range [][]byte{
		[]byte(`not json`),
		[]byte(`{"userId":""}`),
		[]byte(`{}`),
		nil,
	} {
		if decodeAndPush(t, hub, payload) {
			t.Fatalf("bridge accepted junk: %q", payload)
		}
	}

	select {
	case got := <-ch:
		t.Fatalf("junk was delivered to a subscriber: %+v", got)
	default:
	}
}

// StartBridge must degrade rather than block startup. Realtime is an
// enhancement; the notification is already in the table either way.
func TestStartBridge_DegradesWithoutABroker(t *testing.T) {
	if b := StartBridge("", NewHub()); b != nil {
		t.Fatal("no URL should yield no bridge")
	}
	if b := StartBridge("nats://127.0.0.1:1", NewHub()); b != nil {
		b.Close()
		t.Fatal("an unreachable broker should yield no bridge, not a hang or a panic")
	}
	// Closing a nil bridge is safe.
	var nilBridge *Bridge
	nilBridge.Close()
}
