---
id: managing-organizations
title: Managing organizations
sidebar_position: 1
---

# Managing organizations

An **organization** on CivicOS is a public body, agency, NGO, or utility
that citizens can hold to account. Once created, an org can post
announcements, run projects, and take responsibility for issues that
citizens have reported.

This guide is for organization owners and admins.

## How an organization comes to exist

Organizations are minted by **applying at signup and getting approved by
a platform admin**. There is no in-app "New organization" button —
neither citizens nor admins can create an org directly. The application
is the only door, and admin approval is the only key.

### Step 1 — apply at signup

On `/register`, choose **account type = Organization** and fill in the
organization block of the form:

| Field        | Notes                                                                                             |
| ------------ | ------------------------------------------------------------------------------------------------- |
| Name         | The org's public name                                                                             |
| Slug         | URL-friendly identifier, lowercase and hyphenated (e.g. `enugu-water-corp`) — must be unique      |
| Kind         | `GOVERNMENT`, `AGENCY`, `NGO`, `UTILITY`, or `OTHER`                                              |
| Jurisdiction | `NATIONAL`, `STATE`, `LGA`, or `COMMUNITY` — dictates the geographic scope of what you can act on |
| State / LGA  | Required for STATE / LGA jurisdictions                                                            |
| Description  | What the org does                                                                                 |
| Contact      | Public email, phone, website — how citizens reach you                                             |
| Logo         | Public logo URL                                                                                   |

Submitting creates an `OrganizationApplication` with status `PENDING`
and a user account that can log in but is limited to citizen actions
until approval lands.

### Step 2 — admin review

A platform admin sees your application in the admin console's
**Applications** queue. They can **approve**, **request changes**, or
**reject** with a note. When they approve, a single database transaction:

- Creates the public `organizations` row using your submitted details
- Adds you as the `OWNER` in `org_members`
- Sets your platform role to `NGO` (or `GOVERNMENT_ADMIN` for
  government kinds)
- Sends you an in-app + email notification

### Step 3 — you're the owner

After approval, your organization already exists — there is no separate
"Create org" step. Head to your org-owner surface:

- **My organization** in the sidebar (visible only to OWNER/ADMIN
  members of any organization), or directly at `/org/<your-org-id>`

From there you manage announcements, projects, consultations,
assignments, and members — the rest of this page walks through each.

### Fixing details after approval

Something wrong in the approved details? A platform admin can PATCH the
organization via the admin console. Members with OWNER or ADMIN role
can also edit the org's public-facing fields (name, description,
contact, logo) from the org-owner surface. The slug is fixed after
creation.

## The verified badge

Organizations start unverified. A platform admin can grant a **verified
badge** — a citizen-facing trust signal that says "this really is the
body they claim to be." The badge toggle writes a separate audit-log
entry (`org.verified` / `org.unverified`) so trust decisions are
reviewable.

If you believe your org qualifies, contact the platform admins with your
proof (registration certificate, staff directory, etc.).

## Your team

A utility or agency has many people doing different jobs, and CivicOS
records two separate things about each of them.

**Role** is what they may do on CivicOS:

- **OWNER** — full control, including members and payout details.
- **ADMIN** — commits the organization: publishes announcements, creates
  campaigns and consultations, accepts issue assignments, manages the team,
  edits the org.
- **STAFF** — does the work: moves an assignment along (RECEIVED →
  IN PROGRESS → COMPLETED) and posts progress updates on issues and
  projects. Cannot publish announcements, cannot run campaigns, cannot
  manage members, and cannot post a campaign update — that one reports to
  people who gave money.

The line between ADMIN and STAFF is **reporting on a commitment versus
making one**. A field officer saying "we're on site, the main is repaired"
is recording work the organization already took on. Deciding to take the
job on in the first place, or asking the public for money, is a commitment
the organization makes.

**Job title** is their actual job — "Head of Distribution", "Field
Officer". Free text, optional, and shown next to their name. A citizen
reading an update wants to know they're hearing from the Head of
Distribution; that is a different question from whether that person is
allowed to publish.

### Inviting someone

1. Go to **My organization** and scroll to **Team**.
2. Click **Invite someone**.
3. Enter their **email address**, an optional job title, and a role.

They get an email with a link. **They do not need a CivicOS account** —
the link opens a page showing who invited them and to what, and they can
create an account from there. The link is good for 14 days.

Invitations you have sent but nobody has accepted appear under **Team**,
so two admins don't invite the same person twice. If a link expires it
says so — send another, which replaces the old one and stops the previous
link working.

You can **withdraw** a pending invitation at any time.

:::note They must accept with the invited address
Accepting requires being signed in as the address the invitation was sent
to. A forwarded link does not let someone else in — organization admins
can publish in your name and run fundraising campaigns, so the invitation
grants access to the person you chose, not to whoever opens the message.

If someone signs in with a different address, the page tells them which
one to use.
:::

Platform admins can add existing accounts directly from the admin
console's organization page, which also shows each member's current
platform role.

Any platform role can be added, including representatives. A councillor
sitting on a water board's oversight committee is both an elected
representative and a member of that organization — the two are
independent, and being on a utility's team does not make someone an
elected official (or vice versa).

Member changes are audit-logged.

## Announcements

Announcements are the org's public voice — updates you want the
community to see in the feed and on your org page.

**To publish an announcement:**

1. Go to **Announcements → New announcement**.
2. Enter a **title** and **body**.
3. Either **save as draft** (only members see it) or **publish**
   immediately.

Announcements move through **DRAFT → PUBLISHED → ARCHIVED**. Publishing
and archiving both write to the audit log.

## Projects

Projects are the "here's what we're building" primitive — a rehab, a
programme, a rollout. They carry:

