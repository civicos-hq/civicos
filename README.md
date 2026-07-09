# CivicOS

> **An operating system for civic participation.**
> CivicOS is shared digital infrastructure that lets governments, universities, NGOs and communities build trusted civic experiences — organized around the places people actually live.

CivicOS is an open civic engagement platform that bridges the gap between citizens and their governments **between elections**. Citizens report neighbourhood issues, sign petitions, follow their representatives, and see what actually gets done — every action recorded on a public register.

Built for Nigeria first (36 states + FCT, 774 LGAs), designed to work in any democracy.

---

## Features

### Citizen web (`apps/web`, port `5173`)

**Public**

- Animated homepage — live "docket" of civic activity, mission manifesto, procedure explainer, FAQ
- Multilingual — English, Nigerian Pidgin, Yoruba, Igbo, Hausa
- Terms of Service + Privacy notice, linked from footer and auth pages

**Accounts**

- Register with email + password (bcrypt cost 12)
- Email verification via one-time link
- Login with JWT access + refresh-token family rotation
- Forgot / reset password
- Full account deletion — anonymizes PII, revokes all sessions, immediate ban of legacy tokens

**Onboarding**

- Pick your community — Nigerian state → LGA cascade, all 774 LGAs shipped in the bundle

**Civic action**

- **Raise an issue** — title, description, category, location, up to 5 photos (5 MB each)
- **Sign a petition** — signature counter, milestones (10 / 100 / 500 / 1,000), deadline
- **Follow representatives** — pinned to your community, public comment threads
- **Comment** on issues, petitions, and representative pages (rate-limited, verified accounts only)
- **Upvote issues** — a form of endorsement that pushes issues up the community feed
- **Flag content** — 5 reasons (spam, abuse, misinfo, hate, other), reviewed by moderators

**Feed**

- Community-scoped filters — status, category, date, upvotes
- Discover feed — cross-community browsing sorted by tier + kind
- Global search — issues, petitions, representatives, organizations
- Notifications — in-app + Server-Sent-Events realtime push

**Profile**

- Avatar, name, email verification status
- Danger zone — soft-delete account with reason, confirmation, and irreversibility notice

### Admin console (`apps/admin`, port `5174`)

Access requires role `PLATFORM_ADMIN`, `GOVERNMENT_ADMIN`, or `NGO`.

**Overview**

- Real-time health of all 4 backend services
- Platform metrics (30 s refresh) — citizens, communities, issues, petitions, representatives, organizations, verified rate, response rate
- Moderation dashboard — pending flags, hidden all-time, banned users, audit log volume
- Issues by status breakdown with percentages

**People**

- **Users** — filter by role, click through to per-user detail with audit trail + reports filed
- **Communities** — list, create, drill down to per-community stats (citizens, issues by status, petitions, representatives)
- **Representatives** — list, create (10 fields including community linkage), filter by community
- **Organizations** — list, create (with `NATIONAL` / `STATE` / `LGA` / `COMMUNITY` jurisdiction), verify / revoke verified badge, drill down to activity + members

**Trust**

- **Moderation queue** — review flags, hide or dismiss with resolution note
- **Direct hide** — admin utility to hide content by UUID + reason (creates a HIDDEN flag on your behalf, audit-logged)
- **Audit log** — every admin action across every service, searchable and filterable by action type

### Backend

- **api-gateway** (`:3000`) — reverse proxy, JWT validation, tiered rate limiting (Strict / Standard / Lenient) via Redis
- **identity-service** (`:3001`) — auth, users, sessions, refresh-token family rotation with replay detection, email verification, password reset, admin metrics endpoint
- **community-service** (`:3002`) — communities, issues (+ image upload), petitions, representatives, comments, notifications (SSE hub), content flags with placeholder-based hiding, discover feed, search
- **organization-service** (`:3003`) — organizations, membership, announcements, projects, issue assignments, progress updates, verified-badge control

Shared behaviour across services:

- **UUID primary keys** everywhere (no sequential IDs)
- **Structured errors** — every response carries `{success, code, message, data?}`
- **Audit logging** — shared `audit_logs` table, three services INSERT to it
- **Ban / deletion enforcement** — JWT middleware in every service blocks writes from banned or deleted accounts within seconds
- **UTC-only** timestamps, localized only at render

---

## Tech stack

