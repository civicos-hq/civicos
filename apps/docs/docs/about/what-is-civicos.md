---
id: what-is-civicos
title: What is CivicOS?
sidebar_position: 2
---

# What is CivicOS?

CivicOS is an **open-source civic engagement platform** where citizens
report issues, sign petitions, follow their representatives, and see
what actually gets done — with everything on a public record scoped to
the community it affects.

Today (MVP release) it runs as one deployment covering Nigeria's 36
states + FCT and 774 LGAs.

## The three types of people who use it

- **Citizens** — the majority of users. Verified email accounts.
  They report issues in their primary community, sign petitions,
  comment, upvote, and follow representatives.
- **Representatives** — elected officials whose applications have been
  approved by a platform admin. Their public profile carries a
  response-rate metric that's visible to voters.
- **Organizations** — public bodies, agencies, NGOs, or utilities with
  approved applications. They take responsibility for issues, run
  projects, post announcements, and log progress updates.

Behind them, **platform admins** review applications, verify
organizations, and moderate content — every action they take goes on
the audit log.

## What makes it "community-scoped"

Every issue, petition, and representative belongs to a **community** —
a state + LGA pair. Citizens have a **primary community** (their home
constituency, where they can create things) and an **active community**
(what they're viewing/acting in for signatures, comments, upvotes).

Community scoping does two things:

1. **Keeps signal local.** People in Enugu East don't get their feed
   drowned out by chatter from Lagos Island.
2. **Prevents vote-stacking.** You can only create issues in your
   primary community, and you can only change primary community once
   every 30 days.

## What it is not

- **Not a social network.** No follows-for-follows, no algorithmic feed
  optimized for engagement. Content is community-scoped and time-ordered
  with an explicit tier system (community → LGA → state → country).
- **Not anonymous.** Comments, upvotes, signatures, and reports are all
  tied to a verified account. Anonymity works against accountability.
- **Not a replacement for the ballot box.** Elections still matter.
  CivicOS is what happens between them.
- **Not a hotline.** Reports are public. They go on the record, not
  into a private queue.

## The shape of the platform

CivicOS is four Go microservices behind a single API gateway, plus two
React apps (citizen web, admin console). The whole platform is described
declaratively in a single `render.yaml` — one Blueprint deploy brings
it all up.

If you're a developer curious about the internals, see the
[Developer Guide](/developer). Everything from the DI conventions to
the SSE notification hub is documented there.

## How to try it

Local development is fully covered in [Running locally](/developer/development/running-locally).
In production, CivicOS runs at `civicos.ng` (custom domain — pointed
at a Render deploy in front of managed Postgres + Redis).

## Where to next

- [Read the principles](./core-principles.md) — what CivicOS chooses
  to be and refuses to become.
- [See the roadmap](./roadmap.md) — what's next.
- [Create your account](../getting-started/create-account.md) and try it.
