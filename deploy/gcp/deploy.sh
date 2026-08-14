#!/usr/bin/env bash
#
# CivicOS → Google Cloud, cost-minimal topology. See docs/deploy-gcp.md.
#
#   Cloud Run       8 services (5 Go + 3 nginx static), all scale-to-zero
#   Cloud SQL       PostgreSQL 16, db-f1-micro / 10GB HDD / zonal
#   GCS             uploads bucket, mounted into community-service
#   Secret Manager  JWT, SMTP, Gemini, Paystack, Flood Hub, Geocoding, DB password
#
# ~$10-13/month at launch traffic, nearly all of it Cloud SQL.
#
# Idempotent: re-running updates services in place.
set -uo pipefail

PROJECT="${GCP_PROJECT:-civicos-ng-prod}"
REGION="${GCP_REGION:-europe-west1}"
REPO="${REGION}-docker.pkg.dev/${PROJECT}/civicos"
SQL_INSTANCE="${SQL_INSTANCE:-civicos-pg}"
CONN="${PROJECT}:${REGION}:${SQL_INSTANCE}"
UPLOADS_BUCKET="${UPLOADS_BUCKET:-civicos-ng-prod-uploads}"
APP_URL="${APP_URL:-https://civicos.ng}"

# Flood forecasts are optional infrastructure. The secret is referenced only
# if it actually exists, because --set-secrets against a missing secret fails
# the whole deploy — and this feature must never be the reason a release of
# everything else does not ship.
#
# Access is not self-serve: Google must add your account as a Service
# Consumer before floodforecasting.googleapis.com can even be enabled. Until
# then leave the secret uncreated and community-service simply runs without
# the feature. See docs/deploy-gcp.md.
# Same conditional treatment as the flood key: an optional feature must
# never be the reason a deploy of everything else fails.
GEOCODE_SECRETS=""
if gcloud secrets describe GOOGLE_GEOCODING_API_KEY --project="$PROJECT" >/dev/null 2>&1; then
  GEOCODE_SECRETS=",GOOGLE_GEOCODING_API_KEY=GOOGLE_GEOCODING_API_KEY:latest"
  echo "==> admin location lookup: ENABLED (Google Geocoding)"
else
  echo "==> admin location lookup: off (no GOOGLE_GEOCODING_API_KEY secret)"
fi

FLOOD_SECRETS=""
FLOOD_ENV=""
if gcloud secrets describe GOOGLE_FLOOD_API_KEY --project="$PROJECT" >/dev/null 2>&1; then
  FLOOD_SECRETS=",GOOGLE_FLOOD_API_KEY=GOOGLE_FLOOD_API_KEY:latest"
  FLOOD_ENV="##FLOOD_POLL_INTERVAL_MINUTES=${FLOOD_POLL_INTERVAL_MINUTES:-60}##FLOOD_REGION_CODE=${FLOOD_REGION_CODE:-NG}##FLOOD_MATCH_RADIUS_KM=${FLOOD_MATCH_RADIUS_KM:-50}"
  echo "==> flood forecasts: ENABLED (Google Flood Hub)"
else
  echo "==> flood forecasts: off (no GOOGLE_FLOOD_API_KEY secret)"
fi

# The DB password lives in Secret Manager; pull it rather than keeping a copy.
DBPASS="$(gcloud secrets versions access latest --secret=DB_PASSWORD --project="$PROJECT")"

# Cloud Run has no static egress IP, so Cloud SQL is reached through the socket
# the runtime mounts — hence host=/cloudsql/... rather than an address. Requires
# --add-cloudsql-instances on each service that uses it.
DB_URL="postgresql://civicos:${DBPASS}@localhost/civicos?host=/cloudsql/${CONN}"

# min-instances=0 is what makes this cheap: idle services cost nothing, at the
# price of a 1-2s cold start on the first request after a quiet period.
COMMON=(
  --region="$REGION" --project="$PROJECT" --platform=managed
  --min-instances=0 --max-instances=3 --cpu=1
  --concurrency=80 --timeout=300 --allow-unauthenticated --quiet
)

url_of() {
  gcloud run services describe "$1" --region="$REGION" --project="$PROJECT" \
    --format='value(status.url)' 2>/dev/null
}

# NOTE on the delimiter: the database URL contains `user:password@host`, so
# gcloud's default comma split is unusable and `^@^` would truncate the value
# at the password (the service then dies with "invalid port"). `^##^` cannot
# occur in any of these values.
ENVSEP='^##^'

# ── Backends ────────────────────────────────────────────────────────────────
# Order is load-bearing: identity-service refuses to migrate until the tables
# owned by community and organization exist, and the gateway needs every
# backend URL, so it goes last.
#
# All are --allow-unauthenticated: every service validates the caller's JWT
# itself, so the gateway is a convenience layer rather than the only defence.

echo "==> community  (512Mi: GCS volumes require gen2, which needs >=512Mi)"
gcloud run deploy civicos-community --image="$REPO/community-service:latest" "${COMMON[@]}" \
  --memory=512Mi \
  --add-cloudsql-instances="$CONN" \
  --add-volume=name=uploads,type=cloud-storage,bucket="$UPLOADS_BUCKET" \
  --add-volume-mount=volume=uploads,mount-path=/data \
  --set-env-vars="${ENVSEP}DATABASE_URL=${DB_URL}${FLOOD_ENV}" \
  --set-secrets="JWT_SECRET=JWT_SECRET:latest${FLOOD_SECRETS}${GEOCODE_SECRETS}" 2>&1 | tail -2

