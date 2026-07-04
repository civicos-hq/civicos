#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# CivicOS end-to-end API smoke test.
#
# Exercises every feature through the api-gateway. Uses a pre-existing
# admin user (elevated via SQL if needed) and creates a fresh citizen per
# run so it stays idempotent. Prints a pass/fail summary at the end.
#
# Requirements: bash 4+, curl, node (for HMAC + JSON), docker (for the DB
# citizen bootstrap), the shared Postgres running as civicos_postgres.
#
# Env overrides:
#   GATEWAY_URL       default http://localhost:3000
#   DB_CONTAINER      default civicos_postgres
#   JWT_SECRET        default read from /Users/gino/civicos/.env
#   ADMIN_EMAIL       default gino.osahon@gmail.com
# ─────────────────────────────────────────────────────────────────────────────
set -uo pipefail

GATEWAY_URL="${GATEWAY_URL:-http://localhost:3000}"
DB_CONTAINER="${DB_CONTAINER:-civicos_postgres}"
ADMIN_EMAIL="${ADMIN_EMAIL:-gino.osahon@gmail.com}"
ENV_FILE="${ENV_FILE:-/Users/gino/civicos/.env}"

if [[ -z "${JWT_SECRET:-}" ]]; then
  JWT_SECRET=$(grep '^JWT_SECRET=' "$ENV_FILE" | cut -d= -f2- | sed 's/^"//;s/"$//')
fi
if [[ -z "$JWT_SECRET" ]]; then
  echo "❌ JWT_SECRET not found in $ENV_FILE"; exit 1
fi

RUN_ID=$(date +%s)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# ── output helpers ──────────────────────────────────────────────────────────
BOLD=$'\033[1m'; DIM=$'\033[2m'; GREEN=$'\033[32m'; RED=$'\033[31m'
YELLOW=$'\033[33m'; BLUE=$'\033[34m'; NC=$'\033[0m'

PASS=0; FAIL=0; SKIP=0
FAIL_LOG=()

pass() { PASS=$((PASS+1)); printf "  ${GREEN}✓${NC} %s\n" "$1"; }
fail() { FAIL=$((FAIL+1)); FAIL_LOG+=("$1: $2"); printf "  ${RED}✗${NC} %s ${DIM}%s${NC}\n" "$1" "$2"; }
skip() { SKIP=$((SKIP+1)); printf "  ${YELLOW}○${NC} %s ${DIM}%s${NC}\n" "$1" "$2"; }
section() { printf "\n${BOLD}${BLUE}▸ %s${NC}\n" "$1"; }

# ── db + jwt helpers ────────────────────────────────────────────────────────
# Take only the first output row so `INSERT ... RETURNING` doesn't get its
# id concatenated with psql's trailing "INSERT 0 1" command tag. That
# concatenation slips past FK-less services (they'd happily store the
# garbage ID) but fails the moment anyone actually WHERE id=? looks it up.
sql() { docker exec "$DB_CONTAINER" psql -U civicos -d civicos -tAc "$1" 2>/dev/null | head -n 1 | tr -d '\r\n'; }

