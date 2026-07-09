---
id: api-gateway
title: API Gateway
sidebar_position: 1
---

# API Gateway

Port `:3000`. Single public entry point. Every browser request goes
through here — the frontends never talk to the backend services
directly.

## Responsibilities

- **JWT validation** — decodes and verifies the access token; sets
  `userID`, `userName`, `userRole`, `userEmail` on the Gin context so
  downstream services can trust it.
- **Reverse-proxying** — routes each URL to the right backend
  (identity, community, organization).
- **Per-action rate limiting** — Redis-backed sliding windows keyed by
  IP (unauthenticated) or user ID (authenticated).
- **CORS** — allows the three dev frontends (`5173`, `5174`, `5175`)
  and any other origin without credentials.
- **Health aggregation** — `/health`, `/health/identity`,
  `/health/community`, `/health/organization` — used by the admin
  console.
- **Swagger UI** — `/docs` with a service picker; specs are
  `go:embed`ed at build time.
- **SSE pass-through** — the notifications stream is a
  streaming reverse-proxy (`NewStreamingProxy`) that disables buffering.

## Package layout

```
services/api-gateway/
├── cmd/server/main.go             # Route table + wire-up
├── internal/
│   ├── docs/                      # Swagger UI + embedded OpenAPI specs
│   │   ├── docs.go
│   │   └── openapi/*.yaml
│   ├── middleware/
│   │   ├── auth.go                # JWTAuth — decode + validate
│   │   └── ratelimit.go           # Tier definitions + Limit() factory
│   └── proxy/
│       ├── proxy.go               # NewReverseProxy, NewStreamingProxy
│       └── health.go              # NewHealthProxy
└── pkg/
    ├── config/                    # Env loading
    └── ratelimit/                 # Redis sliding-window primitive
```

## Rate-limit tiers

`internal/middleware/ratelimit.go` defines the tiers. Applied _before_
auth for public routes (key by IP), _after_ auth for protected routes
(key by user ID).

| Tier            | Where used                                                                   |
| --------------- | ---------------------------------------------------------------------------- |
| `Strict`        | `/auth/register`, `/auth/login`, forgot-password, flag creation, self-delete |
| `Standard`      | Profile edits, admin PATCH/POST calls                                        |
| `Lenient`       | `/auth/refresh`, `/auth/logout`                                              |
| `CommentMinute` | Comment burst window — few-in-a-minute limit                                 |
| `CommentHour`   | Comment slow-drip window — sustained-hour limit                              |
| `Sign`          | Petition signing                                                             |
| `Create`        | Creating an issue, petition, community, or rep                               |
| `Upvote`        | Toggling upvotes                                                             |

If the limiter can't reach Redis at boot, it fails **open** — the
gateway logs the gap and every request is allowed. Deliberate: a dev
without Redis should still be able to run the gateway.

## Adding a new proxied route

```go
// services/api-gateway/cmd/server/main.go
r.POST("/api/v1/things", authMiddleware, limitCreate, communityProxy)
```

Rules of thumb:

- **Public GET** — no auth, no rate limit (or `limitLenient` if the
  endpoint is expensive).
- **Authenticated GET** — auth, no rate limit unless it's a hot path.
- **Authenticated POST/PATCH/DELETE** — always auth + a rate-limit tier.
- **Auth surface (login, register, password)** — `limitStrict` before
  auth.

## Reverse-proxy behaviour

`proxy.NewReverseProxy(target, prefix)` strips `/api` from the
inbound path before forwarding. So `POST /api/v1/issues` at the
gateway becomes `POST /v1/issues` at community-service. The `/api`
prefix exists solely at the gateway's edge.

## Streaming (SSE)

`GET /api/v1/notifications/stream` uses `proxy.NewStreamingProxy`
which:

- Disables response buffering.
- Sets `X-Accel-Buffering: no` for good measure.
- Doesn't collapse the connection on idle.

The connection is held open by the citizen frontend for realtime
notifications — see [Community Service → Notifications](./community-service.md#notifications--sse).

## Environment

Required:

- `JWT_SECRET` — must match every downstream service.
- `IDENTITY_SERVICE_URL`, `COMMUNITY_SERVICE_URL`,
  `ORGANIZATION_SERVICE_URL` — target URLs.

Optional:

- `REDIS_URL` — enables rate limiting. Empty = limiter disabled.
- `PORT` / `API_GATEWAY_PORT` — defaults to `3000`.
