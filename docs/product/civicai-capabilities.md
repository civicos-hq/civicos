# CivicAI — Full Capabilities Catalog

**Author:** Gino
**Purpose:** Long-range vision for what CivicAI should eventually do. Companion to
[`civicai-plan.md`](./civicai-plan.md), which tracks the near-term build.

This document is intentionally aspirational — it captures the full surface area
we could grow into, not what ships this month. Star ratings (⭐) are the
author's gut-feel priority. The current build plan cherry-picks from this list;
see `civicai-plan.md` for what's actually being built.

## Categories

### 1. AI Citizen Assistant ⭐⭐⭐⭐⭐

Conversational Gemini experience inside CivicOS. Users ask questions like:

- How do I report a broken streetlight?
- What consultations are currently open?
- Show me issues near me.
- Which representative covers my district?
- How do I create a community?

Instead of searching manually.

### 2. AI Organization Assistant ⭐⭐⭐⭐⭐

Administrators ask:

- How many issues were reported this month?
- Which communities are least active?
- What announcements had the highest engagement?
- Which consultation had the highest participation?
- Show unresolved issues older than 30 days.

Instead of opening dashboards.

### 3. Consultation Intelligence ⭐⭐⭐⭐⭐

One of my favorites. Instead of reading 8,000 responses, AI generates:

- Executive summary
- Common themes
- Sentiment
- Most requested changes
- Frequently mentioned concerns
- Recommended actions

**Example**

```
Consultation Summary
────────────────────
Participants        1,824
Main Concerns       Road safety · Parking fees · Traffic congestion
Overall Sentiment   72% Positive
Recommendation      Delay implementation by 30 days.
```

### 4. Issue Intelligence ⭐⭐⭐⭐⭐

AI groups similar reports. Instead of 2,000 pothole reports, it says:
_"These reports refer to the same road."_

Capabilities:

- Detect duplicates
- Categorize issues
- Prioritize urgency
- Estimate severity
- Suggest department

### 5. Smart Search ⭐⭐⭐⭐⭐

Instead of keyword search, users ask:

- Show all education consultations.
- Find road complaints in Abuja.
- Find all announcements about scholarships.

Natural language search.

### 6. Announcement Generator ⭐⭐⭐⭐☆

Admins type:

> The university library will close on Friday for maintenance.

AI creates:

- Professional announcement
- Social media version
- SMS version
- Email version

### 7. Consultation Generator ⭐⭐⭐⭐☆

Admin types:

> We want feedback on new parking rules.

AI generates:

- Title
- Background
- Objectives
- Questions
- Closing statement

Huge time saver.

### 8. Issue Response Assistant ⭐⭐⭐⭐☆

Instead of writing "Thank you for your report…" from scratch, AI drafts
responses. Human approves.

### 9. Meeting Summary ⭐⭐⭐⭐☆

Upload meeting transcript. AI produces:

- Summary
- Decisions
- Action items
- Deadlines

### 10. Representative Assistant ⭐⭐⭐⭐☆

Representatives ask:

> What are the biggest concerns in my constituency?

AI answers with statistics — roads, waste, water, etc.

### 11. Community Insights ⭐⭐⭐⭐⭐

Weekly. AI automatically generates:

```
This Week
─────────
342 new members
28 issues resolved
Top discussion         Waste management
Most active community  Gwarinpa
```

### 12. Smart Notifications ⭐⭐⭐⭐☆

Instead of blasting everyone, AI determines who should receive this —
communities, districts, interest groups.

### 13. Translation ⭐⭐⭐⭐☆

Nigeria alone has hundreds of languages. AI translates between:

- English · Hausa · Igbo · Yoruba · French (eventually)

### 14. Accessibility Assistant ⭐⭐⭐⭐☆

AI simplifies long government policy → plain English, or _"explain this like
I'm 16."_

### 15. AI Moderation ⭐⭐⭐⭐☆

Detect spam, hate speech, abuse, duplicates — **before** publishing.

### 16. Draft Policy Generator ⭐⭐⭐☆☆

Government says: _"Need policy for waste disposal."_ AI drafts. Human edits.

### 17. Analytics Narrator ⭐⭐⭐⭐⭐

Instead of charts, AI explains:

- Citizen engagement increased by 18%.
- Most participation came from students.
- Road infrastructure generated the highest number of reports.
- Response time improved by 12%.

### 18. AI Knowledge Base ⭐⭐⭐⭐⭐

Train AI on org documents, policies, FAQs, regulations, meeting notes. Users
then ask _"What's the school's dress code?"_ and get an instant answer.

### 19. Workflow Assistant ⭐⭐⭐⭐☆

AI reminds admins:

- This consultation closes tomorrow.
- These issues need review.
- Representative hasn't responded.

### 20. CivicOS Copilot ⭐⭐⭐⭐⭐

The eventual endgame. Instead of clicking buttons, users say:

- Create a consultation for students about cafeteria services. → Done.
- Publish this announcement next Monday. → Done.
- Show unresolved issues in my community. → Done.
- Send reminders to everyone who hasn't completed the consultation. → Done.

## Suggested V1

Don't build all of these now. Launch with five capabilities:

1. **AI Chat Assistant** — primary conversational interface for citizens and administrators.
2. **Smart Search** — natural language search across CivicOS.
3. **Consultation Summaries** — automatically summarize responses after a consultation closes.
4. **Announcement Generator** — help administrators write clear, consistent communications.
5. **Analytics Narrator** — turn charts and metrics into plain-language insights.

Achievable, immediately useful, and demonstrates the value of AI without trying
to automate everything.

## Architecture

CivicAI shouldn't be "a chatbot page." It should be an AI platform every CivicOS
service can call:

```
Citizen ──┐
Admin ────┤
Rep ──────┘
     │
     ▼
CivicAI Gateway
     │
─────────────────
│ Chat
│ Search
│ Summarization
│ Recommendations
│ Generation
│ Translation
│ Moderation
│ Analytics
─────────────────
     │
     ▼
LLM Providers
(OpenAI · Anthropic · Gemini · local models)
```

Each CivicOS service sends requests to CivicAI through a well-defined API, and
CivicAI returns intelligence, not business logic.

## How this maps to the current build

|           # | Capability                                       | Status                                                                     |
| ----------: | :----------------------------------------------- | :------------------------------------------------------------------------- |
|           4 | Issue Intelligence (categorize, prioritize, tag) | ✅ Shipped (dedupe still TODO)                                             |
|           3 | Consultation Intelligence                        | ✅ Shipped — `summarize` now supports petitions, issues, and consultations |
|           6 | Announcement Generator                           | ✅ Shipped (multi-channel variants still TODO)                             |
|          11 | Community Insights                               | ✅ Shipped                                                                 |
|          17 | Analytics Narrator                               | ✅ Shipped — admin Overview tile hits `GET /v1/ai/narrate-metrics`         |
|           8 | Issue Response Assistant                         | 🕒 Recommended next                                                        |
| 1, 2, 5, 18 | Chat, Search, Knowledge Base, Copilot            | ⏭️ V1 targets — see next-steps section of `civicai-plan.md`                |

All other rows are on the aspirational roadmap.
