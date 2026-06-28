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
civicos-v2/
├── apps/
│   └── web/                  # React + Vite + TypeScript frontend (port 5173)
├── services/
│   ├── api-gateway/          # NestJS gateway — routes requests to services (port 3000)
│   ├── identity-service/     # NestJS — auth, users, JWT (port 3001)
│   └── community-service/    # NestJS — communities, issues, petitions (port 3002)
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

## Tech stack

- **Frontend**: React 18, Vite, TypeScript, Tailwind CSS, TanStack Query, React Router v6
- **Backend**: NestJS + TypeScript (all services)
- **ORM**: Prisma (PostgreSQL)
- **Cache**: Redis
- **Messaging**: NATS
- **Monorepo**: pnpm workspaces + Turborepo
- **Package manager**: pnpm

## Engineering rules (from Engineering Playbook)

1. **Dependency injection always** — never `new Repository()` inside a service. Use NestJS DI.
2. **No `any`** — TypeScript strict mode is on. Use `unknown` or explicit types.
3. **UUIDs** — all entity IDs are UUIDs (`@default(uuid())` in Prisma). Never expose sequential IDs.
4. **Validate at boundaries** — use `class-validator` DTOs in NestJS. Never trust raw input.
5. **Single responsibility** — each class/service does one thing. `AuthService` does not send emails.
6. **Conventional commits** — `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `ci:` — Husky enforces this.
7. **Files under 300 lines** — if a file grows past 500, consider splitting it.
8. **Store timestamps in UTC** — convert to local time only in the UI.
9. **Error codes, not raw messages** — return `{ code: "ISSUE_NOT_FOUND", message: "..." }` from services.
10. **Log for operators** — include requestId, userId, executionTime. Never log passwords or tokens.

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
2. 🔲 Identity Service — register, login, refresh, /me
3. 🔲 Community Service — create community, join, report issue
4. 🔲 Frontend auth flow — register, login, dashboard shell
5. 🔲 Issue reporting UI + API
6. 🔲 Representative pages
7. 🔲 Notifications
8. 🔲 Petitions

## Running locally

```bash
pnpm install
cp .env.example .env          # fill in JWT_SECRET at minimum
docker compose -f infrastructure/docker-compose.yml up -d
pnpm --filter identity-service db:migrate
pnpm --filter community-service db:migrate
pnpm dev
```
