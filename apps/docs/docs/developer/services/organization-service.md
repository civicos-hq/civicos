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
  announcements, projects, assignments, and consultations _inside_ an org.

Once an org exists, content-authorship writes (create / edit / delete /
publish) are gated **strictly** by org role — `PLATFORM_ADMIN` does
not bypass. Attribution matters: an announcement or consultation
should read as coming from the org, not from the platform.

Three authorization helpers on `organizations.Service`:

- **`CanAdmin(orgID, userID, userRole)`** — strict. Requires OWNER or
  ADMIN membership in the target org. Used for all content-authorship
  writes.
- **`CanClose(orgID, userID, userRole)`** — the emergency lever.
  Allows the same OWNER/ADMIN plus PLATFORM_ADMIN. Wired to
  `consultations.close` and `announcements.archive` so the platform
  can freeze problematic content without joining the org first.
- **`CanReadInternal(orgID, userID, userRole)`** — admin reads. Allows
  any org member (including STAFF) plus PLATFORM_ADMIN. Used for
  response lists, analytics, and org-only announcement drafts.

If PLATFORM_ADMIN legitimately needs to intervene beyond
close/archive/read, the correct path is to join the org first — that
action is audit-logged, so the intervention is properly attributed.

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

## Community Funding — donations and settlement

Organizations raise money against a campaign. CivicOS is **not** the
merchant of record: each organization connects its own Paystack
sub-account, and Paystack settles directly to them. The platform fee is
expressed as the sub-account's transaction charge, so CivicOS never
takes custody of donor money.

That decision has a consequence worth stating plainly: because funds
never pass through CivicOS, there is no milestone-gated withdrawal to
build. **Transparency here is disclosure, not control.** The levers that
remain are publication (pausing a campaign stops new donations) and
reporting.

### Money is always integer minor units

`int64` kobo, never a float, end to end. The platform fee is held in
integer **basis points** (`PLATFORM_FEE_BPS=250` is 2.5%) and the split
is computed with integer division that floors — rounding in the
organization's favour, never the platform's. Each donation stores the
rate that applied **at the time it was made**, so changing the platform
rate later cannot retroactively rewrite what a past donor was told.

### The webhook is the only thing that settles a donation

`POST /v1/webhooks/paystack` authenticates by HMAC SHA-512 over the raw
body and must not sit behind auth middleware. The browser redirect after
checkout is a hint, never proof — a donor can close the tab or type the
return URL themselves.

Two properties hold it together:

- **Replays are a no-op.** Deliveries dedupe on the provider's event id,
  and a campaign's total is recomputed by `SUM` over settled rows rather
  than incremented, so repeats converge instead of inflating.
- **A signature proves origin, not content.** An event whose amount or
  currency disagrees with the opened intent is recorded, noted, and left
  unsettled for a human.

### Reconciliation — the safety net under the webhook

Making the webhook the only settlement path also makes it a single point
of failure. A delivery that never arrives — or arrives while the
endpoint is down — leaves money that genuinely moved sitting `PENDING`
forever. Paystack retries, gives up, and nothing notices.

`internal/donations/reconcile.go` is what notices. It re-reads
transactions from Paystack and runs two sweeps with deliberately
different powers:

| Sweep          | Power                                                   | Why                                                                                        |
| -------------- | ------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| `PENDING` rows | **Repaired** — settled through the ordinary settle path | Idempotent, and `PENDING` means nobody was told anything yet                               |
| `SETTLED` rows | **Reported on only** — never mutated                    | Rewriting banked money would destroy the audit trail that makes the discrepancy explicable |

A repaired row is still reported as drift (`RECOVERED_MISSED_WEBHOOK`).
The money ended up in the right place, but the delivery path failed and
that needs explaining.

A grace period (default 20 minutes) protects donors who are still on the
checkout page — writing off an in-flight payment is worse than waiting.
An admin can pass an explicit `pendingGraceMinutes: 0` to mean "check
everything now", and that is honoured rather than treated as unset.

Drift kinds, roughly in order of seriousness: `SUBACCOUNT_MISMATCH`
(money reached the wrong organization), `AMOUNT_MISMATCH` (public totals
are wrong), `SETTLED_HERE_BUT_NOT_AT_PROVIDER`,
`PENDING_WITH_MISMATCHED_DETAILS`, `RECOVERED_MISSED_WEBHOOK`,
`PROVIDER_UNREACHABLE`.

`PROVIDER_UNREACHABLE` deserves a note: it is not evidence of a problem,
but it means the row was **not checked**. Counting it as clean would let
an unreachable Paystack look like a healthy ledger.

Runs happen two ways:

- On a timer, every `RECONCILE_INTERVAL_MINUTES` (0 disables). Findings
  go to the operator log, one line per drift.
- On demand via `POST /v1/admin/donations/reconcile`, `PLATFORM_ADMIN`
  only — a run can move a donation to `SETTLED`, which changes a
  campaign's public total. Every run is audited against its own run id.

Findings are logged rather than stored in a table. That is a real
tradeoff — drift found overnight lives only in logs until someone looks
— taken because a new table here would put financial records under
`AutoMigrate`, which this repo has no migration tooling to evolve
safely. The on-demand endpoint returns the same report synchronously, so
an admin investigating a complaint never depends on log retention.

### Receipts

A settled donation emails the donor a receipt. It goes out on **fresh**
settlement only, from the shared `applyStatus` path, so it covers both
the webhook and a donation recovered by reconciliation — and a replayed
webhook, which stops at the already-settled check, cannot email anyone
twice.

The governing rule is that **sending a receipt must never affect whether
a donation settles**. Money moving is the fact; the email is a
description of it. An SMTP outage, a bounced address or a slow relay
must not roll back a settlement or fail a webhook (Paystack would just
retry it). Every mail failure is logged and swallowed.

