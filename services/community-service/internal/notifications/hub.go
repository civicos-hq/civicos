package notifications

import (
	"sync"

	"github.com/civicos/community-service/internal/domain"
)

// Hub fans out freshly-created notifications to per-user SSE subscribers.
// A user can have multiple concurrent subscribers (e.g. several browser tabs);
// every subscriber for that user receives the same Notification value.
//
// Buffer is intentionally small: if a subscriber falls behind we drop rather
// than block the producer, since the REST list endpoint is the source of truth
// and the client refetches on reconnect.
type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[chan domain.Notification]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[string]map[chan domain.Notification]struct{}{}}
}

func (h *Hub) Subscribe(userID string) chan domain.Notification {
	ch := make(chan domain.Notification, 8)
	h.mu.Lock()
	if h.subs[userID] == nil {
		h.subs[userID] = map[chan domain.Notification]struct{}{}
	}
	h.subs[userID][ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(userID string, ch chan domain.Notification) {
	h.mu.Lock()
	if set, ok := h.subs[userID]; ok {
		delete(set, ch)
		if len(set) == 0 {
			delete(h.subs, userID)
		}
	}
	h.mu.Unlock()
	close(ch)
}

func (h *Hub) Publish(n domain.Notification) {
	h.mu.RLock()
	set := h.subs[n.UserID]
	channels := make([]chan domain.Notification, 0, len(set))
	for ch := range set {
		channels = append(channels, ch)
	}
	h.mu.RUnlock()
	for _, ch := range channels {
		select {
		case ch <- n:
		default:
		}
	}
}
