# CivicAI — Gemini Integration Plan

**Owner:** Gino
**Status:** In progress (Tier 1 complete — classify, summarize, draft)
**Last updated:** 2026-07-25

CivicAI is the AI intelligence layer for CivicOS. It uses Google Gemini to turn
raw civic input (issue reports, petition comments, consultation responses) into
structured, actionable output for the organizations that serve those communities.

This document is the source of truth for the AI roadmap. It supersedes any
inline notes elsewhere.

## Problem statement

Organizations receive thousands of unstructured community inputs — issue
reports, petition comments, consultation responses, representative Q&A — and
have no scalable way to process them. Manual triage is slow, biased, and
inconsistent. Existing platforms bolt on chatbots that never touch the
decision-making loop.

CivicAI is not a chatbot. It is a decision-support layer wired into the
workflows already in CivicOS: reporting, publishing, deciding.

## Design principles

1. **Human oversight is mandatory.** Every AI output is a _draft_ or
   _suggestion_. Nothing auto-publishes, auto-assigns, or auto-decides.
2. **AI outputs are provenance-tagged.** Every generated string carries
   `model`, `generated_at`, and the prompt hash, stored in Postgres for audit.
   The UI shows an "AI-generated · review before publishing" badge on every
   piece of AI output.
3. **Task-shaped endpoints.** Not one giant `/chat` — each endpoint does one
   thing (`/classify-issue`, `/summarize`, `/draft-announcement`).
4. **Fail open.** If Gemini is unreachable or over quota, the UX degrades
   gracefully — the user can still submit the issue, publish the announcement,
   etc. AI is additive, never load-bearing.
5. **Cache aggressively.** Summaries live in Redis with a short TTL; expensive
   community-intelligence digests live in Postgres with a longer TTL.
6. **Rate-limit hard.** Gemini calls cost real money — per-user + per-org
   budgets at the gateway.

## Architecture

New Go microservice: **`services/civicai-service/`**, port `3004`.

Follows the same layout as the other services (`cmd/server/main.go`, `internal/`,
`pkg/config`, `pkg/response`). Wraps `google.golang.org/genai` and exposes a
small, task-shaped HTTP API. The API gateway proxies `/api/v1/ai/*` to it with
JWT + per-user rate limits.

### Why a separate service (not embedded in community/organization)

- **One place** for prompts, API key, retries, audit, rate limits.
- **Product story** matches the pitch — "CivicAI" is a first-class thing.
- **Blast radius**: quota exhaustion or a bad prompt only degrades AI; the
  civic-loop services keep serving.
- **Trade-off**: needs to pull data (petition comments, consultation responses)
  from sibling services. Solve with in-cluster service-to-service HTTP using a
  shared internal secret — do not duplicate data.

### Data flow

```
citizen ─┐
         ├─► web app ─► gateway ─► civicai-service ─► Gemini API
org admin┘                     └─► community-service (for source data)
                               └─► Postgres (audit + cache of generated output)
                               └─► Redis (short-lived summary cache)
```

### Persistence

Table `ai_generations` in the civicai-service DB (or in a shared `ai` schema):

- `id`, `user_id`, `org_id?`, `resource_type`, `resource_id?`
- `endpoint` (e.g. `classify-issue`)
- `model`, `prompt_hash`, `latency_ms`, `input_tokens`, `output_tokens`
- `input_json`, `output_json`
- `created_at`

Purpose: audit + reproducibility + cost tracking + evaluation.

## Endpoints (v1)

| Method | Path                            | Purpose                                            | Auth       |
| ------ | ------------------------------- | -------------------------------------------------- | ---------- |
| POST   | `/api/v1/ai/classify-issue`     | Suggest category, severity, tags for a draft issue | user       |
| POST   | `/api/v1/ai/summarize`          | Summarize a petition / issue / consultation thread | org admin+ |
| POST   | `/api/v1/ai/draft-announcement` | Draft an accessible announcement from a brief      | org admin+ |
| GET    | `/api/v1/ai/community-insights` | Weekly themes + sentiment + recommended actions    | org admin+ |
| GET    | `/health`                       | Liveness                                           | public     |

