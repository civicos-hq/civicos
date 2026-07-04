import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

// Shared repo-root .env is the source of truth for the JWT secret.
const ENV_FILE = process.env.CIVICOS_ENV_FILE ?? '/Users/gino/civicos/.env';
const DB_CONTAINER = process.env.DB_CONTAINER ?? 'civicos_postgres';

let cachedSecret: string | null = null;
export function jwtSecret(): string {
  if (cachedSecret) return cachedSecret;
  const line = readFileSync(ENV_FILE, 'utf8')
    .split('\n')
    .find((l) => l.startsWith('JWT_SECRET='));
  if (!line) throw new Error(`JWT_SECRET missing from ${ENV_FILE}`);
  cachedSecret = line.slice('JWT_SECRET='.length).replace(/^"/, '').replace(/"$/, '');
  return cachedSecret;
}

// Same pattern as apps/web/e2e — head -n1 avoids psql's "INSERT 0 1"
// command-tag pollution on RETURNING clauses.
export function sql(query: string): string {
  const out = execFileSync(
    'docker',
    ['exec', DB_CONTAINER, 'psql', '-U', 'civicos', '-d', 'civicos', '-tAc', query],
    { encoding: 'utf8' },
  );
  return out.split('\n')[0].replace(/\r/g, '').trim();
}

export function sqlBatch(statements: string): void {
  execFileSync(
    'docker',
    ['exec', '-i', DB_CONTAINER, 'psql', '-U', 'civicos', '-d', 'civicos', '-q'],
    { encoding: 'utf8', input: statements },
  );
}

// Removes every admin-e2e-created record. Uses email/slug prefixes to
// isolate suite data; leaves genuine records untouched.
export function purgeAdminE2eData(): void {
  sqlBatch(`
    DELETE FROM audit_logs WHERE target_id IN (SELECT id FROM users WHERE email LIKE 'admin-e2e-%@civicos.test');
    DELETE FROM content_flags WHERE reporter_id IN (SELECT id FROM users WHERE email LIKE 'admin-e2e-%@civicos.test');
    DELETE FROM content_flags WHERE content_id::text LIKE 'ade2e%';
    DELETE FROM organizations WHERE slug LIKE 'admin-e2e-%';
    DELETE FROM users WHERE email LIKE 'admin-e2e-%@civicos.test';
  `);
}
