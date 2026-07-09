---
id: organization-service
title: Organization Service
sidebar_position: 4
---

# Organization Service

Port `:3003`. The "here's who's responsible" side of the platform —
organizations that can take responsibility for issues, run projects, and
post announcements.

## Responsibilities

- **Organizations** — CRUD, verified-badge toggle (audit-logged
  separately), search by kind / jurisdiction / state / LGA.
- **Members** — org-internal roles (`OWNER`, `ADMIN`, `STAFF`),
  add / update role / remove.
- **Announcements** — DRAFT → PUBLISHED → ARCHIVED lifecycle. Global
  feed of published items plus per-org list.
- **Projects** — planned / active / paused / completed / cancelled,
  optional budget in kobo, optional community link.
- **Issue assignments** — records that an org has claimed an issue.
  Members-only reads on the org's inbox; public reads on the per-issue
  list.
- **Progress updates** — the "respond publicly" primitive. Hangs off
  either an assigned issue or a project.

## Package layout

```
services/organization-service/
├── cmd/server/main.go
├── internal/
│   ├── announcements/          # DRAFT / PUBLISHED / ARCHIVED
│   ├── assignments/            # Org takes on an issue
│   ├── audit/                  # Writes to audit_logs
│   ├── domain/models.go
│   ├── middleware/             # JWTAuth, RequireRole
│   ├── organizations/          # Registry + membership
│   ├── progress/               # Progress updates
│   └── projects/               # Projects
└── pkg/…
```

## Key concepts

### Cross-service references

Issue assignments reference an `issueId` UUID that lives in
community-service. There's **no foreign key** — the two schemas share a
database but not a schema module.

- **Why:** each service is deployable independently. A FK would couple
  their migration order.
- **Cost:** an orphaned assignment can survive if the referenced issue
  is deleted. Acceptable at MVP; a background reconcile job would clean
  up if we ever need to.

### Org role vs. platform role

Two role systems overlap here:

- **Platform role** — on the `User` JWT — determines who can _create_
  a new org (`GOVERNMENT_ADMIN`, `PLATFORM_ADMIN`, `NGO`).
- **Org role** — on the `OrgMember` row — governs who can post
  announcements, assignments, etc. _inside_ an org.

Once an org exists, everything internal (edits, adds, deletes) is
gated by org role, not platform role. `PLATFORM_ADMIN` is the
super-user that bypasses org-role checks (used by
`assignments.Handler.create` and `listByOrg`).

### The verified badge is its own audit action

Toggling `Organization.verified` writes an `org.verified` or
`org.unverified` audit entry — distinct from the plain `org.updated`
action. The verified badge is a citizen-facing trust signal, so its
flip is worth its own action name for review.

### Progress updates — exactly one target

`progress.CreateInput` requires **exactly one** of `issueId` or
`projectId`. `(input.IssueID == nil) == (input.ProjectID == nil)` is
the guard that rejects both empty or both set. Returns `400
INVALID_TARGET` on violation.

## Environment

- `DATABASE_URL`
- `JWT_SECRET`
- `PORT` / `ORGANIZATION_SERVICE_PORT` (default `3003`)
