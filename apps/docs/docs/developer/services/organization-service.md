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
- **Members** — org-internal roles (`OWNER`, `ADMIN`, `STAFF`) plus an
  optional free-text `title` (the person's actual job, e.g. "Head of
  Distribution" — distinct from `role`, which is permissions),
  add / update role / remove.
- **Announcements** — DRAFT → PUBLISHED → ARCHIVED lifecycle. Global
  feed of published items plus per-org list.
- **Projects** — planned / active / paused / completed / cancelled,
  optional budget in kobo, optional community link.
- **Issue assignments** — records that an org has claimed an issue.
  Members-only reads on the org's inbox; public reads on the per-issue
  list.
- **Progress updates** — the "respond publicly" primitive. Hangs off an
  assigned issue, a project, or a campaign.
- **Representative offices** — an elected representative's constituency
  office, provisioned on demand as an organization of kind
  `REPRESENTATIVE_OFFICE`.
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

They are independent, and a person can hold both. A councillor sitting on
a water board's oversight committee is an elected `REPRESENTATIVE` **and**
an `ADMIN` of that org; neither implies the other. `POST
/organizations/:id/members` places no restriction on platform role for
exactly this reason.

Members are added by **email**, resolved server-side. Name and platform
role are read from `users` rather than accepted from the request — the
older shape took both from the client, which meant a UI had to already
know the target's UUID (so none was ever built) and meant the stored name
was whatever the caller typed. `ListMembers` re-reads both from `users` on
every call, because the stored columns are join-time snapshots that go
stale the moment someone is renamed or promoted.

Once an org exists, content-authorship writes (create / edit / delete /
publish) are gated **strictly** by org role — `PLATFORM_ADMIN` does
not bypass. Attribution matters: an announcement or consultation
should read as coming from the org, not from the platform.

Four authorization helpers on `organizations.Service`:

- **`CanAdmin(orgID, userID, userRole)`** — strict. Requires OWNER or
  ADMIN membership in the target org. Used for all content-authorship
  writes.
- **`CanOperate(orgID, userID)`** — the operational tier. Any member,
  STAFF included; no platform role bypasses it. Wired to
  `assignments.updateStatus` and to `progress.create` when the update
  hangs off an issue or a project.
- **`CanClose(orgID, userID, userRole)`** — the emergency lever.
  Allows the same OWNER/ADMIN plus PLATFORM_ADMIN. Wired to
  `consultations.close` and `announcements.archive` so the platform
  can freeze problematic content without joining the org first.
- **`CanReadInternal(orgID, userID, userRole)`** — admin reads. Allows
  any org member (including STAFF) plus PLATFORM_ADMIN. Used for
  response lists, analytics, and org-only announcement drafts.

#### Why `CanOperate` exists

Every operational write used to sit behind `CanAdmin`, which made STAFF
a read-only role that nobody doing actual work could use. A utility with
field officers had to make each of them an ADMIN — which also granted
publishing in the organization's name, creating fundraising campaigns,
and removing colleagues.

The line is **reporting on a commitment versus making one**. Marking a
repair IN_PROGRESS records work the org already took on; accepting the
assignment, publishing, or asking the public for money commits the org,
and those stay at `CanAdmin`.

`progress.create` is the one endpoint that gates on its target rather
than on a fixed role, because it serves three of them:

| Update attached to | Gate         |
| ------------------ | ------------ |
| Issue              | `CanOperate` |
| Project            | `CanOperate` |
| Campaign           | `CanAdmin`   |

A campaign update goes to everyone who donated and forms part of the
spend-accountability trail, so it belongs with the people answerable for
the campaign. The handler binds the request body _before_ authorising,
which is why that one reads out of order compared to its neighbours.

`CanOperate` deliberately refuses PLATFORM_ADMIN non-members, matching
`CanAdmin` rather than `CanReadInternal`: "the water board says it is
fixed" has to come from the water board.

If PLATFORM_ADMIN legitimately needs to intervene beyond
close/archive/read, the correct path is to join the org first — that
action is audit-logged, so the intervention is properly attributed.

### Representative offices

An elected representative gets campaigns, projects, consultations and
announcements by getting an **organization** — one of kind
`REPRESENTATIVE_OFFICE`, linked back to their profile by
`Organization.RepresentativeID`.

`POST /me/representative-office` is fetch-or-create: it requires a claimed
representative profile (`representatives.user_id` = caller), creates the
org with the caller as `OWNER`, and is idempotent. A partial unique index
on `representative_id` settles concurrent calls; the loser re-reads and
returns the winner's office rather than erroring.

The alternative was a polymorphic owner on campaigns, projects,
consultations and announcements. It was rejected because `CanAdmin` is the
single gate in front of all four: making the representative an OWNER
passes it, so **none of those modules changed**, and neither did the
sub-account, split, ledger, reconciliation, payout or admin review. A
second owner type would have meant editing code that moves donations.

Two consequences worth knowing:

- `validKind` deliberately excludes `REPRESENTATIVE_OFFICE`, so nobody can
  relabel an ordinary org as an official's office (acquiring the funding
  substitution below without a claim) or strip the kind off a real one.
