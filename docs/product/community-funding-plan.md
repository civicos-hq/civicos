# Community Funding — Implementation Plan

**Owner:** Gino
**Status:** Planning — nothing built yet
**Last updated:** 2026-07-30
**Spec:** `docs/product/CivicOS Community Funding Feature.pdf` (v1.0)

Community Funding lets **verified organizations** raise money for community
initiatives, with every campaign verified, transparent, trackable and auditable.
The spec's framing is the important constraint: _"The goal is not to allow anyone
to raise money. The goal is to enable trusted institutions to mobilize
communities around legitimate public-interest initiatives."_

This document is the implementation plan. Where it diverges from the PDF, the
divergence is stated and reasoned. It follows the same shape as
[`civicai-plan.md`](./civicai-plan.md).

---

## Read this first: money changes the risk profile

Everything CivicOS handles today is speech — reports, petitions, comments,
consultations. Wrong data is embarrassing but recoverable. Funding is the first
feature where a bug **loses someone's money**, and where the platform becomes a
target for fraud rather than just spam.

Three consequences that shape every decision below:

1. **We never hold funds we can't account for.** Balances are derived from an
   append-only ledger, never from a mutable `amount_raised` column that a
   double-webhook can inflate.
2. **Payments are somebody else's regulated problem.** CivicOS orchestrates and
   records; a PSP moves money. We do not touch card numbers — ever.
3. **Ship the transparency before the money.** A campaign that can't publish
   where funds went is worse than no campaign, because it looks legitimate.

---

## Design principles

1. **Ledger is the source of truth.** `Donation` and `Withdrawal` rows are
   append-only and immutable once settled. Campaign totals are computed from
   them (with a cached projection for reads). Never `UPDATE amount_raised += x`.
2. **Idempotency everywhere.** Every write that a PSP can retry carries an
   idempotency key with a unique index. Webhooks arrive twice; that must be a
   no-op, not a duplicate donation.
3. **Money is integer minor units.** `int64` kobo/cents, never float. Follows
   the existing `Project.BudgetKobo` precedent. Currency is explicit per
   campaign — the spec's examples are in £, the codebase is Nigeria-first, so
   this cannot be implicit.
4. **Verification gates publication, not creation.** Orgs draft freely; only a
   platform admin moves a campaign to `APPROVED`. Reuses the existing
   `ApprovalStatus` vocabulary from identity-service.
5. **AI never gates money.** Per `civicai-plan.md` principle 1, CivicAI output
   is a suggestion. Fraud detection produces a **risk score for a human
   reviewer**; it never auto-rejects or auto-approves a campaign.
6. **Public by default, private where it must be.** Campaign finances are
   public. Donor identity is opt-in — anonymous donation must be supported, and
   a donor's email is never public.
7. **Fail closed on money, open on everything else.** If the PSP is
   unreachable, donation fails loudly. If CivicAI is unreachable, the campaign
   still publishes.

---

## Architecture

### Where it lives: `organization-service`

Campaigns are owned by organizations, are authored by org members, and reuse
org membership/roles for authorization. Putting them anywhere else means
duplicating `OrgMember` role checks across a boundary.

`organization-service` already owns the exact adjacent primitives — `Project`
(with `BudgetKobo` and `CommunityID`), `Announcement`, `ProgressUpdate`,
`Consultation` — and already has the `audit` package the ledger needs.

New packages, following the existing `repository.go` → `service.go` →
`handler.go` DI layout:

```
services/organization-service/internal/
├── campaigns/     # CRUD, lifecycle, verification submission
├── donations/     # intents, PSP webhooks, ledger writes, receipts
├── withdrawals/   # payout requests, admin release, ledger writes
└── milestones/    # milestone + funding-update publishing
```

### Why not a new `funding-service`

Tempting, and the CivicAI precedent argues for separation. But: campaigns are
useless without org identity and membership, and a cross-service boundary for
_authorization on money_ is the worst place to add a network hop and an eventual
consistency window. Extract later if payment volume justifies it — the package
boundaries above are drawn so that extraction is mechanical.