- Title + description
- Status: `PLANNED` / `ACTIVE` / `PAUSED` / `COMPLETED` / `CANCELLED`
- Start and expected-end dates
- Optional **budget** (in kobo — ₦1 = 100 kobo)
- Optional community link

Citizens see the project on your org page. Post
**[progress updates](#progress-updates)** to keep them informed as work
moves forward.

## Assignments — receiving issues

An **assignment** records that your org has taken responsibility for a
citizen-filed issue. Assignments work in two directions:

- **You claim an issue** — go to the issue page, click **Take
  responsibility**, add an optional note. The issue's assignments list
  now shows your org.
- **A platform admin routes an issue to you** — you'll see it in **Your
  organization → Assignments** with status `RECEIVED`.

Once assigned, move the status through `IN_PROGRESS` → `COMPLETED` (or
`REJECTED` with a reason). Every state change is visible to citizens.

Assignments are members-only reads — a curious user can't enumerate
another org's inbox, but the _list of orgs assigned to a given issue_ is
public. Citizens deserve to know who owns their report.

## Progress updates

Progress updates are the "respond publicly" primitive. They hang off
either an **assigned issue** or a **project**.

**To post an update:**

1. Go to the issue or project page.
2. Click **Post progress update**.
3. Write the body (2 characters minimum — usually a sentence or two).
4. Choose **public** (default) or **internal** (members only).

Public updates are readable by anyone. Internal notes are only visible
to org members.

**Who can post one:** any member, STAFF included. This is the operational
record of work in progress, and the person who actually did the work
should be able to say so.

Campaign updates are the exception. An update attached to a **campaign**
goes to everyone who donated and forms part of the spend-accountability
trail, so it stays with owners and admins.

## Running a fundraising campaign

Community Funding lets a verified organization raise money for a
specific piece of work. Before you start, one thing is worth being
clear about, because it shapes everything else:

**CivicOS never holds your money.** Donations settle straight into your
own bank account through a Paystack sub-account. The platform cannot
release funds, cannot hold them back, and cannot see what you spend.
What it provides is a public record — and it will hold you to it.

### Before you can take a single naira

Two things must be true:

1. Your organization is **verified**.
2. You have **connected a payout account** — bank and account number,
   confirmed with Paystack. You enter these once; CivicOS passes them
   to Paystack and does not store the account number.

Until both are done, campaigns can be drafted but not published.

### The lifecycle

1. **Draft.** Create the campaign with a title, summary, description,
   category, goal and location. Edit freely.
2. **Spend plan.** Add milestones — the stages the money will be spent
   in. Their totals cannot exceed the goal. You cannot submit for
   review without one.
3. **Submit for review.** A platform admin reads it.
4. **Approved, or sent back.** An admin either approves, rejects, or
   returns it with **needs changes** and a note saying what to fix.
5. **Publish.** Approval does not publish — you choose when.
6. **Funded / Completed.** Mark it complete when the work is done.
7. **File a final report.** The closing account.

**Once you publish, the content is locked.** Title, description and
goal can no longer be edited. A campaign is a promise people gave money
against, so it stops being a document you can revise.

### While it is live

- **Publish spending** as you go, itemised against your milestones. The
  campaign page shows what came in, what you say you spent, and what
  has not been accounted for.
- **Post updates** for the people who donated. Lead with what has been
  done, and say what has gone wrong or been delayed — donors find out
  eventually, and hearing it from you is better.
- CivicAI can draft both. See [CivicAI for organizations](./civicai.md).

Everything you publish here is a **claim by you**. CivicOS cannot check
it, and every page says so. That is not a slight on your organization —
it is the honest description of a platform that never touched the money.

### The final report

When the work is finished, file a closing account: what was achieved,
what the money went on, and what was not achieved.

If money you raised is still unaccounted for when you file, **the page
will say so and name the amount — permanently.** That figure is frozen
at the moment you file. Publishing more spending afterwards updates the
live accounting, but it does not change the verdict on the report. A
thin report cannot be fixed later, so file it when you can account for
the work.

### If a concern is raised

Donors, and people who live in the LGA your campaign serves, can raise
a concern about it. Concerns go to CivicOS staff, not to you, and
**nothing happens to your campaign automatically** — a person reads
every one. Only a platform admin can pause a campaign, which stops new
donations and takes the page down.

The most common cause of a concern is silence. An organization that
raises money and then publishes nothing for months looks the same from
outside as one that has taken the money. Regular updates are the
cheapest protection you have.

## Funding analytics

The **Analytics** tab on your dashboard shows how your campaigns are doing:
what you have raised, giving over time, repeat donors, average donation, and
a per-campaign breakdown.

Two rates sit below the chart, and each states its denominator on screen
because neither means anything without it:

- **Completion rate** — how many of the campaigns you published are finished.
- **Final reports filed** — how many of your _finished_ campaigns came with an
  account of the money. This is the one worth watching. It is, in effect, the
  question a donor deciding whether to give to you again is asking.

Three things to keep in mind reading them:

- **None of it is a balance.** Every figure is money that settled through
  CivicOS. Donations go straight to your own bank account, so CivicOS has no
  idea what you hold or have spent.
- **Donor counts are a floor.** Someone who gave while signed out cannot be
  linked to any other donation, so your unique and repeat donor numbers are a
  lower bound. The page tells you how many of your donations can be attributed.
- **"People helped" is not shown**, on purpose. Nothing in the record measures
  it, and a number you typed in would be a claim sitting among figures taken
  from a ledger.

---

Related:

- [Running consultations →](./running-consultations.md)
- [CivicAI for organizations →](./civicai.md)
- [Representative dashboard →](../representatives/dashboard.md)
