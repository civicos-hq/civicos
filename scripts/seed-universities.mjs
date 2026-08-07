#!/usr/bin/env node
/**
 * Seeds Nigerian universities as CivicOS communities.
 *
 *   node scripts/seed-universities.mjs --email admin@example.com --password '…'
 *   node scripts/seed-universities.mjs --dry-run
 *
 * Options:
 *   --dry-run          Validate the list and report what would change. No writes.
 *   --gateway <url>    Defaults to $GATEWAY_URL or http://localhost:3000.
 *   --email/--password Credentials for a GOVERNMENT_ADMIN or PLATFORM_ADMIN.
 *                      Falls back to $SEED_ADMIN_EMAIL / $SEED_ADMIN_PASSWORD.
 *
 * ── Why the API and not a migration ──────────────────────────────────
 * A goose migration runs at identity-service boot, and on a fresh database
 * `communities` may not exist yet — community-service creates it via
 * AutoMigrate, and nothing orders the two. The insert would fail, goose
 * would record the version as applied, and the universities would never
 * be seeded: a silent no-op that looks like success.
 *
 * Going through POST /api/v1/communities avoids all of that, runs
 * identically against localhost and production, and passes the same
 * validation every hand-created community does.
 *
 * Idempotent: existing slugs are skipped, so re-running after extending
 * the list only creates what is new.
 */
import { UNIVERSITIES } from './data/universities.mjs';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));

const args = process.argv.slice(2);
const flag = (name, fallback) => {
  const i = args.indexOf(`--${name}`);
  return i !== -1 && args[i + 1] ? args[i + 1] : fallback;
};
const has = (name) => args.includes(`--${name}`);

const GATEWAY = flag('gateway', process.env.GATEWAY_URL ?? 'http://localhost:3000');
const EMAIL = flag('email', process.env.SEED_ADMIN_EMAIL);
const PASSWORD = flag('password', process.env.SEED_ADMIN_PASSWORD);
const DRY_RUN = has('dry-run');

// ─── 1. Validate the list against the canonical LGA data ─────────────
// A wrong LGA is invisible once seeded but mis-tiers the campus in every
// Discover feed in that state, so this runs before anything is written
// and before credentials are even required.

function loadNigeriaData() {
  // nigeria.ts is TypeScript, so it cannot be imported directly here.
  // The shape is a plain literal, so pulling the state/LGA pairs out
  // textually is enough and avoids adding a build step to a seed script.
  const source = readFileSync(join(here, '../apps/web/src/data/nigeria.ts'), 'utf8');
  const states = new Map();
  const blocks = source.matchAll(/name:\s*'([^']+)',\s*code:\s*'[^']+',\s*lgas:\s*\[([\s\S]*?)\]/g);
  for (const [, name, rawLgas] of blocks) {
    // Both quote styles, double-quoted first: names containing an
    // apostrophe ("Jema'a", "Dan Musa") are written with double quotes,
    // and a single-quote-only scan desyncs on them — silently mangling
    // every LGA that follows in the same state.
    const lgas = [...rawLgas.matchAll(/"([^"]*)"|'([^']*)'/g)].map((m) => m[1] ?? m[2]);
    states.set(name, new Set(lgas));
  }
  return states;
}

const states = loadNigeriaData();
if (states.size === 0) {
  console.error('✗ Could not parse apps/web/src/data/nigeria.ts — aborting rather than guessing.');
  process.exit(1);
}

const problems = [];
const slugs = new Set();
for (const u of UNIVERSITIES) {
  if (slugs.has(u.slug)) problems.push(`duplicate slug: ${u.slug}`);
  slugs.add(u.slug);

  const lgas = states.get(u.state);
  if (!lgas) {
    problems.push(`${u.name}: unknown state "${u.state}"`);
    continue;
  }
  if (!lgas.has(u.lga)) {
    problems.push(`${u.name}: "${u.lga}" is not an LGA of ${u.state}`);
  }
}

if (problems.length > 0) {
  console.error(`✗ ${problems.length} problem(s) in the seed list:\n`);
  for (const p of problems) console.error(`   • ${p}`);
  console.error('\nNothing was written. Fix scripts/data/universities.mjs and re-run.');
  process.exit(1);
}
console.log(`✓ ${UNIVERSITIES.length} universities validated against nigeria.ts`);

// ─── 2. Talk to the gateway ──────────────────────────────────────────

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
  return { ok: res.ok, status: res.status, json };
}

const health = await fetch(`${GATEWAY}/health`).catch(() => null);
if (!health?.ok) {
  console.error(`✗ Gateway not reachable at ${GATEWAY}. Is it running?`);
  process.exit(1);
}

// Read the existing communities so a re-run only creates what is missing.
// Paged, because the endpoint is now capped at 100 per request.
const existing = new Set();
for (let offset = 0; ; offset += 100) {
  const res = await api('GET', `/api/v1/communities?limit=100&offset=${offset}`);
  if (!res.ok) {
    console.error(`✗ Could not list communities: ${res.status}`, res.json);
    process.exit(1);
  }
  const page = res.json?.data?.communities ?? [];
  for (const c of page) existing.add(c.slug);
  if (page.length < 100) break;
}

const missing = UNIVERSITIES.filter((u) => !existing.has(u.slug));
console.log(`  ${existing.size} communities already exist; ${missing.length} to create.`);

if (missing.length === 0) {
  console.log('✓ Nothing to do.');
  process.exit(0);
}

if (DRY_RUN) {
  console.log('\nWould create:');
  for (const u of missing) console.log(`   • ${u.name} — ${u.lga}, ${u.state}`);
  console.log('\n(dry run — nothing written)');
  process.exit(0);
}

if (!EMAIL || !PASSWORD) {
  console.error(
    '✗ Admin credentials required to write.\n' +
      '  Pass --email/--password, or set SEED_ADMIN_EMAIL / SEED_ADMIN_PASSWORD.\n' +
      '  Re-run with --dry-run to preview without credentials.',
  );
  process.exit(1);
}

const login = await api('POST', '/api/v1/auth/login', { email: EMAIL, password: PASSWORD });
if (!login.ok) {
  console.error('✗ Login failed:', login.status, login.json);
  process.exit(1);
}
const token = login.json?.data?.tokens?.accessToken ?? login.json?.data?.accessToken;
if (!token) {
  console.error('✗ Login succeeded but no access token in the response:', login.json);
  process.exit(1);
}

const role = login.json?.data?.user?.role;
if (role !== 'PLATFORM_ADMIN' && role !== 'GOVERNMENT_ADMIN') {
  console.error(`✗ ${EMAIL} has role ${role}; creating communities needs an admin role.`);
  process.exit(1);
}

// ─── 3. Create ───────────────────────────────────────────────────────

let created = 0;
const failed = [];
for (const u of missing) {
  const res = await api(
    'POST',
    '/api/v1/communities',
    {
      name: u.name,
      slug: u.slug,
      state: u.state,
      lga: u.lga,
      ...(u.description ? { description: u.description } : {}),
    },
    token,
  );
  if (res.ok) {
    created++;
    console.log(`   + ${u.name}`);
  } else {
    failed.push({ name: u.name, status: res.status, body: res.json });
    console.error(`   ✗ ${u.name} — ${res.status}`);
  }
}

console.log(`\n✓ Created ${created} of ${missing.length}.`);
if (failed.length > 0) {
  console.error(`✗ ${failed.length} failed:`);
  for (const f of failed) console.error(`   • ${f.name}: ${JSON.stringify(f.body)}`);
  process.exit(1);
}