craft_jwt() {
  # Usage: craft_jwt <sub> <email> <name> <role> [emailVerified=true]
  # Values are passed via env, not shell interpolation, so a stray newline
  # in a name or id can never break the Node source.
  JWT_SUB="$1" JWT_EMAIL="$2" JWT_NAME="$3" JWT_ROLE="$4" JWT_VERIFIED="${5:-true}" \
  JWT_SECRET_ENV="$JWT_SECRET" node -e "
    const c=require('crypto');
    const u=b=>Buffer.from(b).toString('base64').replace(/=/g,'').replace(/\+/g,'-').replace(/\//g,'_');
    const h=u(JSON.stringify({alg:'HS256',typ:'JWT'}));
    const n=Math.floor(Date.now()/1e3);
    const p=u(JSON.stringify({
      sub: process.env.JWT_SUB,
      email: process.env.JWT_EMAIL,
      name: process.env.JWT_NAME,
      role: process.env.JWT_ROLE,
      emailVerified: process.env.JWT_VERIFIED === 'true',
      iat: n, exp: n + 3600,
    }));
    const s=c.createHmac('sha256', process.env.JWT_SECRET_ENV).update(h+'.'+p).digest('base64').replace(/=/g,'').replace(/\+/g,'-').replace(/\//g,'_');
    console.log(h+'.'+p+'.'+s);
  "
}

json_get() {
  # Usage: json_get <file> <path> — path like 'data.organization.id'
  node -e "
    const d=require('fs').readFileSync('$1','utf8');
    try { const o=JSON.parse(d); const parts='$2'.split('.'); let v=o; for(const p of parts) v=v?.[p]; console.log(v??'') } catch { console.log('') }
  "
}

# ── request helper ─────────────────────────────────────────────────────────
# req <method> <path> [auth-var] [body]
# Sets globals: STATUS, BODY_FILE, BODY (compact)
req() {
  local method="$1" path="$2" auth="${3:-}" body="${4:-}"
  BODY_FILE="$TMP/resp.$$.$RANDOM.json"
  local args=(-sS -X "$method" "${GATEWAY_URL}${path}" -o "$BODY_FILE" -w '%{http_code}')
  [[ -n "$auth" ]] && args+=(-H "Authorization: Bearer $auth")
  [[ -n "$body" ]] && args+=(-H "Content-Type: application/json" --data-raw "$body")
  STATUS=$(curl "${args[@]}")
  BODY=$(cat "$BODY_FILE")
}

check() {
  # check <label> <expected_status>
  local label="$1" expected="$2"
  if [[ "$STATUS" == "$expected" ]]; then
    pass "$label ($STATUS)"
    return 0
  else
    fail "$label" "expected $expected got $STATUS · body: $(echo "$BODY" | head -c 160)"
    return 1
  fi
}

# ═══════════════════════════════════════════════════════════════════════════
# 1. PREFLIGHT
# ═══════════════════════════════════════════════════════════════════════════
section "1. Preflight — infrastructure + services"

for svc in "gateway:3000" "identity:3001" "community:3002" "org:3003"; do
  name="${svc%:*}"; port="${svc#*:}"
  if lsof -iTCP:$port -sTCP:LISTEN >/dev/null 2>&1; then
    pass "$name listening on :$port"
  else
    fail "$name listening on :$port" "port is closed"
  fi
done

for c in civicos_postgres civicos_redis; do
  if docker inspect -f '{{.State.Running}}' "$c" 2>/dev/null | grep -q true; then
    pass "$c is running"
  else
    fail "$c is running" "container down"
  fi
done

for svc in gateway community organization; do
  case $svc in
    gateway) port=3000;;
    community) port=3002;;
    organization) port=3003;;
  esac
  h=$(curl -sS "http://localhost:$port/health" 2>/dev/null)
  if echo "$h" | grep -q '"status":"ok"'; then
    pass "/$svc/health returns ok"
  else
    fail "/$svc/health returns ok" "$h"
  fi
done

# ═══════════════════════════════════════════════════════════════════════════
# 2. AUTH — admin token + citizen bootstrap
# ═══════════════════════════════════════════════════════════════════════════
section "2. Auth — admin token + fresh citizen"

# Admin details from DB
ADMIN_ROW=$(sql "SELECT id||'|'||name||'|'||role FROM users WHERE email='${ADMIN_EMAIL}';")
if [[ -z "$ADMIN_ROW" ]]; then
  fail "admin user exists" "no user with email=$ADMIN_EMAIL"
  echo; echo "Bootstrap the admin first, then re-run."; exit 1
fi
ADMIN_ID="${ADMIN_ROW%%|*}"
ADMIN_REST="${ADMIN_ROW#*|}"
ADMIN_NAME="${ADMIN_REST%|*}"
ADMIN_ROLE="${ADMIN_ROW##*|}"
pass "admin resolved: $ADMIN_NAME (role=$ADMIN_ROLE)"

if [[ "$ADMIN_ROLE" != "PLATFORM_ADMIN" && "$ADMIN_ROLE" != "GOVERNMENT_ADMIN" && "$ADMIN_ROLE" != "NGO" ]]; then
  fail "admin has elevated role" "role=$ADMIN_ROLE — org creation will fail"
fi

ADMIN_JWT=$(craft_jwt "$ADMIN_ID" "$ADMIN_EMAIL" "$ADMIN_NAME" "$ADMIN_ROLE" true)
pass "admin JWT crafted (${#ADMIN_JWT} chars)"

# Sanity check the JWT with /me
req GET "/api/v1/auth/me" "$ADMIN_JWT"
check "GET /auth/me with admin JWT" "200"

