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

- **Communities** — paginated list with name search (`?q=`), filter by
  state / LGA, resolve a specific set by `?ids=`, create (admin roles
  only). Member counts are computed at query time from
  `user_community_memberships`.
- **Issues** — CRUD, upvote toggle, status changes, comment threads.
- **Petitions** — CRUD, sign, comment threads, milestone notifications.
- **Representatives** — CRUD, follow / unfollow, comment threads,
  official-response notifications to followers, and profile **claiming**
  (`POST /representatives/:id/claim`, PLATFORM_ADMIN only).
- **Notifications** — persisted list plus **realtime SSE** push via an
  in-process hub.
- **Search** — global search across communities, issues, petitions,
  representatives, organizations, consultations, announcements, projects,
  funding campaigns and representative announcements.
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

### Representative profile ownership

`representatives.user_id` is the account that may publish as that
representative. It is set in exactly three places:

1. Approving a representative application (identity-service) — the
   applicant owns the profile their own application created.
2. The `00007` backfill, for profiles that predate the column.
3. `POST /representatives/:id/claim` — PLATFORM_ADMIN only.

The third exists because the first two leave **admin-seeded profiles
permanently unclaimable**: publishing returned `REPRESENTATIVE_UNCLAIMED`
telling the user to ask a platform admin, and no endpoint let that admin
help. Provisioning a constituency office in organization-service keys off
the same claim, so the gap blocked campaigns and consultations too.

Claiming never displaces an existing claim — the repository guards on
`user_id IS NULL`. Reassigning a profile would hand one official's
constituents, and their donors, to a different account; that has to be a
deliberate unlink first. Claiming also does not grant the
`REPRESENTATIVE` platform role: that comes from the approval flow, which
records a reviewer.

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

Passing `tier=` filters to one tier. Passing `kind=` filters the
discriminator to one of `issue`, `petition`, `announcement`, `project`,
`consultation` or `campaign`; an unrecognised value is treated as no
filter, so a client typo shows the whole feed rather than an empty page
that reads as "nothing is happening near you". Under the hood, it scans
up to 1 000 items from the DB and filters in-memory — fine at MVP scale,
replace with `community_id IN (…)` when the dataset grows.

Issues and petitions tier by their own community. Announcements tier by
the publishing org. Projects, consultations and campaigns prefer their
`communityId` when set; campaigns then fall back to their **own**
`state`/`lga` before the org's, because a campaign is often raised for a
specific ward rather than wherever the organization is registered.

### Public activity (unauthenticated)

`GET /discover/public-activity` backs the activity panel on the marketing
homepage. It is the **only unauthenticated aggregate on the platform** —
anyone on the internet can call it without an account — so it returns the
smallest thing that can honestly be shown: kind, title, status, state/LGA,
timestamp.

Deliberately absent: **author names and user ids** (a ticker does not need to
name the citizen who reported a broken transformer, and a public homepage is
a different audience from a signed-in page), **bodies**, and anything not
already public.

It is a separate endpoint rather than a mode of the feed because the feed is
authenticated, personalised, and hydrates whole entities plus their
organizations — none of which is wanted, and all of which would be a larger
surface to get wrong.

Implementation is one query per kind then a merge, not a `UNION`: the six
kinds live in tables owned by two services with different shapes, and a
`UNION` would force a lowest-common-denominator projection that breaks the
moment one of them gains a column. `internal/discover/public_activity_test.go`
seeds every kind in every status against a real Postgres and asserts that no
draft, pending, rejected or archived record is reachable.

An empty list is a valid answer. The homepage renders a real empty state
rather than filling the space, and does not claim "Live" over an empty panel.

### What discovery will not show

Announcements, consultations, projects and campaigns are owned by
`organization-service` and read here from the shared database with
`TableName()` pinned — the same arrangement documented above. For
campaigns the read is an explicit column allow-list, not the whole
model: the review trail (`approvalStatus`, `reviewNote`, `reviewedById`)
is a private conversation between the platform and the organization.

Both surfaces show exactly the statuses that have a public page, and
that list is owned by `organization-service` — its `publicStatuses`
allow-list in `internal/campaigns/repository.go`:

| Status           | Public page | In search | In discover feed |
| ---------------- | ----------- | --------- | ---------------- |
| `DRAFT`          | 404         | no        | no               |
| `PENDING_REVIEW` | 404         | no        | no               |
| `REJECTED`       | 404         | no        | no               |
| `PAUSED`         | 404         | no        | no               |
| `PUBLISHED`      | yes         | yes       | yes              |
| `FUNDED`         | yes         | yes       | yes              |
| `COMPLETED`      | yes         | yes       | yes              |
| `REPORTED`       | yes         | yes       | yes              |

A rejected campaign is hidden because the title alone would reveal that
an organization asked for money and was refused. `PAUSED` is hidden for
a plainer reason: `GetPublicBySlug` returns 404 for it, so surfacing one
in search would be a result that goes nowhere.

**If that allow-list changes, change it in `organization-service` first.**
The two queries here mirror it; they are not independent policy. The
allow-list is deliberately an allow-list, so a status added later
defaults to hidden — the safe direction for a surface that asks people
for money.

## Environment

- `DATABASE_URL`
- `JWT_SECRET` (must match gateway + identity)
- `PORT` / `COMMUNITY_SERVICE_PORT` (default `3002`)
