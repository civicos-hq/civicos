import { createHmac } from 'node:crypto';
import { sql, jwtSecret } from './db';

export interface TestUser {
  id: string;
  email: string;
  name: string;
  role: 'CITIZEN' | 'GOVERNMENT_ADMIN' | 'PLATFORM_ADMIN' | 'NGO';
  emailVerified: boolean;
  accessToken: string;
}

const ADMIN_EMAIL = process.env.CIVICOS_ADMIN_EMAIL ?? 'gino.osahon@gmail.com';

// Reuses the existing admin (elevated to PLATFORM_ADMIN earlier) — same
// account you use in the browser. We just craft a fresh access token for
// them so specs never depend on interactive login.
export function loadAdmin(): TestUser {
  const row = sql(
    `SELECT id||'|'||name FROM users WHERE email='${ADMIN_EMAIL}' AND role='PLATFORM_ADMIN';`,
  );
  if (!row) {
    throw new Error(
      `Admin ${ADMIN_EMAIL} missing or not PLATFORM_ADMIN — run:\n` +
        `  docker exec civicos_postgres psql -U civicos -d civicos ` +
        `-c "UPDATE users SET role='PLATFORM_ADMIN' WHERE email='${ADMIN_EMAIL}';"`,
    );
  }
  const [id, name] = row.split('|');
  return {
    id,
    email: ADMIN_EMAIL,
    name,
    role: 'PLATFORM_ADMIN',
    emailVerified: true,
    accessToken: craftJwt(id, ADMIN_EMAIL, name, 'PLATFORM_ADMIN', true),
  };
}

// Inserts a fresh e2e citizen for this test run. Password is bcrypt of
// 'password' (kept for parity even though we never log in via password
// — the crafted JWT is enough).
const BCRYPT_PW = '$2a$12$0PjLd9ZS/mQEXvUx8LtVjOtaIWTjNXO7v0FBrqXjR3aNVBw/wtFxa';

export function createTestCitizen(tag = 'run'): TestUser {
  const email = `e2e-${tag}-${Date.now()}${Math.floor(Math.random() * 1000)}@civicos.test`;
  const name = `E2E Test ${tag}`;
  const id = sql(
    `INSERT INTO users (id,email,password_hash,name,role,email_verified,created_at,updated_at)
     VALUES (gen_random_uuid(),'${email}','${BCRYPT_PW}','${name}','CITIZEN',true,now(),now())
     RETURNING id;`,
  );
  if (!id) throw new Error(`citizen insert returned no id`);
  return {
    id,
    email,
    name,
    role: 'CITIZEN',
    emailVerified: true,
    accessToken: craftJwt(id, email, name, 'CITIZEN', true),
  };
}

// Ensures a shared e2e community exists (creates it if it doesn't) and
// returns its id. Used by specs that need a citizen with membership so
// the dashboard content pages actually render past the CommunityGate.
export function ensureE2eCommunity(): string {
  const admin = loadAdmin();
  let id = sql(`SELECT id FROM communities WHERE slug='e2e-shared';`);
  if (!id) {
    id = sql(
      `INSERT INTO communities (id,name,slug,state,lga,country,created_by_id,created_at,updated_at)
       VALUES (gen_random_uuid(),'E2E Shared','e2e-shared','Lagos','Ikeja','Nigeria','${admin.id}',now(),now())
       RETURNING id;`,
    );
  }
  return id;
}

// Creates a citizen and pins them to the shared e2e community in the same
// insert — saves the specs from having to click through onboarding.
export function createTestCitizenInCommunity(tag = 'run'): TestUser {
  const communityId = ensureE2eCommunity();
  const email = `e2e-${tag}-${Date.now()}${Math.floor(Math.random() * 1000)}@civicos.test`;
  const name = `E2E Test ${tag}`;
  const id = sql(
    `INSERT INTO users (id,email,password_hash,name,role,community_id,email_verified,created_at,updated_at)
     VALUES (gen_random_uuid(),'${email}','${BCRYPT_PW}','${name}','CITIZEN','${communityId}',true,now(),now())
     RETURNING id;`,
  );
  if (!id) throw new Error('citizen-in-community insert returned no id');
  return {
    id,
    email,
    name,
    role: 'CITIZEN',
    emailVerified: true,
    accessToken: craftJwt(id, email, name, 'CITIZEN', true),
  };
}

// Signs a JWT with the shared secret. Payload shape matches identity-
// service Claims struct so tokens validate at both the gateway and each
// downstream service.
export function craftJwt(
  sub: string,
  email: string,
  name: string,
  role: string,
  emailVerified: boolean,
): string {
  const u = (b: string | Buffer) =>
    Buffer.from(b).toString('base64').replace(/=/g, '').replace(/\+/g, '-').replace(/\//g, '_');
  const header = u(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const now = Math.floor(Date.now() / 1000);
  const payload = u(
    JSON.stringify({
      sub,
      email,
      name,
      role,
      emailVerified,
      iat: now,
      exp: now + 3600,
    }),
  );
  const sig = createHmac('sha256', jwtSecret())
    .update(`${header}.${payload}`)
    .digest('base64')
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_');
  return `${header}.${payload}.${sig}`;
}