- `FundingEligible` substitutes rather than waives. An elected office has
  no entry in a company register, so a **claimed representative profile**
  stands in for `RegistrationNumber`. Every other requirement —
  verification, country, official email, named human, bank verification,
  connected payout account — is identical to an NGO's.

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

## Public campaign browse — filters and sorts

`GET /campaigns` is the citizen-facing browse. Filters narrow the list;
**sorts reorder it without narrowing it**, which matters most for `NEAR_ME`.

`verified` and `country` filter on the **owning organization**, not the
campaign — a campaign has no verification state or country of its own. Both
are expressed as a subquery rather than a join so the result stays
`[]domain.Campaign` and every existing caller of `Repository.Find` is
unaffected.

Sorts live in `applySort` (`campaigns/repository.go`):

| Sort               | Ordering                                                                   |
| ------------------ | -------------------------------------------------------------------------- |
| `RECENT` (default) | `created_at DESC`                                                          |
| `ENDING_SOON`      | future deadlines first, soonest first; expired and undated sort to the END |
| `MOST_FUNDED`      | `raised_minor DESC`                                                        |
| `EMERGENCY`        | `is_emergency DESC`, then recency                                          |
| `NEAR_ME`          | same LGA, then same state, then the rest — using `nearState` / `nearLga`   |

Three things are deliberate:

- **`ENDING_SOON` hides nothing.** A campaign past its deadline is not
  "ending soon", but it may still be accepting money, so it sorts last rather
  than being filtered out.
- **`nearState` / `nearLga` are not the `state` / `lga` filters.** Near-me
  orders the whole country by closeness; reusing the filters would make it
  indistinguishable from filtering to one LGA. With no reference point it
  falls back to recency instead of erroring — the same choice the discover
  feed makes for an unknown kind.
- **Every ordering ends with a deterministic tiebreak.** Without one Postgres
  may return equal rows in any order, and a citizen paging through would see
  items repeat or vanish.

`MOST_FUNDED` is inherently self-reinforcing — the campaigns already carrying
money get the most visibility — which is why it is one option among several
rather than the default.

### A GORM trap worth knowing

`NEAR_ME` builds a `CASE` expression with the caller's state and LGA bound as
parameters. It is applied with `db.Clauses(clause.OrderBy{...})`, **not**
`db.Order(...)`: GORM's `Order` switches on the argument type and accepts only
`string` and `clause.OrderByColumn`. Handed a `clause.OrderBy` it falls
through the switch and **silently discards it** — the query runs with no
`ORDER BY`, returns rows in whatever order Postgres feels like, and looks like
a working sort until you read the generated SQL.

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

## Funding analytics

Two read-only endpoints in `internal/analytics/`:
`GET /organizations/:id/funding-analytics` (gated by `CanReadInternal` on the
owning org) and `GET /admin/funding-analytics` (`PLATFORM_ADMIN`). Nothing
writes, and nothing is cached — the queries are cheap aggregates and a stale
money figure is worse than a slightly slower page.

