---
id: repository-structure
title: Repository structure
sidebar_position: 2
---

# Repository structure

```
civicos/
├── apps/                          # pnpm workspaces (React + docs)
│   ├── web/                       # Citizen app (Vite + React, :5173)
│   ├── admin/                     # Admin console (Vite + React, :5174)
│   └── docs/                      # This Docusaurus site (:5175)
│
├── services/                      # One folder per Go service — separate go.mod each
│   ├── api-gateway/               # :3000 — reverse proxy + JWT + rate limit
│   ├── identity-service/          # :3001 — auth, users, applications, moderation
│   ├── community-service/         # :3002 — communities, issues, petitions, reps
│   └── organization-service/      # :3003 — orgs, projects, announcements
│
├── packages/                      # Shared TS packages (pnpm workspaces)
│   ├── types/                     # @civicos/types — DTOs + enums shared by both apps
│   ├── config/                    # @civicos/config — zod-validated env access for the frontends
│   └── ui/                        # @civicos/ui — shared React components
│
├── infrastructure/
│   └── docker-compose.yml         # Postgres 16 + Redis 7 + NATS + Mailpit
│
├── docs/
│   ├── product/                   # 5 source PDFs (Blueprint, Roadmap, Architecture, UX, Playbook)
│   ├── api/openapi-*.yaml         # Canonical OpenAPI specs — mirrored into api-gateway
│   ├── deploy.md                  # Render deployment playbook
│   └── setup.md                   # Older, partially stale — being superseded by this Docusaurus
│
├── render.yaml                    # Render Blueprint (declarative infra)
├── turbo.json                     # Turborepo pipeline
├── pnpm-workspace.yaml            # Workspace globs — apps/*, services/*, packages/*
├── package.json                   # Root scripts + shared dev deps
├── .env                           # Local secrets (gitignored)
├── CLAUDE.md                      # AI collaborator context (also useful for humans)
└── README.md
```

## Each Go service is a mini-project

```
services/<name>/
├── cmd/server/main.go             # Entry point — wires DI, starts Gin
├── internal/
│   ├── domain/models.go           # GORM models + enums
│   ├── middleware/                # JWT + role + ban/deletion enforcement
│   └── <feature>/                 # repository.go + service.go + handler.go per feature
├── migrations/                    # Optional — GORM AutoMigrate does most of the work
├── pkg/
│   ├── config/config.go           # Env loading + validation
│   ├── database/postgres.go       # GORM connection
│   ├── mailer/                    # SMTP + console mailers (identity only)
│   └── response/response.go       # Success/Error envelope helpers
├── .air.toml                      # Air hot-reload config
├── Dockerfile                     # Multi-stage: golang build → distroless final
├── go.mod
└── go.sum
```

Each service has its own `go.mod` — there's no shared Go module. This
means each service can upgrade dependencies independently, and a
breaking change in one service's model can't accidentally cross into
another. The cost is more duplication in `go.sum` files.

Inside `internal/<feature>/` the standard trio is:

- **`repository.go`** — data access. Only touches GORM. Never talks
  HTTP.
- **`service.go`** — business logic. Consumes the repository interface.
  Never touches Gin.
- **`handler.go`** — HTTP layer. Consumes the service. Never touches
  the database.

Wired together in `cmd/server/main.go` via constructor injection.

## Each frontend app is standard Vite

```
apps/<name>/
├── src/
│   ├── App.tsx
│   ├── main.tsx
│   ├── routes/                   # React Router v6 route trees
│   ├── features/                 # Feature-scoped components + hooks
│   ├── lib/                      # API client, auth helpers, utils
│   └── i18n/                     # Translation resources (web only)
├── public/
├── index.html
├── vite.config.ts
├── tsconfig.json
├── playwright.config.ts          # E2E tests
└── package.json
```

## Where things you might look for live

| Looking for             | Path                                                    |
| ----------------------- | ------------------------------------------------------- |
| JWT middleware          | `services/<svc>/internal/middleware/auth.go`            |
| Response envelope       | `services/<svc>/pkg/response/response.go`               |
| Rate-limit tiers        | `services/api-gateway/internal/middleware/ratelimit.go` |
| GORM models             | `services/<svc>/internal/domain/models.go`              |
| Shared TypeScript enums | `packages/types/src/index.ts`                           |
| Product source of truth | `docs/product/*.pdf`                                    |
| OpenAPI specs           | `docs/api/openapi-*.yaml`                               |
| Swagger UI handler      | `services/api-gateway/internal/docs/docs.go`            |
| Render deploy config    | `render.yaml`                                           |

Next: [Monorepo](./monorepo.md).
