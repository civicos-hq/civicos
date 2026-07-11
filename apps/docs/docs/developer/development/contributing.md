---
id: contributing
title: Contributing
sidebar_position: 3
---

# Contributing

## Conventional commits

Commit messages follow **conventional commits** — Husky rejects
anything else.

```
feat: add primary community change endpoint
fix: prevent duplicate upvotes on issues
chore: bump gin to 1.10
docs: describe representative dashboard flow
test: cover petition milestone thresholds
refactor: split issue handler into subhandlers
ci:    add render build cache
```

Type reference:

- `feat:` — new feature
- `fix:` — bug fix
- `chore:` — dependency bumps, tooling
- `docs:` — documentation only
- `test:` — tests only
- `refactor:` — no behaviour change, code shape only
- `ci:` — pipeline / infra
- `perf:` — performance without behaviour change

Prefer **one commit per logical change**. If a PR ends up with 12
commits, squash on merge.

## The 10 engineering rules

1. **Dependency injection always.** `NewRepository(db)` →
   `NewService(repo)` → `NewHandler(svc)` wired in `main.go`. Never
   instantiate inside a struct.
2. **No stringly-typed IDs.** All entity IDs are UUIDs
   (`uuid.New().String()`).
3. **Validate at boundaries.** Use `binding:"required"` on input
   structs. Never trust raw request data.
4. **Single responsibility per file.** `auth/service.go` does not touch
   HTTP.
5. **Conventional commits** — enforced by Husky.
6. **Files under 300 lines.** Split at 500.
7. **Store timestamps in UTC.** Localize only in the UI.
8. **Error codes, not raw messages.** Every service returns
   `{code, message}`.
9. **Log for operators.** Include service name, route, status. Never
   log passwords or tokens.
10. **`PasswordHash` never serialised.** Use `json:"-"` on any
    sensitive field.

Details in [`CLAUDE.md`](https://github.com/civicos-hq/civicos/blob/main/CLAUDE.md)
at the repo root.

## Adding a Go endpoint

Reach for the standard trio in the target service:

1. **Repository** — write the GORM query in
   `internal/<feature>/repository.go`.
2. **Service** — business logic in `internal/<feature>/service.go`.
   Consumes the repository interface.
3. **Handler** — Gin handler in `internal/<feature>/handler.go`.
   Binds input, calls the service, wraps the response.
4. **Route** — register in `handler.RegisterRoutes(rg)` and mount from
   `cmd/server/main.go`.
5. **Gateway** — add a `POST/GET/PATCH …` line to
   `services/api-gateway/cmd/server/main.go` with the right rate-limit
   tier.
6. **Docs** — add the endpoint to `docs/api/openapi-<svc>.yaml` with
   summary, description, request/response examples, and status codes.
   Then re-sync to the embedded copy:
   ```bash
   scripts/openapi-sync.sh
   ```
   CI runs the same script with `--check` and fails the build if the
   two copies drift.

## Adding a frontend feature

- **State** — TanStack Query for anything the server owns. Local
  `useState` for anything ephemeral. Avoid Redux.
- **Forms** — react-hook-form + zod resolver.
- **Types** — from `@civicos/types` if the value crosses the wire; from
  the feature folder if it's app-local.
- **i18n** — every user-visible string in the citizen app goes into
  `apps/web/src/i18n/locales/en.json`. The admin console is
  English-only for now.

## Running tests

```bash
# Frontend
pnpm --filter @civicos/web test:e2e
pnpm --filter @civicos/admin test:e2e

# Go — from a service directory
go test ./...
```

E2E tests use Playwright. They spin up their own web server, so you
don't need `pnpm dev` running.

## Before you push

```bash
pnpm typecheck   # tsc --noEmit across every TS workspace
pnpm lint        # eslint across every TS workspace
# from a Go service:
go vet ./...
go test ./...
```

Husky runs these on commit staged files. Don't skip hooks —
`--no-verify` is off-limits unless you've discussed with the maintainers.

## Pull requests

- **Small.** Big PRs are hard to review — split.
- **Titled with the type prefix** matching your commit convention.
- **Description explains the _why_.** The diff already shows the
  _what_.
- **Screenshots for UI changes.** Before / after is ideal.

## Where to ask

- Documentation — this Docusaurus site plus `CLAUDE.md` and the five
  PDFs in `docs/product/`.
- Chat — TBD, ask the maintainers.
- Filing an issue — GitHub. Attach reproduction steps.
