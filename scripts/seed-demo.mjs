#!/usr/bin/env node
/**
 * Populates a deployed CivicOS with mock content for the pre-launch test
 * window, so testers who register find a platform in use rather than an
 * empty one.
 *
 *   node scripts/seed-demo.mjs --dry-run
 *   node scripts/seed-demo.mjs --gateway https://api.civicos.ng \
 *        --admin-email you@example.com --admin-password '…'
 *
 * ── Read this before running it against production ───────────────────
 *
 * 1. THERE IS NO UNDO. CivicOS has no DELETE endpoint for issues,
 *    petitions, communities or campaigns. Nothing this script creates can
 *    be removed through the product — removal means SQL against Cloud
 *    SQL, and the plan of record is a full database reset before launch.
 *    See docs/deploy-gcp.md.
 *
 * 2. IT NEEDS ONE SQL STEP. Creating an issue or petition requires a
 *    verified email, there is no admin endpoint to verify one, and
 *    production email delivery is currently broken. So the script pauses
 *    after creating accounts and prints the UPDATE to run. Nothing after
 *    that point works until it has been applied.
 *
 * 3. IT REFUSES TO ARM REAL PAYMENTS. Campaigns are only publishable once
 *    a Paystack sub-account is connected, at which point a stranger can
 *    donate real money to a fake appeal. Pass --paystack-key so the
 *    script can check it, and it will abort on an sk_live_ key.
 *
 * Requires: an existing PLATFORM_ADMIN account on the target deployment.
 */
import { randomUUID } from 'node:crypto';
import {
  CITIZENS,
  COMMUNITIES,
  ISSUES,
  ORGANIZATIONS,
  PETITIONS,
  REPRESENTATIVES,
} from './data/demo-content.mjs';

const args = process.argv.slice(2);
const flag = (name, fallback) => {
  const i = args.indexOf(`--${name}`);
  return i !== -1 && args[i + 1] ? args[i + 1] : fallback;
};
const has = (name) => args.includes(`--${name}`);

const GATEWAY = flag('gateway', process.env.GATEWAY_URL ?? 'http://localhost:3000');
const ADMIN_EMAIL = flag('admin-email', process.env.SEED_ADMIN_EMAIL);
const ADMIN_PASSWORD = flag('admin-password', process.env.SEED_ADMIN_PASSWORD);
const PAYSTACK_KEY = flag('paystack-key', process.env.PAYSTACK_SECRET_KEY ?? '');
const DRY_RUN = has('dry-run');
const SKIP_CAMPAIGNS = has('skip-campaigns');
const NO_LABELS = has('no-labels');

/**
 * One domain for every seeded account, so they are identifiable at a
 * glance in the users table and in any support conversation. Also what
 * the pre-launch verification SQL keys off.
 */
const EMAIL_DOMAIN = flag('email-domain', 'demo.civicos.ng');
const PASSWORD = flag('password', 'CivicOS-Demo-2026');

/**
 * Prefix on organization and representative display names.
 *
 * A fictional "Hon. Someone" attached to a real constituency on a public
 * site is the one part of this that could genuinely mislead a visitor.
 * --no-labels removes it if you want maximum realism and accept that.
 */
const DEMO_LABEL = NO_LABELS ? '' : '[Demo] ';

const email = (key) => `${key}@${EMAIL_DOMAIN}`;

// ─── HTTP ────────────────────────────────────────────────────────────

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

let failures = 0;
let sawRateLimit = false;
function warn(what, res) {
  failures++;
  if (res.status === 429) sawRateLimit = true;
  console.error(`   ✗ ${what} — ${res.status} ${JSON.stringify(res.json?.message ?? res.json)}`);
}

// ─── Safety gates ────────────────────────────────────────────────────

// Live Paystack key + a published campaign = a fake appeal that can take
// real money from a real person. Refuse, rather than warn: a warning in a
// wall of log output is not a safeguard.
const wantsCampaigns = !SKIP_CAMPAIGNS && ORGANIZATIONS.some((o) => o.campaigns.length > 0);

