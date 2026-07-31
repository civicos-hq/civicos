#!/usr/bin/env node
/**
 * Seeds three real, loggable accounts plus a funding-eligible organization,
 * so the whole Community Funding feature can be walked through by hand.
 *
 *   node scripts/seed-funding-demo.mjs
 *
 * Accounts are created through the real register endpoint so the passwords
 * are properly hashed and you can actually sign in. Only the things a user
 * cannot do for themselves — verifying an email, granting PLATFORM_ADMIN,
 * marking an organization verified — are applied directly to the database.
 *
 * Re-runnable: it removes anything it created on a previous run first.
 *
 * Requires: docker compose up, identity + organization + gateway running,
 * and PAYSTACK_SECRET_KEY set (a sandbox sub-account is reused for payouts).
 */
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';

const GATEWAY = process.env.GATEWAY_URL ?? 'http://localhost:3000';
const PAYSTACK = process.env.PAYSTACK_SECRET_KEY;
const PASSWORD = 'CivicOS-Demo-2026';

const psql = (sql) =>
  execFileSync(
    'docker',
    ['exec', 'civicos_postgres', 'psql', '-U', 'civicos', '-d', 'civicos', '-t', '-A', '-c', sql],
    { encoding: 'utf8' },
  ).trim();

async function api(method, path, body, token) {
  const res = await fetch(GATEWAY + path, {
    method,
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(body ? { 'Content-Type': 'application/json' } : {}),
    },
    ...(body ? { body: JSON.stringify(body) } : {}),
  });
  let json = null;
  try {
    json = await res.json();
  } catch {
    /* empty body */
  }
  return { status: res.status, body: json };
}

const ACCOUNTS = [
  { key: 'org', name: 'Ada Bello', email: 'org@civicos.demo', role: 'NGO' },
  { key: 'admin', name: 'Chidi Admin', email: 'admin@civicos.demo', role: 'PLATFORM_ADMIN' },
  { key: 'citizen', name: 'Nosa Citizen', email: 'citizen@civicos.demo', role: 'CITIZEN' },
];

console.log('CivicOS — Community Funding demo seed\n');

// ── Preflight ──────────────────────────────────────────────────────────
const health = await fetch(`${GATEWAY}/health`).catch(() => null);
if (!health?.ok) {
  console.error(`✗ Gateway not reachable at ${GATEWAY}. Start the services first.`);
  process.exit(1);
}
if (!PAYSTACK) {
  console.error('✗ PAYSTACK_SECRET_KEY is not set. Run:  set -a && . .env && set +a');
  process.exit(1);
}
if (!PAYSTACK.startsWith('sk_test_')) {
  console.error('✗ Refusing to run: PAYSTACK_SECRET_KEY is not a test key.');
  process.exit(1);
}

// ── Clean up a previous run ────────────────────────────────────────────
const emails = ACCOUNTS.map((a) => `'${a.email}'`).join(',');
const oldOrg = psql(`SELECT id FROM organizations WHERE slug = 'zaria-relief-demo'`);
if (oldOrg) {
  const cids = psql(
    `SELECT string_agg(quote_literal(id), ',') FROM campaigns WHERE organization_id='${oldOrg}'`,
  );
  if (cids) {
    psql(`DELETE FROM spend_records WHERE campaign_id IN (${cids})`);
    psql(`DELETE FROM progress_updates WHERE campaign_id IN (${cids})`);
    psql(`DELETE FROM donations WHERE campaign_id IN (${cids})`);
    psql(`DELETE FROM milestones WHERE campaign_id IN (${cids})`);
  }
  psql(`DELETE FROM campaigns WHERE organization_id='${oldOrg}'`);
  psql(`DELETE FROM org_members WHERE organization_id='${oldOrg}'`);
  psql(`DELETE FROM organizations WHERE id='${oldOrg}'`);
}
psql(
  `DELETE FROM notifications WHERE user_id IN (SELECT id FROM users WHERE email IN (${emails}))`,
);
psql(`DELETE FROM audit_logs WHERE actor_id IN (SELECT id FROM users WHERE email IN (${emails}))`);
psql(`DELETE FROM users WHERE email IN (${emails})`);
console.log('· cleared any previous demo data');

// ── 1. Create the three accounts through the real register endpoint ────
const ids = {};
for (const acct of ACCOUNTS) {
  const r = await api('POST', '/api/v1/auth/register', {
    name: acct.name,
    email: acct.email,
    password: PASSWORD,
  });
  if (r.status >= 300) {
    console.error(`✗ Could not register ${acct.email}: ${r.status} ${JSON.stringify(r.body)}`);
    process.exit(1);
  }
  ids[acct.key] = psql(`SELECT id FROM users WHERE email='${acct.email}'`);
  // Verify the email and set the role — neither is self-service.
  psql(`UPDATE users SET email_verified=true, role='${acct.role}' WHERE email='${acct.email}'`);
  console.log(`· ${acct.role.padEnd(15)} ${acct.email}`);
}

// ── 2. A funding-eligible organization ─────────────────────────────────
// Every field below is a real requirement of Organization.FundingEligible():
// without all of them the donate endpoint refuses with ORG_NOT_FUNDING_ELIGIBLE.
const subRes = await fetch('https://api.paystack.co/subaccount?perPage=1', {
  headers: { Authorization: `Bearer ${PAYSTACK}` },
});
const subCode = (await subRes.json())?.data?.[0]?.subaccount_code;
if (!subCode) {
  console.error('✗ No Paystack sandbox sub-account found. Create one in the Paystack dashboard.');
  process.exit(1);
}

const orgID = crypto.randomUUID();
psql(`INSERT INTO organizations
  (id,name,slug,kind,jurisdiction,description,state,lga,verified,
   registration_number,country,official_email,representative_name,bank_account_verified,
   psp_provider,psp_subaccount_code,psp_bank_name,psp_account_last4,psp_connected_at,
   created_by_id,created_at,updated_at)
  VALUES ('${orgID}','Zaria Relief Trust','zaria-relief-demo','NGO','LGA',
   'A community relief organization working across Zaria and surrounding wards.',
   'Kaduna','Zaria',true,
   'RC-884213','Nigeria','trust@zaria.demo','Hauwa Bello',true,
   'paystack','${subCode}','Zenith Bank','4417',now(),
   '${ids.org}',now(),now())`);
psql(`INSERT INTO org_members (id,organization_id,user_id,user_name,user_role,role,joined_at)
      VALUES ('${crypto.randomUUID()}','${orgID}','${ids.org}','Ada Bello','NGO','OWNER',now())`);
console.log(`· organization    Zaria Relief Trust (verified, payouts connected)`);

// ── Done ───────────────────────────────────────────────────────────────
const line = '─'.repeat(64);
console.log(`\n${line}\n  Sign in at http://localhost:5173/login\n${line}`);
for (const a of ACCOUNTS) {
  console.log(`  ${a.role.padEnd(15)} ${a.email.padEnd(24)} ${PASSWORD}`);
}
console.log(line);
console.log(`  Organization page: http://localhost:5173/organizations/${orgID}`);
console.log(`  Mailpit (receipts): http://localhost:8025`);
console.log(`${line}\n`);
console.log('Next: follow docs/product/community-funding-demo.md');