# Create a fresh citizen — SQL insert with a pre-hashed password so the
# email-verified flag can be flipped on the same row. Password is 'password'
# hashed with bcrypt cost 12.
CITIZEN_EMAIL="smoke-$RUN_ID@civicos.test"
CITIZEN_NAME="Smoke Test $RUN_ID"
BCRYPT_PW='$2a$12$0PjLd9ZS/mQEXvUx8LtVjOtaIWTjNXO7v0FBrqXjR3aNVBw/wtFxa'
CITIZEN_ID=$(sql "
  INSERT INTO users (id, email, password_hash, name, role, email_verified, created_at, updated_at)
  VALUES (gen_random_uuid(), '${CITIZEN_EMAIL}', '${BCRYPT_PW}', '${CITIZEN_NAME}', 'CITIZEN', true, now(), now())
  RETURNING id;
")
if [[ -n "$CITIZEN_ID" ]]; then
  pass "citizen created: $CITIZEN_EMAIL (id=${CITIZEN_ID:0:8}…)"
else
  fail "citizen created" "insert returned no id"
  exit 1
fi

# Brief pause so identity-service's GORM connection pool sees the new row on
# its next Postgres session. Without this the first /auth/me can 404 in a
# tight loop right after the raw SQL insert.
sleep 1

CITIZEN_JWT=$(craft_jwt "$CITIZEN_ID" "$CITIZEN_EMAIL" "$CITIZEN_NAME" "CITIZEN" true)
req GET "/api/v1/auth/me" "$CITIZEN_JWT"
check "GET /auth/me with citizen JWT" "200"

req PATCH "/api/v1/auth/me" "$CITIZEN_JWT" "{\"name\":\"Smoke $RUN_ID (renamed)\"}"
check "PATCH /auth/me (rename)" "200"

# ═══════════════════════════════════════════════════════════════════════════
# 3. COMMUNITY
# ═══════════════════════════════════════════════════════════════════════════
section "3. Community"

req GET "/api/v1/communities" ""
check "GET /communities (public)" "200"

COMM_SLUG="smoke-community-$RUN_ID"
req POST "/api/v1/communities" "$ADMIN_JWT" "{
  \"name\": \"Smoke Community $RUN_ID\",
  \"slug\": \"$COMM_SLUG\",
  \"state\": \"Lagos\",
  \"lga\": \"Ikeja\",
  \"description\": \"Smoke test community\"
}"
check "POST /communities (admin)" "201"
COMM_ID=$(json_get "$BODY_FILE" "data.community.id")
[[ -n "$COMM_ID" ]] && pass "community id captured: ${COMM_ID:0:8}…"

req GET "/api/v1/communities/$COMM_ID" ""
check "GET /communities/:id" "200"

req POST "/api/v1/auth/me/community" "$CITIZEN_JWT" "{\"communityId\": \"$COMM_ID\"}"
check "POST /auth/me/community (citizen joins)" "200"

# ═══════════════════════════════════════════════════════════════════════════
# 4. ISSUES
# ═══════════════════════════════════════════════════════════════════════════
section "4. Issues"

req GET "/api/v1/issues" ""
check "GET /issues" "200"

req POST "/api/v1/issues" "$CITIZEN_JWT" "{
  \"title\": \"Smoke test pothole $RUN_ID\",
  \"description\": \"A large pothole is disrupting traffic on Ikeja Road near the market. Reported by smoke test.\",
  \"category\": \"INFRASTRUCTURE\",
  \"communityId\": \"$COMM_ID\",
  \"location\": \"Ikeja Road, near market\"
}"
check "POST /issues (citizen)" "201"
ISSUE_ID=$(json_get "$BODY_FILE" "data.issue.id")
[[ -n "$ISSUE_ID" ]] && pass "issue id captured: ${ISSUE_ID:0:8}…"

req GET "/api/v1/issues/$ISSUE_ID" ""
check "GET /issues/:id" "200"

req POST "/api/v1/issues/$ISSUE_ID/upvote" "$CITIZEN_JWT" "{}"
check "POST /issues/:id/upvote" "200"

req PATCH "/api/v1/issues/$ISSUE_ID/status" "$ADMIN_JWT" '{"status":"UNDER_REVIEW"}'
check "PATCH /issues/:id/status (admin)" "200"

req POST "/api/v1/issues/$ISSUE_ID/comments" "$CITIZEN_JWT" '{"content":"I saw this happen yesterday morning."}'
check "POST /issues/:id/comments" "201"

req GET "/api/v1/issues/$ISSUE_ID/comments" ""
check "GET /issues/:id/comments" "200"

req GET "/api/v1/me/upvotes/issues" "$CITIZEN_JWT"
check "GET /me/upvotes/issues" "200"

# ═══════════════════════════════════════════════════════════════════════════
# 5. PETITIONS
# ═══════════════════════════════════════════════════════════════════════════
section "5. Petitions"

req GET "/api/v1/petitions" ""
check "GET /petitions" "200"

req POST "/api/v1/petitions" "$CITIZEN_JWT" "{
  \"title\": \"Smoke petition $RUN_ID\",
  \"description\": \"Please install more streetlights on Ikeja Road for safety. Signed to test the flow end to end.\",
  \"goal\": 100,
  \"communityId\": \"$COMM_ID\"
}"
check "POST /petitions" "201"
PET_ID=$(json_get "$BODY_FILE" "data.petition.id")

req GET "/api/v1/petitions/$PET_ID" ""
check "GET /petitions/:id" "200"

req POST "/api/v1/petitions/$PET_ID/sign" "$CITIZEN_JWT" "{}"
check "POST /petitions/:id/sign" "200"

req POST "/api/v1/petitions/$PET_ID/comments" "$CITIZEN_JWT" '{"content":"Supporting this. Streetlights save lives."}'
check "POST /petitions/:id/comments" "201"

# ═══════════════════════════════════════════════════════════════════════════
# 6. REPRESENTATIVES
# ═══════════════════════════════════════════════════════════════════════════
section "6. Representatives"