### Payments architecture — DECIDED 2026-07-30

**Each organization connects its own Paystack sub-account. CivicOS is not the
merchant of record and never holds funds.**

Money flows payer → Paystack → the organization's settlement account. Four
consequences, and the third is the one that changes the product:

1. **Regulatory posture is much lighter.** CivicOS orchestrates and records;
   it never takes custody, so it is not holding client money.
2. **KYB shifts to Paystack.** The org's bank account is validated by Paystack
   at sub-account creation. CivicOS receives bank details once to create the
   sub-account and then stores **only the returned sub-account code** — never
   the account number.
3. **Milestone-gated withdrawal is no longer possible.** There is no
   CivicOS-held balance to release, so the platform cannot stop an
   organization spending funds that have already settled. The transparency
   model is therefore **disclosure, not control**: record every donation,
   publish it against the spend plan, require a final report. The one lever
   retained is pausing a campaign, which stops _new_ donations rather than
   recovering settled ones. This must be stated plainly on the public page —
   implying a control we do not have would be worse than having none.
4. **Refunds become the org's liability**, since the money has left Paystack's
   float. Policy still open (question 4).

A `PaymentProvider` port still wraps Paystack, so a second provider or the
eventual LinkiSwap crypto rail can be added without touching the ledger.
**LinkiSwap remains an unknown** — no API spec seen — and stays in Phase 6.

So the plan defines an interface and one concrete implementation:

```go
type PaymentProvider interface {
    CreateIntent(ctx, CreateIntentInput) (Intent, error)
    VerifyWebhook(ctx, rawBody []byte, signature string) (Event, error)
    // Payouts may be manual for v1 — see Phase 5.
}
```

Phase 3 ships **one fiat provider** (Paystack or Flutterwave — both are
Nigeria-native, support cards + bank transfer + mobile money, and settle in
NGN). Crypto is Phase 6 behind the same interface, and **is blocked on a
LinkiSwap API spec.** Rationale: crypto adds price-volatility accounting,
irreversible transfers, and sanctions-screening obligations. Shipping it
alongside a first fiat integration multiplies the ways money can go missing.

### Data model sketch

```go
// Campaign — the fundable thing. Belongs to a verified org.
type Campaign struct {
    ID             string
    OrganizationID string          // FK, org must be Verified to publish
    Title          string
    Slug           string          // uniqueIndex, for public URLs
    Description    string
    Category       CampaignCategory
    Status         CampaignStatus
    Currency       string          // ISO-4217, e.g. "NGN"
    GoalMinor      int64           // integer minor units
    // Derived projection, rebuilt from the ledger. Never incremented in place.
    RaisedMinor    int64
    DonorCount     int
    CommunityID    *string         // links a campaign to a community
    ProjectID      *string         // an existing Project can become fundable
    State, LGA     *string
    StartDate, EndDate *time.Time
    IsEmergency    bool            // emergency campaigns need admin approval
    RiskScore      *int            // CivicAI advisory, admin-visible only
    ApprovalStatus ApprovalStatus  // reuses identity-service vocabulary
    ReviewedByID   *string
    ReviewNote     *string
    CreatedByID    string
    CreatedAt, UpdatedAt time.Time
}

// Donation — append-only ledger entry. Immutable once SETTLED.
type Donation struct {
    ID             string
    CampaignID     string
    AmountMinor    int64
    Currency       string
    Status         DonationStatus  // PENDING → SETTLED | FAILED | REFUNDED
    Provider       string          // "paystack", ...
    ProviderRef    string          // uniqueIndex — the idempotency guarantee
    IdempotencyKey string          // uniqueIndex, client-supplied on intent
    DonorUserID    *string         // null for guest donations
    DonorName      *string         // display name, null if anonymous
    IsAnonymous    bool
    DonorEmail     *string         // json:"-", receipts only, never public
    NetMinor       int64           // after PSP fees — needed to reconcile
    FeeMinor       int64
    SettledAt      *time.Time
    CreatedAt      time.Time
}

// Withdrawal — money leaving, gated on published reporting.
type Withdrawal struct {
    ID, CampaignID     string
    AmountMinor        int64
    Status             WithdrawalStatus // REQUESTED → APPROVED → PAID | REJECTED
    MilestoneID        *string
    RequestedByID      string
    ApprovedByID       *string
    ProviderRef        *string
    Reason             string
    CreatedAt, UpdatedAt time.Time
}

// Milestone — the spend plan, and the withdrawal gate.
type Milestone struct {
    ID, CampaignID string
    Title          string
    TargetMinor    int64
    Status         MilestoneStatus // PLANNED → IN_PROGRESS → COMPLETED
    Ordering       int
    CompletedAt    *time.Time
}
```

