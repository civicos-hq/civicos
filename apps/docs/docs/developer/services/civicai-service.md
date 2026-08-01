---
id: civicai-service
title: CivicAI Service
sidebar_position: 5
---

# CivicAI Service

Port `:3004`. Wraps Google Gemini behind a small set of task-shaped HTTP
endpoints. Every other CivicOS service can call CivicAI; CivicAI never
calls anything except upstream data services and the Gemini API.

:::note "Why a separate service"

The other option was embedding Gemini calls in community-service and
organization-service. We chose a dedicated service so prompts, the API
key, retries, rate limits, and the audit story all live in one place.
Blast-radius argument: if Gemini quota is exhausted or a prompt
regresses, only CivicAI degrades — the civic-loop services keep serving.

:::

## Responsibilities

- **Classify** — suggest category / severity / tags for a draft issue.
- **Summarize** — decision-support digest of a single petition, issue,
  or consultation thread.
- **Draft** — turn an announcement brief into a structured draft.
- **Insights** — community-wide aggregate digest (themes + sentiment +
  top asks + recommended actions).
- **Narrate** — plain-language read on admin platform metrics.

Every response is JSON with a strict schema pinned by Gemini's
`response_schema`. We never parse free-form text.

## Package layout

```
services/civicai-service/
├── cmd/server/main.go              # Wires all subpackages, boots Gin
├── internal/
│   ├── gemini/client.go            # Thin SDK wrapper: GenerateJSON(system, prompt, schema, out)
│   ├── middleware/auth.go          # JWT validation (no DB hop; gateway already enforces bans)
│   ├── classify/                   # POST /v1/ai/classify-issue
│   ├── summarize/                  # POST /v1/ai/summarize (petition | issue | consultation)
│   ├── draft/                      # POST /v1/ai/draft-announcement
│   ├── insights/                   # GET  /v1/ai/community-insights
│   └── narrate/                    # GET  /v1/ai/narrate-metrics
└── pkg/
    ├── config/config.go            # GEMINI_API_KEY, GEMINI_MODEL, upstream service URLs, REDIS_URL
    └── response/response.go        # Success/Error envelope helpers
```

Each feature package follows the same convention as the other services:
`service.go` for domain logic, `handler.go` for HTTP wiring, source-side
clients in a sibling file (`source.go`) when the feature reads from
another service. No repository layer — CivicAI holds no state of its
own; audit persistence is on the roadmap.

## Endpoints

