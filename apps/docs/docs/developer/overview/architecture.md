---
id: architecture
title: Architecture
sidebar_position: 1
---

# Architecture

The 10-minute mental model.

## High-level shape

```
┌────────────────┐    ┌────────────────┐
│  Citizen web   │    │ Admin console  │
│  React :5173   │    │  React :5174   │
└───────┬────────┘    └────────┬───────┘
        │                      │
        └──────────┬───────────┘
                   ▼
         ┌──────────────────┐
         │   API Gateway    │  :3000
         │  (Go / Gin)      │  JWT + rate limit + reverse proxy
         └───┬──────────┬───┘
             │          │
   ┌─────────┼──────────┼──────────┐
   ▼         ▼          ▼          ▼
┌───────┐ ┌──────────┐ ┌──────────┐ ┌────────┐
│Identity│ │Community│ │Organization│ │Uploads│ (served by community)
│ :3001 │ │  :3002  │ │   :3003   │ │ (disk)│
└───┬───┘ └────┬─────┘ └────┬──────┘ └────────┘
    │          │            │
    └──────────┼────────────┘
               ▼
         ┌──────────┐   ┌────────┐   ┌────────┐
         │ Postgres │   │ Redis  │   │  NATS  │
         │   :5433  │   │ :6379  │   │ :4222  │
         └──────────┘   └────────┘   └────────┘
```

Every request from the browser hits the **API Gateway** first. The
gateway validates the JWT, applies per-action rate limits, and reverse-
proxies to the appropriate service. The browser never talks to
`identity-service`, `community-service`, or `organization-service`
directly.

## The four Go services

| Service                  | Port | Responsibilities                                                                                                              |
| ------------------------ | ---- | ----------------------------------------------------------------------------------------------------------------------------- |
| **api-gateway**          | 3000 | Reverse proxy, JWT validation, per-action rate limiting, `/health` aggregation, Swagger UI at `/docs`                         |
| **identity-service**     | 3001 | Auth (register / login / refresh with family rotation), users, applications, content flags, audit log, admin metrics          |
| **community-service**    | 3002 | Communities, issues, petitions, representatives, comments, notifications (with SSE hub), search, discover feed, image uploads |
| **organization-service** | 3003 | Organizations, membership, announcements, projects, issue assignments, progress updates                                       |

Notifications and search were spec'd as future standalone services in
the Engineering Playbook but live inside community-service for the MVP
so cross-entity event emission (e.g., a petition signature → notification)
stays in-process. Extract when scale demands it, not before.

## Two React apps

- **`apps/web`** — the citizen-facing app on port 5173. Homepage, feeds,
  issue and petition detail, representative pages, notifications, profile.
- **`apps/admin`** — the admin console on port 5174. Metrics, moderation
  queue, audit log, user administration, applications review.

Both apps consume the API through the gateway. There's a shared UI
package (`@civicos/ui`) and a shared types package (`@civicos/types`)
so a change to a request/response contract flows through the compiler
in both apps at once.

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

Everything above happens in the same process boundary once — no NATS,
no cross-service HTTP. That's the point of colocating notifications and
search in community-service.

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

- **NATS** is provisioned but not driving anything today. Reserved for
  Phase 2 event-driven work.
- **Full-text search** — currently a naive `ILIKE` sweep in
  community-service. Replace with pg_trgm or Meilisearch when the
  dataset justifies it.
- **A separate notification-service** — see the note above about
  colocation.

Next: [Repository structure](./repository-structure.md).
