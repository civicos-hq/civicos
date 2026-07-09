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
- **Consultations** — structured feedback asks with a full DRAFT →
  PUBLISHED → CLOSED lifecycle, question builder, response submission,
  per-question analytics, and outcome publishing (the "close the loop"
  primitive).

## Package layout

```
services/organization-service/
├── cmd/server/main.go
├── internal/
│   ├── announcements/          # DRAFT / PUBLISHED / ARCHIVED
│   ├── assignments/            # Org takes on an issue
│   ├── audit/                  # Writes to audit_logs
│   ├── consultations/          # Structured feedback asks (see below)
│   ├── domain/models.go
│   ├── middleware/             # JWTAuth, RequireRole, RequireVerified
│   ├── notifications/          # Thin writer for the shared notifications table
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

### Consultations — five tables, one package

Five tables under `internal/domain/`:

- `Consultation` — the top-level record with status, target community
  (nullable), author, and denorm `response_count`.
- `ConsultationQuestion` — one row per question, ordered by `position`,
  with a JSON `options` array for choice types.
- `ConsultationResponse` — one row per submitted response, uniquely
  keyed on `(consultation_id, user_id)`.
- `ConsultationAnswer` — one row per (response × question), uniquely
  keyed on `(response_id, question_id)`.
- `ConsultationOutcome` — one row per consultation (unique on
  `consultation_id`) with summary + decisions + next steps.

The single-package trio (`repository.go`, `service.go`, `handler.go`)
holds the lifecycle logic, question validation, response
one-per-user enforcement, and the analytics rollup.

**Frozen after publish.** Questions can only be created / edited /
deleted while `status = DRAFT`. Once `PUBLISHED`, the form is read-only
so early responders and late responders answer the same questions.

**Community is a label, not a gate.** Consultations may carry a
`community_id`, but response submission does **not** require the
responder to be a member of that community. Any verified user can
respond. This is a deliberate departure from issues and petitions
(which require primary-community match for creation and membership for
interaction) — consultation input is more valuable when it's broad,
and organizations often want cross-community perspectives.

### Notifications — DBNotifier

`internal/notifications/DBNotifier` INSERTs directly into the shared
`notifications` table (schema owned by community-service, same
shared-DB pattern as `audit_logs`). Emit sites:

- `consultation.published` → notification to every org member.
- `consultation.closed` → notification to every responder.
- `consultation.outcome_published` → notification to every responder
  with a deep link to the outcome section.

If services move to isolated databases later, `DBNotifier.Emit` becomes
a NATS publish or an HTTP call.

## Environment

- `DATABASE_URL`
- `JWT_SECRET`
- `PORT` / `ORGANIZATION_SERVICE_PORT` (default `3003`)