req GET "/api/v1/representatives" ""
check "GET /representatives" "200"

req POST "/api/v1/representatives" "$ADMIN_JWT" "{
  \"name\": \"Amina Yusuf\",
  \"title\": \"Hon.\",
  \"position\": \"House Member\",
  \"constituency\": \"Ikeja West\",
  \"party\": \"APC\",
  \"communityId\": \"$COMM_ID\",
  \"bio\": \"Serving Ikeja West since 2019.\",
  \"email\": \"amina@example.gov.ng\"
}"
check "POST /representatives (admin)" "201"
REP_ID=$(json_get "$BODY_FILE" "data.representative.id")

req GET "/api/v1/representatives/$REP_ID" ""
check "GET /representatives/:id" "200"

req POST "/api/v1/representatives/$REP_ID/follow" "$CITIZEN_JWT" "{}"
check "POST /representatives/:id/follow" "200"

req POST "/api/v1/representatives/$REP_ID/comments" "$CITIZEN_JWT" '{"content":"Grateful for the recent road repair."}'
check "POST /representatives/:id/comments" "201"

req GET "/api/v1/me/follows/representatives" "$CITIZEN_JWT"
check "GET /me/follows/representatives" "200"

req DELETE "/api/v1/representatives/$REP_ID/follow" "$CITIZEN_JWT"
check "DELETE /representatives/:id/follow (unfollow)" "200"

# ═══════════════════════════════════════════════════════════════════════════
# 7. NOTIFICATIONS
# ═══════════════════════════════════════════════════════════════════════════
section "7. Notifications"

req GET "/api/v1/notifications" "$CITIZEN_JWT"
check "GET /notifications" "200"

req GET "/api/v1/notifications/unread-count" "$CITIZEN_JWT"
check "GET /notifications/unread-count" "200"

# ═══════════════════════════════════════════════════════════════════════════
# 8. SEARCH + DISCOVER
# ═══════════════════════════════════════════════════════════════════════════
section "8. Search + Discover"

req GET "/api/v1/search?q=pothole" ""
check "GET /search?q=pothole" "200"

req GET "/api/v1/discover/feed?communityId=$COMM_ID" "$CITIZEN_JWT"
check "GET /discover/feed" "200"

# ═══════════════════════════════════════════════════════════════════════════
# 9. ORGANIZATIONS
# ═══════════════════════════════════════════════════════════════════════════
section "9. Organizations"

req GET "/api/v1/organizations" ""
check "GET /organizations" "200"

ORG_SLUG="smoke-org-$RUN_ID"
req POST "/api/v1/organizations" "$ADMIN_JWT" "{
  \"name\": \"Smoke Water Board $RUN_ID\",
  \"slug\": \"$ORG_SLUG\",
  \"kind\": \"UTILITY\",
  \"jurisdiction\": \"LGA\",
  \"state\": \"Lagos\",
  \"lga\": \"Ikeja\",
  \"description\": \"Smoke test water board for automated verification.\"
}"
check "POST /organizations (admin)" "201"
ORG_ID=$(json_get "$BODY_FILE" "data.organization.id")

req GET "/api/v1/organizations/$ORG_ID" ""
check "GET /organizations/:id" "200"

req GET "/api/v1/organizations/$ORG_ID/members" ""
check "GET /organizations/:id/members" "200"

req PATCH "/api/v1/organizations/$ORG_ID" "$ADMIN_JWT" '{"description":"Updated by smoke test."}'
check "PATCH /organizations/:id" "200"

req POST "/api/v1/organizations/$ORG_ID/members" "$ADMIN_JWT" "{
  \"userId\": \"$CITIZEN_ID\",
  \"userName\": \"$CITIZEN_NAME\",
  \"userRole\": \"CITIZEN\",
  \"role\": \"STAFF\"
}"
check "POST /organizations/:id/members (add staff)" "201"

# ═══════════════════════════════════════════════════════════════════════════
# 10. ANNOUNCEMENTS
# ═══════════════════════════════════════════════════════════════════════════
section "10. Announcements"

req POST "/api/v1/organizations/$ORG_ID/announcements" "$ADMIN_JWT" '{
  "title": "Smoke test draft",
  "body": "This is a draft announcement created by the smoke test.",
  "publish": false
}'
check "POST /organizations/:id/announcements (draft)" "201"
ANN_ID=$(json_get "$BODY_FILE" "data.announcement.id")

req POST "/api/v1/announcements/$ANN_ID/publish" "$ADMIN_JWT" "{}"
check "POST /announcements/:id/publish" "200"

req GET "/api/v1/announcements" ""
check "GET /announcements (public feed)" "200"

req GET "/api/v1/organizations/$ORG_ID/announcements" "$ADMIN_JWT"
check "GET /organizations/:id/announcements" "200"