`FundingUpdate` reuses the existing `ProgressUpdate` shape rather than inventing
a parallel one — same "respond publicly" primitive, pointed at a campaign.

**Lifecycle** (from the spec, with `REJECTED`/`PAUSED` added — the spec's
Governance section requires pausing but the lifecycle diagram omits it):

```
DRAFT → PENDING_REVIEW → APPROVED → PUBLISHED → FUNDED
                     ↘ NEEDS_CHANGES ↗            ↓
                     ↘ REJECTED              COMPLETED → REPORTED → ARCHIVED
PUBLISHED/FUNDED → PAUSED (admin, reversible)
```

---

## Delivery plan

Ordered so that **nothing can accept money until the accounting and
transparency around it exist.** Each phase is independently shippable.

### Phase 1 — Foundations, no money

- `Campaign` + `Milestone` models, `AutoMigrate`, DI wiring in `main.go`.
- Campaign CRUD for org members (`OWNER`/`ADMIN` create, `STAFF` read).
- Lifecycle transitions with a guard table — illegal transitions return
  `CAMPAIGN_INVALID_TRANSITION`, not a 500.
- Category enum, all 7 groups from the spec.
- `docs/api/openapi-organization.yaml` updated; embedded copy re-synced.
- **Exit:** an org can draft a campaign and submit for review. No public route,
  no payment code.

### Phase 2 — Verification & governance

- Admin review queue in `apps/admin`: approve / needs-changes / reject, with a
  required note on anything but approve.
- Reuse `ApprovalStatus`; extend org verification with the spec's required
  fields (registration number, country, official email, bank account
  verification) — likely on `Organization`, not `Campaign`.
- Pause/suspend with reason codes for the spec's five governance triggers.
- Every state change written through the existing `audit.Auditor`.
- **Exit:** campaigns can be approved and published publicly, read-only, with
  a goal and £0 raised. Transparency surface exists before money does.

### Phase 3 — Donations (one fiat provider)

The high-risk phase. Sequence inside it matters:

1. `PaymentProvider` port + Paystack/Flutterwave adapter.
2. `POST /campaigns/:id/donation-intents` — creates a `PENDING` donation,
   returns provider checkout details. Client-supplied idempotency key.
3. **Webhook receiver**: signature verification, replay-safe via unique
   `ProviderRef`, transitions `PENDING → SETTLED`, writes ledger.
4. Recompute projection (`RaisedMinor`, `DonorCount`) inside the same
   transaction as the ledger write.
5. ✅ Reconciliation job comparing our ledger against the PSP's settlement
   report, surfacing drift to admins. **Not optional** — this is how we find
   out we're wrong before a donor does.

   Shipped as `internal/donations/reconcile.go`: two sweeps, with `PENDING`
   rows repaired through the ordinary settle path and `SETTLED` rows only
   ever reported on. Runs on a timer (`RECONCILE_INTERVAL_MINUTES`) and on
   demand at `POST /v1/admin/donations/reconcile` (`PLATFORM_ADMIN`).

   Verified end-to-end against the real Paystack sandbox, including a
   genuinely **paid** transaction whose webhook was never delivered: the
   sweep found it, settled it, rebuilt the campaign projection from the
   ledger, recorded Paystack's own fee, and reported it as
   `RECOVERED_MISSED_WEBHOOK`. A second run changed nothing.