echo "==> organization"
gcloud run deploy civicos-organization --image="$REPO/organization-service:latest" "${COMMON[@]}" \
  --memory=256Mi \
  --add-cloudsql-instances="$CONN" \
  --set-env-vars="${ENVSEP}DATABASE_URL=${DB_URL}##APP_URL=${APP_URL}##PLATFORM_FEE_BPS=${PLATFORM_FEE_BPS:-250}##RECONCILE_INTERVAL_MINUTES=${RECONCILE_INTERVAL_MINUTES:-60}##DONATION_CALLBACK_URL=${DONATION_CALLBACK_URL:-${APP_URL}/donations/complete}" \
  --set-secrets="JWT_SECRET=JWT_SECRET:latest,PAYSTACK_SECRET_KEY=PAYSTACK_SECRET_KEY:latest,PAYSTACK_PUBLIC_KEY=PAYSTACK_PUBLIC_KEY:latest" 2>&1 | tail -2

echo "==> identity  (after the two above — it migrates against their tables)"
gcloud run deploy civicos-identity --image="$REPO/identity-service:latest" "${COMMON[@]}" \
  --memory=256Mi \
  --add-cloudsql-instances="$CONN" \
  --set-env-vars="${ENVSEP}DATABASE_URL=${DB_URL}##APP_URL=${APP_URL}" \
  --set-secrets="JWT_SECRET=JWT_SECRET:latest,SMTP_HOST=SMTP_HOST:latest,SMTP_PORT=SMTP_PORT:latest,SMTP_USER=SMTP_USER:latest,SMTP_PASSWORD=SMTP_PASSWORD:latest,SMTP_FROM=SMTP_FROM:latest" 2>&1 | tail -2

IDENTITY_URL="$(url_of civicos-identity)"
COMMUNITY_URL="$(url_of civicos-community)"
ORG_URL="$(url_of civicos-organization)"

echo "==> civicai"
gcloud run deploy civicos-civicai --image="$REPO/civicai-service:latest" "${COMMON[@]}" \
  --memory=256Mi \
  --set-env-vars="GEMINI_MODEL=${GEMINI_MODEL:-gemini-flash-latest},IDENTITY_SERVICE_URL=${IDENTITY_URL},COMMUNITY_SERVICE_URL=${COMMUNITY_URL},ORGANIZATION_SERVICE_URL=${ORG_URL}" \
  --set-secrets="GEMINI_API_KEY=GEMINI_API_KEY:latest,JWT_SECRET=JWT_SECRET:latest" 2>&1 | tail -2

CIVICAI_URL="$(url_of civicos-civicai)"

# REDIS_URL is deliberately absent — Memorystore's cheapest tier (~$35/mo) costs
# more than the rest of this stack combined, and the limiter fails OPEN without
# it (pkg/ratelimit). Set REDIS_URL here to switch rate limiting back on.
echo "==> gateway  (last: needs every backend URL)"
gcloud run deploy civicos-gateway --image="$REPO/api-gateway:latest" "${COMMON[@]}" \
  --memory=256Mi \
  --set-env-vars="IDENTITY_SERVICE_URL=${IDENTITY_URL},COMMUNITY_SERVICE_URL=${COMMUNITY_URL},ORGANIZATION_SERVICE_URL=${ORG_URL},CIVICAI_SERVICE_URL=${CIVICAI_URL}" \
  --set-secrets="JWT_SECRET=JWT_SECRET:latest,PAYSTACK_SECRET_KEY=PAYSTACK_SECRET_KEY:latest,PAYSTACK_PUBLIC_KEY=PAYSTACK_PUBLIC_KEY:latest" 2>&1 | tail -2

GATEWAY_URL="$(url_of civicos-gateway)"

# ── Frontends ───────────────────────────────────────────────────────────────
# Three nginx images, each carrying a prebuilt static bundle.
#
# Vite inlines VITE_API_URL at BUILD time, so the gateway URL is a build arg —
# it cannot be set as a runtime env var. Changing the gateway URL therefore
# requires REBUILDING these images, not just redeploying them.
deploy_frontend() {
  local name="$1" dockerfile="$2"
  echo "==> $name (build)"
  gcloud builds submit . --project="$PROJECT" --region="$REGION" --quiet \
    --config=deploy/gcp/cloudbuild-frontend.yaml \
    --substitutions="_IMAGE=${REPO}/${name}:latest,_DOCKERFILE=${dockerfile},_API_URL=${GATEWAY_URL}" 2>&1 | tail -2
  gcloud run deploy "$name" --image="${REPO}/${name}:latest" "${COMMON[@]}" \
    --memory=256Mi --port=8080 2>&1 | tail -2
}

deploy_frontend civicos-web   apps/web/Dockerfile
deploy_frontend civicos-admin apps/admin/Dockerfile
deploy_frontend civicos-docs  apps/docs/Dockerfile

echo
echo "gateway      $GATEWAY_URL"
echo "identity     $IDENTITY_URL"
echo "community    $COMMUNITY_URL"
echo "organization $ORG_URL"
echo "civicai      $CIVICAI_URL"
echo "web          $(url_of civicos-web)"
echo "admin        $(url_of civicos-admin)"
echo "docs         $(url_of civicos-docs)"