| Layer           | Choice                                                                                |
| --------------- | ------------------------------------------------------------------------------------- |
| Frontend        | React 18, Vite, TypeScript, Tailwind CSS, TanStack Query v5, React Router v6, i18next |
| Backend         | Go 1.22, Gin (HTTP), GORM (ORM + AutoMigrate), golang-jwt/jwt/v5                      |
| Database        | PostgreSQL 16                                                                         |
| Cache           | Redis 7                                                                               |
| Messaging       | NATS                                                                                  |
| Email (dev)     | Mailpit (SMTP catcher)                                                                |
| Hot reload (Go) | Air                                                                                   |
| Monorepo        | pnpm workspaces + Turborepo (frontend + shared packages)                              |
| E2E tests       | Playwright (both apps)                                                                |

---

## Repository layout

```
civicos/
├── apps/
│   ├── web/                     # Citizen React app (port 5173)
│   └── admin/                   # Admin React app (port 5174)
├── services/
│   ├── api-gateway/             # Go — reverse proxy + JWT (port 3000)
│   ├── identity-service/        # Go — auth, users (port 3001)
│   ├── community-service/       # Go — communities, issues, petitions (port 3002)
│   └── organization-service/    # Go — organizations, announcements (port 3003)
├── packages/
│   ├── types/                   # Shared TS interfaces + enums (@civicos/types)
│   ├── config/                  # Env validation via zod (@civicos/config)
│   └── ui/                      # Shared React components (@civicos/ui)
├── infrastructure/
│   └── docker-compose.yml       # Postgres 16 + Redis 7 + NATS + Mailpit
├── docs/
│   ├── product/                 # 5 source PDFs (Blueprint, Roadmap, Architecture, UX, Playbook)
│   ├── api/                     # Endpoint reference
│   └── setup.md                 # Extended setup notes
├── CLAUDE.md                    # AI collaborator context
└── README.md
```

Each Go service follows the same layout:

```
service/
├── cmd/server/main.go           # DI wire-up + Gin bootstrap
├── internal/
│   ├── domain/models.go         # GORM models + enums
│   ├── middleware/auth.go       # JWT + role + ban/deletion enforcement
│   └── <feature>/               # repository.go, service.go, handler.go
└── pkg/
    ├── config/config.go
    ├── database/postgres.go     # GORM connection + AutoMigrate
    └── response/response.go
```

---

## Prerequisites

- **Node.js** 20+
- **pnpm** 9+ (`npm install -g pnpm`)
- **Go** 1.22+
- **Docker** Desktop (or compatible engine)
- **Air** — Go hot reload
  ```bash
  go install github.com/air-verse/air@latest
  ```

---

## Getting started

```bash
# 1. Clone and enter
git clone <repo-url> civicos && cd civicos

# 2. Start local infrastructure (Postgres, Redis, NATS, Mailpit)
docker compose -f infrastructure/docker-compose.yml up -d

# 3. Configure environment
cp .env.example .env
# Edit .env — set JWT_SECRET to a 32+ char random string.
# Generate one with:  openssl rand -base64 48

# 4. Install workspace deps (frontends + shared packages)
pnpm install

# 5. Start the 4 Go services — each in its own terminal.
# Air runs from the service dir, so its shell won't find the repo-root .env
# via godotenv. Source it first (or symlink .env into each service dir).
set -a && source .env && set +a

cd services/identity-service     && air
cd services/community-service    && air
cd services/organization-service && air
cd services/api-gateway          && air

# 6. Start both frontends (turbo runs web + admin in parallel)
pnpm dev
```

First-boot notes:

- Each Go service runs `AutoMigrate` on startup — the schema builds itself.
- **Seed an admin user**: register normally via the web app, then in the DB flip their `role` column to `PLATFORM_ADMIN` so the admin console lets you in.
- Verification emails land in **Mailpit** at `http://localhost:8025`.

---

## Running services

