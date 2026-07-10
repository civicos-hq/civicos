---
id: running-consultations
title: Running consultations
sidebar_position: 2
---

# Running consultations

Consultations are how an organization asks citizens a structured set of
questions before making a decision — curriculum changes, budget
priorities, policy reviews. Unlike an announcement (broadcast one-way)
or a project (an ongoing thing you're doing), a consultation is a
listening exercise with a defined start, end, and outcome.

This guide is for organization owners and admins running consultations.

## Who can create a consultation

Anyone with `OWNER` or `ADMIN` role in your organization. `STAFF` can
read drafts but cannot publish. This mirrors how announcements and
projects work.

**Platform admins cannot create consultations on your behalf.** The
platform is deliberately hands-off with authorship — a consultation
carries the org's voice, and citizens should be able to trust that the
questions came from your organization. If a platform admin ever needs
to intervene (compromised owner, urgent correction), they add
themselves to your organization as a member first — that action is
audit-logged, so the intervention is clearly attributed.

Platform admins **do** retain one narrow power: **closing** an active
consultation (see [emergency close](#emergency-close-platform-admin)
below).

## The lifecycle

```
DRAFT  ──▶  PUBLISHED  ──▶  CLOSED  ──▶  Outcome published
  │            │              │
  ▼            ▼              ▼
edit +       responses     no more
delete       collected     responses
```

- **DRAFT** — you're still building the questions. Edits, additions,
  deletions all allowed. Not visible to citizens.
- **PUBLISHED** — form is frozen. Questions can no longer be changed.
  Citizens can respond.
- **CLOSED** — no more responses accepted. You now review what came in.
- **Outcome published** — you post a summary of findings, decisions,
  and next steps. Responders are notified.

Every transition writes to the audit log.

## Creating a consultation

1. From your organization's admin panel, go to **Consultations → New
   consultation**.
2. Fill in:

| Field       | Notes                                                                                                                                                       |
| ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Title       | Short and specific. 5–200 characters                                                                                                                        |
| Summary     | One or two sentences shown on the list page. 10–500 characters                                                                                              |
| Description | The full context. Markdown supported                                                                                                                        |
| Community   | Optional. If set, publish also notifies that community's members (deduplicated against your org's members) — any verified user can still respond regardless |
| Closes at   | Optional. When responses stop. Server enforces this — no auto-close needed                                                                                  |

_The API accepts a cover image URL and `opensAt` timestamp, but the current UI form doesn't expose those yet — they're on the roadmap._

Save. The consultation is now a **DRAFT**.

## Building the question set

Add questions from the draft page. Five question types:

| Type          | When to use                               |
| ------------- | ----------------------------------------- |
| Short text    | Names, dates, single-sentence answers     |
| Long text     | Open-ended feedback — paragraph or two    |
| Single choice | Pick exactly one from a list              |
| Multi choice  | Pick any number from a list               |
| Yes / No      | Binary. Answer is stored as `YES` or `NO` |

Every question has:

- A **prompt** (3–500 chars).
- Optional **help text** shown below the prompt.
- A **required** flag. Required questions must be answered before submit.
- For choice types, **options** — at least two.

Order questions by editing them and setting a position number — a drag-to-reorder UI is on the roadmap.

**Once the consultation is published, questions are frozen.** Change
your mind mid-response window? You'll have to close, publish an outcome
noting the change, and create a new consultation. This is deliberate —
early responders and late responders must answer the same questions.

## Publishing

Click **Publish** on a draft. The system:

1. Requires at least one question. Refuses to publish an empty form.
2. Sets `status = PUBLISHED` and stamps `publishedAt`.
3. Sends a notification to every member of your organization — plus every
   member of the target community if you set one. Overlapping recipients
   (someone who's both in your org and the community) get a single
   notification, not two.
4. Writes `consultation.published` to the audit log.

## While the consultation is open

- Watch the **response count** on the list page — updates in real time.
- The **analytics** tab shows per-question aggregates that update as
  responses come in:
  - Choice / Yes-No questions: option counts (histogram).
  - Text questions: sample responses (up to 100 shown per question).
- The **responses** tab lets you scroll individual submissions.

## Closing the consultation

Click **Close** when the response window is over. The system:

1. Sets `status = CLOSED`, stamps `closedAt`.
2. Rejects any further response submissions.
3. Notifies every responder that the consultation closed and to watch
   for the outcome.
4. Writes `consultation.closed` to the audit log with the final response
   count.

If you set `closesAt` on the consultation, submissions after that time
are automatically rejected — but you still need to click Close to
transition the state. (Automatic state transitions are on the
[roadmap](../about/roadmap.md).)

### Emergency close (platform admin)

Platform admins can close a published consultation from any organization
without being a member of that org. This is the one platform-level
lever on consultation lifecycle — every other write (create, edit,
publish, delete, outcome) is org-only.

The intended use is: a consultation is causing real harm (leading
questions targeting a minority, doxxing in the description, coordinated
misinformation) and needs to be frozen while moderation runs. Every
platform-admin close writes to the audit log with the actor's identity,
so this power stays reviewable.

Ordinary "bad content" moderation goes through the flag system, not
through emergency-close.

## Publishing the outcome (the important part)

This is what makes CivicOS consultations different from a survey tool.

Once the consultation is closed, publish the outcome:

1. **Summary** — what did citizens tell you? (10+ chars, ideally a few
   sentences.)
2. **Decisions** — what has the organization decided as a result?
3. **Next steps** — what happens now, and when?

The outcome is:

- **Public** — anyone can read it, forever. Cite it in follow-up docs.
- **Linked from the consultation** so the record is one click away.
- **A notification event** — every responder gets pinged so they see the
  loop closed.
- **Rewritable** — post it again, and the new content overwrites. Useful
  if new information changes the decision. Every version writes to the
  audit log.

**The outcome is the trust primitive.** A consultation without an
outcome erodes credibility — citizens who took time to respond will
wonder if anyone read what they wrote.

## Deletion

Only DRAFT consultations can be deleted. Once published, the record
must survive so responders can revisit their submissions and outcomes
stay linked.

If a published consultation needs to be retracted, the pattern is:
close it, publish an outcome that says "we're not proceeding with this
question and here's why," and let the record stand.

## Rate limits

- **Creating consultations**: bucketed with issues, petitions, and
  representatives — 5 per hour per user.
- **Responses (from citizens)**: 10 per hour per user. Prevents scripted
  hammering; well above legitimate usage.

## Audit trail

Every action writes to the shared audit log:

- `consultation.created`
- `consultation.published`
- `consultation.closed`
- `consultation.deleted`
- `consultation.outcome_published`

Platform admins can query the log at `/api/v1/audit-logs`.

## Frequently asked

**Can I schedule a publish for later?**
Not yet. Scheduled auto-transitions are on the roadmap. For now, use a
`closesAt` timestamp and click Publish manually.

**Can I export responses?**
Not yet. CSV export is on the roadmap. For now, the analytics tab plus
the responses list is what you have.

**Can citizens edit or withdraw their responses?**
Not in the MVP. One-shot submit; the record is final. Edit / withdraw
is on the roadmap.

**Can I make responses anonymous?**
Not in the MVP. Every response is tied to a verified account. The
public outcome page shows aggregated numbers, not individual answers —
so the anonymity your citizens care about (nobody sees my personal
answer publicly) is already there.

**Can I target multiple communities?**
Not in the MVP. One community per consultation, or whole-org. Multi-
community targeting is on the roadmap.
