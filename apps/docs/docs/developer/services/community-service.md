---
id: community-service
title: Community Service
sidebar_position: 3
---

# Community Service

Port `:3002`. Everything **community-scoped** — communities themselves,
issues, petitions, representatives, comments, and the notification hub.

:::note "There's no separate Issue Service"

Issues live here, alongside petitions and representatives. The MVP
colocates them so cross-entity flows (upvote → notification, petition
signature → milestone → notification) stay in-process. Extract if scale
demands it.

:::

## Responsibilities

- **Communities** — list, filter by state / LGA, create (admin roles
  only).
- **Issues** — CRUD, upvote toggle, status changes, comment threads.
- **Petitions** — CRUD, sign, comment threads, milestone notifications.
- **Representatives** — CRUD, follow / unfollow, comment threads,
  official-response notifications to followers.
- **Notifications** — persisted list plus **realtime SSE** push via an
  in-process hub.
- **Search** — global search across issues, petitions, representatives.
- **Discover feed** — personalized feed tiered by geographic proximity
  (`COMMUNITY` → `LGA` → `STATE` → `COUNTRY`).
- **Uploads** — image upload endpoint (5 MB max, JPG/PNG/GIF/WEBP) plus
  public serve.

## Package layout

```
services/community-service/
├── cmd/server/main.go
├── internal/
│   ├── audit/                     # Writes to audit_logs (schema owned by identity)
│   ├── communities/               # Communities CRUD
│   ├── discover/                  # Discover feed with tier logic
│   ├── domain/models.go
│   ├── issues/                    # Issues + comments + upvotes
│   ├── middleware/                # JWTAuth, RequireVerified, RequirePrimaryCommunityMatch, RequireMembershipInCommunity
│   ├── notifications/             # Hub + service + handler (including SSE stream)
│   ├── petitions/                 # Petitions + signatures + comments + milestones
│   ├── representatives/           # Reps + follows + comments
│   ├── search/                    # Global ILIKE search
│   └── uploads/                   # Multipart upload + static serve
├── migrations/
└── pkg/…
```

## Key concepts

### Content flags → hidden placeholders

Comments, issues, and petitions can be flagged. When a moderator
resolves a flag as `HIDDEN`, the read path in each repository replaces
the content and author name with placeholders — `[Removed by moderator]`
— **without deleting the row**. Conversation flow survives; the audit
trail is preserved; the offending payload is gone.

The `isHidden` field on the DTOs is **computed at query time** — never
stored. If the flag is dismissed later, the row is unaffected.

### Notifications + SSE

- Every notification is persisted to the `notifications` table.
- In parallel, the `notifications.Hub` broadcasts to any connected SSE
  subscribers for that user.
- The SSE handler holds a goroutine per open browser tab; a 25-second
  keep-alive comment prevents intermediaries from closing idle
  connections.
- The frontend uses `EventSource` to subscribe and re-hydrates unread
  state on reconnect.

The hub is a `map[userID]chan *Notification` behind a `sync.RWMutex`.
It's the simplest thing that works. Replace with Redis pub/sub when the
service is horizontally scaled.

### Community scoping middleware

Two middlewares gate community-scoped actions:

- **`RequirePrimaryCommunityMatch`** — used on creation endpoints
  (issue, petition, representative). Confirms the caller's
  `primaryCommunityId` matches the community they're creating in.
- **`RequireMembershipInCommunity`** — used on interaction endpoints
  (comment, upvote, sign). Confirms the caller has a membership row
  for the target's community.

Both live in `internal/middleware/`.

### Petition milestones

`petitions.Handler.sign` fires an extra notification when the new count
crosses **25 %**, **50 %**, or **100 %** of the goal. The crossing is
determined by `(newCount - 1 < threshold) && (newCount >= threshold)` —
each milestone is emitted exactly once per petition per threshold.

### Discover feed tiers

`discover.Service.Feed` labels each item with a tier based on the
caller's active community:

- `COMMUNITY` — same community as caller
- `LGA` — same LGA, different community
- `STATE` — same state, different LGA
- `COUNTRY` — same country, different state

Passing `tier=` filters to one tier. Passing `kind=issue|petition`
filters the discriminator. Under the hood, it scans up to 1 000 items
from the DB and filters in-memory — fine at MVP scale, replace with
`community_id IN (…)` when the dataset grows.

## Environment

- `DATABASE_URL`
- `JWT_SECRET` (must match gateway + identity)
- `PORT` / `COMMUNITY_SERVICE_PORT` (default `3002`)
