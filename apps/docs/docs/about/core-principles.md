---
id: core-principles
title: Core Principles
sidebar_position: 3
---

# Core Principles

These are the principles that guide what we build in CivicOS — and
just as importantly, what we refuse to build. They're not aspirational
statements. Every principle here is currently reflected in the code
you can read in the repo.

---

## 1. Open by default

CivicOS is open source. That's not a marketing choice — it's the only
way public civic infrastructure is worth trusting. Anyone can read the
moderation code, the audit-log guarantees, the rate-limit budgets, the
notification fan-out. The rules of the platform are visible in the
repo, not in a T&C document.

**In practice:** the whole codebase is on GitHub. There is no
proprietary "enterprise" version.

## 2. Community first, feed second

Everything in CivicOS is scoped to a community — a state + LGA pair —
before it's anything else. Issues, petitions, and representatives all
belong to one. Discovery across communities exists (the Discover feed),
but it's a secondary surface, not the primary one.

**In practice:** primary vs. active community, the 30-day primary-change
cooldown, membership-gated interactions on comments, upvotes, and
signatures.

## 3. Transparency builds trust — so log everything admin

Every administrative action across every service writes to a shared
audit log: role changes, bans, moderation decisions, application
approvals, organization verifications. The log is queryable by admins
and forms the public record of who used admin power for what.

**In practice:** the `audit_logs` table (schema in identity-service),
written to by three services, exposed via `GET /api/v1/audit-logs`.

## 4. Simplicity over completeness

We ship the smallest thing that solves the problem. Notifications live
inside community-service, not a separate service, because cross-entity
fan-out (a petition signature triggering a notification) stays
in-process. Search is a naive `ILIKE` sweep because the dataset doesn't
justify Meilisearch yet.

**In practice:** four services, not fifteen. `AutoMigrate` for
additive schema changes. Extraction happens when scale demands it,
not before.

## 5. Privacy by design

Sensitive fields never serialize. Password hashes, refresh-token hashes,
verification-token hashes carry `json:"-"`. Deletion soft-deletes the
row and anonymizes name + email, revoking every refresh token. Public
content the user authored survives with a placeholder author name — the
public record stays intact while PII is scrubbed.

**In practice:** `User.PasswordHash json:"-"`, `deletedAt` + PII
anonymization on self-delete, `[Removed by moderator]` placeholders on
hidden content.

## 6. Security is a responsibility, not a feature

Refresh tokens rotate on every use. Presenting a token whose
`consumed_at` is already set = replay = revoke the entire family. Rate
limits are tiered per action (`Strict` for auth, `Sign` for petitions,
etc.). Verification is required for anything write-heavy — no anonymous
comments, no throwaway signatures.

**In practice:** OWASP refresh-token family rotation, Redis-backed rate
limiter with per-action budgets, `RequireVerified` middleware, bcrypt
cost 12.

## 7. Anonymity is not a feature here

Every action is tied to a verified account. This is a deliberate
tradeoff — anonymous participation is a well-studied problem in civic
tech, and the answer for accountability platforms is: don't. Signatures,
upvotes, comments, and reports all carry a citizen's name.

**In practice:** verified-email gate on all write actions. The audit
trail is real names, not user IDs.

## 8. Errors have codes, not messages

Every failure returns `{success: false, code: "ERR_CODE", message: "…"}`.
The code is the API contract; the message is a localization hint. This
lets the frontend show translated errors in five languages without
brittle string matching.

**In practice:** `response.Error()` in every Go service, `code` fields
matching an enum, i18n in `apps/web` mapping codes → translated
messages.

## 9. UTC-only timestamps

Every timestamp stored in the database is UTC. Localization happens
only at render time. This eliminates a whole class of "why is this
petition dated next week?" bugs.

**In practice:** `time.Now().UTC()` in Go, timezone-aware formatting
in the React apps.

## 10. Ship what works, not what sounds impressive

CivicOS does not have AI features. It does not have a plugin
marketplace. It does not do consultations yet. The docs will not
pretend otherwise. When those features exist and are stable, they
get documented. Not before.

**In practice:** the [roadmap](./roadmap.md) is explicit about what's
built, what's next, and what's aspirational.

---

## What we refuse to build

Principles are as much about no as yes.

- **No dark patterns.** No engagement metrics as a design goal, no
  "streaks," no notifications-for-notifications-sake.
- **No shadow bans.** If content is hidden, the audit log says who
  hid it and why. The placeholder is public.
- **No pay-to-be-verified.** Verification is either "email verified"
  (automatic) or "org/rep application approved by a platform admin"
  (auditable). There is no money path.
- **No selling data.** The whole platform is open source. There's no
  private analytics pipeline.

## Where to next

- [See the roadmap](./roadmap.md) — what's next in this framework.
- [Start using CivicOS](../getting-started/create-account.md).
