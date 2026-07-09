---
id: packages
title: Packages
sidebar_position: 2
---

# Packages

The three shared TypeScript packages under `packages/` back the two
React apps. All are namespaced under `@civicos/*` and consumed via
`workspace:*` — pnpm resolves them to symlinks so a change is visible
immediately without publishing.

## `@civicos/types`

**Purpose:** the DTOs and enums that cross the frontend / backend
boundary. Both apps import from here so a rename or a new enum member
flows through TypeScript across the whole platform.

Contents you'll typically find:

- `UserRole`, `RequestedAccountType`, `ApprovalStatus` — role and
  application enums that mirror the Go domain enums.
- `IssueStatus`, `IssueCategory`, `PetitionStatus`,
  `NotificationType` — community-scoped enums.
- Request/response envelope types (`ApiSuccess<T>`, `ApiError`).
- Public entity shapes: `PublicUser`, `Community`, `Issue`, `Petition`,
  `Representative`, `Notification`.

**Rule:** if a value crosses the wire, its type lives here. Don't
duplicate an interface between `apps/web` and `apps/admin`.

## `@civicos/config`

**Purpose:** zod-validated access to `import.meta.env` for the two
frontends. Fails fast at startup if a required var is missing rather
than exploding later with an undefined URL.

Typical usage:

```ts
import { env } from '@civicos/config';

const apiClient = axios.create({
  baseURL: env.VITE_API_URL, // typed as string, guaranteed non-empty
});
```

Add a new variable by extending the zod schema in
`packages/config/src/index.ts`. If you don't add it to the schema, the
frontend can't read it.

## `@civicos/ui`

**Purpose:** shared React components used by both `apps/web` and
`apps/admin`. Buttons, form primitives, layout shells, empty-state
components — anything both apps benefit from being consistent about.

Keep app-specific components in each app's `src/features/` folder.
Only extract to `@civicos/ui` when a component is _actually_ used in
both apps — premature sharing creates coupling for no gain.

## Consuming from an app

`package.json` in each app already lists the shared packages:

```json
{
  "dependencies": {
    "@civicos/types": "workspace:*",
    "@civicos/config": "workspace:*",
    "@civicos/ui": "workspace:*"
  }
}
```

Import from them like any other package:

```ts
import { IssueStatus, type PublicUser } from '@civicos/types';
import { Button } from '@civicos/ui';
```

## Adding a new shared package

Follow the pattern in [Monorepo → Adding a new frontend package](../overview/monorepo.md#adding-a-new-frontend-package).

## What the packages are _not_

- **Not a component library for external consumers.** They're internal.
  No versioning, no changelog, no publish step — the workspace protocol
  handles everything.
- **Not for backend code.** The Go services don't consume them. If you
  need a shared Go primitive, put it in each service's `pkg/` folder,
  or accept the duplication.

Next: [Contributing](./contributing.md).