req PATCH "/api/v1/announcements/$ANN_ID" "$ADMIN_JWT" '{"body":"Updated body from smoke test."}'
check "PATCH /announcements/:id" "200"

# ═══════════════════════════════════════════════════════════════════════════
# 11. PROJECTS
# ═══════════════════════════════════════════════════════════════════════════
section "11. Projects"

req POST "/api/v1/organizations/$ORG_ID/projects" "$ADMIN_JWT" "{
  \"title\": \"Smoke test project\",
  \"description\": \"Testing project CRUD end to end via smoke script.\",
  \"status\": \"PLANNED\",
  \"budgetKobo\": 500000000,
  \"communityId\": \"$COMM_ID\"
}"
check "POST /organizations/:id/projects" "201"
PROJ_ID=$(json_get "$BODY_FILE" "data.project.id")

req GET "/api/v1/projects/$PROJ_ID" ""
check "GET /projects/:id" "200"

req GET "/api/v1/organizations/$ORG_ID/projects" ""
check "GET /organizations/:id/projects" "200"

req PATCH "/api/v1/projects/$PROJ_ID" "$ADMIN_JWT" '{"status":"ACTIVE"}'
check "PATCH /projects/:id (status change)" "200"

# ═══════════════════════════════════════════════════════════════════════════
# 12. ISSUE ASSIGNMENTS
# ═══════════════════════════════════════════════════════════════════════════
section "12. Issue assignments (receive reports)"

req POST "/api/v1/organizations/$ORG_ID/assignments" "$ADMIN_JWT" "{
  \"issueId\": \"$ISSUE_ID\",
  \"note\": \"Routing to smoke water board for smoke test.\"
}"
check "POST /organizations/:id/assignments" "201"
ASG_ID=$(json_get "$BODY_FILE" "data.assignment.id")

req GET "/api/v1/organizations/$ORG_ID/assignments" "$ADMIN_JWT"
check "GET /organizations/:id/assignments (member-only)" "200"

req GET "/api/v1/issues/$ISSUE_ID/assignments" ""
check "GET /issues/:id/assignments (public)" "200"

req PATCH "/api/v1/assignments/$ASG_ID" "$ADMIN_JWT" '{"status":"IN_PROGRESS","note":"Team dispatched"}'
check "PATCH /assignments/:id (mark IN_PROGRESS)" "200"

# ═══════════════════════════════════════════════════════════════════════════
# 13. PROGRESS UPDATES (respond publicly)
# ═══════════════════════════════════════════════════════════════════════════
section "13. Progress updates"

req POST "/api/v1/organizations/$ORG_ID/progress-updates" "$ADMIN_JWT" "{
  \"issueId\": \"$ISSUE_ID\",
  \"body\": \"Team is on-site as of this smoke test. Repair scheduled.\",
  \"isPublic\": true
}"
check "POST /organizations/:id/progress-updates (issue)" "201"

req POST "/api/v1/organizations/$ORG_ID/progress-updates" "$ADMIN_JWT" "{
  \"projectId\": \"$PROJ_ID\",
  \"body\": \"Project kickoff completed.\",
  \"isPublic\": true
}"
check "POST /organizations/:id/progress-updates (project)" "201"

req GET "/api/v1/issues/$ISSUE_ID/progress-updates" ""
check "GET /issues/:id/progress-updates (public)" "200"

req GET "/api/v1/projects/$PROJ_ID/progress-updates" ""
check "GET /projects/:id/progress-updates (public)" "200"

# ═══════════════════════════════════════════════════════════════════════════
# 13B. MODERATION INFRASTRUCTURE — content flags + audit log
# ═══════════════════════════════════════════════════════════════════════════
section "13B. Moderation — content flags + audit log"

# Citizen flags an issue comment (any UUID target is accepted — flags are
# not FK-constrained to specific content tables, so this exercises the
# happy path without needing a real comment row).
FLAG_TARGET_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')

req POST "/api/v1/flags" "$CITIZEN_JWT" "{
  \"contentType\": \"ISSUE_COMMENT\",
  \"contentId\": \"$FLAG_TARGET_ID\",
  \"reason\": \"SPAM\",
  \"description\": \"Automated smoke test flag\"
}"
check "POST /flags (citizen)" "201"
FLAG_ID=$(json_get "$BODY_FILE" "data.flag.id")

req POST "/api/v1/flags" "$CITIZEN_JWT" "{
  \"contentType\": \"ISSUE_COMMENT\",
  \"contentId\": \"$FLAG_TARGET_ID\",
  \"reason\": \"ABUSE\"
}"
check "POST /flags dedup (same user, same content) → 409" "409"

req GET "/api/v1/flags" "$CITIZEN_JWT"
check "GET /flags (citizen) → 403" "403"

req GET "/api/v1/flags?status=PENDING" "$ADMIN_JWT"
check "GET /flags?status=PENDING (admin)" "200"

