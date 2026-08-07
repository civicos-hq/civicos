---
id: deployment
title: Deployment
sidebar_position: 1
---

# Deployment

Production runs on **Render**. The whole stack (5 Go services + 3
static sites + managed Postgres + managed Redis) is described
declaratively in `render.yaml` at the repo root — one Blueprint apply
brings it all up.

The full playbook lives in
[`docs/deploy.md`](https://github.com/civicos-hq/civicos/blob/main/docs/deploy.md).
This page is the tour, not the replacement.

## What Render provisions

- `civicos-gateway` — the api-gateway, a **free** Web Service (port
  `3000`). Public entry point; spins down after 15 minutes idle.
- `civicos-identity` — identity-service, **starter** Web Service. Also
  runs the shared database's migrations at boot.
- `civicos-community` — community-service, **starter** Web Service.
- `civicos-organization` — organization-service, **starter** Web Service.
- `civicos-civicai` — civicai-service, **starter** Web Service. Holds no
  database of its own; it reads from the other services with the caller's
  token.

The four internal services are `type: web`, not `pserv`. They are reached
by the gateway over Render's **private network** via
`fromService.hostport`, so the traffic never touches the public edge —
that matters, because Render's Cloudflare-fronted edge intermittently
429s the gateway's shared egress IP. They stay `web` rather than becoming
private services because a service's type is immutable on Render:
converting means delete and recreate, with new hostnames and re-wiring.
They remain publicly reachable on their `onrender.com` URLs, and every
route enforces JWT itself.

`starter` also buys two things the free plan does not: instances that
never spin down, and outbound SMTP.

- `civicos-web` — citizen web Static Site (`apps/web` build output).
- `civicos-admin` — admin console Static Site (`apps/admin` build
  output).
- Managed Postgres, managed Redis.

Only the gateway and the two Static Sites are public. The three
backend services are private — they're reachable inside Render's
network from the gateway but never from the public internet.

## Estimated cost at launch

**~$34/mo** on Render's minimum plans. Breakdown in `docs/deploy.md`.

## First-time deploy — the short version

1. Connect Render to the GitHub repo.
2. In Render, **New + → Blueprint** → point at the repo → confirm.
3. Wait ~20–30 minutes for the first Docker builds.
4. Set the following env vars in the Render dashboard on each service
   (Blueprint provides defaults for most, but some are secrets):
   - `JWT_SECRET` (32+ chars) — must be **the same** on gateway,
     identity, community, organization and civicai. The Blueprint mints
     one into the shared `civicos-secrets` group, so this is only manual
     if you override it.
   - `GEMINI_API_KEY` on civicai — from
     [Google AI Studio](https://aistudio.google.com/apikey). The service
     **refuses to boot without it**, which surfaces as a failed deploy
     rather than a service that 500s on every AI call.

     ⚠️ The free tier allows **20 requests per day** across the whole
     project, shared by all eleven CivicAI endpoints. Put the key on a
     paid plan before opening the platform to real users.

   - `PAYSTACK_SECRET_KEY` / `PAYSTACK_PUBLIC_KEY` on organization —
     donations are disabled without them.
   - `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` /
     `SMTP_FROM` on identity (once you have a Resend / Postmark
     account).
   - `APP_URL` on identity — the public URL where email links land.
5. Wait for services to go healthy (each has `/health`).
6. Register the first user via the citizen web app, verify the email,
   then bump their role to `PLATFORM_ADMIN` via `render psql`:
   ```sql
   UPDATE users SET role='PLATFORM_ADMIN' WHERE email='<you>';
   ```

## Zero-downtime deploys

Render does rolling deploys per service by default. Because each
service has `/health`, Render waits for the new instance to answer
`200` before shifting traffic. AutoMigrate runs at startup — if you're
making an additive schema change, that's fine. If you're making a
destructive schema change, apply the SQL migration _first_ (via
`render psql`) then push the code.

## Environment variables — production checklist

Set on **every backend service**:

- `DATABASE_URL` — the Render Postgres string (Blueprint wires this).
  Not set on civicai — it owns no tables.
- `JWT_SECRET` — 32+ chars, identical across the gateway and every
  service that validates a token (identity, community, organization,
  civicai). A mismatch anywhere shows up as 401s on that service alone.
- `PORT` — Render sets this; don't override.

Set on the **gateway**:

- `IDENTITY_SERVICE_URL`, `COMMUNITY_SERVICE_URL`,
  `ORGANIZATION_SERVICE_URL`, `CIVICAI_SERVICE_URL` — Blueprint wires
  these to the private URLs.

  Each has a `localhost` fallback for local development, which is a trap
  in production: an unset variable does not fail loudly, it silently
  routes to a port where nothing is listening. If one family of routes
  returns connection errors and everything else is fine, check this
  first.

- `REDIS_URL` — Render Redis (Blueprint wires this).

Set on **identity**:

- `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` /
  `SMTP_FROM` — real email.
- `APP_URL` — used in email link generation.

Set on the **frontends** (build-time — configured in `apps/*/render.yaml`
section or via Static Site env):

- `VITE_API_URL` — the gateway's public URL.

## Custom domain

Once the platform is live at the Render-issued URL:

1. Add the custom domain in Render on the **gateway** service and both
   Static Sites.
2. Point DNS as Render instructs.
3. Update `APP_URL` on identity and `VITE_API_URL` on the frontends to
   the custom domain.
4. Redeploy the frontends so the new env is baked in.

## Backups

- Render Postgres runs daily snapshots. Retention is plan-dependent —
  check the current plan before relying on it for compliance.
- **User uploads** live on the community-service Web Service's disk
  (`uploads/` directory). Render's ephemeral disks don't survive a
  redeploy on the free tier — for production you must either bind a
  persistent disk to the community-service or move uploads to S3 / R2
  before launch.

## Rollback

Each Render service keeps a small history of past deploys. Roll back
via the Render dashboard on the specific service — the previous image
comes back up in a couple of minutes. Roll back the schema only if
you had a destructive migration; additive changes are safe to leave.
