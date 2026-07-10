---
id: roadmap
title: Roadmap
sidebar_position: 4
---

# Roadmap

Honest snapshot of what's shipped, what's next, and what's aspirational.
Updated as the platform evolves — if a feature moves categories, this
page moves with it.

Last reviewed: **2026-07**.

---

## Shipped (MVP)

Everything in this section is live in the production codebase today
and documented in the User + Developer Guides.

### Identity and access

- Email + password registration with bcrypt cost 12
- Email verification via one-time link
- JWT access tokens + OWASP refresh-token family rotation
- Forgot / reset password
- Full self-service account deletion (soft delete + PII anonymization)
- Representative and organization applications with admin review

### Communities

- All 36 states + FCT and 774 LGAs of Nigeria seeded
- Primary vs. active community
- 30-day cooldown on primary-community changes

### Issues

- Report an issue with title, description, category, location, up to
  5 photos
- OPEN → UNDER_REVIEW → IN_PROGRESS → RESOLVED → CLOSED status lifecycle
- Upvoting (one per account)
- Comment threads with official-response tagging for reps / orgs / admins
- Content flagging (5 reasons) with moderator resolution and hidden
  placeholders

### Petitions

- Create with title, description, goal, deadline, photos
- Signing (one per account) with community-membership requirement
- Milestone notifications at 25 %, 50 %, 100 %
- Comment threads

### Representatives

- Per-community rep profiles with bio, party, contact info
- Follow / unfollow
- Comment thread (official responses fan out to every follower)
- Public **response rate** metric

### Organizations

- Registry with `NATIONAL` / `STATE` / `LGA` / `COMMUNITY` jurisdiction
- Internal roles (OWNER / ADMIN / STAFF)
- Public announcements (DRAFT → PUBLISHED → ARCHIVED)
- Projects with status, budget in kobo, community link
- Issue assignments — orgs take responsibility for citizen reports
- Progress updates on assigned issues and projects
- Verified-badge trust signal (audit-logged separately)

### Notifications

- In-app notification list with unread badge
- Real-time delivery via Server-Sent Events
- Notification types: issue update, petition update, rep response,
  system, community update

### Discover + search

- Global search across issues, petitions, representatives
- Discover feed tiered by geographic proximity
  (COMMUNITY → LGA → STATE → COUNTRY)

### Consultations

- Create structured feedback asks (DRAFT → PUBLISHED → CLOSED lifecycle)
- Question builder with 5 types: short text, long text, single choice, multi choice, yes/no
- Verified-user response submission with one-per-user enforcement
- Per-question analytics rollup (option counts + text samples)
- Outcome publishing — the "close the loop" primitive
- Notification fan-out on publish reaches org members plus the target community's members (deduplicated); responders are also notified on close + outcome-published
- **Citizen-facing UI**: browse open consultations, fill and submit responses
- **Org-owner UI**: create + question builder + publish/close + analytics + outcome publisher

### Announcements, Projects, Assignments — org-owner UI

- Announcements: dashboard tab + create + edit + publish/archive/delete + notify org members on publish
- Projects: dashboard tab + create + edit + status transitions + delete + budget in naira/kobo
- Issue assignments: dashboard tab + "Take responsibility" flow on the citizen issue page + inline status control + drop
- Progress updates: post from issue detail (assigned orgs) or project detail (org admins), visible on the issue and project pages

### Moderation infrastructure

- Content flags with 5 reason categories
- Moderator queue with hide / dismiss actions
- Direct-hide admin shortcut
- Shared audit log across all four services

### Admin tooling

- Platform metrics snapshot
- Per-community stats drill-down
- User administration (list, role change, ban / unban)
- Application review queue

### Platform

- Four Go microservices behind an API gateway
- Per-action rate limiting via Redis
- Interactive Swagger UI at `/docs`
- This documentation site

---

## Next (planned, not started)

Features we intend to build in the next phase. Not in the codebase yet
— treat as commitments, not promises.

- **Drag-to-reorder for consultation questions** — currently reorder is
  positional-integer edits only; a proper drag UI is on the near list.
- **Cover image upload for consultations** — API supports a URL; the UI
  form doesn't collect it yet.
- **Public browse UIs for announcements, projects, and assignments** —
  org-owner surfaces shipped this cycle; the citizen-facing browse pages
  are next.
- **Wider notification channels** — email digests, optional SMS.
- **Full-text search** — replace the current `ILIKE` sweep with
  Postgres `pg_trgm` or Meilisearch as the dataset grows.
- **CI check for OpenAPI mirror sync** — enforce that
  `docs/api/openapi-*.yaml` and the embedded gateway copies stay in
  sync.
- **Uploads on durable storage** — move from local disk to S3-compatible
  object storage before scaling out.

---

## Later (aspirational)

Ideas that fit the mission but haven't been scoped in detail. Some
require significant infrastructure or new engineering primitives.

- **CivicAI** — LLM assistance for summarizing long consultation
  threads, drafting official responses, and improving search recall.
  Assist-not-replace.
- **Plugin architecture** — a way for organizations to extend the
  platform without forking. Non-trivial security surface; will only
  happen with a clear sandbox model.
- **Multi-country deployment** — CivicOS is Nigeria-first by design.
  Second-country deploys require rethinking the primary-community
  cooldown and the state/LGA seed data.
- **Richer analytics** — engagement metrics per representative and
  organization, exposed publicly. Needs privacy review.
- **Managed CivicOS Cloud** — a hosted deployment path for
  organizations that don't want to run their own. Not committed to.

---

## Not on the roadmap

Things we're deliberately not building.

- **Algorithmic engagement feed.** The Discover feed is time-ordered
  and tier-labelled — no engagement optimization.
- **Anonymous participation.** Accountability requires verified
  identities.
- **Paid verification.** Verification is either automatic (email) or
  admin-reviewed (rep / org applications). No money path.
- **Ads.** The platform is open source; there is no revenue model that
  depends on user attention.

---

## Roadmap changes

Category changes (Next → Shipped, Later → Next) happen when:

- **Shipped** — merged to main and released.
- **Next** — an issue exists with a design doc and a rough timeline.
- **Later** — someone has articulated the shape but no work is
  underway.

If you want to move something up, open an issue on GitHub with the
rationale.