// Checked before anything is written, but not before --dry-run: a dry run
// writes nothing, and refusing to describe the plan because a key is
// absent would just make the preview useless.
if (wantsCampaigns && !DRY_RUN) {
  if (PAYSTACK_KEY.startsWith('sk_live_')) {
    console.error(
      '✗ Refusing to seed campaigns against a LIVE Paystack key.\n' +
        '  A published mock campaign can accept real donations from anyone who finds it.\n' +
        '  Either switch the deployment to test keys for the test window, or re-run\n' +
        '  with --skip-campaigns.',
    );
    process.exit(1);
  }
  if (!PAYSTACK_KEY) {
    console.error(
      '✗ Campaigns requested but no Paystack key supplied to check.\n' +
        '  Pass --paystack-key (or set PAYSTACK_SECRET_KEY) so this script can confirm\n' +
        '  it is a test key, or re-run with --skip-campaigns.',
    );
    process.exit(1);
  }
  console.log('✓ Paystack key is a test key — mock campaigns cannot take real money');
}

console.log(`\nTarget: ${GATEWAY}`);
console.log(`Accounts: *@${EMAIL_DOMAIN}  ·  password: ${PASSWORD}`);
if (DEMO_LABEL) console.log(`Labelling orgs and representatives with "${DEMO_LABEL.trim()}"`);

if (DRY_RUN) {
  console.log('\n(dry run) Would create:');
  console.log(`   ${COMMUNITIES.length} communities`);
  console.log(`   ${CITIZENS.length} citizens`);
  console.log(`   ${ISSUES.length} issues, ${PETITIONS.length} petitions`);
  console.log(
    `   ${REPRESENTATIVES.length} representatives with ` +
      `${REPRESENTATIVES.reduce((n, r) => n + r.announcements.length, 0)} announcements`,
  );
  const orgCampaigns = ORGANIZATIONS.reduce((n, o) => n + o.campaigns.length, 0);
  console.log(
    `   ${ORGANIZATIONS.length} organizations, ` +
      `${ORGANIZATIONS.reduce((n, o) => n + o.announcements.length, 0)} announcements, ` +
      `${ORGANIZATIONS.reduce((n, o) => n + o.projects.length, 0)} projects, ` +
      `${ORGANIZATIONS.reduce((n, o) => n + o.consultations.length, 0)} consultations, ` +
      `${SKIP_CAMPAIGNS ? 0 : orgCampaigns} campaigns`,
  );
  if (wantsCampaigns) {
    console.log(
      '\n   Campaigns need a Paystack key to be supplied on the real run, and the\n' +
        '   script will abort on an sk_live_ one — a published mock campaign can\n' +
        '   take real money from anyone who finds it.',
    );
  }
  console.log('\nNothing was written.');
  process.exit(0);
}

const health = await fetch(`${GATEWAY}/health`).catch(() => null);
if (!health?.ok) {
  console.error(`✗ Gateway not reachable at ${GATEWAY}`);
  process.exit(1);
}
if (!ADMIN_EMAIL || !ADMIN_PASSWORD) {
  console.error('✗ Admin credentials required. Pass --admin-email and --admin-password.');
  process.exit(1);
}

const login = await api('POST', '/api/v1/auth/login', {
  email: ADMIN_EMAIL,
  password: ADMIN_PASSWORD,
});
if (!login.ok) {
  console.error('✗ Admin login failed:', login.status, login.json);
  process.exit(1);
}
const ADMIN = login.json?.data?.tokens?.accessToken ?? login.json?.data?.accessToken;
if (login.json?.data?.user?.role !== 'PLATFORM_ADMIN') {
  console.error(`✗ ${ADMIN_EMAIL} is ${login.json?.data?.user?.role}, not PLATFORM_ADMIN.`);
  process.exit(1);
}

// ─── 1. Communities ──────────────────────────────────────────────────