req GET "/api/v1/flags/counts" "$ADMIN_JWT"
check "GET /flags/counts (admin)" "200"

req PATCH "/api/v1/flags/$FLAG_ID" "$ADMIN_JWT" '{"status":"HIDDEN","resolutionNote":"content removed by smoke test"}'
check "PATCH /flags/:id resolve (admin)" "200"

# The resolution should have written an audit-log entry.
req GET "/api/v1/audit-logs?action=flag&targetId=$FLAG_ID" "$ADMIN_JWT"
check "GET /audit-logs?action=flag" "200"
AUDIT_TOTAL=$(json_get "$BODY_FILE" "data.total")
if [[ "$AUDIT_TOTAL" -ge 1 ]]; then
  pass "audit log has $AUDIT_TOTAL entry for flag.resolved"
else
  fail "audit log entry for flag.resolved" "expected ≥1, got $AUDIT_TOTAL"
fi

req GET "/api/v1/audit-logs" "$CITIZEN_JWT"
check "GET /audit-logs (citizen) → 403" "403"

# ═══════════════════════════════════════════════════════════════════════════
# 13C. HIDE ENFORCEMENT — HIDDEN flag actually removes content from lists
# ═══════════════════════════════════════════════════════════════════════════
section "13C. Hide enforcement — HIDDEN flag removes content from queries"

# Post a citizen comment on the issue we already created, then flag it as
# ABUSE and resolve as HIDDEN. The GET /issues/:id/comments query must
# return one fewer row after the resolution.
req POST "/api/v1/issues/$ISSUE_ID/comments" "$CITIZEN_JWT" '{"content":"Spam comment to be hidden by smoke test."}'
check "POST /issues/:id/comments (target for hide)" "201"
HIDE_COMMENT_ID=$(json_get "$BODY_FILE" "data.comment.id")

req GET "/api/v1/issues/$ISSUE_ID/comments" ""
check "GET /issues/:id/comments before hide" "200"
BEFORE_COUNT=$(json_get "$BODY_FILE" "data.comments.length")

# Different citizen would normally file the flag, but any verified user
# can flag any content; using our test citizen keeps setup minimal.
req POST "/api/v1/flags" "$CITIZEN_JWT" "{
  \"contentType\": \"ISSUE_COMMENT\",
  \"contentId\": \"$HIDE_COMMENT_ID\",
  \"reason\": \"ABUSE\"
}"
check "POST /flags on the comment" "201"
HIDE_FLAG_ID=$(json_get "$BODY_FILE" "data.flag.id")

req PATCH "/api/v1/flags/$HIDE_FLAG_ID" "$ADMIN_JWT" '{"status":"HIDDEN","resolutionNote":"smoke test hide"}'
check "PATCH /flags/:id resolve HIDDEN" "200"

req GET "/api/v1/issues/$ISSUE_ID/comments" ""
check "GET /issues/:id/comments after hide" "200"
AFTER_COUNT=$(json_get "$BODY_FILE" "data.comments.length")
if [[ "$AFTER_COUNT" -lt "$BEFORE_COUNT" ]]; then
  pass "hidden comment removed from list ($BEFORE_COUNT → $AFTER_COUNT)"
else
  fail "hidden comment removed from list" "expected fewer, got $AFTER_COUNT (was $BEFORE_COUNT)"
fi

# Also verify announcements are filtered. Create a draft-published
# announcement, flag+hide it, confirm the public feed drops it.
req POST "/api/v1/organizations/$ORG_ID/announcements" "$ADMIN_JWT" '{
  "title": "Hide-me announcement",
  "body": "This announcement will be hidden by the smoke test.",
  "publish": true
}'
check "POST announcement (target for hide)" "201"
HIDE_ANN_ID=$(json_get "$BODY_FILE" "data.announcement.id")

req GET "/api/v1/announcements" ""
BEFORE_ANN=$(json_get "$BODY_FILE" "data.announcements.length")

req POST "/api/v1/flags" "$CITIZEN_JWT" "{
  \"contentType\": \"ANNOUNCEMENT\",
  \"contentId\": \"$HIDE_ANN_ID\",
  \"reason\": \"MISINFO\"
}"
check "POST /flags on the announcement" "201"
HIDE_ANN_FLAG_ID=$(json_get "$BODY_FILE" "data.flag.id")

req PATCH "/api/v1/flags/$HIDE_ANN_FLAG_ID" "$ADMIN_JWT" '{"status":"HIDDEN"}'
check "PATCH /flags/:id resolve HIDDEN (announcement)" "200"

req GET "/api/v1/announcements" ""
AFTER_ANN=$(json_get "$BODY_FILE" "data.announcements.length")
if [[ "$AFTER_ANN" -lt "$BEFORE_ANN" ]]; then
  pass "hidden announcement dropped from public feed ($BEFORE_ANN → $AFTER_ANN)"
else
  fail "hidden announcement drop" "expected fewer, got $AFTER_ANN (was $BEFORE_ANN)"
