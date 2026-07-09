# CivicOS — Claude Code Context

## What is CivicOS?

CivicOS is an open civic infrastructure platform — the "operating system for democratic participation." It bridges the gap between citizens and their governments between elections by enabling continuous civic engagement.

Core users: **citizens**, **elected representatives**, **government admins**, **NGOs**, **moderators**.

## Source documents

All product, architecture, and engineering decisions are driven by five documents in `docs/product/`:

| Document                                  | Purpose                                                       |
| ----------------------------------------- | ------------------------------------------------------------- |
| `CivicOS Blueprint.pdf`                   | Vision, mission, product philosophy, core principles          |
| `CivicOS Product Roadmap.pdf`             | What gets built and in what order (5 phases)                  |
| `CivicOS Technical Architecture v1.0.pdf` | How the system is designed — services, data, security, AI     |
| `CivicOS Experience Architecture.pdf`     | UX, user journeys, screen specs, design system                |
| `CivicOS Engineering Playbook.pdf`        | How software is written — standards, conventions, DI, testing |

**Before implementing any feature, consult the relevant document first.**

## Repository structure

```
civicos/
├── apps/
│   ├── web/                  # Citizen React app (Vite + TS, port 5173)
│   ├── admin/                # Admin console (Vite + TS, port 5174)
│   └── docs/                 # Docusaurus site — user + developer guides (port 5175)
├── services/
│   ├── api-gateway/          # Go — reverse proxy + JWT + Swagger UI at /docs (port 3000)
│   ├── identity-service/     # Go — auth, users, applications, moderation (port 3001)
│   ├── community-service/    # Go — communities, issues, petitions, reps (port 3002)
│   └── organization-service/ # Go — orgs, announcements, projects, assignments (port 3003)
├── packages/
│   ├── types/                # Shared TypeScript interfaces & enums (@civicos/types)
│   ├── config/               # Env validation via zod (@civicos/config)
│   └── ui/                   # Shared React components (@civicos/ui)
├── infrastructure/
│   └── docker-compose.yml    # Postgres 16 (:5433) + Redis 7 + NATS + Mailpit
├── docs/
│   ├── product/              # The five source documents (PDFs)
│   └── api/                  # Canonical OpenAPI 3.0 specs (openapi-*.yaml)
└── CLAUDE.md                 # This file
```

Each Go service follows the layout:

```
service/
├── cmd/server/main.go        # Entry point — wires DI, starts Gin
├── internal/
│   ├── domain/models.go      # GORM models + enums
│   ├── middleware/auth.go    # JWT middleware
│   └── <feature>/            # repository.go, service.go, handler.go
└── pkg/
    ├── config/config.go      # Env loading + validation
    ├── database/postgres.go  # GORM connection + AutoMigrate
    └── response/response.go  # Success/Error helpers
```

## Tech stack

- **Frontend**: React 18, Vite, TypeScript, Tailwind CSS, TanStack Query, React Router v6
- **Backend**: Go 1.22 — Gin (HTTP), GORM (ORM), golang-jwt/jwt/v5 (auth)
- **Database**: PostgreSQL 16 (GORM AutoMigrate)
- **Cache**: Redis 7
- **Messaging**: NATS
- **Hot reload**: Air (`air` CLI, `.air.toml` per service)
- **Monorepo**: pnpm workspaces + Turborepo (frontend/packages only)
- **Package manager**: pnpm (frontend), Go modules (backend)

### Stack note — Go vs the playbook

The Engineering Playbook PDF prescribes **NestJS + TypeScript + Prisma** for backend. We diverged and built in **Go + Gin + GORM** for: smaller deploy artifacts, lower memory at idle, native concurrency for the SSE notification hub, and one binary per service. The playbook's principles (DI, modular services, UUIDs, error codes, UTC) still apply — only the language differs. If a future contributor reads the playbook expecting Node, this file is the source of truth.

### Service boundaries (MVP)

- `identity-service` — users, JWT, `/me`, applications, content flags, audit log, admin metrics
- `community-service` — **everything community-scoped**: communities, issues, petitions, representatives, comments, **notifications** (SSE), search, discover, image uploads
- `organization-service` — orgs, membership, announcements, projects, issue assignments, progress updates
- `api-gateway` — reverse proxy + JWT + per-action rate limiting + Swagger UI at `/docs`

Notifications and search were spec'd by the playbook as future standalone services. For the MVP they live inside `community-service` so cross-entity event emission (e.g., a petition signature → notification) stays in-process and doesn't need NATS. Extract when scale demands it, not before.

