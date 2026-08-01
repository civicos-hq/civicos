---
id: civicai
title: CivicAI for staff
sidebar_position: 3
---

# CivicAI for staff

CivicAI is the AI intelligence layer of CivicOS. This page covers the
four AI surfaces available to staff — representatives, government
admins, NGO staff, moderators, and platform admins.

Every CivicAI output is a **suggestion** or **draft**. Nothing publishes,
assigns, or decides on its own. Every AI-generated string in the UI
carries an **AI-generated · review** badge so you always know what came
from CivicAI and what came from a human.

## The four surfaces

| Surface                    | Where                                                  | For                                                         |
| -------------------------- | ------------------------------------------------------ | ----------------------------------------------------------- |
| **Summarize discussion**   | Petition, issue, and consultation detail pages         | Reading a long thread quickly.                              |
| **Draft with AI**          | New announcement page (`/org/:id/announcements/new`)   | Turning a brief into a publishable announcement.            |
| **Community Intelligence** | Top of the organization dashboard (`/org/:id`)         | Understanding what's happening across your whole community. |
| **Analytics Narrator**     | Top of the admin Overview page (`/admin`, admins only) | Reading the platform's health without staring at charts.    |

## Summarize discussion

When a petition, issue, or consultation has real conversation attached,
you'll see a **Summarize discussion** button. Click it and CivicAI reads
the entire thread (up to 200 comments, oldest first) and produces:

- A one-paragraph **TL;DR** you can read in ten seconds.
- **Themes** — the recurring topics people are raising.
- A **sentiment mix** — positive · neutral · negative across the whole
  thread.
- **Top asks** — what the community is specifically requesting, in their
  own words where possible.
- **Recommended actions** — 2–4 imperative-mood suggestions ordered by
  impact.
- **Official responders** — who from your side has already engaged.

The summary is cached for **30 minutes**. Clicking again within that
window returns the same summary instantly — you'll see a `Cached` badge
instead of `Fresh`. This keeps costs down and lets you revisit the
summary while you draft a response.

**Consultations are supported too.** When you summarize a consultation,
CivicAI reads the structured responses (Q&A) rather than free-form
comments, but the output is the same shape.

**Who sees the button.** Representatives, government admins, NGO staff,
moderators, and platform admins. Citizens don't see it — this is
decision-support for people who act on the discussion.

## Draft with AI

On the **New announcement** page, above the title/body form, you'll find
a **Draft with CivicAI** panel. Type a rough brief (at least 20
characters — the more concrete, the better), pick a tone and audience,
and click **Draft with AI**.

CivicAI returns:

- A short **title** (max 80 characters, no emoji).
- A **body** — 2 to 5 short paragraphs in the tone you asked for.
- A few **key points** — scannable one-liners covering what changed,
  what you're doing, and what the reader should do or expect next.

Click **Apply to form** to fill the title + body fields on the actual
announcement form. Both fields get an **AI · edit to remove** chip that
disappears the moment you edit — so you always know at a glance what's
still AI-generated and what you've made your own.

**Tones.**

| Tone           | Use when                                             |
| -------------- | ---------------------------------------------------- |
| **Formal**     | Policy changes, official notices, regulatory items.  |
| **Friendly**   | Community events, milestones, weekly updates.        |
| **Urgent**     | Time-sensitive service disruptions, safety notices.  |
| **Empathetic** | Announcements about outages, apologies, disruptions. |

**Audiences.**

- **General public** — assume the reader may not know your org.
- **Members only** — assume they already do.

**Ground rules the model follows.** CivicAI is instructed never to
invent facts, dates, quotes, phone numbers, or statistics you didn't
put in the brief. If you don't mention a repair date, the draft won't
promise one. The output is only as specific as your brief.

Drafts are **not cached** — a second click on the same brief gives you
a fresh variation to compare.

## Community Intelligence

On your organization dashboard (`/org/:id`), the **Community Intelligence**
tile lets you generate an aggregate digest of what's happening across
your whole community — not one thread, but the story the numbers +
discussions tell together.

Click **Generate insights** and CivicAI pulls recent issues, petitions,
and their comments from your community (capped at 25 issues, 15
petitions, 12 comments per thread), analyzes them, and returns:

- Activity counts — issues, petitions, comments in scope.
- A one-paragraph **TL;DR** — the top story right now.
- Recurring **themes** across all items.
- An overall **sentiment mix**.
- **Top asks** across the whole corpus.
- **Recommended actions** — 3–5 next steps ordered by impact.

