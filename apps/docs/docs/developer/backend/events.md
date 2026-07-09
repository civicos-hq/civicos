---
id: events
title: Events
sidebar_position: 2
---

# Events

CivicOS has **two** event surfaces today plus one provisioned for
Phase 2:

1. **In-process fan-out** inside a service (petition sign → milestone
   check → notification insert → SSE push).
2. **Server-Sent Events (SSE)** from community-service to the browser
   for realtime notifications.
3. **NATS** — running in the Docker compose but not driving anything
   yet. Reserved for cross-service events when we split notifications
   into its own service or add a search indexer.

## In-process fan-out

Notifications are the only "eventful" workflow at MVP. The pattern:

- A handler completes the primary write (sign the petition, add a
  comment, change an issue status).
- If side effects are needed (notify the creator, milestone, etc.),
  the handler calls `notifier.Emit(userID, type, title, body, linkURL)`.
- `notifications.Service.Emit` does two things:
  1. INSERT into the `notifications` table.
  2. Broadcast to the `notifications.Hub`, which pushes to any
     open SSE subscribers for that user.

This is deliberately in-process and synchronous. Not durable across
service instances yet — see the horizontal-scale note below.

## SSE — the realtime channel

**Endpoint:** `GET /api/v1/notifications/stream` (proxied through the
gateway with buffering disabled).

**Wire format:**

```
: connected

event: notification
data: {"id":"…","type":"ISSUE_UPDATE","title":"…","body":"…","createdAt":"…"}

: ping

event: notification
data: {…}
```

- `: comment` frames are keep-alives (25-second interval) so
  intermediaries don't kill the idle connection.
- `event: notification` frames carry the JSON payload.

**Frontend:**

```ts
const src = new EventSource(`${API}/api/v1/notifications/stream`, {
  withCredentials: true,
});
src.addEventListener('notification', (e) => {
  const n = JSON.parse(e.data);
  queryClient.setQueryData(['notifications'], (prev) => (prev ? [n, ...prev] : [n]));
  queryClient.invalidateQueries(['notifications', 'unread-count']);
});
```

**Auth:** SSE can't send custom headers via native `EventSource`. The
gateway accepts the JWT either in `Authorization: Bearer …` (fetch
polyfill) or as a `token` query param — see
`services/api-gateway/internal/middleware/auth.go` for the exact
handling in your version.

## The Hub, briefly

`notifications.Hub` is a `map[string]chan *Notification` guarded by a
`sync.RWMutex`. `Subscribe(userID)` returns a channel; `Emit(userID, n)`
sends non-blocking (drops if the buffered channel is full — the client
will pick it up on next poll). `Unsubscribe(userID, ch)` closes and
removes.

At one instance this is fine. At two instances, a citizen connected to
instance A won't receive a notification emitted by instance B. Two
options when we scale horizontally:

- **Redis pub/sub** — cheapest; the Hub becomes a Redis subscriber.
- **NATS** — leverages the already-provisioned broker.

## NATS — provisioned, not wired

`infrastructure/docker-compose.yml` runs a NATS server on `4222` (HTTP
monitor on `8222`). Nothing publishes or subscribes today. It's there
so that when we split notifications into its own service or add a
search indexer, we already have the broker.

If you're the person doing that split, the natural topic layout is:

```
civicos.notifications.emit    # published by any service, subscribed by notifications-service
civicos.audit.append          # published by any service, subscribed by audit-log projector
civicos.search.reindex        # published by community-service on write, subscribed by search indexer
```

Keep the payload backward-compatible; every subscriber should tolerate
extra fields.

## What isn't an event yet

- **Issue status changes** — handled in-process; no external event.
- **Audit-log inserts** — direct writes; no bus.
- **Rate-limit counters** — Redis, not events.
- **Search indexing** — none; queries hit the primary DB via `ILIKE`.

All of these could become events if scale requires it. None of them
need to be one yet.
