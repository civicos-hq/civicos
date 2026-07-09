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
- **Content flags** — citizens report content; moderators resolve.
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
