---
id: deployment
title: Deployment
sidebar_position: 1
---

# Deployment

Production runs on **Google Cloud**. Everything — 5 Go services and 3
static sites — runs on Cloud Run with scale-to-zero, backed by Cloud SQL
for Postgres and a GCS bucket for uploads.

The full playbook lives in
[`docs/deploy-gcp.md`](https://github.com/civicos-hq/civicos/blob/main/docs/deploy-gcp.md).
This page is the tour, not the replacement.

## What runs where

| Cloud Run service      | Image                    | Notes                                    |
| ---------------------- | ------------------------ | ---------------------------------------- |
| `civicos-gateway`      | `api-gateway`            | The only backend the browser talks to    |
| `civicos-identity`     | `identity-service`       | Also runs the shared database migrations |
| `civicos-community`    | `community-service`      | 512Mi — its GCS volume forces gen2       |
| `civicos-organization` | `organization-service`   | —                                        |
| `civicos-civicai`      | `civicai-service`        | Holds no tables of its own               |
| `civicos-web`          | nginx + `apps/web` build | Static                                   |
| `civicos-admin`        | nginx + `apps/admin`     | Static                                   |
| `civicos-docs`         | nginx + `apps/docs`      | This site                                |

Plus a `db-f1-micro` Cloud SQL instance in `europe-west1` — tier-1
pricing, and roughly half the latency to Nigerian users versus a US
region.

All services are `--allow-unauthenticated`. That is deliberate: every one
validates the caller's JWT itself, so the gateway is a convenience layer
rather than the only defence.

## Deploying

```bash
./deploy/gcp/deploy.sh
```

Idempotent — re-running updates services in place.

### Order matters

`identity-service` refuses to migrate until the tables owned by community
and organization exist, and says so explicitly rather than failing
obscurely. So:

```
community → organization → identity → civicai → gateway
```

The gateway is last because it needs every backend URL. Deploy it before a
backend exists and it comes up with an empty `*_SERVICE_URL`, returning
502s until redeployed.

### Frontends are built, not configured

Vite inlines `VITE_API_URL` at **build** time. The frontends take the
gateway URL as a Docker build arg, so changing it means **rebuilding the
images**, not just redeploying them.

## Environment variables — production checklist

Secrets live in Secret Manager and are wired with `--set-secrets`; plain
configuration goes through `--set-env-vars`.

Set on **every backend service**:

- `DATABASE_URL` — points at the Cloud SQL socket, not an address. Cloud
  Run has no static egress IP, so the form is
  `postgresql://civicos:PASS@localhost/civicos?host=/cloudsql/PROJECT:REGION:INSTANCE`,
  and each service needs `--add-cloudsql-instances`. Not set on civicai —
  it owns no tables.
- `JWT_SECRET` — 32+ chars, identical across the gateway and every service
  that validates a token. A mismatch anywhere shows up as 401s on that
  service alone.
- `PORT` — Cloud Run injects it; don't override.

Set on the **gateway**:

- `IDENTITY_SERVICE_URL`, `COMMUNITY_SERVICE_URL`,
  `ORGANIZATION_SERVICE_URL`, `CIVICAI_SERVICE_URL`.

  Each has a `localhost` fallback for local development, which is a trap
  in production: an unset variable does not fail loudly, it silently
  routes to a port where nothing is listening. If one family of routes
  returns connection errors and everything else is fine, check this first.

- `REDIS_URL` — **intentionally unset.** Memorystore's cheapest tier costs
  more than the rest of the stack combined, and the rate limiter fails
  open without it. Rate limiting is therefore off until a cheaper Redis is
  wired in.

Set on **identity**: `SMTP_*` for real email, and `APP_URL` for links
inside those emails.

Set on **community** (all optional): `GOOGLE_FLOOD_API_KEY` and the
`FLOOD_*` tuning vars for flood forecasts, `GOOGLE_GEOCODING_API_KEY` for
admin location lookup. Each is referenced only if its secret exists, so a
missing one never fails the deploy.

Set on the **frontends** at build time: `VITE_API_URL`.

## Custom domain

1. Map the domain to the **gateway** service and both static sites in
   Cloud Run.
2. Point DNS as Cloud Run instructs.
3. Update `APP_URL` on identity and `VITE_API_URL` on the frontends.
4. **Rebuild** the frontend images — `VITE_API_URL` is baked in at build
   time, so a redeploy alone will not pick it up.

## Backups

- Cloud SQL takes automated daily backups. Check the retention on the
  current tier before relying on it for compliance.
- **User uploads** live in a GCS bucket mounted into community-service at
  `/data`, so they survive redeploys — unlike a container filesystem.

## Rollback

Cloud Run keeps every revision. Roll back by routing traffic to the
previous one:

```bash
gcloud run services update-traffic civicos-community \
  --to-revisions=PREVIOUS_REVISION=100 --region=europe-west1
```

That is instant and needs no rebuild. Roll the schema back only if you
shipped a destructive migration; additive changes are safe to leave.
