# CivicOS

> The open civic operating system — connecting citizens, representatives, and communities.

## Structure

```
civicos/
├── apps/
│   └── web/                  # React + Vite frontend
├── services/
│   ├── api-gateway/          # NestJS API Gateway
│   ├── identity-service/     # NestJS auth & user management
│   └── community-service/    # NestJS communities & issues
├── packages/
│   ├── types/                # Shared TypeScript types
│   ├── config/               # Shared env validation
│   └── ui/                   # Shared React component library
├── infrastructure/
│   └── docker-compose.yml    # Local dev stack
└── docs/
```

## Prerequisites

- Node.js 20+
- pnpm 9+
- Docker Desktop

## Getting Started

```bash
# 1. Install all dependencies
pnpm install

# 2. Configure environment
cp .env.example .env

# 3. Start local infrastructure (Postgres, Redis, NATS)
docker compose -f infrastructure/docker-compose.yml up -d

# 4. Run migrations
pnpm --filter identity-service db:migrate
pnpm --filter community-service db:migrate

# 5. Start all apps
pnpm dev
```

| App | URL |
|-----|-----|
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
| Backend | NestJS + TypeScript |
| ORM | Prisma |
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