console.log('\n== Communities ==');
const communityIds = {};
{
  const existing = await api('GET', '/api/v1/communities?limit=100');
  const bySlug = new Map((existing.json?.data?.communities ?? []).map((c) => [c.slug, c.id]));

  for (const c of COMMUNITIES) {
    if (bySlug.has(c.slug)) {
      communityIds[c.slug] = bySlug.get(c.slug);
      console.log(`   = ${c.name} (exists)`);
      continue;
    }
    const res = await api('POST', '/api/v1/communities', c, ADMIN);
    if (!res.ok) {
      warn(`community ${c.name}`, res);
      continue;
    }
    communityIds[c.slug] = res.json.data.community.id;
    console.log(`   + ${c.name}`);
  }
}

// ─── 2. Accounts ─────────────────────────────────────────────────────
//
// Registered through the real endpoint so passwords are properly hashed
// and every persona can actually be signed into.

console.log('\n== Accounts ==');
const tokens = {};

async function register(key, name, extra = {}) {
  const res = await api('POST', '/api/v1/auth/register', {
    name,
    email: email(key),
    password: PASSWORD,
    ...extra,
  });
  if (res.ok) {
    tokens[key] = res.json?.data?.tokens?.accessToken;
    console.log(`   + ${name} <${email(key)}>`);
    return true;
  }
  if (res.json?.code === 'EMAIL_ALREADY_IN_USE') {
    const again = await api('POST', '/api/v1/auth/login', {
      email: email(key),
      password: PASSWORD,
    });
    if (again.ok) {
      tokens[key] = again.json?.data?.tokens?.accessToken;
      console.log(`   = ${name} (exists)`);
      return true;
    }
  }
  warn(`register ${name}`, res);
  return false;
}

for (const c of CITIZENS) await register(c.key, c.name);
for (const r of REPRESENTATIVES) {
  await register(r.key, `${DEMO_LABEL}${r.name}`, {
    requestedAccountType: 'REPRESENTATIVE',
    representativeApplication: {
      fullName: `${DEMO_LABEL}${r.name}`,
      title: r.title,
      position: r.position,
      constituency: r.constituency,
      communityId: communityIds[r.community],
      party: r.party,
      bio: r.bio,
    },
  });
}
for (const o of ORGANIZATIONS) {
  await register(`org-${o.key}`, o.ownerName, {
    requestedAccountType: 'ORGANIZATION',
    organizationApplication: {
      name: `${DEMO_LABEL}${o.name}`,
      slug: o.slug,
      kind: o.kind,
      jurisdiction: o.jurisdiction,
      state: o.state,
      lga: o.lga,
      description: o.description,
    },
  });
}

// ─── The manual step ─────────────────────────────────────────────────
//
// Everything past this point needs verified accounts, and nothing in the
// API can verify one. Stop here rather than emitting a wall of 403s.

const VERIFY_SQL = `UPDATE users SET email_verified = true, email_verified_at = now() WHERE email LIKE '%@${EMAIL_DOMAIN}';`;

if (!has('verified')) {
  console.log(`
${'─'.repeat(70)}
PAUSE — accounts exist but are unverified.

Creating issues, petitions and representative announcements all require a
verified email. There is no admin endpoint for it, so run this against
Cloud SQL, then re-run this script with --verified:

  ${VERIFY_SQL}

  gcloud sql connect civicos-pg --user=civicos --database=civicos \\
    --project=civicos-ng-prod

Re-running is safe: existing accounts are detected and reused.
${'─'.repeat(70)}
`);
  process.exit(failures > 0 ? 1 : 0);
}

// ─── 3. Approve the applications ─────────────────────────────────────

