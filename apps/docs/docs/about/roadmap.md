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
- Representative and organization applications with admin review — the
  only path to a rep or org record; no admin-side direct create
- Sybil resistance — Phase 1: disposable / temporary email domains
  rejected at registration and on email change (250-domain embedded
  blocklist). Phases 2 (SMS OTP as a gate on write actions) and 3
  (optional NIN verification) are deferred.

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

- Global search across eight kinds — issues, petitions, representatives,
  organizations, consultations, announcements, projects, and funding
  campaigns (drafts, non-published announcements, and campaigns with no
  public page are hidden)
- Discover feed tiered by geographic proximity
  (COMMUNITY → LGA → STATE → COUNTRY), covering issues, petitions,
  announcements, projects, consultations, and campaigns — announcements
  and un-scoped projects/consultations tier by the publishing org's
  state/lga; campaigns tier by their own state/lga first, because a
  campaign is often raised for a specific ward rather than wherever the
  organization is registered
- Campaign browse with category, emergency and verified-organization
  filters, and five sorts: recently added, ending soon, most funded,
  emergency first, near me

### Consultations

- Create structured feedback asks (DRAFT → PUBLISHED → CLOSED lifecycle)
- Question builder with 5 types: short text, long text, single choice, multi choice, yes/no
- Verified-user response submission with one-per-user enforcement
- Per-question analytics rollup (option counts + text samples)
- Drag-to-reorder questions in the org-side builder
- Cover image upload — shown on list cards and both detail pages
- Outcome publishing — the "close the loop" primitive
- Notification fan-out on publish reaches org members plus the target community's members (deduplicated); responders are also notified on close + outcome-published
- **Citizen-facing UI**: browse open consultations, fill and submit responses
- **Org-owner UI**: create + question builder + publish/close + analytics + outcome publisher

### Announcements, Projects, Assignments — org-owner UI

- Announcements: dashboard tab + create + edit + publish/archive/delete + notify org members on publish
- Projects: dashboard tab + create + edit + status transitions + delete + budget in naira/kobo
- Issue assignments: dashboard tab + "Take responsibility" flow on the citizen issue page + inline status control + drop
- Progress updates: post from issue detail (assigned orgs) or project detail (org admins), visible on the issue and project pages

### Community Funding

Six phases. The governing constraint: **CivicOS is not the merchant of
record.** Donations settle straight into each organization's own bank
account via a Paystack sub-account, so the platform never holds the
money. Everything below follows from that.

- Campaigns with a goal, a spend plan of milestones, and a
  DRAFT → PENDING_REVIEW → PUBLISHED → FUNDED → COMPLETED → REPORTED
  lifecycle, plus REJECTED / NEEDS_CHANGES / PAUSED
- Admin review before a campaign can publish; content is locked once it
  does
- Donations via Paystack — card, bank transfer, mobile money — with a
  disclosed 2.5% platform fee shown in naira before the donor confirms
- **Anonymous giving.** A donor's name never appears publicly if they
  ask; the ledger row still carries everything reconciliation needs
- Email receipts on settlement, explicitly not tax receipts — the
  organization received the gift, not CivicOS
- Public transparency dashboard: goal, raised, milestones, reported
  spend, and what has _not_ been accounted for
- Final reports, with the unaccounted shortfall **frozen at filing
  time** so a thin report cannot be made to look complete later
- Reconciliation sweep against Paystack with a drift queue for admins
- Payout account connection and funding-eligibility gating
- Citizen-raised **campaign concerns** — restricted to donors and people
  in the campaign's LGA, evidence required, and they never auto-act

Two figures from the product spec are **deliberately not shown**:
"funds withdrawn" and "remaining balance" (CivicOS never holds the
money, so it cannot know either), and "people helped" (nothing in the
record measures it, and a self-reported number would be a claim sitting
among figures taken from a ledger).

### CivicAI