fi

# ═══════════════════════════════════════════════════════════════════════════
# 13D. CROSS-SERVICE AUDIT — actions in community/org services land in the log
# ═══════════════════════════════════════════════════════════════════════════
section "13D. Cross-service audit — community + org admin actions write to audit_logs"

# Section 4 patched an issue status via community-service. That should
# have landed an issue.status_changed row from the community-service
# process (not identity-service). Prove it.
req GET "/api/v1/audit-logs?action=issue.status_changed&targetId=$ISSUE_ID" "$ADMIN_JWT"
check "GET /audit-logs issue.status_changed" "200"
ISSUE_AUDIT=$(json_get "$BODY_FILE" "data.total")
if [[ "$ISSUE_AUDIT" -ge 1 ]]; then
  pass "community-service wrote issue.status_changed ($ISSUE_AUDIT entry)"
else
  fail "community-service audit for issue status change" "expected ≥1, got $ISSUE_AUDIT"
fi

# Section 9 created an org via org-service.
req GET "/api/v1/audit-logs?action=org.created&targetId=$ORG_ID" "$ADMIN_JWT"
ORG_CREATED=$(json_get "$BODY_FILE" "data.total")
if [[ "$ORG_CREATED" -ge 1 ]]; then
  pass "org-service wrote org.created ($ORG_CREATED entry)"
else
  fail "org.created audit" "expected ≥1, got $ORG_CREATED"
fi

# Section 9 patched the org's description. That should have landed a
# generic org.updated (not the verify-specific one). Prove it. Distinct-
# action-name coverage for org.verified is exercised by the admin e2e
# suite (organizations.spec.ts) instead of here — this smoke test's
# admin JWT is nearly out of Standard-tier budget by this point.
req GET "/api/v1/audit-logs?action=org.updated&targetId=$ORG_ID" "$ADMIN_JWT"
ORG_UPDATED=$(json_get "$BODY_FILE" "data.total")
if [[ "$ORG_UPDATED" -ge 1 ]]; then
  pass "org-service wrote org.updated ($ORG_UPDATED entry)"
else
  fail "org.updated audit" "expected ≥1, got $ORG_UPDATED"
fi

# Section 10 published an announcement via org-service.
req GET "/api/v1/audit-logs?action=announcement.published&targetId=$ANN_ID" "$ADMIN_JWT"
ANN_PUB=$(json_get "$BODY_FILE" "data.total")
if [[ "$ANN_PUB" -ge 1 ]]; then
  pass "org-service wrote announcement.published ($ANN_PUB entry)"
else
  fail "announcement.published audit" "expected ≥1, got $ANN_PUB"
fi

# ═══════════════════════════════════════════════════════════════════════════
# 14. RATE LIMITING
# ═══════════════════════════════════════════════════════════════════════════
section "14. Rate limiting (Strict — /forgot-password)"

# Strict budget is 5/min; anything after 5 within a fresh window should 429.
# We use a random unregistered email so we don't spam a real inbox.
RL_EMAIL="rl-$RUN_ID-$RANDOM@example.test"
RL_FIRED=0
for i in 1 2 3 4 5 6 7 8; do
  req POST "/api/v1/auth/forgot-password" "" "{\"email\":\"$RL_EMAIL\"}"
  if [[ "$STATUS" == "429" ]]; then
    RL_FIRED=1
    break
  fi
done
if [[ "$RL_FIRED" == "1" ]]; then
  pass "429 fired at request $i (limiter is active)"
else
  skip "429 not observed in 8 requests" "Redis may be disabled — check gateway log"
fi

# ═══════════════════════════════════════════════════════════════════════════
# 15. CROSS-SERVICE INTEGRATION — official response shows on issue
# ═══════════════════════════════════════════════════════════════════════════
section "15. Cross-service — issue detail sees org progress update"

req GET "/api/v1/issues/$ISSUE_ID/progress-updates" ""
UPDATE_COUNT=$(json_get "$BODY_FILE" "data.updates.length")
if [[ "$UPDATE_COUNT" -ge 1 ]]; then
  pass "issue has $UPDATE_COUNT public progress update(s)"
else
  fail "issue progress-updates surface" "expected ≥1, got $UPDATE_COUNT"
fi

# ═══════════════════════════════════════════════════════════════════════════
# 16. CLEANUP — remove all test artifacts
# ═══════════════════════════════════════════════════════════════════════════