console.log('\n== Approving applications ==');
const repProfileIds = {};
const orgIds = {};
{
  // One list endpoint for both kinds; each row carries its own `kind` and
  // nests the applicant. Only demo-domain applicants are touched, so this
  // can never approve a real person's pending application by accident.
  const list = await api('GET', '/api/v1/admin/applications?status=PENDING', null, ADMIN);
  if (!list.ok) warn('list pending applications', list);

  for (const app of list.json?.data?.applications ?? []) {
    const applicantEmail = app.applicant?.email ?? '';
    if (!applicantEmail.endsWith(`@${EMAIL_DOMAIN}`)) continue;

    const kind = String(app.kind ?? '').toLowerCase();
    const res = await api(
      'PATCH',
      `/api/v1/admin/applications/${kind}/${app.id}`,
      { status: 'APPROVED' },
      ADMIN,
    );
    if (!res.ok) {
      warn(`approve ${kind} for ${applicantEmail}`, res);
      continue;
    }
    console.log(`   ✓ approved ${kind}: ${applicantEmail}`);
  }

  // Re-read so the approved profiles and orgs can be addressed by id.
  const reps = await api('GET', '/api/v1/representatives');
  for (const r of REPRESENTATIVES) {
    const match = (reps.json?.data?.representatives ?? []).find(
      (x) => x.name === `${DEMO_LABEL}${r.name}`,
    );
    if (match) repProfileIds[r.key] = match.id;
  }
  const orgs = await api('GET', '/api/v1/organizations');
  for (const o of ORGANIZATIONS) {
    const match = (orgs.json?.data?.organizations ?? []).find((x) => x.slug === o.slug);
    if (match) orgIds[o.key] = match.id;
  }
}

// ─── 4. Citizens join their communities ──────────────────────────────

console.log('\n== Memberships ==');
for (const c of CITIZENS) {
  if (!tokens[c.key]) continue;
  const res = await api(
    'POST',
    '/api/v1/auth/me/communities',
    {
      communityIds: [communityIds[c.community]],
      primaryCommunityId: communityIds[c.community],
    },
    tokens[c.key],
  );
  if (!res.ok && res.json?.code !== 'ALREADY_MEMBER') warn(`join ${c.name}`, res);
}
console.log(`   ✓ ${CITIZENS.length} citizens placed in their communities`);

// ─── 5. Issues ───────────────────────────────────────────────────────

console.log('\n== Issues ==');

// Skip anything already present.
//
// Two reasons this matters more than it looks. Re-running is expected —
// the script pauses mid-way for the verification SQL, and any section can
// fail and need another go. And issue/petition creation is rate limited to
// 5 per user per hour, so a blind re-run burns a persona's budget on
// duplicates and then 429s on the content that actually mattered.
const existingIssueTitles = new Set(
  ((await api('GET', '/api/v1/issues')).json?.data?.issues ?? []).map((i) => i.title),
);
const existingPetitionTitles = new Set(
  ((await api('GET', '/api/v1/petitions')).json?.data?.petitions ?? []).map((p) => p.title),
);

for (const issue of ISSUES) {
  if (existingIssueTitles.has(issue.title)) {
    console.log(`   = ${issue.title} (exists)`);
    continue;
  }
  const token = tokens[issue.author];
  if (!token) continue;

  const res = await api(
    'POST',
    '/api/v1/issues',
    {
      title: issue.title,
      description: issue.description,
      category: issue.category,
      communityId: communityIds[issue.community],
      location: issue.location,
    },
    token,
  );
  if (!res.ok) {
    warn(`issue "${issue.title}"`, res);
    continue;
  }
  const id = res.json.data.issue.id;

  // Upvoters must be members of the issue's community to upvote, which
  // the personas already are.
  for (const key of issue.upvoters) {
    if (tokens[key]) await api('POST', `/api/v1/issues/${id}/upvote`, {}, tokens[key]);
  }
  for (const c of issue.comments) {
    if (tokens[c.author]) {
      await api('POST', `/api/v1/issues/${id}/comments`, { content: c.body }, tokens[c.author]);
    }
  }
  // Status is admin-set, so the lifecycle and the timeline both have
  // something to show rather than every issue sitting at OPEN.
  if (issue.status !== 'OPEN') {
    await api('PATCH', `/api/v1/issues/${id}/status`, { status: issue.status }, ADMIN);
  }
  console.log(`   + ${issue.title} (${issue.status})`);
}

