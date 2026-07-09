---
id: index
title: Developer Guide
slug: /developer
sidebar_position: 1
---

# Developer Guide

This is the guide for engineers building on, deploying, or contributing
to CivicOS. If you're a citizen or an organization admin looking to
_use_ CivicOS, head to the **[User Guide](/)** instead.

## Where to start

- **New to the codebase?** Start with [Architecture](./overview/architecture.md) —
  the 10-minute mental model.
- **Setting up your machine?** Follow [Running locally](./development/running-locally.md).
- **Working in a specific service?** See the [Services](./services/api-gateway.md)
  section — one page per service, focused on responsibilities and
  package layout.
- **Deploying?** See [Deployment](./operations/deployment.md) — pointer to the
  Render blueprint in `docs/deploy.md`.

## What CivicOS is, technically

CivicOS is a **microservice-oriented civic engagement platform** built in
**Go** on the backend and **React + TypeScript** on the frontend. Four
Go services (identity, community, organization, api-gateway) sit behind
Postgres 16 and Redis 7. Two React apps (citizen web, admin console)
consume the gateway.

- One binary per service. Single Docker image per service.
- pnpm workspaces + Turborepo hold the frontend and shared TS packages.
- Go modules per service — no shared Go module, on purpose.
- Everything talks HTTP + JSON. NATS is provisioned for future
  event-driven work.

## Design principles

1. **Dependency injection everywhere** — `NewRepository(db)` →
   `NewService(repo)` → `NewHandler(svc)` in `main.go`. No hidden
   globals.
2. **UUIDs, not sequential IDs** — `uuid.New().String()` at construction
   time.
3. **Validate at boundaries** — `binding:"required"` on input structs.
4. **Error codes, not raw messages** — every error returns
   `{code, message}` so the frontend can localize.
5. **Timestamps in UTC** — localize only in the UI.
6. **Files under 300 lines** — split at 500.
7. **Never serialize secrets** — `json:"-"` on password hashes, reset
   tokens, refresh tokens.

Details: [Contributing](./development/contributing.md).
