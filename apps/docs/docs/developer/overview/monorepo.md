---
id: monorepo
title: Monorepo
sidebar_position: 3
---

# Monorepo

CivicOS is a **partial monorepo**: pnpm workspaces + Turborepo cover the
TypeScript side (`apps/*` + `packages/*`), and Go modules stand alone
per service. There's no unifying Go workspace on top.

## Why two build systems?

- **pnpm + Turborepo** is idiomatic for TS monorepos. Deduped
  `node_modules`, workspace-relative imports, incremental builds.
- **Go modules per service** is idiomatic for Go microservices.
  Independent dependency graphs, no accidental cross-service coupling,
  smaller build contexts for Docker.

Trying to unify them (e.g. Bazel) would be more machinery than a
four-service platform justifies. If the service count doubles, revisit.

## Workspace globs

```yaml
# pnpm-workspace.yaml
packages:
  - 'apps/*'
  - 'services/*'
  - 'packages/*'
```

Yes, `services/*` is in there — but the Go services don't have
`package.json` files, so pnpm ignores them at install time. It's there
because a few services will eventually get sidecar tooling (E2E
harnesses, benchmark scripts) and we want them under one glob when they
do.

## Turborepo pipeline

`turbo.json` defines `dev`, `build`, `lint`, `typecheck`, `test` tasks.
Each task is a function of workspace inputs, so Turbo caches outputs
and only re-runs what changed.

Root commands wire straight through:

```bash
pnpm dev         # turbo run dev        — starts every workspace's dev script
pnpm build       # turbo run build      — builds every workspace's build
pnpm typecheck   # turbo run typecheck  — tsc --noEmit across every TS workspace
pnpm lint        # turbo run lint       — eslint across every TS workspace
```

Turbo won't touch Go — you still run `air` per service, or
`go build ./...` from a service directory.

## Package naming

TS workspaces are all under the `@civicos/` scope:

- `@civicos/web` — citizen app
- `@civicos/admin` — admin console
- `@civicos/docs` — this Docusaurus site
- `@civicos/types` — shared types
- `@civicos/config` — shared frontend env validation
- `@civicos/ui` — shared React components

Referenced from `dependencies` as `"@civicos/types": "workspace:*"` —
pnpm rewrites this to a symlink at install time.

## Adding a new frontend package

```bash
# 1. Create the folder
mkdir -p packages/new-pkg/src

# 2. Add package.json
cat > packages/new-pkg/package.json <<EOF
{
  "name": "@civicos/new-pkg",
  "version": "0.0.1",
  "private": true,
  "main": "src/index.ts",
  "types": "src/index.ts"
}
EOF

# 3. Add a tsconfig
cat > packages/new-pkg/tsconfig.json <<EOF
{
  "extends": "../../tsconfig.base.json",
  "include": ["src/**/*"]
}
EOF

# 4. Consume it
pnpm add --filter @civicos/web @civicos/new-pkg
```

## Adding a new Go service

1. `mkdir services/new-service` and copy the layout from an existing
   service (`cmd/server/`, `internal/`, `pkg/`, `.air.toml`,
   `Dockerfile`).
2. `cd services/new-service && go mod init github.com/civicos/new-service`.
3. Wire it into `render.yaml` and the gateway (`services/api-gateway/cmd/server/main.go`
   plus its `pkg/config`) so requests can route to it.
4. Add it to the running-services table in the root README.

## Running one workspace at a time

```bash
# Frontend
pnpm --filter @civicos/web dev
pnpm --filter @civicos/admin dev
pnpm --filter @civicos/docs dev

# TS lint / typecheck for one workspace
pnpm --filter @civicos/web typecheck
pnpm --filter @civicos/ui lint
```

## Prettier + husky

The root `package.json` has a `prepare` script wired to husky. On
`pnpm install`, husky installs the git hooks in `.husky/`. Commit hooks
run `lint-staged`, which runs Prettier on staged files. Commit messages
are enforced by `commitlint` against the conventional-commits config.

If a hook blocks you, fix the underlying issue rather than bypassing
with `--no-verify`.

Next: [Running locally](../development/running-locally.md).
