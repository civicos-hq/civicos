---
id: architecture
title: Architecture
sidebar_position: 1
---

# Architecture

The 10-minute mental model.

## High-level shape

```
┌────────────────┐   ┌────────────────┐   ┌─────────────────┐
│  Citizen web   │   │ Admin console  │   │    Docs site    │
│  React  :5173  │   │  React  :5174  │   │ Docusaurus :5175│
└───────┬────────┘   └───────┬────────┘   └─────────────────┘
        │                    │
        └─────────┬──────────┘
                  ▼
    ┌───────────────────────────────────────────┐
    │             API Gateway  :3000            │
    │   JWT · per-action rate limit · proxy     │
    │   Swagger UI /docs · /health aggregation  │
    └───┬───────────┬─────────────┬──────────┬──┘
        ▼           ▼             ▼          ▼
  ┌──────────┐ ┌──────────┐ ┌────────────┐ ┌──────────┐
  │ Identity │ │Community │ │Organization│ │ CivicAI  │
  │  :3001   │ │  :3002   │ │   :3003    │ │  :3004   │
  │          │ │ + uploads│ │            │ │  no DB   │
  │          │ │ from disk│ │            │ │          │
  └──────────┘ └──────────┘ └────────────┘ └──────────┘

  ┌──────────┐   ┌────────┐   ┌────────┐
  │ Postgres │   │  NATS  │   │ Redis  │
  │  :5433   │   │ :4222  │   │ :6379  │
  └──────────┘   └────────┘   └────────┘
```

Every request from the browser hits the **API Gateway** first. It applies
per-action rate limits and reverse-proxies to the appropriate service; the
browser never talks to a service directly. Most routes also carry JWT
validation, but **not all of them are authenticated** — community lists,
public campaign pages and the homepage activity ticker are reachable signed
out, by design.

Who talks to what, since the boxes above don't say:

- **Postgres** — identity, community and organization share one database.
  Ownership is by table, not by schema; a service reading another's table
  pins `TableName()` and treats it as read-only.
- **Redis** — the gateway (rate-limit budgets) and CivicAI (response cache).
- **NATS** — organization → community, one subject. See below.
- **CivicAI has no database of its own.** It fetches platform metrics from
  identity-service over HTTP with the caller's forwarded JWT, then caches the
  result. It is the one service that calls another over HTTP.

## The five Go services

| Service                  | Port | Responsibilities                                                                                                                                                                               |
| ------------------------ | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **api-gateway**          | 3000 | Reverse proxy, JWT validation, per-action rate limiting, `/health` aggregation, Swagger UI at `/docs`                                                                                          |
| **identity-service**     | 3001 | Auth (register / login / refresh with family rotation), users, applications, content flags (incl. campaign concerns), audit log, admin metrics, DB migrations                                  |
| **community-service**    | 3002 | Communities, issues, petitions, representatives and their announcements, comments, notifications (with SSE hub), search, discover feed, public activity ticker, image uploads                  |
| **organization-service** | 3003 | Organizations, membership, announcements, projects, issue assignments, progress updates, consultations, **Community Funding** (campaigns, donations, spend, reconciliation), funding analytics |
| **civicai-service**      | 3004 | Gemini-backed advisory endpoints — classification, summarization, drafting, community insights, campaign risk. No database of its own. Every response is a suggestion; nothing auto-acts       |

Notifications and search were spec'd as future standalone services in
the Engineering Playbook but live inside community-service for the MVP
so cross-entity event emission (e.g., a petition signature → notification)
stays in-process. Extract when scale demands it, not before.

## The frontend workspaces

- **`apps/web`** — the citizen-facing app on port 5173. Homepage, feeds,
  issue and petition detail, representative pages, consultations,
  community funding (browse, campaign pages, donate), org dashboard
  including campaign management and funding analytics, notifications,
  profile.