6. ✅ Receipts via the existing Mailpit-in-dev mail path.

   Emailed on fresh settlement (webhook _or_ reconciliation recovery),
   never on a replay. Mail failure can never block settlement, and
   `Donation.ReceiptSentAt` lets the reconciliation sweep re-send
   anything missed during an SMTP outage.

   **Delivered as email, not PDF.** A PDF attachment would need a
   rendering dependency, and the ₦ sign is not in the core PDF font set
   — it would need an embedded TTF to avoid rendering as garbage on the
   one document a donor keeps. The HTML email prints to PDF from any
   client, so the remaining gain is small. Worth revisiting if
   organizations ask for a formal attachment.

7. Guest + anonymous donation.

- **Exit:** a donor can fund a published campaign; totals are provably derived
  from the ledger; a replayed webhook changes nothing.

### Phase 4 — Transparency dashboard

- Public campaign page: goal, progress, funds received/withdrawn/remaining,
  milestones, timeline, photos, reports, completion %.
- Funding updates (text + images + documents + financial reports), reusing
  `ProgressUpdate`.
- Public donor list honouring anonymity flags.
- Notifications: extend `NotificationType` with `CAMPAIGN_APPROVED`,
  `DONATION_RECEIVED`, `MILESTONE_COMPLETED`, `CAMPAIGN_UPDATE`,
  `FUNDING_GOAL_REACHED`, `CAMPAIGN_COMPLETED`. Fan-out via the existing SSE
  hub in community-service.
- **Exit:** a donor can answer "where did my money go?" without asking anyone.

### Phase 5 — Settlement reconciliation & completion

Substantially reshaped by the merchant-of-record decision. There are no
CivicOS-issued payouts to build: Paystack settles directly to the org, so
this phase is about **accounting for money that has already moved**, not
releasing it.

- ✅ Reconcile the donation ledger against Paystack; surface drift to
  admins. Delivered in Phase 3 rather than deferred to here, because the
  gap it covers opens the moment the first donation is taken.
- Per-milestone reported spend, published by the org against the plan.
- Final report required before `ARCHIVED`.
- Public "reported vs unreported" state on the campaign page, since
  reporting is the only accountability lever that remains.
- **Optional spike:** Paystack sub-accounts carry a settlement schedule. If
  the API supports platform-triggered settlement on a `manual` schedule,
  milestone-gated release could be restored without CivicOS taking custody.
  Unverified against their docs — worth a spike before assuming it.
- **Exit:** a campaign can complete its full lifecycle, and its public page
  reconciles against Paystack.

### Phase 6 — Discovery, analytics, CivicAI, crypto

- Discovery: browse by category/country/org/community/verified, plus the spec's
  Emergency / Recently Added / Ending Soon / Most Funded / Near Me sorts. Reuse
  the existing proximity-tiered discover feed.
- Org + platform analytics per the spec's metric lists.
- **CivicAI surfaces**, all advisory, following `civicai-plan.md`:
  - `POST /ai/draft-campaign` — campaign assistant
  - `POST /ai/assess-campaign-risk` — fraud signals + risk score, **admin-only,
    never auto-acting**
  - `POST /ai/classify-campaign` — smart categorisation
  - `POST /ai/summarize-campaign-impact` — public progress summaries
  - `POST /ai/draft-donor-update` — donor communication
  - Completion reports
- **Crypto via LinkiSwap** — blocked on an API spec. Same `PaymentProvider`
  port. Needs a decision on custody, volatility accounting, and screening.

---

## Cross-cutting work

