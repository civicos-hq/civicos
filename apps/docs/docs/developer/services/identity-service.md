---
id: identity-service
title: Identity Service
sidebar_position: 2
---

# Identity Service

Port `:3001`. Owns everything about **who** — users, sessions,
applications for elevated roles, and the platform's moderation
infrastructure.

## Responsibilities

- **Auth** — register, login, refresh with family rotation, logout,
  email verification, forgot / reset password.
- **Profile** — `GET /me`, `PATCH /me`, `DELETE /me` (soft delete with
  PII anonymization).
- **Community membership** — join, set active community, change primary
  community (30-day cooldown).
- **Applications** — the queue where citizens apply to be
  representatives or organizations. Admins review.
- **Content flags** — citizens report content; moderators resolve. Also
  carries **campaign concerns**, which share the table but not the rules —
  see below.
- **Audit log** — the immutable trail of admin actions across every
  service. The schema is owned here; three services INSERT to it.
- **Admin metrics** — platform-wide snapshot + per-community drill-down.
- **User administration** — list, change role, ban, unban.

## Package layout

```
services/identity-service/
├── cmd/server/main.go                     # DI + Gin bootstrap
├── internal/
│   ├── adminmetrics/                      # /admin/metrics + /admin/communities/:id/stats
│   ├── applications/                      # Rep + org applications and admin review
│   ├── audit/                             # Shared Auditor writer (imported by other services too)
│   ├── auditlogs/                         # Read surface for admins
│   ├── auth/                              # Register, login, refresh, JWT signing
│   ├── domain/models.go                   # GORM models + enums
│   ├── flags/                             # Content flag queue and moderator resolution
│   │   └── eligibility.go                 # Who may raise a campaign concern
│   ├── middleware/                        # JWTAuth, RequireVerified, RequireRole
│   └── users/                             # Admin user administration
├── migrations/                            # Optional — AutoMigrate handles most of it
└── pkg/
    ├── config/
    ├── database/
    ├── mailer/                            # SMTP + console mailers
    └── response/
```

## Key domain concepts

### Refresh token family rotation (OWASP)

- Every `POST /auth/refresh` **consumes** the presented token (marks
  `consumed_at`) and issues a fresh one in the **same family**
  (`family_id` is stable).
- Presenting an already-consumed token = **replay = theft**. The
  service revokes every row where `family_id` matches, forcing the
  legitimate user and the attacker to sign in again.
- The raw token is 32 bytes of `crypto/rand` hex — never stored. Only
  `SHA256(raw)` in `token_hash`. Leaking the DB can't hijack live
  sessions.

### Application approval flow

- Citizen signs up with `requestedAccountType: REPRESENTATIVE` or
  `ORGANIZATION`.
- Their `User` row starts at `approvalStatus: PENDING`; the citizen
  can act as a citizen while pending.
- An admin lists the queue: `GET /admin/applications?status=PENDING`.
- Admin approves / rejects / requests changes: `PATCH
/admin/applications/{kind}/{id}`.
- On approval, the citizen's `role` and `approvalStatus` are updated
  and a notification fires.
- Every review is written to the `application_review_events` table
  and to the audit log.

### Primary vs. active community

- **Primary** community is the citizen's home constituency — where they
  can _create_ issues, petitions, and rep profiles.
- **Active** community is what they're currently viewing/acting in for
  signatures, comments, and upvotes.
- Both are stored on the `User` row. First join sets both.
- `PATCH /auth/me/primary-community` enforces a **30-day cooldown**.
  Returns `429` with `nextEligibleAt` if you try earlier.

### Ban and self-delete enforcement

- `bannedAt` and `deletedAt` on `User`. The JWT middleware in every
  service blocks writes from either.
- Refresh path returns `403` with `ACCOUNT_BANNED` or `ACCOUNT_DELETED`
  so the client can clear state and stop retrying.
- Self-delete anonymizes name and email, revokes every refresh token,
  and marks `deleted_at`. Content the user authored stays in place
  with a placeholder author name.

## Environment

Required:

- `DATABASE_URL` — the shared Postgres instance.
- `JWT_SECRET` — 32+ chars.
- Ports — `PORT` / `IDENTITY_SERVICE_PORT` (default `3001`).

Optional but strongly recommended:

- `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` /
  `SMTP_FROM` — enables real email. Without these, the mailer prints
  to stdout (development mode).
- `APP_URL` — used in verification and reset email links.

## Campaign concerns

`contentType: CAMPAIGN` reuses the flags table and the moderator queue, but
almost none of the moderation rules. The differences are all downstream of one
fact: a campaign is reviewed by an admin before it publishes and locked
afterwards (`campaigns/service.go` permits edits only in `DRAFT` and
`NEEDS_CHANGES`), so a concern is never "this should not have been approved".
It is a claim about conduct after approval — spending, progress, silence —
which CivicOS has no way to verify, because donations settle straight to the
organization's own bank account.

**Who may file one** — `flags/eligibility.go`. Two groups, and no others:

- someone with a **SETTLED** donation to that campaign (PENDING does not
  count; anyone can open a checkout and walk away), or
- someone whose community is in the **same LGA**. Same state, different LGA
  does not qualify — state-wide is too coarse to mean local knowledge.

Everyone else gets `403 NOT_ELIGIBLE_TO_REPORT`. This is the first place in
the product where a community tag acts as a **gate** rather than an audience
label, and it is confined to this one decision deliberately. The reason is
asymmetry: an open report button on a fundraiser is a lever a rival
organization or a political opponent would be glad to have, and unlike a
comment flag it points at money that has already moved and cannot be
recovered.

A campaign with no public page returns `404 CAMPAIGN_NOT_FOUND`, matching
`organization-service`'s `publicStatuses`, so the endpoint cannot be used to
probe for draft or rejected campaigns by id.

**Separate vocabularies.** A `CAMPAIGN` takes only `FUNDS_MISUSE`,
`WORK_NOT_DONE`, `MISREPRESENTED`, `NO_UPDATES`, `OTHER`; everything else
takes only the moderation reasons. Enforced both ways by `validReasonFor`.
This is not tidiness — it stops a funding concern arriving in the queue
labelled `SPAM`, where a moderator would triage it as a nuisance post rather
than a claim about money. `description` is required, minimum 20 characters.

**Nothing auto-acts.** `HIDDEN` is rejected for campaign flags with
`USE_PAUSE_INSTEAD`, and `/flags/direct-hide` refuses them outright. There is
no such thing as hiding a campaign; the real action is **pause**, which stops
money moving, and it stays a separate deliberate act on the campaign with its
own reason code and audit trail. If resolving a flag could pause a fundraiser,
a coordinated set of reports would become a way to shut down a rival — which
is the specific outcome this design refuses to make possible.

The admin surface is `apps/admin` → **Campaign concerns**, grouped by campaign
and ordered by how many _distinct_ people raised one. Corroboration between
unconnected observers is most of the signal available when the platform cannot
check the books itself.