Full OpenAPI at [`/docs`](http://localhost:3000/docs?spec=/docs/openapi/civicai.yaml)
on the gateway. Summary:

| Method | Path                        | Purpose                                            | Role gate         |
| ------ | --------------------------- | -------------------------------------------------- | ----------------- |
| POST   | `/v1/ai/classify-issue`     | Suggest category, severity, tags for a draft issue | Any authenticated |
| POST   | `/v1/ai/summarize`          | Summarize petition / issue / consultation          | Staff roles       |
| POST   | `/v1/ai/draft-announcement` | Draft an announcement from a brief                 | Staff roles       |
| GET    | `/v1/ai/community-insights` | Aggregate digest across a community                | Staff roles       |
| GET    | `/v1/ai/narrate-metrics`    | Plain-language read on platform metrics            | `PLATFORM_ADMIN`  |
| GET    | `/health`                   | Liveness                                           | Public            |

Community Funding surfaces (`internal/campaignai/`):

| Method | Path                               | Purpose                                       | Role gate        |
| ------ | ---------------------------------- | --------------------------------------------- | ---------------- |
| POST   | `/v1/ai/classify-campaign`         | Suggest category, emergency flag, tags        | Staff roles      |
| POST   | `/v1/ai/draft-campaign`            | Draft title, summary, description, milestones | Staff roles      |
| POST   | `/v1/ai/summarize-campaign-impact` | Public read on what a campaign has achieved   | Staff roles      |
| POST   | `/v1/ai/draft-donor-update`        | Draft an update for people who donated        | Staff roles      |
| POST   | `/v1/ai/draft-completion-report`   | Draft the closing account                     | Staff roles      |
| POST   | `/v1/ai/assess-campaign-risk`      | Review-priority signals for a queue           | `PLATFORM_ADMIN` |

Staff roles = `REPRESENTATIVE`, `GOVERNMENT_ADMIN`, `PLATFORM_ADMIN`,
`NGO`, `MODERATOR`. Enforced inside civicai-service, not the gateway.

## The `gemini` package

One method matters:

```go
func (c *Client) GenerateJSON(
    ctx context.Context,
    systemInstruction, userPrompt string,
    schema *genai.Schema,
    out any,
) error
```

- 15-second overall deadline — Gemini flash is fast but a slow tail
  wedges citizen forms.
- System instruction pins tone + task; response schema constrains the
  shape so we never get free-form text back.
- Decodes into `out` (a pointer to a per-feature struct). Callers never
  touch the SDK directly.

## Source clients (talking to other services)

Two features need to pull data from CivicOS's own services before
prompting Gemini:

- **Summarize** — reads petition / issue / consultation detail +
  discussion. `community-service` for petitions and issues,
  `organization-service` for consultations.
- **Insights** — fans out to `community-service` for issues + petitions
  - comments in a bounded worker pool.
- **Narrate** — reads `identity-service` `/v1/admin/metrics`.

All three forward the caller's JWT. This is deliberate: authorization
cascades naturally. A user who can't read the underlying resource
upstream can't summarize it either. There's no service-to-service secret
to rotate.

## Caching

Redis-backed. Each feature keys its own namespace; the cache is
fail-open (a Redis outage degrades to fresh Gemini calls, never a 500).

| Feature     | Key                               | TTL    | Why                                                                                             |
| ----------- | --------------------------------- | ------ | ----------------------------------------------------------------------------------------------- |
| `summarize` | `civicai:summary:<resource>:<id>` | 30 min | Thread themes shift slowly; re-clicking should be near-instant.                                 |
| `insights`  | `civicai:insights:<communityId>`  | 1 h    | Community-wide stories change even more slowly + fan-out is heavy.                              |
| `narrate`   | `civicai:narrate:<scope>`         | 15 min | Metrics move continuously; short TTL keeps freshness reasonable.                                |
| `classify`  | (none)                            | —      | Debounced per keystroke; caching same-title inputs across sessions gives no meaningful benefit. |
| `draft`     | (none)                            | —      | A second call is meant to give a different variation to compare.                                |

## Rate limiting

Every AI route sits behind the gateway's `limitStandard` middleware
(same tier as org authoring). AI is expensive; the shared budget stops
runaway loops in the FE from burning quota.

`AI_UNAVAILABLE` (HTTP 502) is the standard failure code when Gemini
itself fails — the FE treats it as a soft error and lets the user
continue without AI.

## Config

```bash
# ─── civicai-service ────────────────────────────────────────────────
CIVICAI_SERVICE_PORT=3004
CIVICAI_SERVICE_URL="http://localhost:3004"   # gateway → civicai

# Where CivicAI reads source data from (forwards caller's JWT).
COMMUNITY_SERVICE_URL="http://localhost:3002"
ORGANIZATION_SERVICE_URL="http://localhost:3003"
IDENTITY_SERVICE_URL="http://localhost:3001"

# Auth (shared with all services).
JWT_SECRET="..."

# ─── AI (CivicAI) ───────────────────────────────────────────────────
GEMINI_API_KEY=""                       # https://aistudio.google.com/apikey
GEMINI_MODEL="gemini-flash-latest"      # evergreen alias — swap only if you know why

# Cache backend (optional; unset disables caching).
REDIS_URL="redis://localhost:6379"
```

## Design principles

Enforced in code across every subpackage:

1. **Human oversight is mandatory.** Every response is a _suggestion_
   or _draft_; nothing auto-publishes, auto-assigns, or auto-decides.
2. **Provenance-tagged.** Every output includes `model` +
   `generatedAt`. Every FE surface renders an `AI-generated · review`
   badge until a human edits or approves.
3. **Task-shaped endpoints.** Not one giant `/chat` — each endpoint
   does one thing with a strict response schema.
4. **Fail open.** Gemini outages, quota exhaustion, and upstream 500s
   all return `AI_UNAVAILABLE`. The civic loops (submit issue, publish
   announcement) never depend on CivicAI.
5. **Cache aggressively.** See the table above.
6. **Rate-limit hard.** All AI routes share the Standard authoring
   budget at the gateway.

## Product docs

- Product roadmap for CivicAI: [`docs/product/civicai-plan.md`](https://github.com/civicos-hq/civicos/blob/main/docs/product/civicai-plan.md) in the repo.
- Full capabilities catalog: [`docs/product/civicai-capabilities.md`](https://github.com/civicos-hq/civicos/blob/main/docs/product/civicai-capabilities.md).
- User-facing overviews:
  [Citizens](/citizens/civicai) · [Organizations & Staff](/organizations/civicai).

## Community Funding surfaces

Six endpoints in `internal/campaignai/`. They differ from the rest of this
service in one respect that shapes all of them: they write about money other
people gave, from claims CivicOS cannot verify. Donations settle straight to
the organization's bank account, so reported spending is the organization's
assertion, not an observed fact.

Three consequences are enforced in code rather than left to the prompt:

**The fact sheet says whose claim it is.** `factSheet()` introduces spend
records under "WHAT THE ORGANIZATION SAYS IT HAS SPENT", followed by an
explicit line that CivicOS never held the money and cannot verify it. A model
handed a list of expenses under a neutral heading will summarise them as
established fact, and that summary is shown to donors.

**Arithmetic about unexplained money is computed, never generated.**
`CompletionReport` returns `unaccountedMinor` from `Context.UnaccountedMinor()`.
That figure is frozen into the public record when the report is filed, and a
hallucinated one would be indistinguishable from a true one.
`UnaccountedMinor()` is also allowed to go negative — an organization reporting
more spending than it raised here is normal, and clamping it to zero would hand
the prompt a tidier picture than the truth.

**Authorization cascades rather than being re-implemented.** `SourceClient`
forwards the caller's own Bearer token to organization-service, where
`CanReadInternal` applies. There is deliberately no service-to-service
credential: adding one would silently widen who can have a campaign assessed.

### assess-campaign-risk

The one that needed the most care. It produces fraud signals about a named
organization asking the public for money, and the funding plan is explicit
about the stakes: _"an AI that can block fundraising is an AI that can be wrong
about someone's flood relief."_

- **`PLATFORM_ADMIN` only.** Not `NGO` — an organization must never be able to
  run a fraud probe on a rival's appeal.
- **It never writes.** Nothing sets `Campaign.RiskScore`, changes a status, or
  notifies anyone. A reviewer acts through the ordinary review and pause
  endpoints, which carry their own audit trails.
- **Observations, not verdicts.** The system instruction tells the model it is
  not deciding anything. A model asked to judge produces judgments, and a
  reviewer who reads "LIKELY FRAUD" is anchored before opening the campaign.
  The bands are review priorities — `ROUTINE`, `WORTH_A_LOOK`,
  `REVIEW_CLOSELY` — and there is no "FRAUDULENT" band, because the word would
  follow the organization around the admin console.
- **Every signal must cite evidence and offer the innocent reading of the same
  fact.** Both are schema-required, and signals arriving with an empty evidence
  string are dropped server-side — "required" only means present, and a model
  can satisfy it with `""`. If no signal survives, the band is forced back to
  `ROUTINE`.
- The prompt also names what is **not** a signal on its own: a small or new
  organization, simple writing, a large goal for genuinely expensive work, or
  no spending reported by a campaign that has only just published.

The honest description of what it is for: a reviewer with forty campaigns in a
queue wants to know which three to open first.