// ─── 6. Petitions ────────────────────────────────────────────────────

console.log('\n== Petitions ==');
for (const p of PETITIONS) {
  if (existingPetitionTitles.has(p.title)) {
    console.log(`   = ${p.title} (exists)`);
    continue;
  }
  const token = tokens[p.author];
  if (!token) continue;

  const res = await api(
    'POST',
    '/api/v1/petitions',
    {
      title: p.title,
      description: p.description,
      goal: p.goal,
      communityId: communityIds[p.community],
    },
    token,
  );
  if (!res.ok) {
    warn(`petition "${p.title}"`, res);
    continue;
  }
  const id = res.json.data.petition.id;
  for (const key of p.signers) {
    if (tokens[key]) await api('POST', `/api/v1/petitions/${id}/sign`, {}, tokens[key]);
  }
  console.log(`   + ${p.title} (${p.signers.length}/${p.goal})`);
}

// ─── 7. Representative announcements ─────────────────────────────────

console.log('\n== Representative announcements ==');
for (const r of REPRESENTATIVES) {
  const repId = repProfileIds[r.key];
  const token = tokens[r.key];
  if (!repId || !token) {
    console.error(`   ✗ ${r.name}: profile not linked — skipping announcements`);
    failures++;
    continue;
  }
  for (const a of r.announcements) {
    const res = await api(
      'POST',
      `/api/v1/representatives/${repId}/announcements`,
      { title: a.title, body: a.body },
      token,
    );
    if (!res.ok) {
      warn(`rep announcement "${a.title}"`, res);
      continue;
    }
    if (a.publish) {
      await api(
        'POST',
        `/api/v1/representatives/${repId}/announcements/${res.json.data.announcement.id}/publish`,
        {},
        token,
      );
    }
    console.log(`   + ${r.name}: ${a.title}`);
  }
}

// ─── 8. Organization content ─────────────────────────────────────────

console.log('\n== Organization content ==');
for (const o of ORGANIZATIONS) {
  const orgId = orgIds[o.key];
  const token = tokens[`org-${o.key}`];
  if (!orgId || !token) {
    console.error(`   ✗ ${o.name}: organization not found — skipping its content`);
    failures++;
    continue;
  }

  for (const a of o.announcements) {
    const res = await api(
      'POST',
      `/api/v1/organizations/${orgId}/announcements`,
      { title: a.title, body: a.body },
      token,
    );
    if (!res.ok) {
      warn(`org announcement "${a.title}"`, res);
      continue;
    }
    if (a.publish) {
      await api(
        'POST',
        `/api/v1/announcements/${res.json.data.announcement.id}/publish`,
        {},
        token,
      );
    }
  }

  for (const p of o.projects) {
    const res = await api(
      'POST',
      `/api/v1/organizations/${orgId}/projects`,
      {
        title: p.title,
        description: p.description,
        status: p.status,
        budgetKobo: p.budgetNaira * 100,
        communityId: communityIds[p.community],
      },
      token,
    );
    if (!res.ok) warn(`project "${p.title}"`, res);
  }

  for (const c of o.consultations) {
    const res = await api(
      'POST',
      `/api/v1/organizations/${orgId}/consultations`,
      {
        title: c.title,
        summary: c.summary,
        description: c.description,
        communityId: communityIds[COMMUNITIES[0].slug],
      },
      token,
    );
    if (!res.ok) {
      warn(`consultation "${c.title}"`, res);
      continue;
    }
    const cid = res.json.data.consultation.id;
    for (const [i, q] of c.questions.entries()) {
      await api(
        'POST',
        `/api/v1/consultations/${cid}/questions`,
        { prompt: q.prompt, type: q.type, options: q.options, position: i },
        token,
      );
    }
    if (c.publish) await api('POST', `/api/v1/consultations/${cid}/publish`, {}, token);
  }

  console.log(
    `   + ${o.name}: ${o.announcements.length} announcements, ` +
      `${o.projects.length} projects, ${o.consultations.length} consultations`,
  );
}

