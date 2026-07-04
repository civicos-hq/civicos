import { createHmac } from 'node:crypto';
import { sql, jwtSecret } from './db';

export interface TestUser {
  id: string;
  email: string;
  name: string;
  role: string;
  emailVerified: boolean;
  accessToken: string;
}

const ADMIN_EMAIL = process.env.CIVICOS_ADMIN_EMAIL ?? 'gino.osahon@gmail.com';

export function loadAdmin(): TestUser {
  const row = sql(
    `SELECT id||'|'||name FROM users WHERE email='${ADMIN_EMAIL}' AND role='PLATFORM_ADMIN';`,
  );
  if (!row) {
    throw new Error(
      `Admin ${ADMIN_EMAIL} missing or not PLATFORM_ADMIN — the admin e2e suite requires a live PLATFORM_ADMIN account`,
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

const BCRYPT_PW = '$2a$12$0PjLd9ZS/mQEXvUx8LtVjOtaIWTjNXO7v0FBrqXjR3aNVBw/wtFxa';

export function createTestCitizen(tag = 'run'): TestUser {
  const email = `admin-e2e-${tag}-${Date.now()}${Math.floor(Math.random() * 1000)}@civicos.test`;
  const name = `E2E Target ${tag}`;
  const id = sql(
    `INSERT INTO users (id,email,password_hash,name,role,email_verified,created_at,updated_at)
     VALUES (gen_random_uuid(),'${email}','${BCRYPT_PW}','${name}','CITIZEN',true,now(),now())
     RETURNING id;`,
  );
  if (!id) throw new Error('citizen insert returned no id');
  return {
    id,
    email,
    name,
    role: 'CITIZEN',
    emailVerified: true,
    accessToken: craftJwt(id, email, name, 'CITIZEN', true),
  };
}

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
    JSON.stringify({ sub, email, name, role, emailVerified, iat: now, exp: now + 3600 }),
  );
  const sig = createHmac('sha256', jwtSecret())
    .update(`${header}.${payload}`)
    .digest('base64')
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_');
  return `${header}.${payload}.${sig}`;
}