All responses ride the standard envelope:

```json
{ "success": true, "data": { ... } }
```

## Features & delivery plan

### Tier 1 — hackathon must-ship

**1. Issue auto-classification** (Day 1 – 2)

- Endpoint: `POST /api/v1/ai/classify-issue`
- Input: `{ title, description, communityId? }`
- Output: `{ category, severity, suggestedTags[], confidence, model, generatedAt }`
- FE: debounced call from the "Report an Issue" modal → chip beneath the
  category dropdown ("AI suggests: **Roads** · click to accept"). Never
  overrides user; only suggests.
- Prompt strategy: system prompt pins the eight `IssueCategory` enum values;
  Gemini responds in strict JSON via response schema.

**2. Petition / consultation summary** (Day 2 – 3)

- Endpoint: `POST /api/v1/ai/summarize`
- Input: `{ resource: "petition"|"issue"|"consultation", id }`
- Output: `{ tldr, themes[], sentiment: {positive, neutral, negative}, topAsks[], recommendedActions[] }`
- Cached in Redis for 30 min per `(resource, id)`.
- FE: "Summarize discussion" button on petition + issue detail pages, visible
  to org admins + reps + platform admins.

**3. AI-assisted announcement drafting** (Day 3)

- Endpoint: `POST /api/v1/ai/draft-announcement`
- Input: `{ brief, tone: "formal"|"friendly"|"urgent", audience: "all"|"members", orgId }`
- Output: `{ title, body, plainLanguageVersion, model, generatedAt }`
- FE: "Draft with AI" panel in `OrgAnnouncementCreatePage` — editable before
  publish, badge stays on until human edits.

### Tier 2 — if time permits

**4. Community intelligence dashboard tile**

- Endpoint: `GET /api/v1/ai/community-insights?communityId=X&window=7d`
- Weekly digest: top themes across issues+petitions+consultations, sentiment
  trend, three recommended actions.
- Rendered on the org-owner dashboard (`/org/*`).

**5. Semantic Q&A over community content** — pgvector RAG, deferred.

### Tier 3 — post-hackathon roadmap

- Multilingual: Yoruba, Igbo, Hausa, Pidgin — Gemini can translate + classify.
- Predictive alerts: spike detection in negative sentiment for a topic.
- AI evaluation harness: golden-set prompts + regression tracking on each model bump.

## Model choice

Default: **`gemini-2.5-flash`** — fast, cheap, plenty capable for
classification + summarization. Reserve larger models for offline batch work
(community insights) if quality demands it.

Configure via env `GEMINI_MODEL`; fall back to `gemini-2.5-flash`.

## Env vars

Added to `.env.example`:

```bash
# ─── AI (CivicAI) ──────────────────────────────────────────────────
GEMINI_API_KEY=""
GEMINI_MODEL="gemini-2.5-flash"
CIVICAI_SERVICE_PORT=3004
CIVICAI_SERVICE_URL="http://localhost:3004"
```

## Delivery checklist

- [x] Plan doc committed
- [x] `services/civicai-service/` scaffolded
- [x] `POST /v1/ai/classify-issue` shipped
- [x] Gateway routes `/api/v1/ai/*`
- [x] `.env.example` + CLAUDE.md updated
- [x] FE classification chip on issue modal
- [x] `POST /v1/ai/summarize` + FE button (Redis-cached, 30min TTL)
- [x] `POST /v1/ai/draft-announcement` + FE panel
- [ ] `GET /v1/ai/community-insights` + dashboard tile
- [ ] Persistence table `ai_generations` (Day 2)
- [ ] OpenAPI spec `openapi-civicai.yaml` (Day 2)

## Open questions

- Should AI summaries be visible to citizens on public petition/issue detail
  pages, or org-admin-only? (Current default: org-admin only.)
- Do we log the full input text to `ai_generations`, or only the hash? Full
  text helps debugging but grows fast and includes PII.
- When we introduce multilingual, do we translate at ingest time (index the
  English translation for search) or at read time?
