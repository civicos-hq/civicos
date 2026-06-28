# CivicOS

> The open civic operating system — connecting citizens, representatives, and communities.

## Structure

```
civicos/
├── apps/
│   └── web/                  # React + Vite frontend (port 5173)
├── services/
│   ├── api-gateway/          # Go — reverse proxy, JWT validation (port 3000)
│   ├── identity-service/     # Go — auth & user management (port 3001)
│   └── community-service/    # Go — communities, issues, petitions (port 3002)
├── packages/
│   ├── types/                # Shared TypeScript types
│   ├── config/               # Shared env validation
│   └── ui/                   # Shared React component library
├── infrastructure/
│   └── docker-compose.yml    # Local dev stack (Postgres, Redis, NATS)
└── docs/
    └── product/              # Product, architecture & engineering PDFs
```

## Prerequisites

- Node.js 20+
- pnpm 9+
- Go 1.22+
- Docker Desktop
- Air (Go hot reload): `go install github.com/air-verse/air@latest`

## Getting Started

```bash
# 1. Start local infrastructure (Postgres, Redis, NATS)
docker compose -f infrastructure/docker-compose.yml up -d

# 2. Install frontend/package deps
pnpm install

# 3. Configure environment
cp .env.example .env   # fill in JWT_SECRET (min 32 chars)

# 4. Start Go services — each in a separate terminal
cd services/identity-service && air
cd services/community-service && air
cd services/api-gateway && air

# 5. Start frontend
pnpm dev
```

| Service | URL |
|---------|-----|
| Frontend | http://localhost:5173 |
| API Gateway | http://localhost:3000 |
| Identity Service | http://localhost:3001 |
| Community Service | http://localhost:3002 |

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | React 18 + Vite + TypeScript |
| Styling | Tailwind CSS |
| Data fetching | TanStack Query |
| Routing | React Router v6 |
| Backend | Go 1.22 + Gin |
| ORM | GORM |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Messaging | NATS |
| Monorepo | pnpm workspaces + Turborepo |

## Commit Conventions

This project follows [Conventional Commits](https://www.conventionalcommits.org/).

```
feat: add issue upvote endpoint
fix: resolve token refresh race condition
chore: update dependencies
docs: add API reference for petitions
```

Husky enforces this on every commit via commitlint.