Eleven task-shaped endpoints wrapping Google Gemini. Every output is a
draft or a suggestion; nothing auto-publishes, auto-assigns, or
auto-decides, and every response is provenance-tagged with the model
and timestamp.

- Civic engagement: classify issue, summarize discussion, draft
  announcement, community insights, analytics narrator
- Community Funding: draft campaign, draft donor update, draft
  completion report, assess campaign risk, classify campaign,
  summarize campaign impact
- **AI never gates money.** Risk assessment is admin-only, cites the
  evidence behind every signal alongside the innocent reading of that
  same evidence, and writes nothing — a person makes every decision

### Funding analytics

- Per-organization: funds raised, giving over time, repeat donors,
  average donation, per-campaign performance, completion and reporting
  rates
- Platform-wide: totals, campaigns by category and country, verified
  and funding-eligible organizations, emergency response times, review
  queue latency
- Caveats travel in the API payload, not only in the UI, because these
  figures get lifted into board packs without whatever qualification was
  on the screen

### Moderation infrastructure

- Content flags with 5 reason categories
- Moderator queue with hide / dismiss actions
- Direct-hide admin shortcut
- Shared audit log across every service that changes state

### Admin tooling

- Platform metrics snapshot
- Per-community stats drill-down
- User administration (list, role change, ban / unban)
- Application review queue

### Platform

- Four Go microservices behind an API gateway (identity, community,
  organization, CivicAI)
- Schema migrations run in-process at identity-service boot, behind a
  Postgres advisory lock so concurrent deploys cannot race
- Per-action rate limiting via Redis
- Interactive Swagger UI at `/docs`
- CI check that keeps the embedded gateway copies of the OpenAPI specs
  in lock-step with `docs/api/`
- This documentation site

---

## Next (planned, not started)

Features we intend to build in the next phase. Not in the codebase yet
— treat as commitments, not promises.

- **Citizen browse for standalone assignments** — announcements and
  projects landed as their own list pages plus in the Discover feed;
  assignments still only surface on the specific issue they're tied to.
- **Wider notification channels** — email digests, optional SMS.
- **Full-text search** — replace the current `ILIKE` sweep with
  Postgres `pg_trgm` or Meilisearch as the dataset grows.
- **Uploads on durable storage** — move from local disk to S3-compatible
  object storage before scaling out.
- **Crypto donations via LinkiSwap** — designed behind the same
  `PaymentProvider` port as Paystack, but **blocked**: no API spec yet,
  and it needs decisions on custody, volatility accounting and
  sanctions screening before any code is written.

---

## Later (aspirational)

Ideas that fit the mission but haven't been scoped in detail. Some
require significant infrastructure or new engineering primitives.

- **Plugin architecture** — a way for organizations to extend the
  platform without forking. Non-trivial security surface; will only
  happen with a clear sandbox model.
- **Multi-country deployment** — CivicOS is Nigeria-first by design.
  Second-country deploys require rethinking the primary-community
  cooldown and the state/LGA seed data.
- **Managed CivicOS Cloud** — a hosted deployment path for
  organizations that don't want to run their own. Not committed to.

---

## Not on the roadmap

Things we're deliberately not building.

- **Algorithmic engagement feed.** The Discover feed is time-ordered
  and tier-labelled — no engagement optimization.
- **Anonymous civic participation.** Issues, petitions, comments,
  upvotes and signatures all carry a real name; accountability requires
  verified identities. **Donating is the deliberate exception** — a
  donor may give without their name appearing publicly, because naming
  someone who gave ₦2,000 to a flood appeal holds nobody to account.
  See [Core Principles](./core-principles.md#7-anonymity-is-not-a-feature-here--with-one-exception).
- **Paid verification.** Verification is either automatic (email) or
  admin-reviewed (rep / org applications). Money cannot buy a badge or
  move an application up a queue. The 2.5% fee on donations is a
  transaction fee, not a price on standing.
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