- **`apps/admin`** — the admin console on port 5174. Metrics, moderation
  queue, audit log, user administration, applications review, campaign
  review, reconciliation drift, campaign concerns, funding analytics.
- **`apps/docs`** — this Docusaurus site on port 5175. Not a React app in
  the same sense, but it lives in the same workspace and ships with the
  monorepo.

Both apps consume the API through the gateway, and both depend on the shared
types package (`@civicos/types`) — so a change to a request/response contract
flows through the compiler in both at once. The shared UI package
(`@civicos/ui`) is currently used by `apps/web` only; the admin console has
its own components.

## Data flow example — signing a petition

1. Citizen clicks **Sign** on a petition in `apps/web`.
2. Browser: `POST /api/v1/petitions/{id}/sign` with `Authorization: Bearer …`.
3. **Gateway (`:3000`)**:
   - `JWTAuth` middleware validates the token and sets `userID` in the
     context.
   - `Limit(Sign)` middleware checks Redis for the per-user sign budget.
   - Reverse-proxies to `community-service:3002/api/v1/petitions/{id}/sign`.
4. **Community service (`:3002`)**:
   - `RequireVerified` middleware confirms email verification.
   - `Handler.sign` fetches the petition.
   - `RequireMembershipInCommunity` middleware confirms membership.
   - `Service.Sign` inserts a `petition_signatures` row (unique index
     enforces one signature per user).
   - Handler checks if the new count crosses a **milestone** (25 %, 50 %,
     100 %); if so, `Notifier.Emit` fans out to the creator.
   - Notification insert into `notifications` table AND push through the
     in-process SSE hub to any open browser tab for that user.
5. Response returns up through the gateway to the browser.

**This particular flow** happens entirely inside one process — no NATS, no
cross-service HTTP. That's the point of colocating notifications and search
in community-service.

It is not the whole story. When **organization-service** writes a
notification — an announcement, a consultation, a campaign event — it writes
the row itself, then publishes `civicos.notifications.created` on NATS.
community-service subscribes and pushes the event to the SSE hub, because it
owns the hub and the row never passed through its `Emit`. The bridge
deliberately persists nothing: the publisher already committed, so writing
again would duplicate every cross-service notification. Losing an event costs
a realtime push and nothing more — the row is in the table and the REST list
stays the source of truth.

The subject string is declared separately in both services, because they are
separate Go modules. That string is the contract.

## Why Go, not the Playbook's NestJS?

The Engineering Playbook prescribes NestJS + TypeScript + Prisma. We
diverged for four reasons:

1. **One binary per service** — no runtime dependency on Node, no
   package.json to build in production images.
2. **Lower memory at idle** — Go services sit at ~20 MB RSS; a Node
   equivalent would sit at 80–150 MB. Matters on a small Render plan.
3. **Native concurrency for the SSE hub** — goroutines make the
   notification fan-out cheap. Node needs an event-loop reshape or a
   Worker.
4. **Deployable Docker image is under 50 MB** — the scratch/distroless
   final stage is easy.

The Playbook's _principles_ (DI, modular services, UUIDs, error codes,
UTC) still apply — only the language differs.

## Shared behaviour across services

- **UUID primary keys** everywhere — never sequential IDs.
- **Structured errors** — every response is
  `{success: bool, code, message, data?}`.
- **Audit logging** — `audit_logs` table lives in identity-service's
  schema but three services INSERT to it.
- **Ban / deletion enforcement** — JWT middleware in every service
  blocks writes from banned or deleted accounts.
- **UTC-only** timestamps, localized only at render.

## What's _not_ here yet

- **Broad event-driven work.** NATS carries exactly one subject today (see
  above); everything else is still an in-process call or a direct table
  write.
- **Full-text search** — currently a naive `ILIKE` sweep in
  community-service. Replace with pg_trgm or Meilisearch when the
  dataset justifies it.
- **A separate notification-service** — see the note above about
  colocation.

Next: [Repository structure](./repository-structure.md).