### What these numbers are not

Every money figure is what settled **through CivicOS**. It is not what an
organization holds, has spent, or has left; donations settle straight into its
own bank account and CivicOS has no view of it afterwards.

Two metrics the product spec asks for are **deliberately absent rather than
approximated**:

- **People helped / Beneficiaries reached.** Nothing in the schema records a
  beneficiary count. It could only be a number an organization typed in, which
  is a claim — and placing it beside figures derived from a ledger would lend
  it a precision it has not earned. Adding a self-reported field is a product
  decision, not a reporting one.
- **Funds withdrawn / Remaining balance.** CivicOS is not the merchant of
  record, so neither is knowable. Already recorded in the funding plan.

### Choices that would be easy to get wrong

- **`completionRate` divides by campaigns that EVER published**, tracked via
  `published_at IS NOT NULL` rather than `status = 'PUBLISHED'`. Using the
  current status would make the rate climb as work finishes, since a completed
  campaign leaves the PUBLISHED state.
- **`reportingRate`** is reported-over-completed — the share of finished work
  that came with an account of the money. The single most telling number here.
- **Donor counts are a floor.** A donation made while signed out carries no
  `donor_user_id`, so it cannot be tied to any other donation.
  `attributableDonations` is returned beside the counts; without it they read
  as totals. The `notes` array in every response says so in words.
- **Averages are over all settled donations**, attributable or not — an
  average does not need to know who gave.
- **Emergency response uses a median**, not a mean. One appeal that sat
  unfunded for a month would drag an average away from the typical experience.
- **`review.oldestWaitingHours`** exists because an average wait stays
  comfortable while one campaign sits for a fortnight.
- **Trend buckets include empty weeks**, via `generate_series`. A series that
  skipped them would let a chart draw a line straight through a silence.
- **Money is grouped by currency.** One entry in practice today, but summing
  across currencies would be a defect the moment a second appears.

### Where these surface

- `apps/admin` → **Funding analytics** (platform-wide, under Money).
- `apps/web` → the **Analytics** tab on the org dashboard (`/org/:id?tab=analytics`).

Both render the `notes` array rather than hard-coding their own copy, and both
show each rate's denominator on screen.

### The `notes` array

Every response carries its caveats in the payload, not only in the docs. These
figures get lifted into board packs and funding applications, where they
arrive without whatever qualification was on the screen — so the qualification
travels with the data.

### Who pays Paystack's fee

`bearer: "subaccount"` — the organization does.

This was `"account"` (CivicOS) until it was measured. Paystack's Nigerian
fee is 1.5% + ₦100, which exceeds the 2.5% platform cut on most donations:
in the sandbox a ₦10,000 gift left CivicOS with exactly **₦0.00**. Worse, when
the platform share was too small to cover the fee Paystack charged the
organization instead, so which party paid depended on the amount.

Verified with `fees_split`, which Paystack returns on any initialised
transaction — no payment has to complete:

| Donation | Organization | CivicOS | Paystack |
| -------- | ------------ | ------- | -------- |
| ₦1,000   | ₦960.00      | ₦25.00  | ₦15.00   |
| ₦2,500   | ₦2,300.00    | ₦62.50  | ₦137.50  |
| ₦10,000  | ₦9,500.00    | ₦250.00 | ₦250.00  |

The same probe settled a second question Paystack's own documentation
contradicts itself on: `percentage_charge` names the **platform's** cut, not
the sub-account's share. `percentage_charge: 2.5` returns
`fees_split.subaccount = 975000` on ₦10,000.

**`net_minor` is the split allocation, not what arrived.** Paystack allocates
`gross - platform_fee` to the sub-account and _then_ charges its fee to it, so
what the organization actually received is `net_minor - psp_fee_minor`. The
receipt and the donate form both compute it that way. Reconciliation compares
the **gross** only, so it would not catch a mistake here — which is why the
sandbox probe exists rather than a unit test asserting our own arithmetic back
to us.
