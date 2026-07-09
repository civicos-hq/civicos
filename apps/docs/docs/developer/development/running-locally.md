---
id: running-locally
title: Running locally
sidebar_position: 1
---

# Running locally

The end-to-end local dev loop is: bring up Docker infra → install pnpm
workspaces → install Go tools → start each Go service with Air → start
the frontends with Vite.

## Prerequisites

- **Node.js** 20+
- **pnpm** 9+ (`npm install -g pnpm`)
- **Go** 1.22+
- **Docker Desktop** (or compatible)
- **Air** — Go hot-reload tool:
  ```bash
  go install github.com/air-verse/air@latest
  ```
  Make sure `~/go/bin` is on your `PATH`.

## First-time setup

```bash
# 1. Clone
git clone <repo-url> civicos && cd civicos

# 2. Start infrastructure
docker compose -f infrastructure/docker-compose.yml up -d

# 3. Configure environment
cp .env.example .env
# Edit .env — set JWT_SECRET to a 32+ char random string:
#   openssl rand -base64 48

# 4. Install pnpm workspaces
pnpm install
```

## Starting the services

Each Go service calls `godotenv.Load()` in its own process, which looks
for `.env` in the process's current working directory. `air` runs from
inside the service directory, so it won't find the repo-root `.env`
unless you source it first (or symlink `.env` into each service dir).

**Recommended one-liner per terminal:**

```bash
set -a && source /path/to/civicos/.env && set +a

cd services/identity-service      && air
cd services/community-service     && air
cd services/organization-service  && air
cd services/api-gateway           && air
```

Then in one more terminal:

```bash
pnpm dev
```

This starts every TS workspace's `dev` script in parallel via Turbo —
the two frontends and this Docusaurus site.

## Ports at a glance

| Component            | URL                        | Notes                                            |
| -------------------- | -------------------------- | ------------------------------------------------ |
| Citizen web          | http://localhost:5173      | Vite dev server                                  |
| Admin console        | http://localhost:5174      | Vite dev server                                  |
| User Guide (docs)    | http://localhost:5175      | Docusaurus dev server (this site)                |
| API gateway          | http://localhost:3000      | Single entry point for all API calls             |
| Identity service     | http://localhost:3001      | Auth, users, applications, moderation            |
| Community service    | http://localhost:3002      | Communities, issues, petitions                   |
| Organization service | http://localhost:3003      | Organizations, projects, assignments             |
| Swagger UI           | http://localhost:3000/docs | API reference — served by the gateway            |
| Postgres             | localhost:5433             | Mapped from container's 5432 → host's 5433       |
| Redis                | localhost:6379             |                                                  |
| NATS                 | localhost:4222             | HTTP monitor at `:8222`                          |
| Mailpit UI           | http://localhost:8025      | Dev SMTP catcher — verification emails land here |

## Seeding an admin user

1. Register normally in the citizen web app.
2. Verify your email — it lands in Mailpit at `http://localhost:8025`.
3. Bump your role to `PLATFORM_ADMIN` in the database:

```bash
docker exec -it civicos_postgres psql -U civicos -d civicos \
  -c "UPDATE users SET role='PLATFORM_ADMIN' WHERE email='<your-email>';"
```

Now the admin console at `:5174` lets you in.

## Health checks

Every service exposes `/health`. The gateway also proxies per-service
health probes:

```bash
curl http://localhost:3000/health              # gateway itself
curl http://localhost:3000/health/identity     # via gateway
curl http://localhost:3000/health/community
curl http://localhost:3000/health/organization
```

Use these when something feels off.

## Common resets

**Wipe Redis (e.g. after rate-limiting yourself out):**

```bash
docker exec civicos_redis redis-cli FLUSHDB > /dev/null
```

**Wipe the database (destructive — dev only):**

```bash
docker compose -f infrastructure/docker-compose.yml down -v
docker compose -f infrastructure/docker-compose.yml up -d
```

Each Go service re-runs `AutoMigrate` on next startup, so the schema
rebuilds itself.

**Kill stuck Go processes on service ports:**

```bash
lsof -tiTCP:3000,3001,3002,3003 -sTCP:LISTEN | xargs -r kill
```

## Common failure modes

- **`missing required env var: DATABASE_URL`** — you started a service
  without sourcing `.env`. See the section above.
- **`JWT_SECRET must be at least 32 characters`** — same fix.
- **`connect: connection refused` from a service** — Postgres or Redis
  isn't running (`docker compose ps` to check).
- **Frontend showing CORS errors** — the gateway's allowed origins are
  hardcoded to 5173/5174/5175 in `services/api-gateway/cmd/server/main.go`.
  If you moved the frontend to another port, edit that list.

Next: [Packages](./packages.md).