The tile picks the community from your active community (or your primary
community, if you haven't switched). If you don't have a community set,
the tile prompts you to pick one first.

Cached for **1 hour** — because community-wide themes don't flip in
an afternoon.

## Analytics Narrator (platform admins)

At the top of the admin Overview page (`/admin` on the admin console),
platform admins have a **CivicAI narrator** tile. Click **Narrate the
numbers** and CivicAI reads the same metrics shown below (users,
communities, issues, petitions, moderation queue) and returns:

- A one-sentence **headline** — the top story in the numbers.
- A **narrative** — 2–3 short paragraphs weaving specific numbers into
  sentences.
- **Highlights** — 3–5 standout figures worth surfacing.
- **Trends** — where things are improving, plateauing, or slipping.
- **Recommendations** — 2–4 next steps grounded in what the numbers say.

Cached for **15 minutes**. This is the surface that turns "here are 40
counters" into "here's what's happening on the platform right now."

## Campaign surfaces (Community Funding)

Five surfaces for organizations running a fundraising campaign. All of
them are drafts you edit, and none of them is available yet in the
interface — the endpoints exist, the buttons do not.

| Surface                | For                                                            |
| ---------------------- | -------------------------------------------------------------- |
| **Classify campaign**  | Suggesting a category and whether this is really an emergency. |
| **Draft campaign**     | Turning a brief into a title, description and milestone plan.  |
| **Campaign impact**    | A plain summary of where the campaign stands.                  |
| **Draft donor update** | Writing to the people who gave you money.                      |
| **Draft final report** | The closing account, when the work is done.                    |

Three things about these are worth knowing, because money makes them
different from the announcement drafting above.

**CivicAI knows what you reported, not what you spent.** Donations settle
straight into your bank account — CivicOS never holds the money and
cannot check any of it. So every draft describes your spending as what
your organization _reports_, never as verified fact. It will not write
that something has been confirmed, because nothing has been.

**It will tell you what is missing.** Campaign drafts come with
`warnings`: costings a reviewer will ask for, claims your published
record does not support, a goal that does not match the described work.
Hearing that from a draft is cheaper than hearing it from a rejection
two days later. The impact summary similarly reports **gaps** — money
raised but not accounted for, milestones with no progress, long
silences. That is deliberate. A summary that flattered a campaign which
had gone quiet would be worth nothing to a donor deciding whether to
give again.

**It never does the arithmetic on unexplained money.** In a final report
draft, the amount still unaccounted for is calculated from the ledger,
not written by the AI. That figure is frozen onto your public campaign
page the moment you file, and it does not change afterwards even if you
publish more spending later — so it is not a number anyone should be
generating.

## What CivicAI does _not_ do

- **It doesn't auto-publish.** Every draft, summary, insight, and
  narration requires a human click before anything ships.
- **It doesn't route or assign.** Assignment still runs through the
  normal community + organization flow.
- **It doesn't send messages.** No notifications, no emails, nothing on
  behalf of the org.
- **It doesn't remember previous chats.** Each call is independent — no
  conversation history, no personalization, no user profile.
- **It doesn't decide anything about money.** No campaign is approved,
  rejected, paused, or prioritised by CivicAI. Platform admins have an
  advisory tool that flags campaigns worth opening first in a review
  queue; it cites the evidence behind every observation, offers the
  innocent explanation alongside it, and changes nothing. A person makes
  every decision.

## Trust and audit

Every AI response includes the **model name** and the **timestamp** it
was generated at. Every UI surface that renders AI output shows an
**AI-generated · review** badge until a human has explicitly edited or
approved the content.

Full audit persistence (`ai_generations` table with prompt hash, tokens,
latency per call) is on the roadmap.

## Rate limits

CivicAI endpoints share the same "Standard authoring" rate-limit budget
as the org authoring endpoints. If you hit the limit while iterating on
a draft, you'll see the standard rate-limit toast — wait a minute and
try again.

## Costs and provider

CivicAI runs on Google's `gemini-flash-latest` by default (an evergreen
alias so the platform always uses the most current fast tier). Model
selection is a deploy-time config, not a per-request choice — see the
[developer guide](/developer/services/civicai-service) for details.

## Coming next

Roadmap features you'll see later:

- **AI-drafted responses** on issues — same pattern as announcement
  drafting, but on every discussion thread.
- **Multi-channel announcements** — SMS / social / email variants
  alongside the main body.
- **Multilingual** — Yoruba, Igbo, Hausa, Pidgin.
- **Duplicate detection** at report time — reduce noise before it
  reaches your inbox.