There is intentionally **no separate issue-service** — issues live in `community-service` alongside petitions and representatives for the same in-process reason.

## Documentation surfaces

Three places to look, each with a distinct audience:

- **Swagger UI** — `http://localhost:3000/docs`. Interactive API reference served by the gateway. Backed by hand-written specs at `docs/api/openapi-*.yaml`; embedded copies at `services/api-gateway/internal/docs/openapi/` via `go:embed`. Re-sync with `cp docs/api/openapi-*.yaml services/api-gateway/internal/docs/openapi/`.
- **Docusaurus site** — `http://localhost:5175` (workspace at `apps/docs/`). Two tabs in the nav: **User Guide** (citizens, orgs, reps) and **Developer Guide** (architecture, running locally, per-service pages, database, events, contributing, deployment).
- **`docs/product/`** — the five source PDFs. Product / architecture / UX / playbook source of truth for design decisions.

When adding a feature: update the OpenAPI spec + the Docusaurus page for the affected role. Don't leave the API docs and the user docs out of sync.

## Engineering rules

1. **Dependency injection always** — `NewRepository(db)` → `NewService(repo)` → `NewHandler(svc)` in `main.go`. Never instantiate inside a struct.
2. **No stringly-typed IDs** — all entity IDs are UUIDs (`uuid.New().String()`). Never use sequential integers.
3. **Validate at boundaries** — use `binding:"required"` tags on input structs. Never trust raw request data.
4. **Single responsibility** — each file does one thing. `auth/service.go` does not touch HTTP.
5. **Conventional commits** — `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `ci:` — Husky enforces this.
6. **Files under 300 lines** — if a file grows past 500, consider splitting it.
7. **Store timestamps in UTC** — convert to local time only in the UI.
8. **Error codes, not raw messages** — return `{ code: "ISSUE_NOT_FOUND", message: "..." }` from all services.
9. **Log for operators** — include service name, route, status. Never log passwords or tokens.
10. **PasswordHash never serialised** — use `json:"-"` on any sensitive field.

## Domain model (key entities)

- **User** — citizens, reps, admins. Role-based (`UserRole` enum).
- **Community** — geographic unit (state + LGA). Everything belongs to a community.
- **Issue** — community problem reported by a citizen. Has status lifecycle (OPEN → RESOLVED).
- **Petition** — citizen-created, has a signature goal and deadline.
- **Representative** — a User with `role: REPRESENTATIVE`, linked to a constituency.
- **Notification** — in-app alerts for issue updates, responses, petition milestones.

All types are in `packages/types/src/index.ts`.

## MVP build order

1. ✅ Monorepo scaffold
2. ✅ Identity Service — register, login, refresh, /me, PATCH /me (Go)
3. ✅ Community Service — communities, issues, petitions, representatives, comments, notifications, search, discover (Go)
4. ✅ API Gateway — reverse proxy, JWT validation (Go)
5. ✅ Frontend auth flow — register, login, dashboard shell, profile
6. ✅ Issue reporting UI + API (incl. filters, status timeline, admin status change)
7. ✅ Representative pages (incl. public comments, contact links, admin edit)
8. ✅ Notifications (incl. SSE realtime push)
9. ✅ Petitions (incl. sign + milestone fan-out)

Beyond the original list, also shipped: search, discover feed (tier + kind + pagination), list-page filters, image lightbox, share button, public homepage.

## Running locally

```bash
# 1. Start infrastructure
docker compose -f infrastructure/docker-compose.yml up -d

# 2. Install workspace deps (frontends + Docusaurus site)
pnpm install

# 3. Copy env and fill in JWT_SECRET (min 32 chars)
cp .env.example .env

# 4. Source .env in the shell — air runs from the service directory,
#    so godotenv.Load() won't find the repo-root .env otherwise.
set -a && source .env && set +a

# 5. Start Go services (each in a separate terminal, requires `air` installed)
cd services/identity-service     && air
cd services/community-service    && air
cd services/organization-service && air
cd services/api-gateway          && air

# 6. Start frontends + docs (runs web :5173, admin :5174, docs :5175)
pnpm dev

# Install air for hot reload (one-time):
go install github.com/air-verse/air@latest
```

## CI gates

PRs must pass:

- **`pnpm format:check`** — Prettier over `**/*.{ts,tsx,json,md}`. Fix with `pnpm exec prettier --write <file>`.
- **`gofmt -l services/`** — must return empty. Fix with `gofmt -w services/`.

Husky runs Prettier on staged files at commit time. Merges from other branches can still introduce drift — run both locally before pushing.
