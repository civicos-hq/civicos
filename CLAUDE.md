# CivicOS — Claude Code Context

## What is CivicOS?

CivicOS is an open civic infrastructure platform — the "operating system for democratic participation." It bridges the gap between citizens and their governments between elections by enabling continuous civic engagement.

Core users: **citizens**, **elected representatives**, **government admins**, **NGOs**, **moderators**.

## Source documents

All product, architecture, and engineering decisions are driven by five documents in `docs/product/`:

| Document | Purpose |
|----------|---------|
| `CivicOS Blueprint.pdf` | Vision, mission, product philosophy, core principles |
| `CivicOS Product Roadmap.pdf` | What gets built and in what order (5 phases) |
| `CivicOS Technical Architecture v1.0.pdf` | How the system is designed — services, data, security, AI |
| `CivicOS Experience Architecture.pdf` | UX, user journeys, screen specs, design system |
| `CivicOS Engineering Playbook.pdf` | How software is written — standards, conventions, DI, testing |

**Before implementing any feature, consult the relevant document first.**

## Repository structure

```
civicos/
├── apps/
│   └── web/                  # React + Vite + TypeScript frontend (port 5173)
├── services/
│   ├── api-gateway/          # Go — reverse proxy, JWT validation (port 3000)
│   ├── identity-service/     # Go — auth, users, JWT (port 3001)
│   └── community-service/    # Go — communities, issues, petitions (port 3002)
├── packages/
│   ├── types/                # Shared TypeScript interfaces & enums (@civicos/types)
│   ├── config/               # Env validation via zod (@civicos/config)
│   └── ui/                   # Shared React components (@civicos/ui)
├── infrastructure/
│   └── docker-compose.yml    # Postgres 16 + Redis 7 + NATS
├── docs/
│   ├── product/              # The five source documents (PDFs)
│   └── setup.md              # Getting started guide
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
2. ✅ Identity Service — register, login, refresh, /me (Go)
3. ✅ Community Service — communities, issues (Go)
4. ✅ API Gateway — reverse proxy, JWT validation (Go)
5. 🔲 Frontend auth flow — register, login, dashboard shell
6. 🔲 Issue reporting UI + API
7. 🔲 Representative pages
8. 🔲 Notifications
9. 🔲 Petitions

## Running locally

```bash
# 1. Start infrastructure
docker compose -f infrastructure/docker-compose.yml up -d

# 2. Install frontend/packages deps
pnpm install

# 3. Copy env and fill in JWT_SECRET (min 32 chars)
cp .env.example .env

# 4. Start Go services (each in a separate terminal, requires `air` installed)
cd services/identity-service && air
cd services/community-service && air
cd services/api-gateway && air

# 5. Start frontend
pnpm dev

# Install air for hot reload (one-time):
go install github.com/air-verse/air@latest
```
