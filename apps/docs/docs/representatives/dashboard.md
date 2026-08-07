---
id: dashboard
title: Representative dashboard
sidebar_position: 1
---

# Representative dashboard

This guide is for elected representatives who have been approved on
CivicOS. Your dashboard is where you see what citizens in your
constituency are saying, respond publicly, and track how you're being
held to account.

## Getting your representative badge

Representatives don't self-onboard as reps directly. The flow is:

1. Sign up with **account type = Representative** (see [Create an
   account](../getting-started/create-account.md)).
2. Fill in your **application**: full name, title, position,
   constituency, community, and any proof URLs (ID card, official
   directory listing, etc.).
3. A platform admin reviews your application. Outcomes:
   - **Approved** — your rep profile goes live, citizens can follow you.
   - **Needs changes** — you get a note explaining what's missing; you
     can edit and resubmit.
   - **Rejected** — with a reason. If you disagree, contact platform admins.

While pending, you can still use the platform as a citizen.

## Your representative page

Once approved, you have a public representative page tied to your
constituency and community. It shows:

- Your title, position, party (optional), and photo.
- **Bio** — free-form.
- **Contact** — official email, phone, and website (all optional but
  strongly encouraged).
- **Follower count** — citizens who chose to follow you.
- **Response rate** — how often you post an **official response** on
  content you're tagged in. This is a public accountability metric.
- A **comment thread** where citizens post publicly on your page.

Keep your bio and contact info fresh — a stale page reads as
disengaged. Any admin in your community can also edit your profile.

## Announcements — speaking to your constituents

Everything else on this page is you _replying_ to something a citizen
started. Announcements are the one place you raise something yourself.

They appear on your public page, and **publishing notifies everyone
following you**.

**To post one:**

1. Open your own representative page. You will see an **Announcements**
   section with a **New announcement** button — it appears only on your
   own page.
2. Write a title and body, then **Save draft**. Saving does not publish
   anything and nobody is notified.
3. When you are ready, click **Publish to followers**.

**Three things to know:**

- **A draft is private.** Nobody sees it until you publish — not
  citizens, not other representatives, not admins.
- **Published words cannot be edited.** Your constituents were notified
  about specific words; quietly changing them afterwards would make a
  public statement into something nobody can rely on having read. If you
  need to correct something, archive it and publish a new one.
- **Archiving is not deleting.** It takes the announcement off your
  public page, but the record that you made the statement remains. Only
  an unpublished draft can be deleted outright.

**Where it goes.** A published announcement reaches your constituents
three ways: everyone following you is notified, it appears in the
**Discover feed** for your community, and it is findable in search.
Drafts and archived announcements appear in none of those.

**People can reply.** Each published announcement has its own thread, so
a constituent answering _this_ announcement is not posting into a
general discussion about something else. Your own replies there carry
the **official response** badge, the same as on issues and petitions.
Replies are moderated like any other comment, and your announcement can
be reported.

**Only you can post as you.** Not another representative, not a platform
admin. The profile is linked to your account when your application is
approved, and that link is what the check is against — so nobody can put
words in your mouth.

:::note Profile seeded by an admin?
If your profile was created by a platform admin rather than by your own
approved application, it starts **unclaimed** — it isn't tied to any
account yet, and publishing will tell you so. Ask a platform admin to link
it to your account. Until they do, you can't post announcements or open a
constituency office.
:::

## Your constituency office

Everything an organization can do on CivicOS, you can do too — through
your **constituency office**. Open **My office** in the sidebar and create
it once; it takes a click.

The office is the entity your constituents see behind what you publish
("Office of Senator Ada Okafor"), and it gives you:

- **Campaigns** — raise money for something your constituency needs.
  Donations settle to your own account, not to CivicOS.
- **Projects** — publish what you're delivering, with budgets and progress
  updates.
- **Consultations** — put a question to your constituents and collect
  structured responses.
- **Announcements from the office** — distinct from the personal
  announcements on your profile above.

### Before your office can take money

Creating the office lets you **draft**. It does not let you collect
anything. Your office starts unverified with no payout account, exactly
as a newly registered organization does. Before a campaign can accept a
single naira:

1. A platform admin verifies your office.
2. You connect a payout account.
3. Your campaign passes review.

One thing differs from an organization, and only one: an NGO proves it
exists by its registration number, and an elected office has no such
entry in any company register. In its place, your office must be linked
to your claimed representative profile — a platform admin has confirmed
that you are the person holding the seat. Every other requirement is
identical. Raising money from your own constituents is held to at least
the standard an NGO is held to, not a lower one.

## Reading what citizens are saying

Three feeds are relevant to you:

- **Your constituency feed** — issues and petitions in the community
  you're tied to. Sort by upvotes to see what's rising.
- **Comments on your page** — the public wall citizens post to.
- **Your notifications** — new comments, new followers, and issues that
  admins have specifically flagged for your attention.

## Responding officially

An **official response** is a comment posted by a representative,
government admin, NGO, moderator, or platform admin. The system labels
it so citizens can distinguish your voice from another citizen posting
on your page.

**How to respond:**

1. Open the issue, petition, or your rep page's comment thread.
2. Write a comment as you normally would.
3. Post it.

The comment is automatically flagged `isOfficialResponse: true` because
of your role — you don't need to toggle anything.

**When you post an official response on your rep page**, every follower
gets a notification. Ordinary comments from other citizens on your page
don't fan out — only official ones do.

## Answering issues

Citizens will file issues in your constituency. You can:

- **Comment** to acknowledge, ask for detail, or explain the situation.
- **Change status** — if you're also a `GOVERNMENT_ADMIN`, you can move
  the issue through the lifecycle (`OPEN` → `UNDER_REVIEW` → `IN_PROGRESS`
  → `RESOLVED` / `CLOSED`).
- **Ping the right org** — a platform admin or org admin can assign the
  issue to the body responsible for handling it.

## Answering petitions

Petitions are structured asks. Your options:

- **Comment publicly** to state your position — for, against, needs
  refinement, etc.
- **Post a milestone response** when a petition crosses 25 %, 50 %, or
  100 %. Citizens are already getting a milestone notification; your
  reply is the follow-up they're expecting.

## Response rate

Every representative has a **response rate** metric visible on their
public profile. It's the share of issues (in your community) that
received either an official comment from you or a public progress update
from an assigned organization.

:::warning Not yet calculated

This metric is **not currently computed**. The field exists and shows 0%
on every profile. Nothing in the platform updates it yet, so it should
not be read as a signal about any representative.

:::

## Etiquette

- **Answer, don't campaign.** Your rep page isn't a campaign channel —
  it's a public accountability venue. Save the campaigning for elsewhere.
- **Be specific.** "We're looking into it" ages badly. "Team dispatched
  to Junction 4 at 9 AM, ETA 45 minutes" ages well.
- **Never delete a hard question.** Comment on it. Deleted questions
  hurt trust more than any answer ever could.
- **Correct the record publicly.** If you were wrong about something,
  say so in a follow-up comment. The audit trail is public anyway.

---

Related: [Notifications →](../citizens/notifications.md)
