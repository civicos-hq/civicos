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

## Members

Organizations have three internal roles:

- **OWNER** — full control, including adding admins and deleting the org.
- **ADMIN** — can post announcements, projects, assignments, progress
  updates, and manage members below the owner tier.
- **STAFF** — read-only member; can see internal drafts and org-only
  content but can't publish.

To add someone:

1. Go to **Your organization → Members**.
2. Click **Add member**.
3. Enter their user ID and role.

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

---

Related:

- [Running consultations →](./running-consultations.md)
- [Representative dashboard →](../representatives/dashboard.md)