The cost of that is a window: the process can settle a donation and die
before the mail goes out. `Donation.ReceiptSentAt` closes it —
reconciliation re-sends anything settled without a receipt, after a
short delay so it cannot race the inline send. A receipt is only ever
recorded after a send actually succeeded, so a failure stays retryable.

Two content decisions worth keeping:

- **It is not a tax receipt, and it says so.** CivicOS is not the
  merchant of record; the organization is the recipient of the gift.
  Implying the document could be filed for tax relief would be a claim
  made on another entity's behalf, and a donor acting on it could be
  misled into a filing they cannot support.
- **Amounts are shown to the kobo** — `₦62.50`, never `₦63`. A donor who
  adds up a rounded receipt gets a different answer than the ledger
  holds, and a receipt whose arithmetic does not reconcile is worse than
  no receipt at all.

Without `SMTP_HOST` the service falls back to a console mailer that
prints receipts to the log, so a dev environment still exercises the
whole path. In local development Mailpit catches them — read them at
`http://localhost:8025`.

### Reported spend — the transparency dashboard

Phase 4's answer to "where did my money go?".

Organizations publish spend records against the milestones donors were
shown before giving. Each entry is dated, itemised, attributed to a named
person, and optionally carries a receipt link.

**CivicOS cannot verify any of it.** Donations settle straight to the
organization's own Paystack sub-account, so the platform never holds the
money and cannot check a line of this against a bank statement. A spend
record is a _claim published under the organization's name_, not a fact
the platform attests to, and every surface that shows one says so.

That is a weaker guarantee than holding funds and releasing them against
milestones — it is the guarantee the merchant-of-record decision left in
place. The design leans into it: make the claim specific enough that the
people who paid can check it, even though we cannot enforce it.

Three decisions worth keeping:

- **Reporting more than was raised is allowed and surfaced, not clamped.**
  An organization may legitimately top up from other funds. Silently
  capping the figure would make the arithmetic on the page fail to add up,
  so `exceedsReceived` flags it instead.
- **There is no "remaining balance" figure.** CivicOS does not hold the
  money and cannot know an organization's actual balance — only what came
  through this platform and what they say they spent. A derived balance
  presented as fact would be an invented number.
- **Spend can still be reported while a campaign is PAUSED.** Pausing
  stops new donations; an organization under investigation is exactly who
  should be accounting for what it already took.

Amendments and withdrawals are audited with the figures before and after,
so a published claim cannot be quietly rewritten.

The citizen page shows amounts to the kobo — `₦62.50`, not `₦63` — using
a separate formatter from the rounded one used for progress bars. Three
rows that are supposed to sum must visibly sum.

### Funding updates

The evidence half of the same page. `ProgressUpdate` gained a
`CampaignID`, a title and attachment URLs, so a campaign feed reuses the
moderation and hide-filter machinery issues and projects already have.

One authorisation subtlety: the create route authorises against the
organization in the **path**, while `campaignId` arrives in the **body**.
The service verifies the campaign belongs to that organization — without
it, an admin of one org could publish updates onto another org's campaign
page, under its name, to its donors. A campaign owned by someone else
returns `404`, not `403`, so ids cannot be probed.

### Notifications and the realtime bridge

Six campaign notification types: `CAMPAIGN_APPROVED`, `DONATION_RECEIVED`,
`MILESTONE_COMPLETED`, `CAMPAIGN_UPDATE`, `FUNDING_GOAL_REACHED`,
`CAMPAIGN_COMPLETED`.

Who hears what is resolved once, in `internal/audience`, and shared by
campaigns, milestones, donations and progress:

- **Only approvals are announced.** `NEEDS_CHANGES` and `REJECTED` carry
  the reviewer's note, which is a private conversation between platform
  and organization — the same reasoning that keeps `reviewNote` out of the
  public DTO.
- **Donors are not told about other people's donations.** The org hears
  that money arrived; donors already have their receipt. Otherwise a busy
  campaign makes the notification tray unusable.
- **Anonymous donors are still notified.** Anonymity governs the public
  donor list, not whether someone hears what happened to their own money.

`community-service` owns the notifications schema; this service mirrors
the enum. Nothing structural keeps the two in sync, so a test reads the
canonical source and compares — it skips rather than fails when the
sibling checkout is absent.

**The realtime bridge.** The SSE hub lives in community-service, but this
service writes notification rows directly, so before Phase 4 nothing it
emitted pushed live — announcements and consultations included. Now:

1. organization-service writes the row (exactly once, as before),
2. publishes the committed row to NATS on `civicos.notifications.created`,
3. community-service subscribes and **only pushes to the hub** — it never
   writes.

A dead broker therefore costs realtime and nothing else: the notification
is still in the table and still appears on the next fetch. That fallback
is what allows core NATS with no persistence — the hub already documents
itself as lossy, with the REST list as the source of truth.

## Environment

- `DATABASE_URL`
- `JWT_SECRET`
- `PORT` / `ORGANIZATION_SERVICE_PORT` (default `3003`)

Community Funding (all optional — without a Paystack key the service
still starts and serves campaigns; donation endpoints return `503`):

- `PAYSTACK_SECRET_KEY` / `PAYSTACK_PUBLIC_KEY`
- `PLATFORM_FEE_BPS` — integer basis points, default `0`
- `DONATION_CALLBACK_URL` — where Paystack returns the donor
- `RECONCILE_INTERVAL_MINUTES` — default `60`, `0` disables the sweep
- `NATS_URL` — realtime notification bus. Optional: without it,
  notifications still persist and appear on the user's next fetch.
