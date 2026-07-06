# Deploying CivicOS to Render

This is the first-time deployment playbook. The whole stack — 4 Go services, 2 static frontends, managed Postgres, managed Redis — is described declaratively in `render.yaml` at the repo root. One click applies it all.

Estimated time end-to-end: **20–30 minutes**, most of that waiting for the first Docker builds.

Monthly cost at launch: **~$34/mo** (breakdown at the bottom).

---

## Prerequisites

- The repo must be on GitHub (or another Git provider Render supports). CivicOS lives at [`civicos-hq/civicos`](https://github.com/civicos-hq/civicos).
- A Render account. Sign up at https://render.com — the free tier is fine to start.
- A domain (optional for launch, required before public release). We'll wire it up at the end.
- SMTP credentials (optional for the very first deploy — verification emails won't send until set). We recommend **Resend** or **Postmark**.

---

## Step 1 — Connect Render to GitHub

1. In Render's dashboard, click your avatar → **Account settings** → **Integrations** → **GitHub** → **Connect**.
2. In GitHub, grant Render access to the `civicos-hq/civicos` repo (single-repo access is fine — no need to grant the whole org).

---

## Step 2 — Apply the Blueprint

1. In Render's dashboard, click **New +** → **Blueprint**.
2. Pick the `civicos-hq/civicos` repo.
3. Render reads `render.yaml` at the repo root and previews what it'll create:
   - **civicos-postgres** — Postgres 16, `basic-256mb` plan
   - **civicos-redis** — Key Value, free tier
   - **civicos-gateway** — public web service (only backend on the internet)
   - **civicos-identity** — private service
   - **civicos-community** — private service with 10 GB persistent disk for uploads
   - **civicos-organization** — private service
   - **civicos-web** — citizen static site
   - **civicos-admin** — admin console static site
   - **civicos-secrets** — env-var group with `JWT_SECRET` (auto-generated), SMTP fields (blank)
4. Give the group a name — e.g. `civicos-production`.
5. Click **Apply**.

Render now provisions everything. First deploy takes **10–20 minutes** because it's building 4 Go services + 2 frontends from cold cache in parallel. Follow along in the dashboard — each service has a **Logs** tab so you can watch the Docker build.

---

## Step 3 — Fill in the SMTP secrets

The `civicos-secrets` env-var group has three blank fields Render can't auto-populate: `SMTP_HOST`, `SMTP_USER`, `SMTP_PASSWORD`.

1. Sign up for **Resend** (https://resend.com) — free tier covers 3,000 emails/mo, more than enough for a launch.
2. Add and verify a sender domain (or use the free `onboarding@resend.dev` for testing).
3. Create an API key in the Resend dashboard.
4. In Render → **Env Groups** → **civicos-secrets**, set:

| Variable        | Value                                |
| --------------- | ------------------------------------ |
| `SMTP_HOST`     | `smtp.resend.com`                    |
| `SMTP_PORT`     | `587` (already set)                  |
| `SMTP_USER`     | `resend`                             |
| `SMTP_PASSWORD` | `re_...` (your Resend API key)       |
| `SMTP_FROM`     | `CivicOS <no-reply@your-domain.com>` |

Render auto-restarts the identity-service the moment you save.

---

## Step 4 — Seed the first admin

The admin console is gated by the `PLATFORM_ADMIN` role, but nobody has it yet. Bootstrap:

1. Visit the citizen web URL Render assigned (e.g. `https://civicos-web.onrender.com`) — check the **Overview** tab of the `civicos-web` service.
2. Register normally with your email.
3. Verify your email (Mailpit is dev-only; on Render the verification email lands in your real inbox via Resend).
4. In Render → `civicos-postgres` → **Connect** → **PSQL Command**. It hands you a `psql` command with credentials embedded. Copy-paste into your terminal.
5. Promote yourself:

```sql
UPDATE users SET role = 'PLATFORM_ADMIN' WHERE email = 'you@example.com';
```

6. Visit the admin URL (e.g. `https://civicos-admin.onrender.com`). Sign in. You should land on the Overview page.

---

## Step 5 — Wire your custom domain (optional but recommended)

Do this once you have a domain like `civicos.ng`.

**In Render:**

1. On `civicos-web`, go to **Settings** → **Custom Domain** → **Add Custom Domain** → enter `civicos.ng` (or your subdomain).
2. On `civicos-admin`, do the same for `admin.civicos.ng`.
3. On `civicos-gateway`, do the same for `api.civicos.ng`.

Render shows you which DNS records to add.

**At your DNS provider** (Cloudflare, Namecheap, etc.):

| Type           | Name         | Value                                   |
| -------------- | ------------ | --------------------------------------- |
| `A` or `ALIAS` | `civicos.ng` | (value from Render for civicos-web)     |
| `CNAME`        | `admin`      | (value from Render for civicos-admin)   |
| `CNAME`        | `api`        | (value from Render for civicos-gateway) |

Certificates provision automatically (Let's Encrypt) within a few minutes of DNS resolving.

**Then update `APP_URL`** on the identity service to your citizen domain — this is what verification emails link to. In Render → `civicos-identity` → **Environment** → set `APP_URL` = `https://civicos.ng`.

---

## Step 6 — Verify everything works

Smoke-test in this order:

1. **Citizen web loads** — `https://civicos.ng` (or the onrender URL) shows the homepage.
2. **Register a fresh test account** — receive verification email → click link → land on dashboard.
3. **Sign in to admin** — `https://admin.civicos.ng` — sign in with your promoted account.
4. **Create a community** in admin → check it appears in the citizen web's onboarding LGA picker.
5. **Raise an issue** from a citizen account → check it shows in the community feed and in the admin's overview metrics.

If any of those fail, jump to the troubleshooting section below.

---

## How auto-deploy works

- **Backend services** (`civicos-gateway`, `civicos-identity`, `civicos-community`, `civicos-organization`) — Render watches `main`. On every push, only the services whose Docker context changed rebuild. Deploy on success.
- **Frontends** (`civicos-web`, `civicos-admin`) — Render rebuilds on every push to `main` (Vite build outputs land in the CDN).
- **`render.yaml` changes** — you must re-apply the Blueprint manually in the dashboard. Push doesn't do this automatically. Prevents accidental infra changes.

If a build produces a bad revision, Render keeps serving the previous one — a failed deploy doesn't take the site down. But bad code can still ship to production if CI fails to catch it. See below for how to gate deploys on green CI.

### Optional: gate deploys on green CI

By default Render deploys the moment you push to `main`, without waiting for GitHub Actions to finish. If you want deploys to require green CI:

1. In each Render service's **Settings** → **Auto-Deploy**, switch from **On Commit** to **Off**.
2. In each service's **Settings**, copy its **Deploy Hook** URL (a webhook Render exposes). You'll get 6 URLs total.
3. In your GitHub repo → **Settings** → **Secrets and variables** → **Actions**, add:
   - `RENDER_HOOK_GATEWAY`, `RENDER_HOOK_IDENTITY`, `RENDER_HOOK_COMMUNITY`, `RENDER_HOOK_ORGANIZATION`, `RENDER_HOOK_WEB`, `RENDER_HOOK_ADMIN`
4. Add this job to `.github/workflows/ci.yml`:
   ```yaml
   deploy:
     name: Deploy
     runs-on: ubuntu-latest
     needs: ci
     if: github.ref == 'refs/heads/main' && github.event_name == 'push'
     steps:
       - name: Trigger Render deploys
         env:
           HOOKS: |
             ${{ secrets.RENDER_HOOK_GATEWAY }}
             ${{ secrets.RENDER_HOOK_IDENTITY }}
             ${{ secrets.RENDER_HOOK_COMMUNITY }}
             ${{ secrets.RENDER_HOOK_ORGANIZATION }}
             ${{ secrets.RENDER_HOOK_WEB }}
             ${{ secrets.RENDER_HOOK_ADMIN }}
         run: |
           set -eu
           echo "$HOOKS" | while IFS= read -r hook; do
             [ -z "$hook" ] && continue
             echo "→ POST $(echo "$hook" | head -c 60)…"
             curl -fsSL -X POST "$hook" > /dev/null
           done
   ```
   Now deploys only fire after every test job passes.

---

## Troubleshooting

### First deploy: a service is stuck "Deploying" for >15 min

Open its **Logs** tab. Common causes:

- **Docker build fails** — usually a `go mod download` hitting a network hiccup. Click **Manual Deploy** → **Clear build cache & deploy**.
- **Health check timing out** — the service is trying to reach `DATABASE_URL` before Postgres is fully ready. Wait — the first Postgres provision takes 2–3 min. Once Postgres is `Available`, the failing service will retry.

### "invalid connection URL" errors from Go services

Almost always because `DATABASE_URL` starts with `postgres://` but GORM's Postgres driver prefers `postgresql://`. Render's `fromDatabase.connectionString` uses `postgres://` — GORM handles both, but if you see this, verify the env var value in the service's dashboard.

### Emails aren't arriving

Check the identity service logs — it prints the exact SMTP error. Common:

- **Resend rejects the `From:` address** because you haven't verified the sender domain. Verify in Resend dashboard, or temporarily use `onboarding@resend.dev`.
- **`SMTP_PASSWORD` is set to the Resend UI key, not an API key.** They're different — API keys start with `re_`.

### Rate-limited immediately

The Redis instance may not be linked. Verify `REDIS_URL` env var on `civicos-gateway`. If missing, redeploy from Blueprint.

### Uploads disappear across deploys

Uploads must live on the persistent disk mounted at `/data`. Verify `civicos-community` has the `uploads` disk attached (Blueprint provisions this, but a manual dashboard edit could remove it).

---

## Cost breakdown

| Line item                        | Plan        | $/mo        |
| -------------------------------- | ----------- | ----------- |
| civicos-gateway                  | Starter     | $7          |
| civicos-identity                 | Starter     | $7          |
| civicos-community                | Starter     | $7          |
| civicos-organization             | Starter     | $7          |
| civicos-postgres                 | Basic-256mb | $6          |
| civicos-redis                    | Free        | $0          |
| civicos-web (static)             | Free        | $0          |
| civicos-admin (static)           | Free        | $0          |
| Persistent disk (uploads, 10 GB) |             | ~$1         |
| **Total**                        |             | **~$35/mo** |

Cost-shaving options:

- Bundle the 3 private services (identity + community + organization) into one Docker image behind a single Starter plan — saves $14/mo. Requires a small `main.go` that mounts each service's routes on distinct URL prefixes. Doable but reduces microservice hygiene.
- Skip the persistent disk entirely — store uploads in Render Object Storage or S3 instead. Adds a small storage cost but removes the disk fee and enables horizontal scaling.
- Use Render's free web-service tier for early testing. It sleeps after 15 min of inactivity, so it's not viable for real users, but fine for initial demos.

---

## What's not in the Blueprint

Deliberate omissions:

- **NATS** — CivicOS's inter-service messaging bus. A codebase grep shows nothing publishes or subscribes yet; it was scaffolded for future cross-service events. Add later as a Render Private Service (~$7/mo) when needed.
- **Object storage (S3-compatible)** for uploads — currently uploads land on the persistent disk. Fine for < 10,000 photos; migrate to Render Object Storage when volume grows.
- **CDN in front of the gateway** — Render's static-site tier already includes CDN caching for `civicos-web` and `civicos-admin`. The gateway is dynamic and can't be CDN-cached without careful cache-control headers.
- **Error tracking (Sentry / equivalent)** — worth adding before real users hit the platform. `apps/web/src/main.tsx` is the obvious injection point.

---

## Next steps after launch

1. Set up **automated Postgres backups** — Render includes daily backups on paid plans; verify they're retained for at least 7 days.
2. Add **uptime monitoring** (Better Stack, UptimeRobot) — free tier covers 5 endpoints.
3. **Rotate `JWT_SECRET`** every 90 days. Render supports env-var updates without code changes.
4. **Enable branch preview environments** — Render can auto-deploy PRs to disposable environments for review. Useful once contributors show up.