// ─── 9. Campaigns ────────────────────────────────────────────────────
//
// Left until last: it is the only section that touches money, and if
// anything above went wrong you want to see that before arming this.

if (SKIP_CAMPAIGNS) {
  console.log('\n== Campaigns == (skipped)');
} else {
  console.log('\n== Campaigns ==');
  for (const o of ORGANIZATIONS) {
    if (!o.campaigns.length) continue;
    const orgId = orgIds[o.key];
    const token = tokens[`org-${o.key}`];
    if (!orgId || !token) continue;

    if (o.fundingReady) {
      // The paperwork FundingEligible() checks. Verification and bank
      // attestation are platform-admin acts, so they use the admin token.
      await api(
        'PATCH',
        `/api/v1/organizations/${orgId}`,
        {
          verified: true,
          registrationNumber: `DEMO-${randomUUID().slice(0, 8).toUpperCase()}`,
          country: 'Nigeria',
          officialEmail: email(`org-${o.key}`),
          representativeName: o.ownerName,
        },
        ADMIN,
      );
      await api(
        'PATCH',
        `/api/v1/organizations/${orgId}/funding-verification`,
        { bankAccountVerified: true },
        ADMIN,
      );
      console.log(`   ✓ ${o.name} marked funding-eligible`);
      console.log(
        `     ⚠ still needs a payout account: connect one in the org dashboard\n` +
          `       (Paystack test mode) before a campaign can be published.`,
      );
    }

    for (const c of o.campaigns) {
      const res = await api(
        'POST',
        `/api/v1/organizations/${orgId}/campaigns`,
        {
          title: c.title,
          summary: c.summary,
          description: c.description,
          category: c.category,
          goalMinor: c.goalNaira * 100,
          currency: 'NGN',
          state: o.state,
          lga: o.lga,
          isEmergency: c.isEmergency,
          communityId: communityIds[c.community],
        },
        token,
      );
      if (!res.ok) {
        warn(`campaign "${c.title}"`, res);
        continue;
      }
      const campaignId = res.json.data.campaign.id;
      for (const m of c.milestones) {
        await api(
          'POST',
          `/api/v1/organizations/${orgId}/campaigns/${campaignId}/milestones`,
          { title: m.title, targetMinor: m.targetNaira * 100 },
          token,
        );
      }
      console.log(`   + ${c.title} (draft — submit and approve it from the console)`);
    }
  }
}

// ─── Done ────────────────────────────────────────────────────────────

console.log(`
${'─'.repeat(70)}
Seeding ${failures === 0 ? 'complete' : `finished with ${failures} failure(s)`}.

Sign in as any persona:  <key>@${EMAIL_DOMAIN} / ${PASSWORD}
  citizens        ${CITIZENS.map((c) => c.key).join(', ')}
  representatives ${REPRESENTATIVES.map((r) => r.key).join(', ')}
  organizations   ${ORGANIZATIONS.map((o) => `org-${o.key}`).join(', ')}

Before launch: reset the database. None of this can be deleted through
the product — see docs/deploy-gcp.md.
${'─'.repeat(70)}
`);

if (sawRateLimit) {
  console.error(
    'Some writes were rate limited (429). Issue and petition creation is\n' +
      'capped at 5 per user per hour by the gateway.\n\n' +
      '  Locally:    docker exec civicos_redis redis-cli FLUSHDB\n' +
      '  Production: REDIS_URL is unset, so the limiter fails open and this\n' +
      '              should not happen — if it did, something is wired up.\n\n' +
      'Then re-run. Content that already exists is skipped, so only the\n' +
      'missing items are retried.',
  );
}
process.exit(failures > 0 ? 1 : 0);
