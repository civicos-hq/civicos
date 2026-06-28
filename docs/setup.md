# Getting Started

## Prerequisites
- Node.js 20+
- pnpm 9+ (`npm install -g pnpm`)
- Docker Desktop

## First-time setup

```bash
# 1. Install all workspace dependencies
pnpm install

# 2. Set up environment
cp .env.example .env
# Edit .env — at minimum set JWT_SECRET to a 32+ char string

# 3. Start Postgres, Redis, and NATS
docker compose -f infrastructure/docker-compose.yml up -d

# 4. Generate Prisma clients and run migrations
pnpm --filter identity-service db:generate
pnpm --filter identity-service db:migrate

pnpm --filter community-service db:generate
pnpm --filter community-service db:migrate

# 5. Start all services in dev/watch mode
pnpm dev
```

## Running services

| Service | URL | Description |
|---------|-----|-------------|
| Frontend (Vite) | http://localhost:5173 | React app |
| API Gateway | http://localhost:3000 | Entry point for all API calls |
| Identity Service | http://localhost:3001 | Auth, users |
| Community Service | http://localhost:3002 | Communities, issues, petitions |
| NATS Monitor | http://localhost:8222 | Messaging dashboard |
| Prisma Studio (identity) | Run `pnpm --filter identity-service db:studio` | |
| Prisma Studio (community) | Run `pnpm --filter community-service db:studio` | |

## Commit conventions

```bash
feat: add petition signing endpoint
fix: resolve JWT refresh race condition
chore: update prisma to 5.17
docs: update API reference
test: add auth service unit tests
```

Husky blocks commits that don't follow this format.
