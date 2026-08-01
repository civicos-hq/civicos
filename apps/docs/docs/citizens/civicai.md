---
id: civicai
title: CivicAI for citizens
sidebar_position: 6
---

# CivicAI for citizens

CivicAI is the AI layer built into CivicOS. As a citizen, you don't
interact with a chatbot — CivicAI shows up quietly inside features you
already use, to help you file better issues and reduce the "did anyone
even read this?" friction that kills civic engagement.

Everything CivicAI does is a **suggestion**. It never edits your report,
overrides your category, or posts on your behalf.

## What you'll see today

### Category suggestions when reporting an issue

When you open **Report an issue** and start typing a title + description,
a small chip appears under the category dropdown a moment later:

```
✨ CivicAI suggests: Utilities  · apply
                    AI-generated · review
```

The chip is doing one thing — reading what you just wrote and guessing
which of the eight CivicOS categories fits best (Infrastructure, Health,
Education, Security, Environment, Utilities, Transport, Other). If you
click the chip, the dropdown updates to that category. If you'd rather
pick your own, ignore it — CivicAI never overrides a choice you've
already made.

**Why this exists.** Getting the category right means the right
organization gets your report faster. When categories are wrong, reports
sit in the wrong inbox and go unanswered. This is the quietest way to
help you help yourself.

**What it uses.** Only the title + description you're typing. Nothing
else about you is sent. The suggestion appears client-side; if the
suggestion service is down or slow, the form still works exactly the
same.

## What CivicAI does _not_ do

- **It doesn't file your issue.** You still click **Post issue**.
- **It doesn't rewrite your words.** The description you write is the
  description that gets posted.
- **It doesn't decide priority.** It surfaces a severity hint (LOW /
  MEDIUM / HIGH / CRITICAL) to help staff triage, but that hint is
  never shown to other citizens and never changes who sees your issue.
- **It doesn't route your issue to any specific person.** Assignment
  still runs through the normal community + organization flow.

## Trust and transparency

Every AI-generated output in CivicOS carries an **AI-generated · review**
tag. If you ever see a chip, badge, or paragraph without one, that's a
bug — please [flag it](./notifications.md) so we can fix it.

CivicAI runs on Google Gemini. The details of how it's built, what data
it reads, and what it doesn't read live in the
[developer guide](/developer/services/civicai-service).

## What's coming next

Features you may see later, in rough order of arrival:

- **Multilingual** — write your report in Yoruba, Igbo, Hausa, or
  Pidgin; staff still get an English summary.
- **Duplicate hint at report time** — "3 similar issues already exist on
  this street — sign these instead?"
- **Plain-language summaries** of long policy documents attached to
  consultations.

If you have ideas for what CivicAI should help with, tell us in the
[feedback channel](../about/roadmap.md).
