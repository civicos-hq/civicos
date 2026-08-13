# Deploying CivicOS to Google Cloud

The whole platform — 5 Go services, 3 static sites, Postgres — runs on Cloud
Run and Cloud SQL for roughly **$10–13/month** at launch traffic.

Everything below is already provisioned in project **`civicos-ng-prod`**
(billing account `civicos`, region `europe-west1`). This document is both the
record of what exists and the playbook for rebuilding it.

---

## Why this shape

| Concern       | Choice                                   | Reason                                                                                                |
| ------------- | ---------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Compute       | Cloud Run, `min-instances=0`             | Idle services cost nothing. The trade is a 1–2s cold start on the first request after a quiet period. |
| Database      | Cloud SQL `db-f1-micro`, 10GB HDD, zonal | Cheapest managed Postgres. ~$9/mo — the bulk of the bill. Regional HA doubles it.                     |
| Region        | `europe-west1`                           | Cloud Run tier-1 pricing (same as US regions) but ~half the latency to Nigerian users.                |
| Uploads       | GCS bucket mounted at `/data`            | Cloud Run's filesystem is read-only. A bucket volume keeps photos across restarts for pennies.        |
| Secrets       | Secret Manager                           | Injected at deploy; never in the image or the repo.                                                   |
| Rate limiting | **off**                                  | Memorystore's cheapest tier is ~$35/mo — more than everything else combined. See below.               |

### The Redis decision

`REDIS_URL` is deliberately unset. The gateway's limiter is written to **fail
open** when Redis is unreachable (`services/api-gateway/pkg/ratelimit`), so the
platform behaves correctly without it — it simply does not rate-limit.

That is a real security trade, taken because Memorystore would roughly quadruple
the monthly bill. To turn limiting back on, provision any reachable Redis and
set `REDIS_URL` on `civicos-gateway`; no code change is needed.

---

## What exists

**Cloud Run** (all `europe-west1`, all scale-to-zero)

| Service                | Image                    | Notes                                 |
| ---------------------- | ------------------------ | ------------------------------------- |
| `civicos-gateway`      | `api-gateway`            | The only backend the browser talks to |
| `civicos-identity`     | `identity-service`       | Auth, users, SMTP                     |
| `civicos-community`    | `community-service`      | 512Mi + GCS volume (see below)        |
| `civicos-organization` | `organization-service`   | Paystack keys                         |
| `civicos-civicai`      | `civicai-service`        | Gemini                                |
| `civicos-web`          | nginx + Vite build       | Citizen site                          |
| `civicos-admin`        | nginx + Vite build       | Admin console                         |
| `civicos-docs`         | nginx + Docusaurus build | Documentation                         |

**Other**: Cloud SQL `civicos-pg` (database `civicos`, user `civicos`),
Artifact Registry `civicos`, GCS bucket `civicos-ng-prod-uploads`, and ten
Secret Manager secrets.

---

## Deploying

```bash
./deploy/gcp/deploy.sh
```

Idempotent — re-running updates services in place.

### Order matters

`identity-service` refuses to run its migrations until the tables owned by
community and organization exist, and it says so explicitly rather than failing
obscurely. So the order is:

```
community → organization → identity → civicai → gateway
```

The gateway goes last because it needs every backend URL. If you deploy it
before a backend exists it will come up with an empty `*_SERVICE_URL` and
return 502s until redeployed.

### Frontends are built, not configured

Vite inlines `VITE_API_URL` at **build** time. The frontends therefore take the
gateway URL as a Docker build arg, and **changing the gateway URL means
rebuilding the images**, not just redeploying them.

---

## Gotchas worth knowing

**`--set-env-vars` and the `@` in a database URL.** The Postgres URL contains
`user:password@host`. Using `^@^` as gcloud's delimiter silently truncates the
value at the password and the service dies with `invalid port`. This deploy
uses `^##^`.

**Cloud SQL over a unix socket.** Cloud Run has no static egress IP, so the
services connect through the mounted socket, not TCP:

```
postgresql://civicos:PASSWORD@localhost/civicos?host=/cloudsql/PROJECT:REGION:INSTANCE
```

paired with `--add-cloudsql-instances`.

**GCS volumes force gen2, which needs ≥512Mi.** `civicos-community` runs at
512Mi for this reason; everything else fits in 256Mi. A 256Mi service with a
volume fails at deploy with `Total memory < 512 Mi is not supported with gen2`.

**Backends are `--allow-unauthenticated`.** Every service validates the caller's
JWT itself, so the gateway is a convenience layer rather than the only defence —
the same posture as the previous Render deployment. Locking them to IAM would
require the gateway to mint ID tokens per request (a code change) or a VPC
connector (~$8/mo).

---

## Cost breakdown

| Item                                  | Monthly     |
| ------------------------------------- | ----------- |
| Cloud SQL `db-f1-micro` + 10GB HDD    | ~$9         |
| Cloud Run (8 services, scale-to-zero) | ~$0–3       |
| Artifact Registry (~2GB)              | ~$0.20      |
| GCS uploads                           | ~$0.05      |
| Secret Manager (10 secrets)           | ~$0.36      |
| **Total**                             | **~$10–13** |

Cloud Run's free tier (2M requests, 360k GB-seconds/month) covers a
launch-sized workload, so the compute line stays near zero until real traffic
arrives.

### If cost needs to drop further

Cloud SQL is the only meaningful line. Stopping the instance when idle, or
moving to a Postgres provider with a free tier, is the only large saving
available — everything else is already at or near zero.