| Service              | URL                        | Purpose                                                                     |
| -------------------- | -------------------------- | --------------------------------------------------------------------------- |
| Citizen web          | http://localhost:5173      | Public homepage + citizen app                                               |
| Admin console        | http://localhost:5174      | Moderation, metrics, entity management                                      |
| API gateway          | http://localhost:3000      | Single entry point for all API calls                                        |
| Identity service     | http://localhost:3001      | Auth, users, admin metrics                                                  |
| Community service    | http://localhost:3002      | Communities, issues, petitions, reps, flags                                 |
| Organization service | http://localhost:3003      | Organizations, announcements, projects                                      |
| Swagger UI           | http://localhost:3000/docs | Interactive API docs — picker for identity / community / organization specs |
| Postgres             | localhost:5433             | Data                                                                        |
| Redis                | localhost:6379             | Rate-limit counters + SSE fan-out                                           |
| NATS                 | localhost:4222             | Inter-service messaging (monitor: `:8222`)                                  |
| Mailpit UI           | http://localhost:8025      | Dev SMTP catcher — verification + reset emails                              |

Every citizen and admin request goes through the gateway (`:3000`); the frontends never call service ports directly.

---

## Common tasks

**Reset a rate-limited account**

```bash
docker exec civicos_redis redis-cli FLUSHDB > /dev/null
```

**Reset the database** (development only — destroys all data)

```bash
docker compose -f infrastructure/docker-compose.yml down -v
docker compose -f infrastructure/docker-compose.yml up -d
```

**Run e2e tests**

```bash
pnpm --filter web test:e2e
pnpm --filter admin test:e2e
```

**Add a new locale**

1. Copy `apps/web/src/i18n/locales/en.json` to `apps/web/src/i18n/locales/<code>.json`
2. Translate every key
3. Register the locale in `apps/web/src/i18n/index.ts`
4. Add it to the `LanguageSwitcher` label map

**Regenerate Go modules after adding a dep**

```bash
cd services/<service>
go mod tidy
```

---

## Environment variables

Minimum set required in `.env`:

| Variable                    | Example                                               | Notes                               |
| --------------------------- | ----------------------------------------------------- | ----------------------------------- |
| `JWT_SECRET`                | (32+ char random string)                              | Required. `openssl rand -base64 48` |
| `DATABASE_URL`              | `postgresql://civicos:civicos@localhost:5433/civicos` | Set by docker-compose defaults      |
| `REDIS_URL`                 | `redis://localhost:6379`                              |                                     |
| `NATS_URL`                  | `nats://localhost:4222`                               |                                     |
| `API_GATEWAY_PORT`          | `3000`                                                |                                     |
| `IDENTITY_SERVICE_PORT`     | `3001`                                                |                                     |
| `COMMUNITY_SERVICE_PORT`    | `3002`                                                |                                     |
| `ORGANIZATION_SERVICE_PORT` | `3003`                                                |                                     |
| `SMTP_HOST`                 | `localhost`                                           | Mailpit in dev                      |
| `SMTP_PORT`                 | `1025`                                                | Mailpit in dev                      |
| `VITE_API_URL`              | `http://localhost:3000`                               | Gateway URL for both frontends      |

See `.env.example` for the complete list.

---

## Documentation

- **Swagger UI** at `http://localhost:3000/docs` — browseable API docs with a picker for identity / community / organization
- `docs/product/` — the five source documents that drive every product and architectural decision (Blueprint, Roadmap, Architecture, Experience, Engineering Playbook)
- `docs/api/openapi-*.yaml` — canonical OpenAPI 3.0 specs, one per service (mirrored into the gateway for embedding)
- `docs/setup.md` — extended local-dev notes and troubleshooting
- `CLAUDE.md` — context file for AI assistants working in this repo (also useful as a human onboarding brief)

---

## Engineering conventions (short version)

1. **Dependency injection always** — `NewRepository(db)` → `NewService(repo)` → `NewHandler(svc)` in `main.go`
2. **UUIDs everywhere** — never sequential IDs
3. **Validate at boundaries** — `binding:"required"` on input structs; never trust raw request data
4. **Error codes, not raw messages** — every error returns `{code, message}`
5. **Conventional commits** — `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `ci:` — Husky enforces
6. **Files under 300 lines** — split at 500
7. **Timestamps in UTC** — localize only in the UI
8. **Sensitive fields carry `json:"-"`** — passwords, tokens, reset codes never serialize
9. **Log for operators** — service, route, status; never log passwords or tokens

The Go architecture diverges from the Engineering Playbook PDF (which specifies NestJS). See `CLAUDE.md` for the rationale.

---

## Status

MVP complete. Currently in local-development phase — deployment and CI are the next launch-blockers. See `docs/product/CivicOS Product Roadmap.pdf` for the five-phase build order.

## License

TBD — check with the maintainers before redistributing.