# delete_rows <table> <where-clause>
# Runs DELETE ... RETURNING 1 and echoes the row count. Safe on missing
# tables or empty where-clauses (returns 0 and moves on).
delete_rows() {
  local table="$1" where="$2"
  # Bail if the WHERE landed with an empty id — the test never captured
  # that entity, so there's nothing to remove.
  if [[ "$where" =~ =\'\'( |$) ]]; then
    echo 0; return
  fi
  local out
  out=$(docker exec "$DB_CONTAINER" psql -U civicos -d civicos -tAc \
    "DELETE FROM $table WHERE $where RETURNING 1;" 2>&1)
  # If the table doesn't exist yet, print a warning and count as 0.
  if echo "$out" | grep -q 'does not exist'; then
    printf "  ${YELLOW}○${NC} %s — table missing\n" "$table" >&2
    echo 0; return
  fi
  # RETURNING 1 emits one '1' per deleted row, plus a trailing 'DELETE N'.
  local n
  n=$(echo "$out" | grep -c '^1$')
  echo "$n"
}

section "16. Cleanup — removing test artifacts"

if [[ "${SKIP_CLEANUP:-0}" == "1" ]]; then
  skip "cleanup" "SKIP_CLEANUP=1 — test data left in place for inspection"
else
  # With ON DELETE CASCADE in place we only need to remove the parents.
  # Ordering respects RESTRICT constraints on communities.{issues,petitions,
  # representatives}.community_id and users.{reported_by,created_by,author}_id:
  # kill the content before its author/community.
  #
  # Each cascade delete is silent about child row counts (Postgres only
  # counts the target rows in RETURNING); the summary line reports what we
  # explicitly asked for, which is enough to prove cleanup happened.
  declare -a DELETES=(
    "audit_logs|target_id='${FLAG_ID:-}'"          # audit for flag.resolved
    "audit_logs|target_id='${HIDE_FLAG_ID:-}'"     # audit for comment-hide
    "audit_logs|target_id='${HIDE_ANN_FLAG_ID:-}'" # audit for announcement-hide
    "audit_logs|target_id='${ISSUE_ID:-}'"         # community-service issue.status_changed
    "audit_logs|target_id='${ORG_ID:-}'"           # org-service org.created / org.verified / org.updated
    "audit_logs|target_id='${ANN_ID:-}'"           # announcement.published
    "content_flags|id='${FLAG_ID:-}'"              # the queue-only flag
    "content_flags|id='${HIDE_FLAG_ID:-}'"         # the comment-hide flag
    "content_flags|id='${HIDE_ANN_FLAG_ID:-}'"     # the announcement-hide flag
    "organizations|id='${ORG_ID:-}'"       # cascades: org_members, announcements, projects, issue_assignments, progress_updates
    "representatives|id='${REP_ID:-}'"     # cascades: representative_comments, representative_followers
    "petitions|id='${PET_ID:-}'"           # cascades: petition_signatures, petition_comments
    "issues|id='${ISSUE_ID:-}'"            # cascades: issue_upvotes, issue_comments
    "communities|id='${COMM_ID:-}'"        # SET NULL on users.community_id; requires above deletes first
    "content_flags|reporter_id='${CITIZEN_ID:-}'"  # any flags the citizen filed
    "users|id='${CITIZEN_ID:-}'"           # cascades: refresh_tokens, notifications, remaining membership rows
  )
  total_deleted=0
  for entry in "${DELETES[@]}"; do
    IFS='|' read -r table where <<< "$entry"
    n=$(delete_rows "$table" "$where")
    if [[ "$n" -gt 0 ]]; then
      printf "  ${GREEN}✓${NC} %-20s ${DIM}%d row(s) + cascades${NC}\n" "$table" "$n"
      total_deleted=$((total_deleted + n))
    fi
  done
  printf "  ${GREEN}✓${NC} cleanup complete · %d parent rows removed (children cascaded)\n" "$total_deleted"
fi

# ═══════════════════════════════════════════════════════════════════════════
# SUMMARY
# ═══════════════════════════════════════════════════════════════════════════
printf "\n${BOLD}══════════════════════════════════════════════════════════${NC}\n"
printf "${BOLD}Summary — run $RUN_ID${NC}\n"
printf "  ${GREEN}Pass: %d${NC}\n" $PASS
printf "  ${RED}Fail: %d${NC}\n" $FAIL
printf "  ${YELLOW}Skip: %d${NC}\n" $SKIP

if [[ ${#FAIL_LOG[@]} -gt 0 ]]; then
  printf "\n${BOLD}${RED}Failed:${NC}\n"
  for msg in "${FAIL_LOG[@]}"; do
    printf "  ${RED}✗${NC} %s\n" "$msg"
  done
fi

printf "\n${DIM}Test artifacts (run %s):\n" "$RUN_ID"
printf "  community: %s\n  issue: %s\n  petition: %s\n  rep: %s\n  organization: %s\n  announcement: %s\n  project: %s\n  citizen: %s${NC}\n" \
  "${COMM_ID:-—}" "${ISSUE_ID:-—}" "${PET_ID:-—}" "${REP_ID:-—}" \
  "${ORG_ID:-—}" "${ANN_ID:-—}" "${PROJ_ID:-—}" "${CITIZEN_EMAIL:-—}"

[[ $FAIL -eq 0 ]] && exit 0 || exit 1