**Frontend.** `apps/web`: public campaign list + detail, donate flow, donor
receipts, org campaign management under the existing org dashboard.
`apps/admin`: verification queue, risk review, withdrawal release, reconciliation
drift. All new strings through i18n in **all five locales** (en/pcm/ha/ig/yo) —
the landing-page work showed how quickly English-only copy leaks into a
translated page.

**Gateway.** New proxy routes for `/api/v1/campaigns/*`, `/api/v1/donations/*`.
Rate limits: donation intents get their own tighter bucket. The **webhook route
must bypass JWT auth** (PSPs don't carry our tokens) and instead verify by
signature — call this out explicitly in review, it's an easy place to create an
unauthenticated write.

**Docs.** OpenAPI spec + Docusaurus pages for both the citizen/donor guide and
the org guide, per the repo rule that API docs and user docs move together.

**Testing.** Unit tests on money arithmetic and transition guards. E2E specs in
`apps/web/e2e/` for the donate flow against a PSP sandbox. Explicit tests for:
duplicate webhook, out-of-order webhook, refund after settle, withdrawal
exceeding balance.

---

## Explicit divergences from the spec

| Spec says                                   | Plan does                                               | Why                                                                                                                     |
| ------------------------------------------- | ------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| 11 payment methods incl. 5 crypto assets    | One fiat provider in Phase 3; crypto Phase 6            | Each rail is its own failure/compliance surface; parallel first integrations multiply risk                              |
| Amounts shown in £                          | Currency explicit per campaign, NGN-first               | Codebase is Nigeria-first (`kobo`, `NIGERIAN_STATES`); implicit currency on money is a defect                           |
| "Withdraw funds" as an org capability       | Withdrawal requires admin release + published milestone | Unilateral withdrawal defeats the transparency the spec is built on                                                     |
| Fraud detection listed as a CivicAI feature | Advisory risk score for human reviewers only            | `civicai-plan.md` principle 1; an AI that can block fundraising is an AI that can be wrong about someone's flood relief |
| Lifecycle diagram has no failure states     | Adds `REJECTED`, `NEEDS_CHANGES`, `PAUSED`              | The spec's own Governance section requires pausing                                                                      |

---

## Open questions

These need answers from product/legal before the phases they gate:

1. ~~**Who is the merchant of record?**~~ **DECIDED 2026-07-30: each
   organization connects its own PSP sub-account.** CivicOS never takes
   custody of funds — money flows payer → Paystack → the org's settlement
   account. See "Payments architecture" above for what this changes.
2. **What is LinkiSwap?** Internal product, third party, or intended? Is there
   an API spec? **Blocks Phase 6.**
3. ~~**Platform fee?**~~ **DECIDED: a percentage fee**, taken via the
   Paystack split. Stored in integer basis points, disclosed publicly on the
   campaign page — a donor should be able to see what reaches the
   organization before giving. The rate itself is still a product call.
4. **Refund policy.** Donor-initiated refunds, or PSP-dispute only? Affects
   whether `REFUNDED` needs partial amounts.
5. **Regulatory scope.** Fundraising by NGOs in Nigeria may require specific
   registration; "International Organization" as an org type implies
   cross-border flows and sanctions screening. Needs a legal read.
6. **Failed campaigns.** If a campaign misses its goal, are donations refunded,
   kept, or redirected? The spec doesn't say, and donors will ask.
7. **Do emergency campaigns get a fast path?** The spec has admins "approve
   emergency campaigns" — a flood needs hours, not days. What's the SLA, and
   does it relax any checks?

---

## What I'd build first

If the goal is a demonstrable slice rather than the whole module: **Phases 1, 2
and 4** — draft a campaign, get it verified, publish it publicly with milestones
and a transparency dashboard showing a £0 balance and a real spend plan.

That is the actual differentiator in the spec, it's the part no crowdfunding
platform does well, and it carries none of the payment risk. Then answer open
question 1 and do Phase 3 properly.
