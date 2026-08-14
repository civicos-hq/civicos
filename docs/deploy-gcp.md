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

### Flood forecasts (Google Flood Hub) — optional

Off unless a `GOOGLE_FLOOD_API_KEY` secret exists. The deploy script checks
for it and, if absent, deploys `community-service` without the feature — a
missing secret referenced by `--set-secrets` fails the whole deploy, and
flood forecasts must never be the reason a release of everything else
doesn't ship.

**Access is not self-serve.** The Flood Forecasting API is in pilot and
Google must add your account as a _Service Consumer_ before
`floodforecasting.googleapis.com` can be enabled at all. If the enable page
404s or says you lack access, that is what has not happened yet — it is not
a project misconfiguration. Merging the code does not turn the feature on;
Google granting access does.

Once you have access:

```bash
# 1. Enable the API in the same project the rest of CivicOS runs in.
gcloud services enable floodforecasting.googleapis.com --project=civicos-ng-prod

# 2. Create a key and restrict it to that one API. An unrestricted key that
#    leaks is a key to every API enabled on the project.
gcloud alpha services api-keys create \
  --display-name="CivicOS Flood Hub" \
  --api-target=service=floodforecasting.googleapis.com \
  --project=civicos-ng-prod

# 3. Store it. The key string is in the `keyString` field of the output above.
printf '%s' 'AIza...' | gcloud secrets create GOOGLE_FLOOD_API_KEY \
  --data-file=- --project=civicos-ng-prod

# 4. Let Cloud Run read it.
gcloud secrets add-iam-policy-binding GOOGLE_FLOOD_API_KEY \
  --member="serviceAccount:$(gcloud projects describe civicos-ng-prod \
     --format='value(projectNumber)')-compute@developer.gserviceaccount.com" \
  --role=roles/secretmanager.secretAccessor --project=civicos-ng-prod

# 5. Redeploy. The script picks the secret up automatically.
./deploy/gcp/deploy.sh
```

Three tuning variables ride along as plain env vars, not secrets — they are
configuration, not credentials:

| Variable                      | Default | What it does                                                                                                                                                                           |
| ----------------------------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `FLOOD_POLL_INTERVAL_MINUTES` | `60`    | Sweep cadence. **`0` is the kill switch** — the API is in pilot and Google say breaking changes should be expected, so an operator must be able to stop consuming it without a deploy. |
| `FLOOD_REGION_CODE`           | `NG`    | CLDR region swept. One call covers the whole country.                                                                                                                                  |
| `FLOOD_MATCH_RADIUS_KM`       | `50`    | How far a gauge can be from a community and still cover it.                                                                                                                            |

### Location lookup for admins — optional

`GOOGLE_GEOCODING_API_KEY`, treated exactly like the flood key: referenced
only if the secret exists, so a missing one never fails the deploy. Enable
the Geocoding API on the same project.

It powers the "suggest a point" behaviour when an admin picks an LGA while
creating a community. Without it the admin types coordinates by hand and
the affordance is hidden rather than offered and broken.

It produces a **suggestion, not an answer** — an LGA is a polygon and the
geocoder returns roughly its centre, which can sit in farmland well away
from the town or the river. The admin confirms on a map before saving.

**No community gets a forecast until it has coordinates.** Everything else
about a community is an administrative name; flood gauges are points. Set
them per community with `PATCH /api/v1/communities/:id` as a
`GOVERNMENT_ADMIN` or `PLATFORM_ADMIN`:

```bash
curl -X PATCH https://api.civicos.ng/api/v1/communities/<id> \
  -H "Authorization: Bearer <admin token>" \
  -H 'Content-Type: application/json' \
  -d '{"latitude": 7.7322, "longitude": 8.5391}'
```

A community with no coordinates is skipped rather than matched
approximately. That is deliberate: warning the wrong town is worse than
warning nobody.

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
